package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var routerHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   4 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   4 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          128,
	},
}

func proxyToProviderBaseURL(w http.ResponseWriter, r *http.Request, providerBaseURL string) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	base := strings.TrimRight(strings.TrimSpace(providerBaseURL), "/")
	if base == "" {
		writeJSONError(w, http.StatusBadGateway, "provider base url is empty", "")
		return
	}

	target := base + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create provider request", err.Error())
		return
	}
	req.Header = r.Header.Clone()
	req.Header.Set(gatewayAuthHeader, gatewayToProviderAuthToken())

	resp, err := routerHTTPClient.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to contact provider", err.Error())
		return
	}
	defer resp.Body.Close()

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	// Ensure CORS headers are always present on responses returned via the router.
	setCORS(w)
	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		return
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func tryProxyToProviderBaseURL(w http.ResponseWriter, r *http.Request, providerBaseURL string) (bool, error) {
	base := strings.TrimRight(strings.TrimSpace(providerBaseURL), "/")
	if base == "" {
		return false, fmt.Errorf("provider base url is empty")
	}

	target := base + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, target, nil)
	if err != nil {
		return false, err
	}
	req.Header = r.Header.Clone()
	req.Header.Set(gatewayAuthHeader, gatewayToProviderAuthToken())

	resp, err := routerHTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Slot mismatch is a router concern (Mode 2): treat it as a routing miss so we can
	// try another provider without leaking a confusing 400 back to the client.
	if resp.StatusCode == http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		body := strings.TrimSpace(string(bodyBytes))
		if strings.Contains(body, "provider slot mismatch") {
			return false, fmt.Errorf("provider slot mismatch: %s", body)
		}

		// Not a routing miss: forward the 400 response to the client.
		for k, vals := range resp.Header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		setCORS(w)
		w.WriteHeader(resp.StatusCode)
		_, copyErr := w.Write(bodyBytes)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return true, copyErr
	}

	// If this is an on-chain session fetch, a non-assigned provider will reject the
	// session unless deputy mode is enabled. Treat that as a routing miss so the router
	// can retry the request with `deputy=1` (or try other providers).
	if resp.StatusCode == http.StatusForbidden {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		body := strings.TrimSpace(string(bodyBytes))
		if strings.TrimSpace(r.Header.Get("X-Nil-Session-Id")) != "" && strings.Contains(body, "session provider mismatch") {
			return false, fmt.Errorf("session provider mismatch: %s", body)
		}

		for k, vals := range resp.Header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		setCORS(w)
		w.WriteHeader(resp.StatusCode)
		_, copyErr := w.Write(bodyBytes)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return true, copyErr
	}

	// If the provider is reachable but returns a 5xx, attempt failover to the next candidate.
	if resp.StatusCode >= http.StatusInternalServerError {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		body := strings.TrimSpace(string(bodyBytes))
		if body == "" {
			body = resp.Status
		}
		return false, fmt.Errorf("provider returned %d: %s", resp.StatusCode, body)
	}

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	setCORS(w)
	w.WriteHeader(resp.StatusCode)

	_, copyErr := io.Copy(w, resp.Body)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return true, copyErr
}

type proxyAskRequest struct {
	DealID     uint64 `json:"deal_id"`
	Provider   string `json:"provider"`
	FilePath   string `json:"file_path"`
	RangeStart uint64 `json:"range_start"`
	RangeLen   uint64 `json:"range_len"`
	MaxPrice   uint64 `json:"max_price"`
}

type proxyAskResponse struct {
	RequestID      string `json:"request_id"`
	DeputyPeerID   string `json:"deputy_peer_id"`
	DeputyEndpoint string `json:"deputy_endpoint"`
	Price          uint64 `json:"price"`
}

var proxyFailureNow = time.Now

func proxyMaxPrice() uint64 {
	raw := strings.TrimSpace(envDefault("NIL_PROXY_MAX_PRICE", "0"))
	if raw == "" {
		return 0
	}
	if v, err := strconv.ParseUint(raw, 10, 64); err == nil {
		return v
	}
	return 0
}

