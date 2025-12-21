package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const routerMirrorTimeout = 45 * time.Second

type routerMirrorResult struct {
	Status          string   `json:"status"`
	Forwarded       int      `json:"forwarded"`
	FailedProviders []string `json:"failed_providers,omitempty"`
	Error           string   `json:"error,omitempty"`
}

func RouterMirrorMdu(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	dealIDStr := strings.TrimSpace(r.Header.Get("X-Nil-Deal-ID"))
	mduIndexStr := strings.TrimSpace(r.Header.Get("X-Nil-Mdu-Index"))
	rawManifestRoot := strings.TrimSpace(r.Header.Get("X-Nil-Manifest-Root"))
	if dealIDStr == "" || mduIndexStr == "" || rawManifestRoot == "" {
		http.Error(w, "X-Nil-Deal-ID, X-Nil-Mdu-Index, and X-Nil-Manifest-Root headers are required", http.StatusBadRequest)
		return
	}

	dealID, err := strconv.ParseUint(dealIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid deal_id", http.StatusBadRequest)
		return
	}

	// Validate deal exists on-chain.
	if _, _, err := fetchDealOwnerAndCID(dealID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to validate deal", err.Error())
		return
	}

	manifestRoot, err := parseManifestRoot(rawManifestRoot)
	if err != nil {
		http.Error(w, "invalid manifest root", http.StatusBadRequest)
		return
	}

	body, err := readLimitedBody(w, r, 10<<20)
	if err != nil {
		return
	}

	if err := storeRouterObject(manifestRoot.Key, fmt.Sprintf("mdu_%s.bin", mduIndexStr), body); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to store mdu", err.Error())
		return
	}

	providers, err := fetchDealProvidersFromLCD(r.Context(), dealID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve providers", err.Error())
		return
	}

	forwarded, failed := forwardToProviders(r.Context(), providers, func(base string) (*http.Request, error) {
		ctx, cancel := context.WithTimeout(r.Context(), routerMirrorTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/sp/upload_mdu", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Nil-Deal-ID", dealIDStr)
		req.Header.Set("X-Nil-Mdu-Index", mduIndexStr)
		req.Header.Set("X-Nil-Manifest-Root", manifestRoot.Canonical)
		req.Header.Set("Content-Type", "application/octet-stream")
		return req, nil
	})

	respondMirrorResult(w, forwarded, failed, "")
}

func RouterMirrorManifest(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	dealIDStr := strings.TrimSpace(r.Header.Get("X-Nil-Deal-ID"))
	rawManifestRoot := strings.TrimSpace(r.Header.Get("X-Nil-Manifest-Root"))
	if dealIDStr == "" || rawManifestRoot == "" {
		http.Error(w, "X-Nil-Deal-ID and X-Nil-Manifest-Root headers are required", http.StatusBadRequest)
		return
	}

	dealID, err := strconv.ParseUint(dealIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid deal_id", http.StatusBadRequest)
		return
	}

	if _, _, err := fetchDealOwnerAndCID(dealID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to validate deal", err.Error())
		return
	}

	manifestRoot, err := parseManifestRoot(rawManifestRoot)
	if err != nil {
		http.Error(w, "invalid manifest root", http.StatusBadRequest)
		return
	}

	body, err := readLimitedBody(w, r, 2<<20)
	if err != nil {
		return
	}

	if err := storeRouterObject(manifestRoot.Key, "manifest_blob.bin", body); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to store manifest", err.Error())
		return
	}

	providers, err := fetchDealProvidersFromLCD(r.Context(), dealID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve providers", err.Error())
		return
	}

	forwarded, failed := forwardToProviders(r.Context(), providers, func(base string) (*http.Request, error) {
		ctx, cancel := context.WithTimeout(r.Context(), routerMirrorTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/sp/upload_manifest", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Nil-Deal-ID", dealIDStr)
		req.Header.Set("X-Nil-Manifest-Root", manifestRoot.Canonical)
		req.Header.Set("Content-Type", "application/octet-stream")
		return req, nil
	})

	respondMirrorResult(w, forwarded, failed, "")
}

