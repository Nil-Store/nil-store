package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"nilchain/x/crypto_ffi"
	"nilchain/x/nilchain/types"
)

type mode2IngestResult struct {
	manifestRoot    ManifestRoot
	manifestBlob    []byte
	allocatedLength uint64
	totalSizeBytes  uint64
	fileSize        uint64
	witnessMdus     uint64
	userMdus        uint64
}

type mode2LayoutMeta struct {
	Version             int    `json:"version"`
	MaxUserMdus         uint64 `json:"max_user_mdus"`
	CommitmentsPerMdu   uint64 `json:"commitments_per_mdu"`
	StripeK             uint64 `json:"stripe_k"`
	StripeM             uint64 `json:"stripe_m"`
	StripeLeafCount     uint64 `json:"stripe_leaf_count"`
	StripeSlotCount     uint64 `json:"stripe_slot_count"`
	StripeRows          uint64 `json:"stripe_rows"`
	WitnessMdus         uint64 `json:"witness_mdus"`
	CreatedAtUnixMillis int64  `json:"created_at_unix_millis"`
}

func encodePayloadToMdu(raw []byte) []byte {
	if len(raw) > RawMduCapacity {
		raw = raw[:RawMduCapacity]
	}
	encoded := make([]byte, types.MDU_SIZE)
	scalarIdx := 0
	for i := 0; i < len(raw) && scalarIdx < nilfsScalarsPerMdu; i += nilfsScalarPayloadBytes {
		end := i + nilfsScalarPayloadBytes
		if end > len(raw) {
			end = len(raw)
		}
		chunk := raw[i:end]
		pad := nilfsScalarBytes - len(chunk)
		offset := scalarIdx*nilfsScalarBytes + pad
		copy(encoded[offset:offset+len(chunk)], chunk)
		scalarIdx++
	}
	return encoded
}

func decodePayloadFromMdu(encoded []byte) ([]byte, error) {
	if len(encoded) != types.MDU_SIZE {
		return nil, fmt.Errorf("invalid mdu size: %d", len(encoded))
	}
	out := make([]byte, RawMduCapacity)
	for i := 0; i < nilfsScalarsPerMdu; i++ {
		srcOff := i*nilfsScalarBytes + (nilfsScalarBytes - nilfsScalarPayloadBytes)
		dstOff := i * nilfsScalarPayloadBytes
		copy(out[dstOff:dstOff+nilfsScalarPayloadBytes], encoded[srcOff:srcOff+nilfsScalarPayloadBytes])
	}
	return out, nil
}

func mode2MetaPath(dealID uint64) string {
	return filepath.Join(uploadDir, "deals", strconv.FormatUint(dealID, 10), "mode2_layout.json")
}

func loadMode2LayoutMeta(dealID uint64) (*mode2LayoutMeta, error) {
	path := mode2MetaPath(dealID)
	bz, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta mode2LayoutMeta
	if err := json.Unmarshal(bz, &meta); err != nil {
		return nil, err
	}
	if meta.Version != 1 || meta.MaxUserMdus == 0 || meta.CommitmentsPerMdu == 0 || meta.WitnessMdus == 0 {
		return nil, fmt.Errorf("invalid mode2_layout.json")
	}
	return &meta, nil
}