func proxyReportFailuresEnabled() bool {
	return strings.TrimSpace(envDefault("NIL_PROXY_REPORT_FAILURES", "1")) == "1"
}

func askForProxyDeputy(ctx context.Context, req proxyAskRequest) (*proxyAskResponse, error) {
	base := strings.TrimRight(strings.TrimSpace(envDefault("NIL_P2P_PROXY_URL", "")), "/")
	if base == "" {
		return nil, errors.New("proxy discovery disabled")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/proxy/ask", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := routerHTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("proxy discovery %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var payload proxyAskResponse
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, err
	}
	payload.DeputyEndpoint = strings.TrimSpace(payload.DeputyEndpoint)
	if payload.DeputyEndpoint == "" {
		return nil, fmt.Errorf("proxy discovery returned empty deputy endpoint")
	}
	return &payload, nil
}

func proxyFailureHash(dealID uint64, provider string, filePath string, rangeStart uint64, rangeLen uint64) []byte {
	now := proxyFailureNow().UnixNano()
	h := sha256.New()
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], dealID)
	_, _ = h.Write(buf[:])
	_, _ = h.Write([]byte(provider))
	_, _ = h.Write([]byte(filePath))
	binary.BigEndian.PutUint64(buf[:], rangeStart)
	_, _ = h.Write(buf[:])
	binary.BigEndian.PutUint64(buf[:], rangeLen)
	_, _ = h.Write(buf[:])
	binary.BigEndian.PutUint64(buf[:], uint64(now))
	_, _ = h.Write(buf[:])
	return h.Sum(nil)
}

func submitProofOfFailure(ctx context.Context, dealID uint64, provider string, proofHash []byte) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(proofHash) != 32 {
		return "", fmt.Errorf("proof hash must be 32 bytes")
	}
	providerKeyName := envDefault("NIL_PROVIDER_KEY", "faucet")
	proofHashHex := "0x" + hex.EncodeToString(proofHash)
	dealIDStr := strconv.FormatUint(dealID, 10)

	out, err := runTxWithRetry(
		ctx,
		"tx", "nilchain", "submit-proof-of-failure",
		dealIDStr,
		provider,
		proofHashHex,
		"--from", providerKeyName,
		"--chain-id", chainID,
		"--home", homeDir,
		"--keyring-backend", "test",
		"--yes",
		"--gas", "auto",
		"--gas-adjustment", "1.6",
		"--gas-prices", gasPrices,
	)
	outStr := string(out)
	if err != nil {
		return "", fmt.Errorf("submit-proof-of-failure failed: %w (%s)", err, outStr)
	}
	return extractTxHash(outStr), nil
}

func proxyRequestRange(r *http.Request) (uint64, uint64) {
	if r == nil {
		return 0, 0
	}
	rangeHeader := strings.TrimSpace(r.Header.Get("Range"))
	if rangeHeader == "" {
		return 0, 0
	}
	start, length, err := parseHTTPRange(rangeHeader)
	if err != nil {
		return 0, 0
	}
	return start, length
}

func bufferRequestBody(r *http.Request) (*os.File, int64, error) {
	tmpFile, err := os.CreateTemp("", "nil-router-upload-*")
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if err != nil {
			_ = tmpFile.Close()
		}
	}()

	size, copyErr := io.Copy(tmpFile, r.Body)
	if copyErr != nil {
		err = copyErr
		return nil, 0, err
	}
	if closeErr := r.Body.Close(); closeErr != nil && err == nil {
		err = closeErr
		return nil, 0, err
	}
	if _, seekErr := tmpFile.Seek(0, io.SeekStart); seekErr != nil {
		err = seekErr
		return nil, 0, err
	}
	return tmpFile, size, nil
}

