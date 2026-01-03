package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type gatewayUploadLocalRequest struct {
	LocalPath    string `json:"local_path"`
	FilePath     string `json:"file_path,omitempty"`
	Owner        string `json:"owner,omitempty"`
	MaxUserMdus  uint64 `json:"max_user_mdus,omitempty"`
	Description  string `json:"description,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	DeclaredSize uint64 `json:"file_size_bytes,omitempty"`
}

func localImportEnabled() bool {
	raw := strings.TrimSpace(os.Getenv("NIL_LOCAL_IMPORT_ENABLED"))
	return raw == "1" || strings.EqualFold(raw, "true")
}

func localImportAllowAbs() bool {
	raw := strings.TrimSpace(os.Getenv("NIL_LOCAL_IMPORT_ALLOW_ABS"))
	return raw == "1" || strings.EqualFold(raw, "true")
}

func localImportRoot() string {
	return strings.TrimSpace(os.Getenv("NIL_LOCAL_IMPORT_ROOT"))
}

func pathWithinRoot(root string, path string) (bool, error) {
	// Best-effort symlink evaluation to prevent trivial escapes.
	if resolvedRoot, err := filepath.EvalSymlinks(root); err == nil {
		root = resolvedRoot
	}
	if resolvedPath, err := filepath.EvalSymlinks(path); err == nil {
		path = resolvedPath
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false, err
	}
	if rel == "." {
		return true, nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}
	return true, nil
}

func resolveLocalImportPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("local_path is required")
	}

	root := localImportRoot()
	if filepath.IsAbs(raw) {
		if !localImportAllowAbs() {
			return "", fmt.Errorf("absolute local_path requires NIL_LOCAL_IMPORT_ALLOW_ABS=1")
		}
		absPath, err := filepath.Abs(raw)
		if err != nil {
			return "", err
		}
		if root == "" {
			return absPath, nil
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		ok, err := pathWithinRoot(absRoot, absPath)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("local_path is outside NIL_LOCAL_IMPORT_ROOT")
		}
		return absPath, nil
	}

	if root == "" {
		return "", fmt.Errorf("relative local_path requires NIL_LOCAL_IMPORT_ROOT")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(filepath.Join(absRoot, raw))
	if err != nil {
		return "", err
	}
	ok, err := pathWithinRoot(absRoot, absPath)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("local_path is outside NIL_LOCAL_IMPORT_ROOT")
	}
	return absPath, nil
}

func GatewayUploadLocal(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if !localImportEnabled() {
		writeJSONError(w, http.StatusBadRequest, "local import disabled", "set NIL_LOCAL_IMPORT_ENABLED=1 to enable /gateway/upload-local")
		return
	}

	dealID, ok := requireDealIDQuery(w, r)
	if !ok {
		return
	}
	uploadID, ok := requireUploadIDQuery(w, r)
	if !ok {
		return
	}

	var req gatewayUploadLocalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON", "")
		return
	}

	resolvedPath, err := resolveLocalImportPath(req.LocalPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid local_path", err.Error())
		return
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to stat local_path", err.Error())
		return
	}
	if !info.Mode().IsRegular() {
		writeJSONError(w, http.StatusBadRequest, "local_path is not a regular file", "")
		return
	}

	fileName := filepath.Base(resolvedPath)
	fileSize := uint64(info.Size())
	fileRecordPath := strings.TrimSpace(req.FilePath)
	if fileRecordPath == "" {
		fileRecordPath = fileName
	}
	if validated, err := validateNilfsFilePath(fileRecordPath); err == nil {
		fileRecordPath = validated
	}
	if strings.Contains(fileRecordPath, "/") {
		fileRecordPath = filepath.Base(fileRecordPath)
	}

	job := newUploadJob(dealID, uploadID)
	job.setFile(fileName, fileSize)
	job.setPhase(uploadJobPhaseQueued, "Local import queued...")
	job.setBytes(0, fileSize)
	storeUploadJob(job)

	dealIDStr := strconv.FormatUint(dealID, 10)
	maxUserMdusStr := ""
	if req.MaxUserMdus > 0 {
		maxUserMdusStr = strconv.FormatUint(req.MaxUserMdus, 10)
	}

	go func() {
		jobCtx, cancel := context.WithTimeout(context.Background(), uploadIngestTimeout)
		defer cancel()

		res, err := computeGatewayUploadResult(jobCtx, gatewayUploadComputeInput{
			owner:          req.Owner,
			dealIDStr:      dealIDStr,
			fileRecordPath: fileRecordPath,
			maxUserMdusStr: maxUserMdusStr,
			uploadedPath:   resolvedPath,
			job:            job,
		})
		if err != nil {
			job.setError(err.Error())
			return
		}

		job.setResult(uploadJobResult{
			ManifestRoot:    res.manifestRoot,
			SizeBytes:       res.sizeBytes,
			FileSizeBytes:   res.fileSizeBytes,
			AllocatedLength: res.allocatedLength,
		})
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "accepted",
		"deal_id":    dealIDStr,
		"upload_id":  uploadID,
		"file_name":  fileName,
		"file_path":  fileRecordPath,
		"local_path": req.LocalPath,
		"status_url": "/gateway/upload-status?deal_id=" + url.QueryEscape(dealIDStr) + "&upload_id=" + url.QueryEscape(uploadID),
	})
}