func writeMode2LayoutMeta(dealID uint64, meta mode2LayoutMeta) error {
	path := mode2MetaPath(dealID)
	tmp, err := os.CreateTemp(filepath.Dir(path), "mode2_layout_*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	bz, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(bz); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func inferMode2LayoutFromDisk(dealID uint64, dir string, stripe stripeParams, commitmentsPerMdu uint64) (mode2LayoutMeta, error) {
	witness := uint64(0)
	for i := uint64(1); ; i++ {
		if _, err := os.Stat(filepath.Join(dir, fmt.Sprintf("mdu_%d.bin", i))); err != nil {
			break
		}
		witness++
	}
	if witness == 0 {
		return mode2LayoutMeta{}, fmt.Errorf("could not infer witness mdus")
	}

	commitBytesPerMdu := commitmentsPerMdu * 48
	if commitBytesPerMdu == 0 {
		return mode2LayoutMeta{}, fmt.Errorf("invalid commitments_per_mdu")
	}
	upper := (witness*RawMduCapacity + commitBytesPerMdu - 1) / commitBytesPerMdu
	if upper == 0 {
		upper = 1
	}

	return mode2LayoutMeta{
		Version:             1,
		MaxUserMdus:         upper,
		CommitmentsPerMdu:   commitmentsPerMdu,
		StripeK:             stripe.k,
		StripeM:             stripe.m,
		StripeLeafCount:     stripe.leafCount,
		StripeSlotCount:     stripe.slotCount,
		StripeRows:          stripe.rows,
		WitnessMdus:         witness,
		CreatedAtUnixMillis: time.Now().UnixMilli(),
	}, nil
}

func mode2BuildArtifacts(ctx context.Context, filePath string, dealID uint64, hint string, fileRecordPath string, maxUserMdus uint64) (*mode2IngestResult, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	stripe, err := stripeParamsFromHint(hint)
	if err != nil {
		return nil, "", fmt.Errorf("parse service_hint: %w", err)
	}
	if stripe.mode != 2 || stripe.k == 0 || stripe.m == 0 || stripe.rows == 0 {
		return nil, "", fmt.Errorf("deal is not Mode 2")
	}

	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, "", err
	}
	fileSize := uint64(fi.Size())
	userMdus := uint64(1)
	if fileSize > 0 {
		userMdus = (fileSize + RawMduCapacity - 1) / RawMduCapacity
		if userMdus == 0 {
			userMdus = 1
		}
	}

	if fileRecordPath == "" {
		fileRecordPath = filepath.Base(filePath)
	}
	if len(fileRecordPath) > 40 {
		fileRecordPath = fileRecordPath[:40]
	}

	commitmentsPerMdu := stripe.leafCount
	if maxUserMdus == 0 {
		maxUserMdus = userMdus
	}
	builder := crypto_ffi.NewMdu0BuilderWithCommitments(maxUserMdus, commitmentsPerMdu)
	if builder == nil {
		return nil, "", fmt.Errorf("failed to create MDU0 builder")
	}
	defer builder.Free()
	witnessCount := builder.GetWitnessCount()

	// Stage artifacts under uploads/deals/<dealID>/.staging-<ts>/, then atomically rename to the manifest-root key.
	baseDealDir := filepath.Join(uploadDir, "deals", strconv.FormatUint(dealID, 10))
	if err := os.MkdirAll(baseDealDir, 0o755); err != nil {
		return nil, "", err
	}
	stagingDir, err := os.MkdirTemp(baseDealDir, "staging-")
	if err != nil {
		return nil, "", err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	userRoots := make([][]byte, 0, userMdus)
	commitBytesPerMdu := int(commitmentsPerMdu * 48)
	if commitBytesPerMdu <= 0 {
		return nil, "", fmt.Errorf("invalid commitmentsPerMdu")
	}
	rawWitnessBytes := make([]byte, int(witnessCount*RawMduCapacity))

	f, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	rawBuf := make([]byte, RawMduCapacity)
	for i := uint64(0); i < userMdus; i++ {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		n, readErr := io.ReadFull(f, rawBuf)
		if readErr != nil {
			if readErr == io.ErrUnexpectedEOF || readErr == io.EOF {
				// Last chunk is short (or empty).
			} else {
				return nil, "", readErr
			}
		}
		chunk := rawBuf[:n]
		encoded := encodePayloadToMdu(chunk)

		witnessFlat, shards, err := crypto_ffi.ExpandMduRs(encoded, stripe.k, stripe.m)
		if err != nil {
			return nil, "", fmt.Errorf("expand mdu %d: %w", i, err)
		}
		root, err := crypto_ffi.ComputeMduRootFromWitnessFlat(witnessFlat)
		if err != nil {
			return nil, "", fmt.Errorf("compute mdu root %d: %w", i, err)
		}
		userRoots = append(userRoots, root)
		if err := builder.SetRoot(witnessCount+i, root); err != nil {
			return nil, "", fmt.Errorf("set user root %d: %w", i, err)
		}
		start := int(i) * commitBytesPerMdu
		end := start + len(witnessFlat)
		if end > len(rawWitnessBytes) {
			return nil, "", fmt.Errorf("witness buffer overflow (need %d, have %d)", end, len(rawWitnessBytes))
		}
		copy(rawWitnessBytes[start:end], witnessFlat)

		slabIndex := uint64(1) + witnessCount + i
		for slot := uint64(0); slot < stripe.slotCount; slot++ {
			if int(slot) >= len(shards) {
				return nil, "", fmt.Errorf("missing shard for slot %d", slot)
			}
			name := fmt.Sprintf("mdu_%d_slot_%d.bin", slabIndex, slot)
			if err := os.WriteFile(filepath.Join(stagingDir, name), shards[slot], 0o644); err != nil {
				return nil, "", err
			}
		}
	}

	// Build witness MDUs from the concatenated witness commitments.
	witnessRoots := make([][]byte, 0, witnessCount)
	for i := uint64(0); i < witnessCount; i++ {
		start := i * RawMduCapacity
		end := start + RawMduCapacity
		chunk := rawWitnessBytes[start:end]
		encoded := encodePayloadToMdu(chunk)
		root, err := crypto_ffi.ComputeMduMerkleRoot(encoded)
		if err != nil {
			return nil, "", fmt.Errorf("compute witness root %d: %w", i, err)
		}
		witnessRoots = append(witnessRoots, root)
		if err := builder.SetRoot(i, root); err != nil {
			return nil, "", fmt.Errorf("set witness root %d: %w", i, err)
		}
		if err := os.WriteFile(filepath.Join(stagingDir, fmt.Sprintf("mdu_%d.bin", 1+i)), encoded, 0o644); err != nil {
			return nil, "", err
		}
	}

	// Append the file record (naive single-file mapping at offset 0 for now).
	if err := builder.AppendFile(fileRecordPath, fileSize, 0); err != nil {
		return nil, "", err
	}

	// Write MDU #0 and compute its root.
	mdu0Bytes, err := builder.Bytes()
	if err != nil {
		return nil, "", err
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "mdu_0.bin"), mdu0Bytes, 0o644); err != nil {
		return nil, "", err
	}
	mdu0Root, err := crypto_ffi.ComputeMduMerkleRoot(mdu0Bytes)
	if err != nil {
		return nil, "", fmt.Errorf("compute mdu0 root: %w", err)
	}

	roots := make([][]byte, 0, 1+len(witnessRoots)+len(userRoots))
	roots = append(roots, mdu0Root)
	roots = append(roots, witnessRoots...)
	roots = append(roots, userRoots...)

	commitment, manifestBlob, err := crypto_ffi.ComputeManifestCommitment(roots)
	if err != nil {
		return nil, "", fmt.Errorf("compute manifest commitment: %w", err)
	}
	manifestRootHex := "0x" + hex.EncodeToString(commitment)
	parsedRoot, err := parseManifestRoot(manifestRootHex)
	if err != nil {
		return nil, "", err
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "manifest.bin"), manifestBlob, 0o644); err != nil {
		return nil, "", err
	}

	finalDir := dealScopedDir(dealID, parsedRoot)
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return nil, "", err
	}
	if err := os.Rename(stagingDir, finalDir); err != nil {
		return nil, "", err
	}
	rollback = false

	// Persist mode2 sizing metadata under uploads/deals/<dealID>/ so append can keep a stable witness layout.
	_ = writeMode2LayoutMeta(dealID, mode2LayoutMeta{
		Version:             1,
		MaxUserMdus:         maxUserMdus,
		CommitmentsPerMdu:   commitmentsPerMdu,
		StripeK:             stripe.k,
		StripeM:             stripe.m,
		StripeLeafCount:     stripe.leafCount,
		StripeSlotCount:     stripe.slotCount,
		StripeRows:          stripe.rows,
		WitnessMdus:         witnessCount,
		CreatedAtUnixMillis: time.Now().UnixMilli(),
	})

	return &mode2IngestResult{
		manifestRoot:    parsedRoot,
		manifestBlob:    manifestBlob,
		allocatedLength: uint64(len(roots)),
		totalSizeBytes:  totalSizeBytesFromMdu0(builder),
		fileSize:        fileSize,
		witnessMdus:     witnessCount,
		userMdus:        userMdus,
	}, finalDir, nil
}