func tryProxyUploadToProviderBaseURL(w http.ResponseWriter, r *http.Request, providerBaseURL string, bodyFile *os.File, contentLength int64) (bool, error) {
	base := strings.TrimRight(strings.TrimSpace(providerBaseURL), "/")
	if base == "" {
		return false, fmt.Errorf("provider base url is empty")
	}

	// Use a SectionReader so the HTTP client doesn't close the shared temp file
	// across provider retries.
	bodyReader := io.NewSectionReader(bodyFile, 0, contentLength)

	target := base + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}

	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, r.Method, target, bodyReader)
	if err != nil {
		return false, err
	}
	req.Header = r.Header.Clone()
	req.Header.Set(gatewayAuthHeader, gatewayToProviderAuthToken())
	req.ContentLength = contentLength

	resp, err := routerHTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// If the provider is reachable but returns a 5xx, attempt failover to the next candidate.
	if resp.StatusCode >= http.StatusInternalServerError {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		body := strings.TrimSpace(string(bodyBytes))
		if body == "" {
			body = resp.Status
		}
		return false, fmt.Errorf("provider returned %d: %s", resp.StatusCode, body)
	}

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	setCORS(w)
	w.WriteHeader(resp.StatusCode)

	_, copyErr := io.Copy(w, resp.Body)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return true, copyErr
}

func requireDealIDQuery(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("deal_id"))
	if raw == "" {
		writeJSONError(w, http.StatusBadRequest, "deal_id query parameter is required", "")
		return 0, false
	}
	dealID, err := parseDealID(raw)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid deal_id", "")
		return 0, false
	}
	return dealID, true
}

func requireUploadIDQuery(w http.ResponseWriter, r *http.Request) (string, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("upload_id"))
	if raw == "" {
		writeJSONError(w, http.StatusBadRequest, "upload_id query parameter is required", "")
		return "", false
	}
	return raw, true
}

func parseDealID(raw string) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty")
	}
	return strconv.ParseUint(raw, 10, 64)
}

func RouterGatewayFetch(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	dealID, ok := requireDealIDQuery(w, r)
	if !ok {
		return
	}
	providers, err := resolveDealProviders(r.Context(), dealID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrDealNotFound) {
			status = http.StatusNotFound
		}
		writeJSONError(w, status, "failed to resolve deal providers", err.Error())
		return
	}

	// For Mode 2 deals, the router doesn't have enough local context to compute the
	// exact slot provider for an arbitrary file range. Instead, it tries each
	// assigned provider until it finds one that can serve the request. If all
	// providers reject with a slot mismatch (or if the correct provider is down),
	// fall back to "deputy" mode which allows any provider to reconstruct from K
	// shards and serve the request.
	isFetch := strings.HasPrefix(r.URL.Path, "/gateway/fetch/")
	origRawQuery := r.URL.RawQuery
	primaryProvider := ""
	if len(providers) > 0 {
		primaryProvider = providers[0]
	}
	rangeStart, rangeLen := proxyRequestRange(r)

	var lastErr error
	for _, providerAddr := range providers {
		baseURL, err := resolveProviderHTTPBaseURL(r.Context(), providerAddr)
		if err != nil {
			lastErr = err
			continue
		}
		ok, err := tryProxyToProviderBaseURL(w, r, baseURL)
		if ok {
			dealProviderCache.Store(dealID, &dealProviderCacheEntry{
				provider: providerAddr,
				expires:  time.Now().Add(dealProviderTTL),
			})
			if err != nil {
				// The provider response already started streaming to the client.
				// We can't safely failover, but keep the error for logging/visibility.
				lastErr = err
			}
			return
		}
		if err != nil {
			lastErr = err
		}
	}

	if isFetch {
		q := r.URL.Query()
		if strings.TrimSpace(q.Get("deputy")) == "" {
			q.Set("deputy", "1")
			r.URL.RawQuery = q.Encode()
		}

		for _, providerAddr := range providers {
			baseURL, err := resolveProviderHTTPBaseURL(r.Context(), providerAddr)
			if err != nil {
				lastErr = err
				continue
			}
			ok, err := tryProxyToProviderBaseURL(w, r, baseURL)
			if ok {
				dealProviderCache.Store(dealID, &dealProviderCacheEntry{
					provider: providerAddr,
					expires:  time.Now().Add(dealProviderTTL),
				})
				if err != nil {
					lastErr = err
				}
				r.URL.RawQuery = origRawQuery
				return
			}
			if err != nil {
				lastErr = err
			}
		}
	}

	if isFetch {
		filePath := strings.TrimSpace(r.URL.Query().Get("file_path"))
		proxyResp, err := askForProxyDeputy(r.Context(), proxyAskRequest{
			DealID:     dealID,
			Provider:   primaryProvider,
			FilePath:   filePath,
			RangeStart: rangeStart,
			RangeLen:   rangeLen,
			MaxPrice:   proxyMaxPrice(),
		})
		if err == nil && proxyResp != nil {
			q := r.URL.Query()
			if strings.TrimSpace(q.Get("deputy")) == "" {
				q.Set("deputy", "1")
				r.URL.RawQuery = q.Encode()
			}
			ok, proxyErr := tryProxyToProviderBaseURL(w, r, proxyResp.DeputyEndpoint)
			if ok {
				r.URL.RawQuery = origRawQuery
				return
			}
			if proxyErr != nil {
				lastErr = proxyErr
			}
			if proxyReportFailuresEnabled() && primaryProvider != "" {
				hash := proxyFailureHash(dealID, primaryProvider, filePath, rangeStart, rangeLen)
				if _, err := submitProofOfFailure(r.Context(), dealID, primaryProvider, hash); err != nil {
					lastErr = err
				}
			}
		} else if err != nil {
			lastErr = err
		}
	}

	msg := "failed to contact provider"
	detail := ""
	if lastErr != nil {
		detail = lastErr.Error()
	}
	writeJSONError(w, http.StatusBadGateway, msg, detail)

	if isFetch {
		r.URL.RawQuery = origRawQuery
	}
}

