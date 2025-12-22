package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
)

type mirrorProvider struct {
	addr         string
	multiaddr    string
	mu           sync.Mutex
	mduCalls     int
	manifestCalls int
	shardCalls   int
	lastAuth     string
	lastBody     []byte
	lastPath     string
}

func newMirrorProvider(t *testing.T, addr string) *mirrorProvider {
	t.Helper()
	p := &mirrorProvider{addr: addr}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.lastAuth = strings.TrimSpace(r.Header.Get(gatewayAuthHeader))
		p.lastPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		p.lastBody = body
		switch r.URL.Path {
		case "/sp/upload_mdu":
			p.mduCalls++
		case "/sp/upload_manifest":
			p.manifestCalls++
		case "/sp/upload_shard":
			p.shardCalls++
		default:
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	p.multiaddr = mustHTTPMultiaddr(t, srv.URL)
	return p
}

func TestRouterGatewayMirror_BroadcastsMetadataAndTargetsShardSlot(t *testing.T) {
	dealProviderCache = sync.Map{}
	providerBaseCache = sync.Map{}

	// Providers in deal order correspond to slot indices.
	p0 := newMirrorProvider(t, "nil1p0")
	p1 := newMirrorProvider(t, "nil1p1")
	p2 := newMirrorProvider(t, "nil1p2")
	providers := []*mirrorProvider{p0, p1, p2}

	lcdSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/nilchain/nilchain/v1/deals/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"deal": map[string]any{
					"providers": []string{p0.addr, p1.addr, p2.addr},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/nilchain/nilchain/v1/providers/"):
			addr := strings.TrimPrefix(r.URL.Path, "/nilchain/nilchain/v1/providers/")
			for _, p := range providers {
				if p.addr == addr {
					_ = json.NewEncoder(w).Encode(map[string]any{
						"provider": map[string]any{
							"endpoints": []string{p.multiaddr},
						},
					})
					return
				}
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer lcdSrv.Close()

	oldLCD := lcdBase
	lcdBase = lcdSrv.URL
	t.Cleanup(func() { lcdBase = oldLCD })

	r := mux.NewRouter()
	r.HandleFunc("/gateway/mirror_mdu", RouterGatewayMirrorMdu).Methods("POST", "OPTIONS")
	r.HandleFunc("/gateway/mirror_manifest", RouterGatewayMirrorManifest).Methods("POST", "OPTIONS")
	r.HandleFunc("/gateway/mirror_shard", RouterGatewayMirrorShard).Methods("POST", "OPTIONS")

	const dealID = "1"
	const manifestRoot = "0xabc"

	// Mirror MDU to all providers.
	{
		req := httptest.NewRequest(http.MethodPost, "/gateway/mirror_mdu", bytes.NewReader([]byte("mdu")))
		req.Header.Set("X-Nil-Deal-ID", dealID)
		req.Header.Set("X-Nil-Mdu-Index", "0")
		req.Header.Set("X-Nil-Manifest-Root", manifestRoot)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("mirror_mdu: expected 200, got %d (%s)", w.Code, w.Body.String())
		}
		for _, p := range providers {
			p.mu.Lock()
			calls := p.mduCalls
			auth := p.lastAuth
			p.mu.Unlock()
			if calls != 1 {
				t.Fatalf("mirror_mdu: provider %s expected 1 call, got %d", p.addr, calls)
			}
			if strings.TrimSpace(auth) != strings.TrimSpace(gatewayToProviderAuthToken()) {
				t.Fatalf("mirror_mdu: provider %s expected auth header", p.addr)
			}
		}
	}

	// Mirror manifest to all providers.
	{
		req := httptest.NewRequest(http.MethodPost, "/gateway/mirror_manifest", bytes.NewReader([]byte("manifest")))
		req.Header.Set("X-Nil-Deal-ID", dealID)
		req.Header.Set("X-Nil-Manifest-Root", manifestRoot)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("mirror_manifest: expected 200, got %d (%s)", w.Code, w.Body.String())
		}
		for _, p := range providers {
			p.mu.Lock()
			calls := p.manifestCalls
			p.mu.Unlock()
			if calls != 1 {
				t.Fatalf("mirror_manifest: provider %s expected 1 call, got %d", p.addr, calls)
			}
		}
	}

	// Mirror shard targets only the requested slot provider.
	{
		req := httptest.NewRequest(http.MethodPost, "/gateway/mirror_shard", bytes.NewReader([]byte("shard")))
		req.Header.Set("X-Nil-Deal-ID", dealID)
		req.Header.Set("X-Nil-Mdu-Index", "1")
		req.Header.Set("X-Nil-Slot", "1")
		req.Header.Set("X-Nil-Manifest-Root", manifestRoot)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("mirror_shard: expected 200, got %d (%s)", w.Code, w.Body.String())
		}
		p0.mu.Lock()
		p0Calls := p0.shardCalls
		p0.mu.Unlock()
		p1.mu.Lock()
		p1Calls := p1.shardCalls
		p1.mu.Unlock()
		p2.mu.Lock()
		p2Calls := p2.shardCalls
		p2.mu.Unlock()
		if p0Calls != 0 || p2Calls != 0 || p1Calls != 1 {
			t.Fatalf("mirror_shard: expected only slot provider to receive shard (p0=%d p1=%d p2=%d)", p0Calls, p1Calls, p2Calls)
		}
	}
}
