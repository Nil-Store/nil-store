package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"nilchain/x/crypto_ffi"
)

func TestGateway_Mode2_Append_PreservesExistingFiles(t *testing.T) {
	dealProviderCache = sync.Map{}
	providerBaseCache = sync.Map{}

	useTempUploadDir(t)
	if err := crypto_ffi.Init(trustedSetup); err != nil {
		t.Fatalf("crypto_ffi.Init failed: %v", err)
	}

	oldReqSig := requireRetrievalReqSig
	requireRetrievalReqSig = false
	t.Cleanup(func() { requireRetrievalReqSig = oldReqSig })

	dealID := uint64(99)
	owner := testDealOwner(t)

	// 12 providers: slot 0..11
	providers := make([]string, 0, 12)
	endpoints := map[string]string{}
	for i := 0; i < 12; i++ {
		addr := "nil1provider" + strconv.Itoa(i)
		providers = append(providers, addr)
		srv, _ := newProviderServer(t)
		endpoints[addr] = srv.URL
	}

	state := &mode2DealState{
		owner:       owner,
		cid:         "",
		serviceHint: "General:replicas=12:rs=8+4",
		providers:   providers,
		endpoints:   endpoints,
	}

	lcdSrv := newMode2LCDServer(t, dealID, state)
	defer lcdSrv.Close()
	oldLCD := lcdBase
	lcdBase = lcdSrv.URL
	t.Cleanup(func() { lcdBase = oldLCD })

	// Gateway acts as the slot 0 provider for proof headers.
	t.Setenv("NIL_PROVIDER_ADDRESS", providers[0])

	upload := func(name string, payload []byte) string {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", name)
		_, _ = part.Write(payload)
		_ = writer.WriteField("deal_id", strconv.FormatUint(dealID, 10))
		_ = writer.WriteField("file_path", name)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/gateway/upload?deal_id="+strconv.FormatUint(dealID, 10), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		testRouter().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GatewayUpload(%s) failed: %d %s", name, w.Code, w.Body.String())
		}
		var resp struct {
			ManifestRoot string `json:"manifest_root"`
			CID          string `json:"cid"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		root := strings.TrimSpace(resp.ManifestRoot)
		if root == "" {
			root = strings.TrimSpace(resp.CID)
		}
		if root == "" {
			t.Fatalf("missing manifest_root in upload response: %s", w.Body.String())
		}
		return root
	}

	// First upload (new deal content).
	cidA := upload("a.txt", bytes.Repeat([]byte("A"), 32*1024))
	state.setCID(cidA)

	// Second upload appends.
	cidB := upload("b.txt", bytes.Repeat([]byte("B"), 32*1024))
	state.setCID(cidB)

	// List files using the latest manifest root.
	listReq := httptest.NewRequest(http.MethodGet, "/gateway/list-files/"+cidB+"?deal_id="+strconv.FormatUint(dealID, 10)+"&owner="+owner, nil)
	listW := httptest.NewRecorder()
	testRouter().ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("GatewayListFiles failed: %d %s", listW.Code, listW.Body.String())
	}
	var payload struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &payload); err != nil {
		t.Fatalf("list-files JSON invalid: %v (%s)", err, listW.Body.String())
	}
	seen := map[string]bool{}
	for _, f := range payload.Files {
		seen[f.Path] = true
	}
	if !seen["a.txt"] || !seen["b.txt"] {
		t.Fatalf("expected both a.txt and b.txt, got %v", seen)
	}
}