func mode2UploadArtifactsToProviders(
	ctx context.Context,
	dealID uint64,
	manifestRoot ManifestRoot,
	hint string,
	finalDir string,
	witnessCount uint64,
	userMdus uint64,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	stripe, err := stripeParamsFromHint(hint)
	if err != nil {
		return fmt.Errorf("parse service_hint: %w", err)
	}
	if stripe.mode != 2 || stripe.k == 0 || stripe.m == 0 || stripe.rows == 0 {
		return fmt.Errorf("deal is not Mode 2")
	}
	if witnessCount == 0 || stripe.slotCount == 0 {
		return fmt.Errorf("invalid Mode 2 state")
	}

	// Upload to assigned providers as a dumb pipe: bytes-in/bytes-out.
	providers, err := fetchDealProvidersFromLCD(ctx, dealID)
	if err != nil {
		return err
	}
	if len(providers) < int(stripe.slotCount) {
		return fmt.Errorf("not enough providers for Mode 2 (need %d, got %d)", stripe.slotCount, len(providers))
	}
	slotBases := make([]string, 0, stripe.slotCount)
	for slot := uint64(0); slot < stripe.slotCount; slot++ {
		base, err := resolveProviderHTTPBaseURL(ctx, providers[slot])
		if err != nil {
			return err
		}
		slotBases = append(slotBases, strings.TrimRight(base, "/"))
	}

	client := &http.Client{Timeout: 60 * time.Second}
	manifestRootCanonical := manifestRoot.Canonical
	dealIDStr := strconv.FormatUint(dealID, 10)

	uploadBlob := func(ctx context.Context, url string, headers map[string]string, path string, maxBytes int64) error {
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if maxBytes > 0 && int64(len(body)) > maxBytes {
			return fmt.Errorf("artifact too large: %s (%d bytes)", filepath.Base(path), len(body))
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			msg, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
			return fmt.Errorf("upload failed: %s (%s)", resp.Status, strings.TrimSpace(string(msg)))
		}
		return nil
	}

	// Replicated metadata: mdu_0..mdu_witnessCount + manifest.bin to all slots.
	for _, base := range slotBases {
		for mduIndex := uint64(0); mduIndex <= witnessCount; mduIndex++ {
			path := filepath.Join(finalDir, fmt.Sprintf("mdu_%d.bin", mduIndex))
			if err := uploadBlob(ctx, base+"/sp/upload_mdu", map[string]string{
				"X-Nil-Deal-ID":       dealIDStr,
				"X-Nil-Mdu-Index":     strconv.FormatUint(mduIndex, 10),
				"X-Nil-Manifest-Root": manifestRootCanonical,
			}, path, 10<<20); err != nil {
				return err
			}
		}
		if err := uploadBlob(ctx, base+"/sp/upload_manifest", map[string]string{
			"X-Nil-Deal-ID":       dealIDStr,
			"X-Nil-Manifest-Root": manifestRootCanonical,
		}, filepath.Join(finalDir, "manifest.bin"), 512<<10); err != nil {
			return err
		}
	}

	// Striped user shards.
	for i := uint64(0); i < userMdus; i++ {
		slabIndex := uint64(1) + witnessCount + i
		for slot, base := range slotBases {
			path := filepath.Join(finalDir, fmt.Sprintf("mdu_%d_slot_%d.bin", slabIndex, slot))
			if err := uploadBlob(ctx, base+"/sp/upload_shard", map[string]string{
				"X-Nil-Deal-ID":       dealIDStr,
				"X-Nil-Mdu-Index":     strconv.FormatUint(slabIndex, 10),
				"X-Nil-Slot":          strconv.Itoa(slot),
				"X-Nil-Manifest-Root": manifestRootCanonical,
			}, path, 10<<20); err != nil {
				return err
			}
		}
	}

	return nil
}