func RouterMirrorShard(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	dealIDStr := strings.TrimSpace(r.Header.Get("X-Nil-Deal-ID"))
	mduIndexStr := strings.TrimSpace(r.Header.Get("X-Nil-Mdu-Index"))
	slotStr := strings.TrimSpace(r.Header.Get("X-Nil-Slot"))
	rawManifestRoot := strings.TrimSpace(r.Header.Get("X-Nil-Manifest-Root"))
	if dealIDStr == "" || mduIndexStr == "" || slotStr == "" || rawManifestRoot == "" {
		http.Error(w, "X-Nil-Deal-ID, X-Nil-Mdu-Index, X-Nil-Slot, and X-Nil-Manifest-Root headers are required", http.StatusBadRequest)
		return
	}

	dealID, err := strconv.ParseUint(dealIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid deal_id", http.StatusBadRequest)
		return
	}

	slot, err := strconv.ParseUint(slotStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid slot", http.StatusBadRequest)
		return
	}

	if _, _, err := fetchDealOwnerAndCID(dealID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to validate deal", err.Error())
		return
	}

	manifestRoot, err := parseManifestRoot(rawManifestRoot)
	if err != nil {
		http.Error(w, "invalid manifest root", http.StatusBadRequest)
		return
	}

	body, err := readLimitedBody(w, r, 10<<20)
	if err != nil {
		return
	}

	if err := storeRouterObject(manifestRoot.Key, fmt.Sprintf("mdu_%s_slot_%d.bin", mduIndexStr, slot), body); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to store shard", err.Error())
		return
	}

	providers, err := fetchDealProvidersFromLCD(r.Context(), dealID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve providers", err.Error())
		return
	}
	if int(slot) >= len(providers) {
		writeJSONError(w, http.StatusBadRequest, "slot out of range for deal providers", fmt.Sprintf("slot=%d providers=%d", slot, len(providers)))
		return
	}

	target := providers[slot]
	base, err := resolveProviderHTTPBaseURL(r.Context(), target)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve provider endpoint", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), routerMirrorTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/sp/upload_shard", bytes.NewReader(body))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to build forward request", err.Error())
		return
	}
	req.Header.Set("X-Nil-Deal-ID", dealIDStr)
	req.Header.Set("X-Nil-Mdu-Index", mduIndexStr)
	req.Header.Set("X-Nil-Slot", slotStr)
	req.Header.Set("X-Nil-Manifest-Root", manifestRoot.Canonical)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "forward shard failed", err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		writeJSONError(w, http.StatusBadGateway, "forward shard failed", fmt.Sprintf("%s: %s", resp.Status, strings.TrimSpace(string(b))))
		return
	}

	respondMirrorResult(w, 1, nil, "")
}

func readLimitedBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return nil, err
	}
	if len(body) == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return nil, fmt.Errorf("empty body")
	}
	return body, nil
}

func storeRouterObject(rootKey string, filename string, body []byte) error {
	rootDir := filepath.Join(uploadDir, rootKey)
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(rootDir, filename)

	tmp, err := os.CreateTemp(rootDir, filename+".tmp_*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func forwardToProviders(ctx context.Context, providers []string, buildReq func(base string) (*http.Request, error)) (int, []string) {
	forwarded := 0
	var failed []string

	for _, providerAddr := range providers {
		base, err := resolveProviderHTTPBaseURL(ctx, providerAddr)
		if err != nil || strings.TrimSpace(base) == "" {
			failed = append(failed, providerAddr)
			continue
		}
		req, err := buildReq(base)
		if err != nil {
			failed = append(failed, providerAddr)
			continue
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			failed = append(failed, providerAddr)
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			failed = append(failed, providerAddr)
			continue
		}
		forwarded++
	}
	return forwarded, failed
}

func respondMirrorResult(w http.ResponseWriter, forwarded int, failed []string, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	if len(failed) > 0 || errMsg != "" {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(routerMirrorResult{
			Status:          "error",
			Forwarded:       forwarded,
			FailedProviders: failed,
			Error:           strings.TrimSpace(errMsg),
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(routerMirrorResult{
		Status:    "success",
		Forwarded: forwarded,
	})
}
