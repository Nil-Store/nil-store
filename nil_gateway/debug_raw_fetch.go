package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

// GatewayDebugRawFetch serves file bytes directly from an on-disk NilFS slab without
// requiring retrieval-session / receipt flows. This is intended for devnet debugging.
//
// Query params:
// - deal_id, owner: optional but recommended; when provided they are validated against chain state.
// - file_path: required NilFS path.
// - range_start, range_len: optional byte range within the file; len=0 means "to EOF".
func GatewayDebugRawFetch(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	vars := mux.Vars(r)
	rawManifestRoot := strings.TrimSpace(vars["cid"])
	if rawManifestRoot == "" {
		writeJSONError(w, http.StatusBadRequest, "manifest_root path parameter is required", "")
		return
	}
	manifestRoot, err := parseManifestRoot(rawManifestRoot)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid manifest_root", err.Error())
		return
	}

	// Optional deal_id/owner validation (shared semantics with manifest-info / mdu-kzg).
	dealID, _, status, err := validateDealOwnerCidQuery(r, manifestRoot)
	hasDealQuery := strings.TrimSpace(r.URL.Query().Get("deal_id")) != ""
	if err != nil {
		writeJSONError(w, status, err.Error(), "")
		return
	}

	q := r.URL.Query()
	filePath, err := validateNilfsFilePath(q.Get("file_path"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid file_path", "")
		return
	}

	var rangeStart uint64
	if raw := strings.TrimSpace(q.Get("range_start")); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid range_start", "")
			return
		}
		rangeStart = v
	}
	var rangeLen uint64
	if raw := strings.TrimSpace(q.Get("range_len")); raw != "" {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid range_len", "")
			return
		}
		rangeLen = v
	}

	var dealDir string
	if hasDealQuery {
		dealDir, err = resolveDealDirForDeal(dealID, manifestRoot, rawManifestRoot)
	} else {
		dealDir, err = resolveDealDir(manifestRoot, rawManifestRoot)
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSONError(w, http.StatusNotFound, "slab not found on disk", "")
			return
		}
		if errors.Is(err, ErrDealDirConflict) {
			writeJSONError(w, http.StatusConflict, "deal directory conflict", err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve slab directory", err.Error())
		return
	}

	// Mode2 stores user data as striped shards. The normal fetch path can reconstruct
	// the required MDUs on-demand, but raw-fetch decodes directly from mdu_<idx>.bin.
	// Ensure the required user MDUs exist to avoid silent truncation on large files.
	if hasDealQuery {
		serviceHint, serr := fetchDealServiceHintFromLCD(r.Context(), dealID)
		if serr != nil {
			writeJSONError(w, http.StatusBadGateway, "failed to query deal service hint", serr.Error())
			return
		}
		stripe, perr := stripeParamsFromHint(serviceHint)
		if perr != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to parse deal service hint", perr.Error())
			return
		}
		if stripe.mode == 2 {
			startOffset, fileLen, witnessCount, metaErr := GetFileMetaByPath(dealDir, filePath)
			if metaErr != nil {
				if errors.Is(metaErr, os.ErrNotExist) {
					writeJSONError(w, http.StatusNotFound, "file not found in deal", "")
					return
				}
				writeJSONError(w, http.StatusInternalServerError, "failed to resolve file metadata", metaErr.Error())
				return
			}
			if fileLen == 0 || rangeStart >= fileLen {
				writeJSONError(w, http.StatusRequestedRangeNotSatisfiable, "range not satisfiable", "")
				return
			}

			effectiveLen := rangeLen
			if effectiveLen == 0 || effectiveLen > fileLen-rangeStart {
				effectiveLen = fileLen - rangeStart
			}
			if effectiveLen == 0 {
				writeJSONError(w, http.StatusRequestedRangeNotSatisfiable, "range not satisfiable", "")
				return
			}

			absStart := startOffset + rangeStart
			absEnd := absStart + effectiveLen - 1
			startMdu := uint64(1) + witnessCount + (absStart / RawMduCapacity)
			endMdu := uint64(1) + witnessCount + (absEnd / RawMduCapacity)

			// Reconstruct MDUs concurrently but keep a conservative limit to avoid
			// overwhelming local providers during large downloads.
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()

			sem := make(chan struct{}, 4)
			errOnce := make(chan error, 1)
			var wg sync.WaitGroup

			reconstruct := func(idx uint64) {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					return
				}
				defer func() { <-sem }()

				if _, err := ensureMode2MduOnDisk(ctx, dealID, manifestRoot, idx, dealDir, stripe); err != nil {
					select {
					case errOnce <- fmt.Errorf("mdu_%d.bin: %w", idx, err):
						cancel()
					default:
					}
				}
			}

			for idx := startMdu; idx <= endMdu; idx++ {
				wg.Add(1)
				go reconstruct(idx)
			}
			wg.Wait()
			select {
			case err := <-errOnce:
				writeJSONError(w, http.StatusBadGateway, "failed to reconstruct Mode2 MDU", err.Error())
				return
			default:
			}
		}
	}

	content, _, _, _, servedLen, _, err := resolveNilfsFileSegmentForFetch(dealDir, filePath, rangeStart, rangeLen)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSONError(w, http.StatusNotFound, "file not found in deal", "")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to resolve file", err.Error())
		return
	}
	defer content.Close()
	if servedLen == 0 {
		writeJSONError(w, http.StatusRequestedRangeNotSatisfiable, "range not satisfiable", "")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	// Provide a small, parseable hint for debug tooling (not used by the main UI yet).
	w.Header().Set("X-Nil-Debug-Range-Start", strconv.FormatUint(rangeStart, 10))
	w.Header().Set("X-Nil-Debug-Range-Len", strconv.FormatUint(servedLen, 10))
	w.Header().Set("Content-Length", strconv.FormatUint(servedLen, 10))

	written, copyErr := io.CopyN(w, content, int64(servedLen))
	if copyErr != nil {
		log.Printf("GatewayDebugRawFetch: stream error after %d/%d bytes: %v", written, servedLen, copyErr)
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