func RouterGatewayListFiles(w http.ResponseWriter, r *http.Request) { RouterGatewayFetch(w, r) }
func RouterGatewaySlab(w http.ResponseWriter, r *http.Request)      { RouterGatewayFetch(w, r) }
func RouterGatewayManifestInfo(w http.ResponseWriter, r *http.Request) {
	RouterGatewayFetch(w, r)
}
func RouterGatewayMduKzg(w http.ResponseWriter, r *http.Request) { RouterGatewayFetch(w, r) }
func RouterGatewayDebugRawFetch(w http.ResponseWriter, r *http.Request) {
	RouterGatewayFetch(w, r)
}
func RouterGatewayPlanRetrievalSession(w http.ResponseWriter, r *http.Request) {
	RouterGatewayFetch(w, r)
}
func RouterGatewayOpenSession(w http.ResponseWriter, r *http.Request) {
	RouterGatewayFetch(w, r)
}

func RouterGatewayUpload(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// NOTE: In router mode we require deal_id in the URL query string so we can
	// route without parsing the multipart body. The provider will still accept
	// deal_id in the multipart form for compatibility.
	dealID, ok := requireDealIDQuery(w, r)
	if !ok {
		return
	}
	providers, err := resolveDealProviders(r.Context(), dealID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrDealNotFound) {
			status = http.StatusNotFound
		}
		writeJSONError(w, status, "failed to resolve deal providers", err.Error())
		return
	}

	tmpFile, size, err := bufferRequestBody(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read upload body", err.Error())
		return
	}
	defer func() {
		tmpPath := tmpFile.Name()
		if err := tmpFile.Close(); err == nil {
			_ = os.Remove(tmpPath)
		}
	}()

	var lastErr error
	for _, providerAddr := range providers {
		baseURL, err := resolveProviderHTTPBaseURL(r.Context(), providerAddr)
		if err != nil {
			lastErr = err
			continue
		}
		ok, err := tryProxyUploadToProviderBaseURL(w, r, baseURL, tmpFile, size)
		if ok {
			dealProviderCache.Store(dealID, &dealProviderCacheEntry{
				provider: providerAddr,
				expires:  time.Now().Add(dealProviderTTL),
			})
			return
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no providers available")
	}
	writeJSONError(w, http.StatusBadGateway, "failed to contact provider", lastErr.Error())
}