func mode2IngestAndUploadNewDeal(ctx context.Context, filePath string, dealID uint64, hint string, fileRecordPath string, maxUserMdus uint64) (*mode2IngestResult, error) {
	res, finalDir, err := mode2BuildArtifacts(ctx, filePath, dealID, hint, fileRecordPath, maxUserMdus)
	if err != nil {
		return nil, err
	}
	if err := mode2UploadArtifactsToProviders(ctx, dealID, res.manifestRoot, hint, finalDir, res.witnessMdus, res.userMdus); err != nil {
		return nil, err
	}
	return res, nil
}

var mode2ShardNameRe = regexp.MustCompile(`^mdu_(\d+)_slot_(\d+)\.bin$`)

func mode2CountExistingUserMdus(dir string, witnessCount uint64) (uint64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	maxIdx := uint64(0)
	found := false
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := mode2ShardNameRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		idx, err := strconv.ParseUint(m[1], 10, 64)
		if err != nil {
			continue
		}
		if idx > maxIdx {
			maxIdx = idx
		}
		found = true
	}
	if !found {
		return 0, nil
	}
	start := uint64(1) + witnessCount
	if maxIdx < start {
		return 0, nil
	}
	return maxIdx - start + 1, nil
}

func linkOrCopyFile(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func mode2StageExistingShards(oldDir, stagingDir string) error {
	entries, err := os.ReadDir(oldDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if mode2ShardNameRe.MatchString(e.Name()) {
			if err := linkOrCopyFile(filepath.Join(oldDir, e.Name()), filepath.Join(stagingDir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func mode2LoadOrInferLayout(dealID uint64, oldDir string, stripe stripeParams, commitmentsPerMdu uint64, maxUserMdusHint uint64) (mode2LayoutMeta, error) {
	if meta, err := loadMode2LayoutMeta(dealID); err == nil {
		// Ensure stripe params still match (service hint is the source of truth).
		if meta.StripeK == stripe.k && meta.StripeM == stripe.m && meta.CommitmentsPerMdu == commitmentsPerMdu {
			return *meta, nil
		}
	}

	meta, err := inferMode2LayoutFromDisk(dealID, oldDir, stripe, commitmentsPerMdu)
	if err != nil {
		return mode2LayoutMeta{}, err
	}
	if maxUserMdusHint > meta.MaxUserMdus {
		// Best-effort: allow increasing the max for older deals, but never shrink.
		meta.MaxUserMdus = maxUserMdusHint
		commitBytes := commitmentsPerMdu * 48
		meta.WitnessMdus = (meta.MaxUserMdus*commitBytes + RawMduCapacity - 1) / RawMduCapacity
		if meta.WitnessMdus == 0 {
			meta.WitnessMdus = 1
		}
	}
	_ = writeMode2LayoutMeta(dealID, meta)
	return meta, nil
}

func mode2AppendArtifacts(
	ctx context.Context,
	filePath string,
	dealID uint64,
	hint string,
	fileRecordPath string,
	previousManifestRoot ManifestRoot,
	maxUserMdusHint uint64,
) (*mode2IngestResult, string, uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	stripe, err := stripeParamsFromHint(hint)
	if err != nil {
		return nil, "", 0, fmt.Errorf("parse service_hint: %w", err)
	}
	if stripe.mode != 2 || stripe.k == 0 || stripe.m == 0 || stripe.rows == 0 {
		return nil, "", 0, fmt.Errorf("deal is not Mode 2")
	}
	commitmentsPerMdu := stripe.leafCount

	oldDir, err := resolveDealDirForDeal(dealID, previousManifestRoot, previousManifestRoot.Canonical)
	if err != nil {
		return nil, "", 0, err
	}

	meta, err := mode2LoadOrInferLayout(dealID, oldDir, stripe, commitmentsPerMdu, maxUserMdusHint)
	if err != nil {
		return nil, "", 0, err
	}
	witnessCount := meta.WitnessMdus

	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, "", 0, err
	}
	fileSize := uint64(fi.Size())
	newUserMdus := uint64(1)
	if fileSize > 0 {
		newUserMdus = (fileSize + RawMduCapacity - 1) / RawMduCapacity
		if newUserMdus == 0 {
			newUserMdus = 1
		}
	}

	oldUserMdus, err := mode2CountExistingUserMdus(oldDir, witnessCount)
	if err != nil {
		return nil, "", 0, err
	}
	if oldUserMdus+newUserMdus > meta.MaxUserMdus {
		return nil, "", 0, fmt.Errorf("deal capacity exceeded: have %d user_mdus, need %d more (max %d)", oldUserMdus, newUserMdus, meta.MaxUserMdus)
	}

	if fileRecordPath == "" {
		fileRecordPath = filepath.Base(filePath)
	}
	if len(fileRecordPath) > 40 {
		fileRecordPath = fileRecordPath[:40]
	}

	mdu0Data, err := os.ReadFile(filepath.Join(oldDir, "mdu_0.bin"))
	if err != nil {
		return nil, "", 0, err
	}
	builder, err := crypto_ffi.LoadMdu0BuilderWithCommitments(mdu0Data, meta.MaxUserMdus, commitmentsPerMdu)
	if err != nil {
		return nil, "", 0, err
	}
	defer builder.Free()

	// Stage artifacts under uploads/deals/<dealID>/.staging-<ts>/, then atomically rename to the new manifest-root key.
	baseDealDir := filepath.Join(uploadDir, "deals", strconv.FormatUint(dealID, 10))
	if err := os.MkdirAll(baseDealDir, 0o755); err != nil {
		return nil, "", 0, err
	}
	stagingDir, err := os.MkdirTemp(baseDealDir, "staging-")
	if err != nil {
		return nil, "", 0, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	// Copy/link existing shards into the new slab directory.
	if err := mode2StageExistingShards(oldDir, stagingDir); err != nil {
		return nil, "", 0, err
	}

	// Decode current witness MDUs to raw witness bytes, then append new witness commitments in-place.
	rawWitnessBytes := make([]byte, int(witnessCount*RawMduCapacity))
	for i := uint64(0); i < witnessCount; i++ {
		bz, err := os.ReadFile(filepath.Join(oldDir, fmt.Sprintf("mdu_%d.bin", 1+i)))
		if err != nil {
			return nil, "", 0, err
		}
		raw, err := decodePayloadFromMdu(bz)
		if err != nil {
			return nil, "", 0, err
		}
		copy(rawWitnessBytes[int(i*RawMduCapacity):int((i+1)*RawMduCapacity)], raw)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, "", 0, err
	}
	defer f.Close()

	commitBytesPerMdu := int(commitmentsPerMdu * 48)
	if commitBytesPerMdu <= 0 {
		return nil, "", 0, fmt.Errorf("invalid commitmentsPerMdu")
	}

	rawBuf := make([]byte, RawMduCapacity)
	for i := uint64(0); i < newUserMdus; i++ {
		if err := ctx.Err(); err != nil {
			return nil, "", 0, err
		}
		n, readErr := io.ReadFull(f, rawBuf)
		if readErr != nil {
			if readErr == io.ErrUnexpectedEOF || readErr == io.EOF {
				// Last chunk is short (or empty).
			} else {
				return nil, "", 0, readErr
			}
		}
		encoded := encodePayloadToMdu(rawBuf[:n])

		witnessFlat, shards, err := crypto_ffi.ExpandMduRs(encoded, stripe.k, stripe.m)
		if err != nil {
			return nil, "", 0, fmt.Errorf("expand mdu %d: %w", i, err)
		}
		root, err := crypto_ffi.ComputeMduRootFromWitnessFlat(witnessFlat)
		if err != nil {
			return nil, "", 0, fmt.Errorf("compute mdu root %d: %w", i, err)
		}
		userIndex := oldUserMdus + i
		if err := builder.SetRoot(witnessCount+userIndex, root); err != nil {
			return nil, "", 0, fmt.Errorf("set user root %d: %w", userIndex, err)
		}

		wStart := int(userIndex) * commitBytesPerMdu
		wEnd := wStart + len(witnessFlat)
		if wEnd > len(rawWitnessBytes) {
			return nil, "", 0, fmt.Errorf("witness buffer overflow (need %d, have %d)", wEnd, len(rawWitnessBytes))
		}
		copy(rawWitnessBytes[wStart:wEnd], witnessFlat)

		slabIndex := uint64(1) + witnessCount + userIndex
		for slot := uint64(0); slot < stripe.slotCount; slot++ {
			if int(slot) >= len(shards) {
				return nil, "", 0, fmt.Errorf("missing shard for slot %d", slot)
			}
			name := fmt.Sprintf("mdu_%d_slot_%d.bin", slabIndex, slot)
			if err := os.WriteFile(filepath.Join(stagingDir, name), shards[slot], 0o644); err != nil {
				return nil, "", 0, err
			}
		}
	}

	// Append file record (naive MDU-boundary packing).
	startOffset := oldUserMdus * RawMduCapacity
	if err := builder.AppendFile(fileRecordPath, fileSize, startOffset); err != nil {
		return nil, "", 0, err
	}

	// Rebuild witness MDUs.
	witnessRoots := make([][]byte, 0, witnessCount)
	for i := uint64(0); i < witnessCount; i++ {
		start := i * RawMduCapacity
		end := start + RawMduCapacity
		encoded := encodePayloadToMdu(rawWitnessBytes[start:end])
		root, err := crypto_ffi.ComputeMduMerkleRoot(encoded)
		if err != nil {
			return nil, "", 0, fmt.Errorf("compute witness root %d: %w", i, err)
		}
		witnessRoots = append(witnessRoots, root)
		if err := builder.SetRoot(i, root); err != nil {
			return nil, "", 0, fmt.Errorf("set witness root %d: %w", i, err)
		}
		if err := os.WriteFile(filepath.Join(stagingDir, fmt.Sprintf("mdu_%d.bin", 1+i)), encoded, 0o644); err != nil {
			return nil, "", 0, err
		}
	}

	// Write MDU #0 and compute its root.
	mdu0Bytes, err := builder.Bytes()
	if err != nil {
		return nil, "", 0, err
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "mdu_0.bin"), mdu0Bytes, 0o644); err != nil {
		return nil, "", 0, err
	}
	mdu0Root, err := crypto_ffi.ComputeMduMerkleRoot(mdu0Bytes)
	if err != nil {
		return nil, "", 0, fmt.Errorf("compute mdu0 root: %w", err)
	}

	totalUserMdus := oldUserMdus + newUserMdus
	userRoots := make([][]byte, 0, totalUserMdus)
	for i := uint64(0); i < totalUserMdus; i++ {
		root, err := builder.GetRoot(witnessCount + i)
		if err != nil {
			return nil, "", 0, fmt.Errorf("get user root %d: %w", i, err)
		}
		userRoots = append(userRoots, root)
	}
	roots := make([][]byte, 0, 1+len(witnessRoots)+len(userRoots))
	roots = append(roots, mdu0Root)
	roots = append(roots, witnessRoots...)
	roots = append(roots, userRoots...)

	commitment, manifestBlob, err := crypto_ffi.ComputeManifestCommitment(roots)
	if err != nil {
		return nil, "", 0, fmt.Errorf("compute manifest commitment: %w", err)
	}
	manifestRootHex := "0x" + hex.EncodeToString(commitment)
	parsedRoot, err := parseManifestRoot(manifestRootHex)
	if err != nil {
		return nil, "", 0, err
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "manifest.bin"), manifestBlob, 0o644); err != nil {
		return nil, "", 0, err
	}

	finalDir := dealScopedDir(dealID, parsedRoot)
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return nil, "", 0, err
	}
	if err := os.Rename(stagingDir, finalDir); err != nil {
		return nil, "", 0, err
	}
	rollback = false

	return &mode2IngestResult{
		manifestRoot:    parsedRoot,
		manifestBlob:    manifestBlob,
		allocatedLength: uint64(len(roots)),
		totalSizeBytes:  totalSizeBytesFromMdu0(builder),
		fileSize:        fileSize,
		witnessMdus:     witnessCount,
		userMdus:        totalUserMdus,
	}, finalDir, oldUserMdus, nil
}

func mode2UploadArtifactsToProvidersAppend(
	ctx context.Context,
	dealID uint64,
	manifestRoot ManifestRoot,
	hint string,
	finalDir string,
	witnessCount uint64,
	totalUserMdus uint64,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	stripe, err := stripeParamsFromHint(hint)
	if err != nil {
		return fmt.Errorf("parse service_hint: %w", err)
	}
	if stripe.mode != 2 || stripe.k == 0 || stripe.m == 0 || stripe.rows == 0 {
		return fmt.Errorf("deal is not Mode 2")
	}

	// Upload to assigned providers as a dumb pipe.
	providers, err := fetchDealProvidersFromLCD(ctx, dealID)
	if err != nil {
		return err
	}
	if len(providers) < int(stripe.slotCount) {
		return fmt.Errorf("not enough providers for Mode 2 (need %d, got %d)", stripe.slotCount, len(providers))
	}
	slotBases := make([]string, 0, stripe.slotCount)
	for slot := uint64(0); slot < stripe.slotCount; slot++ {
		base, err := resolveProviderHTTPBaseURL(ctx, providers[slot])
		if err != nil {
			return err
		}
		slotBases = append(slotBases, strings.TrimRight(base, "/"))
	}

	client := &http.Client{Timeout: 60 * time.Second}
	manifestRootCanonical := manifestRoot.Canonical
	dealIDStr := strconv.FormatUint(dealID, 10)

	uploadBlob := func(ctx context.Context, url string, headers map[string]string, path string, maxBytes int64) error {
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if maxBytes > 0 && int64(len(body)) > maxBytes {
			return fmt.Errorf("artifact too large: %s (%d bytes)", filepath.Base(path), len(body))
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			msg, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
			return fmt.Errorf("upload failed: %s (%s)", resp.Status, strings.TrimSpace(string(msg)))
		}
		return nil
	}

	// Replicated metadata: mdu_0..mdu_witnessCount + manifest.bin to all slots.
	for _, base := range slotBases {
		for mduIndex := uint64(0); mduIndex <= witnessCount; mduIndex++ {
			path := filepath.Join(finalDir, fmt.Sprintf("mdu_%d.bin", mduIndex))
			if err := uploadBlob(ctx, base+"/sp/upload_mdu", map[string]string{
				"X-Nil-Deal-ID":       dealIDStr,
				"X-Nil-Mdu-Index":     strconv.FormatUint(mduIndex, 10),
				"X-Nil-Manifest-Root": manifestRootCanonical,
			}, path, 10<<20); err != nil {
				return err
			}
		}
		if err := uploadBlob(ctx, base+"/sp/upload_manifest", map[string]string{
			"X-Nil-Deal-ID":       dealIDStr,
			"X-Nil-Manifest-Root": manifestRootCanonical,
		}, filepath.Join(finalDir, "manifest.bin"), 512<<10); err != nil {
			return err
		}
	}

	// Striped user shards. Even if only a suffix changed, upload the full set under the
	// new manifest_root so remote repairs/fetches can address shards by (manifest_root, index).
	for userIndex := uint64(0); userIndex < totalUserMdus; userIndex++ {
		slabIndex := uint64(1) + witnessCount + userIndex
		for slot, base := range slotBases {
			path := filepath.Join(finalDir, fmt.Sprintf("mdu_%d_slot_%d.bin", slabIndex, slot))
			if err := uploadBlob(ctx, base+"/sp/upload_shard", map[string]string{
				"X-Nil-Deal-ID":       dealIDStr,
				"X-Nil-Mdu-Index":     strconv.FormatUint(slabIndex, 10),
				"X-Nil-Slot":          strconv.Itoa(slot),
				"X-Nil-Manifest-Root": manifestRootCanonical,
			}, path, 10<<20); err != nil {
				return err
			}
		}
	}

	return nil
}

func mode2IngestAndUploadAppendDeal(
	ctx context.Context,
	filePath string,
	dealID uint64,
	hint string,
	fileRecordPath string,
	previousManifestRoot ManifestRoot,
	maxUserMdusHint uint64,
) (*mode2IngestResult, error) {
	res, finalDir, _, err := mode2AppendArtifacts(ctx, filePath, dealID, hint, fileRecordPath, previousManifestRoot, maxUserMdusHint)
	if err != nil {
		return nil, err
	}
	if err := mode2UploadArtifactsToProvidersAppend(ctx, dealID, res.manifestRoot, hint, finalDir, res.witnessMdus, res.userMdus); err != nil {
		return nil, err
	}
	return res, nil
}