func RouterGatewayUploadStatus(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	dealID, ok := requireDealIDQuery(w, r)
	if !ok {
		return
	}
	if _, ok := requireUploadIDQuery(w, r); !ok {
		return
	}

	providerAddr, err := resolveDealAssignedProvider(r.Context(), dealID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrDealNotFound) {
			status = http.StatusNotFound
		}
		writeJSONError(w, status, "failed to resolve deal provider", err.Error())
		return
	}
	baseURL, err := resolveProviderHTTPBaseURL(r.Context(), providerAddr)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to resolve provider endpoint", err.Error())
		return
	}
	proxyToProviderBaseURL(w, r, baseURL)
}

func forwardJSONToProviderBase(w http.ResponseWriter, r *http.Request, providerBaseURL string, path string, body []byte) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	target := strings.TrimRight(strings.TrimSpace(providerBaseURL), "/") + path
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create provider request", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(gatewayAuthHeader, gatewayToProviderAuthToken())

	resp, err := routerHTTPClient.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to contact provider", err.Error())
		return
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	if ct := strings.TrimSpace(resp.Header.Get("Content-Type")); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(out)
}

func RouterGatewaySubmitReceipt(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read body", err.Error())
		return
	}

	var env struct {
		Receipt struct {
			Provider string `json:"provider"`
		} `json:"receipt"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON", "")
		return
	}
	providerAddr := strings.TrimSpace(env.Receipt.Provider)
	if providerAddr == "" {
		writeJSONError(w, http.StatusBadRequest, "receipt.provider is required", "")
		return
	}

	baseURL, err := resolveProviderHTTPBaseURL(r.Context(), providerAddr)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to resolve provider endpoint", err.Error())
		return
	}
	forwardJSONToProviderBase(w, r, baseURL, "/sp/receipt", body)
}

func RouterGatewaySubmitReceipts(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read body", err.Error())
		return
	}

	var env struct {
		Receipts []struct {
			Receipt struct {
				Provider string `json:"provider"`
			} `json:"receipt"`
		} `json:"receipts"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON", "")
		return
	}
	providerAddr := ""
	for _, item := range env.Receipts {
		p := strings.TrimSpace(item.Receipt.Provider)
		if p == "" {
			continue
		}
		if providerAddr == "" {
			providerAddr = p
			continue
		}
		if providerAddr != p {
			writeJSONError(w, http.StatusBadRequest, "batch must target a single provider", "")
			return
		}
	}
	if providerAddr == "" {
		writeJSONError(w, http.StatusBadRequest, "receipt.provider is required", "")
		return
	}

	baseURL, err := resolveProviderHTTPBaseURL(r.Context(), providerAddr)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to resolve provider endpoint", err.Error())
		return
	}
	forwardJSONToProviderBase(w, r, baseURL, "/sp/receipts", body)
}

func RouterGatewaySubmitSessionReceipt(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read body", err.Error())
		return
	}

	var env struct {
		Receipt struct {
			Provider string `json:"provider"`
		} `json:"receipt"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON", "")
		return
	}
	providerAddr := strings.TrimSpace(env.Receipt.Provider)
	if providerAddr == "" {
		writeJSONError(w, http.StatusBadRequest, "receipt.provider is required", "")
		return
	}

	baseURL, err := resolveProviderHTTPBaseURL(r.Context(), providerAddr)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to resolve provider endpoint", err.Error())
		return
	}
	forwardJSONToProviderBase(w, r, baseURL, "/sp/session-receipt", body)
}

func RouterGatewaySubmitRetrievalSessionProof(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read body", err.Error())
		return
	}

	var env struct {
		SessionID string `json:"session_id"`
		Provider  string `json:"provider"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON", "expected {session_id, provider}")
		return
	}

	providerAddr := strings.TrimSpace(env.Provider)
	if providerAddr == "" {
		writeJSONError(w, http.StatusBadRequest, "provider is required", "")
		return
	}

	baseURL, err := resolveProviderHTTPBaseURL(r.Context(), providerAddr)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to resolve provider endpoint", err.Error())
		return
	}
	forwardJSONToProviderBase(w, r, baseURL, "/sp/session-proof", body)
}

func isGatewayRouterMode() bool {
	raw := strings.TrimSpace(os.Getenv("NIL_GATEWAY_ROUTER"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("NIL_GATEWAY_ROUTER_MODE"))
	}
	return raw == "1" || strings.EqualFold(raw, "true")
}
