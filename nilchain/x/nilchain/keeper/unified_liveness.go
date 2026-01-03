package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"nilchain/x/nilchain/types"
)

const (
	epochSeedTag   = "nilstore/epoch/v1"
	challengeTag   = "nilstore/chal/v1"
	creditIDTag    = "nilstore/credit/v1"
	syntheticIDTag = "nilstore/synth/v1"
	repairTag      = "nilstore/repair/v1"
)

func epochIDForHeight(height uint64, epochLenBlocks uint64) (uint64, error) {
	if epochLenBlocks == 0 {
		return 0, fmt.Errorf("epoch_len_blocks must be non-zero")
	}
	if height == 0 {
		return 0, fmt.Errorf("block height must be >= 1")
	}
	return (height - 1) / epochLenBlocks, nil
}

func epochStartHeight(epochID uint64, epochLenBlocks uint64) (uint64, error) {
	if epochLenBlocks == 0 {
		return 0, fmt.Errorf("epoch_len_blocks must be non-zero")
	}
	if epochID > (^(uint64(0))-1)/epochLenBlocks {
		return 0, fmt.Errorf("epoch_id overflow")
	}
	start := epochID*epochLenBlocks + 1
	if start == 0 {
		return 0, fmt.Errorf("epoch_start_height overflow")
	}
	return start, nil
}

func computeEpochSeed(chainID string, epochID uint64, blockHash []byte) []byte {
	var epochBuf [8]byte
	binary.BigEndian.PutUint64(epochBuf[:], epochID)

	buf := make([]byte, 0, len(epochSeedTag)+len(chainID)+8+len(blockHash))
	buf = append(buf, []byte(epochSeedTag)...)
	buf = append(buf, []byte(chainID)...)
	buf = append(buf, epochBuf[:]...)
	buf = append(buf, blockHash...)

	sum := sha256.Sum256(buf)
	out := make([]byte, 32)
	copy(out, sum[:])
	return out
}

// BeginBlock persists the epoch seed at epoch boundaries so that subsequent
// tx handlers don't need historical header hashes.
func (k Keeper) BeginBlock(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	height := uint64(ctx.BlockHeight())
	epochID, err := epochIDForHeight(height, params.EpochLenBlocks)
	if err != nil {
		return err
	}
	startHeight, err := epochStartHeight(epochID, params.EpochLenBlocks)
	if err != nil {
		return err
	}
	if height != startHeight {
		return nil
	}

	if _, err := k.EpochSeeds.Get(ctx, epochID); err == nil {
		return nil
	} else if !errors.Is(err, collections.ErrNotFound) {
		return fmt.Errorf("failed to read epoch seed: %w", err)
	}

	seed := computeEpochSeed(ctx.ChainID(), epochID, ctx.HeaderHash())
	if err := k.EpochSeeds.Set(ctx, epochID, seed); err != nil {
		return fmt.Errorf("failed to persist epoch seed: %w", err)
	}
	return nil
}

func (k Keeper) EndBlock(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	height := uint64(ctx.BlockHeight())
	epochID, err := epochIDForHeight(height, params.EpochLenBlocks)
	if err != nil {
		return err
	}
	startHeight, err := epochStartHeight(epochID, params.EpochLenBlocks)
	if err != nil {
		return err
	}
	endHeight := startHeight + params.EpochLenBlocks - 1
	if endHeight < startHeight {
		return fmt.Errorf("epoch_end_height overflow")
	}
	if height != endHeight {
		return nil
	}

	epochSeed, err := k.mustEpochSeed(ctx, epochID)
	if err != nil {
		return err
	}

	return k.Deals.Walk(goCtx, nil, func(dealID uint64, deal types.Deal) (stop bool, err error) {
		if deal.TotalMdus == 0 {
			return false, nil
		}
		if len(deal.ManifestRoot) != 48 {
			return false, nil
		}
		if height < deal.StartBlock || height > deal.EndBlock {
			return false, nil
		}

		stripe, err := stripeParamsForDeal(deal)
		if err != nil {
			return false, nil
		}

		quotaBlobs, err := quotaBlobsForAssignment(params, deal, stripe)
		if err != nil {
			return false, nil
		}

		creditCap := creditCapBlobs(quotaBlobs, params.CreditCapBps)
		epochKey := collections.Join(epochID, deal.Id)

		if stripe.mode == 2 {
			if len(deal.Mode2Slots) == 0 {
				return false, nil
			}

			reserved := make(map[string]struct{}, len(deal.Mode2Slots))
			for _, slot := range deal.Mode2Slots {
				if slot == nil {
					continue
				}
				if strings.TrimSpace(slot.Provider) != "" {
					reserved[strings.TrimSpace(slot.Provider)] = struct{}{}
				}
				if strings.TrimSpace(slot.PendingProvider) != "" {
					reserved[strings.TrimSpace(slot.PendingProvider)] = struct{}{}
				}
			}

			updated := false

			for idx, slot := range deal.Mode2Slots {
				if slot == nil {
					continue
				}
				slotNum := uint64(slot.Slot)

				if slot.Status != types.SlotStatus_SLOT_STATUS_ACTIVE {
					continue
				}

				creditsSeen, err := k.EpochCreditsMode2.Get(ctx, collections.Join(epochKey, slotNum))
				if err != nil && !errors.Is(err, collections.ErrNotFound) {
					return false, err
				}
				synthSeen, err := k.EpochSyntheticMode2.Get(ctx, collections.Join(epochKey, slotNum))
				if err != nil && !errors.Is(err, collections.ErrNotFound) {
					return false, err
				}

				creditsApplied := creditsSeen
				if creditsApplied > creditCap {
					creditsApplied = creditCap
				}
				if creditsApplied > quotaBlobs {
					creditsApplied = quotaBlobs
				}
				satisfied := creditsApplied + synthSeen

				missedKey := collections.Join(deal.Id, slotNum)
				if satisfied >= quotaBlobs {
					if err := k.MissedEpochsMode2.Remove(ctx, missedKey); err != nil && !errors.Is(err, collections.ErrNotFound) {
						return false, err
					}
					continue
				}

				missedEpochs, err := k.MissedEpochsMode2.Get(ctx, missedKey)
				if err != nil && !errors.Is(err, collections.ErrNotFound) {
					return false, err
				}
				missedEpochs++
				if err := k.MissedEpochsMode2.Set(ctx, missedKey, missedEpochs); err != nil {
					return false, err
				}

				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						"liveness_quota_shortfall",
						sdk.NewAttribute(types.AttributeKeyDealID, fmt.Sprintf("%d", deal.Id)),
						sdk.NewAttribute("epoch_id", fmt.Sprintf("%d", epochID)),
						sdk.NewAttribute("mode", "2"),
						sdk.NewAttribute("slot", fmt.Sprintf("%d", slotNum)),
						sdk.NewAttribute("quota_blobs", fmt.Sprintf("%d", quotaBlobs)),
						sdk.NewAttribute("credits_seen", fmt.Sprintf("%d", creditsSeen)),
						sdk.NewAttribute("credits_applied", fmt.Sprintf("%d", creditsApplied)),
						sdk.NewAttribute("synthetic_seen", fmt.Sprintf("%d", synthSeen)),
						sdk.NewAttribute("missed_epochs", fmt.Sprintf("%d", missedEpochs)),
					),
				)

				if params.EvictAfterMissedEpochs == 0 || missedEpochs < params.EvictAfterMissedEpochs {
					continue
				}
				if slot.Status != types.SlotStatus_SLOT_STATUS_ACTIVE || strings.TrimSpace(slot.PendingProvider) != "" {
					continue
				}

				candidate, ok, err := k.selectRepairCandidate(ctx, deal, reserved, epochSeed, slotNum)
				if err != nil {
					return false, err
				}
				if !ok {
					continue
				}

				slot.Status = types.SlotStatus_SLOT_STATUS_REPAIRING
				slot.PendingProvider = candidate
				slot.StatusSinceHeight = ctx.BlockHeight()
				slot.RepairTargetGen = deal.CurrentGen
				deal.Mode2Slots[idx] = slot
				reserved[candidate] = struct{}{}
				updated = true

				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						"auto_start_slot_repair",
						sdk.NewAttribute(types.AttributeKeyDealID, fmt.Sprintf("%d", deal.Id)),
						sdk.NewAttribute("slot", fmt.Sprintf("%d", slotNum)),
						sdk.NewAttribute("provider", slot.Provider),
						sdk.NewAttribute("pending_provider", candidate),
						sdk.NewAttribute("repair_target_gen", fmt.Sprintf("%d", slot.RepairTargetGen)),
					),
				)
			}

			if updated {
				if err := k.Deals.Set(ctx, deal.Id, deal); err != nil {
					return false, fmt.Errorf("failed to persist mode2 repair update: %w", err)
				}
			}

			return false, nil
		}

		for _, provider := range deal.Providers {
			creditsSeen, err := k.EpochCreditsMode1.Get(ctx, collections.Join(epochKey, provider))
			if err != nil && !errors.Is(err, collections.ErrNotFound) {
				return false, err
			}
			synthSeen, err := k.EpochSyntheticMode1.Get(ctx, collections.Join(epochKey, provider))
			if err != nil && !errors.Is(err, collections.ErrNotFound) {
				return false, err
			}

			creditsApplied := creditsSeen
			if creditsApplied > creditCap {
				creditsApplied = creditCap
			}
			if creditsApplied > quotaBlobs {
				creditsApplied = quotaBlobs
			}
			satisfied := creditsApplied + synthSeen

			missedKey := collections.Join(deal.Id, provider)
			if satisfied >= quotaBlobs {
				if err := k.MissedEpochsMode1.Remove(ctx, missedKey); err != nil && !errors.Is(err, collections.ErrNotFound) {
					return false, err
				}
				continue
			}

			missedEpochs, err := k.MissedEpochsMode1.Get(ctx, missedKey)
			if err != nil && !errors.Is(err, collections.ErrNotFound) {
				return false, err
			}
			missedEpochs++
			if err := k.MissedEpochsMode1.Set(ctx, missedKey, missedEpochs); err != nil {
				return false, err
			}

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"liveness_quota_shortfall",
					sdk.NewAttribute(types.AttributeKeyDealID, fmt.Sprintf("%d", deal.Id)),
					sdk.NewAttribute("epoch_id", fmt.Sprintf("%d", epochID)),
					sdk.NewAttribute("mode", "1"),
					sdk.NewAttribute("provider", provider),
					sdk.NewAttribute("quota_blobs", fmt.Sprintf("%d", quotaBlobs)),
					sdk.NewAttribute("credits_seen", fmt.Sprintf("%d", creditsSeen)),
					sdk.NewAttribute("credits_applied", fmt.Sprintf("%d", creditsApplied)),
					sdk.NewAttribute("synthetic_seen", fmt.Sprintf("%d", synthSeen)),
					sdk.NewAttribute("missed_epochs", fmt.Sprintf("%d", missedEpochs)),
				),
			)
		}

		return false, nil
	})
}

func (k Keeper) selectRepairCandidate(ctx sdk.Context, deal types.Deal, reserved map[string]struct{}, epochSeed []byte, slot uint64) (string, bool, error) {
	parsedHint, _ := types.ParseServiceHint(deal.ServiceHint)
	serviceHint := strings.TrimSpace(parsedHint.Base)

	var candidates []string
	err := k.Providers.Walk(ctx, nil, func(key string, provider types.Provider) (stop bool, err error) {
		if provider.Status != "Active" {
			return false, nil
		}

		if serviceHint == "Hot" && provider.Capabilities != "General" && provider.Capabilities != "Edge" {
			return false, nil
		}
		if serviceHint == "Cold" && provider.Capabilities != "Archive" && provider.Capabilities != "General" {
			return false, nil
		}

		addr := strings.TrimSpace(provider.Address)
		if addr == "" {
			return false, nil
		}
		if _, ok := reserved[addr]; ok {
			return false, nil
		}
		candidates = append(candidates, addr)
		return false, nil
	})
	if err != nil {
		return "", false, fmt.Errorf("failed to walk providers: %w", err)
	}
	if len(candidates) == 0 {
		return "", false, nil
	}

	buf := make([]byte, 0, len(repairTag)+len(epochSeed)+8+8+8)
	buf = append(buf, []byte(repairTag)...)
	buf = append(buf, epochSeed...)
	buf = append(buf, u64be(deal.Id)...)
	buf = append(buf, u64be(deal.CurrentGen)...)
	buf = append(buf, u64be(slot)...)
	sum := sha256.Sum256(buf)
	idx := binary.BigEndian.Uint64(sum[0:8]) % uint64(len(candidates))
	return candidates[idx], true, nil
}

func (k Keeper) mustEpochSeed(ctx sdk.Context, epochID uint64) ([]byte, error) {
	seed, err := k.EpochSeeds.Get(ctx, epochID)
	if err == nil {
		return seed, nil
	}
	if !errors.Is(err, collections.ErrNotFound) {
		return nil, fmt.Errorf("failed to load epoch seed: %w", err)
	}

	// Fallback: allow computing the seed if called at the epoch boundary (useful in tests).
	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())
	expectedEpoch, errEpoch := epochIDForHeight(height, params.EpochLenBlocks)
	if errEpoch != nil {
		return nil, errEpoch
	}
	startHeight, errStart := epochStartHeight(epochID, params.EpochLenBlocks)
	if errStart != nil {
		return nil, errStart
	}
	if expectedEpoch != epochID || height != startHeight {
		return nil, fmt.Errorf("epoch seed missing for epoch %d", epochID)
	}

	seed = computeEpochSeed(ctx.ChainID(), epochID, ctx.HeaderHash())
	if err := k.EpochSeeds.Set(ctx, epochID, seed); err != nil {
		return nil, fmt.Errorf("failed to persist epoch seed: %w", err)
	}
	return seed, nil
}

func dealMduCounts(deal types.Deal) (metaMdus uint64, userMdus uint64, err error) {
	metaMdus = 1 + deal.WitnessMdus
	if deal.TotalMdus == 0 {
		return metaMdus, 0, fmt.Errorf("deal.total_mdus is not initialized")
	}
	if deal.TotalMdus <= metaMdus {
		return metaMdus, 0, fmt.Errorf("deal.total_mdus (%d) must exceed meta_mdus (%d)", deal.TotalMdus, metaMdus)
	}
	userMdus = deal.TotalMdus - metaMdus
	return metaMdus, userMdus, nil
}

func quotaBpsForDeal(params types.Params, deal types.Deal) uint64 {
	hint, err := types.ParseServiceHint(deal.ServiceHint)
	if err == nil && strings.EqualFold(strings.TrimSpace(hint.Base), "cold") {
		return params.QuotaBpsPerEpochCold
	}
	return params.QuotaBpsPerEpochHot
}

func quotaBlobsForAssignment(params types.Params, deal types.Deal, stripe stripeParams) (uint64, error) {
	_, userMdus, err := dealMduCounts(deal)
	if err != nil {
		return 0, err
	}

	var slotBytes math.Int
	switch stripe.mode {
	case 2:
		// slot_bytes = user_mdus * rows * BLOB_SIZE
		slotBytes = math.NewIntFromUint64(userMdus).
			MulRaw(int64(stripe.rows)).
			MulRaw(int64(types.BlobSizeBytes))
	default:
		// slot_bytes = user_mdus * MDU_SIZE (BLOB_SIZE * 64)
		slotBytes = math.NewIntFromUint64(userMdus).
			MulRaw(int64(types.BlobSizeBytes)).
			MulRaw(int64(types.BlobsPerMdu))
	}

	quotaBps := quotaBpsForDeal(params, deal)
	targetBytes := slotBytes.MulRaw(int64(quotaBps)).AddRaw(9999).QuoRaw(10000) // ceil
	targetBlobs := targetBytes.AddRaw(int64(types.BlobSizeBytes - 1)).QuoRaw(int64(types.BlobSizeBytes))
	if !targetBlobs.IsUint64() {
		return 0, fmt.Errorf("target blob count overflow")
	}

	quota := targetBlobs.Uint64()
	if quota < params.QuotaMinBlobs {
		quota = params.QuotaMinBlobs
	}
	if params.QuotaMaxBlobs > 0 && quota > params.QuotaMaxBlobs {
		quota = params.QuotaMaxBlobs
	}
	if quota == 0 {
		return 0, fmt.Errorf("quota_blobs resolved to 0")
	}
	return quota, nil
}

func creditCapBlobs(quotaBlobs uint64, creditCapBps uint64) uint64 {
	if quotaBlobs == 0 {
		return 0
	}
	if creditCapBps >= 10000 {
		return quotaBlobs
	}
	return (quotaBlobs*creditCapBps + 9999) / 10000 // ceil
}

func u64be(v uint64) []byte {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], v)
	return tmp[:]
}

func assignmentBytesMode1(provider string) ([]byte, error) {
	addr, err := sdk.AccAddressFromBech32(strings.TrimSpace(provider))
	if err != nil {
		return nil, fmt.Errorf("invalid provider address: %w", err)
	}
	if len(addr) != 20 {
		return nil, fmt.Errorf("provider address must be 20 bytes (got %d)", len(addr))
	}
	out := make([]byte, 20)
	copy(out, addr.Bytes())
	return out, nil
}

func assignmentBytesMode2(slot uint64) []byte {
	return u64be(slot)
}

func computeCreditID(epochID uint64, dealID uint64, currentGen uint64, assignment []byte, mduIndex uint64, blobIndex uint64) []byte {
	buf := make([]byte, 0, len(creditIDTag)+8*5+len(assignment))
	buf = append(buf, []byte(creditIDTag)...)
	buf = append(buf, u64be(epochID)...)
	buf = append(buf, u64be(dealID)...)
	buf = append(buf, u64be(currentGen)...)
	buf = append(buf, assignment...)
	buf = append(buf, u64be(mduIndex)...)
	buf = append(buf, u64be(blobIndex)...)
	sum := sha256.Sum256(buf)
	out := make([]byte, 32)
	copy(out, sum[:])
	return out
}

func computeSyntheticID(epochID uint64, dealID uint64, currentGen uint64, assignment []byte, mduIndex uint64, blobIndex uint64) []byte {
	buf := make([]byte, 0, len(syntheticIDTag)+8*5+len(assignment))
	buf = append(buf, []byte(syntheticIDTag)...)
	buf = append(buf, u64be(epochID)...)
	buf = append(buf, u64be(dealID)...)
	buf = append(buf, u64be(currentGen)...)
	buf = append(buf, assignment...)
	buf = append(buf, u64be(mduIndex)...)
	buf = append(buf, u64be(blobIndex)...)
	sum := sha256.Sum256(buf)
	out := make([]byte, 32)
	copy(out, sum[:])
	return out
}

func deriveMode1Challenge(epochSeed []byte, dealID uint64, currentGen uint64, provider20 []byte, userMdus uint64, metaMdus uint64, ordinal uint64) (mduIndex uint64, blobIndex uint64) {
	buf := make([]byte, 0, len(challengeTag)+len(epochSeed)+8+8+20+8)
	buf = append(buf, []byte(challengeTag)...)
	buf = append(buf, epochSeed...)
	buf = append(buf, u64be(dealID)...)
	buf = append(buf, u64be(currentGen)...)
	buf = append(buf, provider20...)
	buf = append(buf, u64be(ordinal)...)
	sum := sha256.Sum256(buf)

	mduOrdinal := binary.BigEndian.Uint64(sum[0:8]) % userMdus
	blobIndex = binary.BigEndian.Uint64(sum[8:16]) % types.BlobsPerMdu
	mduIndex = metaMdus + mduOrdinal
	return mduIndex, blobIndex
}

func deriveMode2Challenge(epochSeed []byte, dealID uint64, currentGen uint64, slot uint64, userMdus uint64, metaMdus uint64, rows uint64, ordinal uint64) (mduIndex uint64, leafIndex uint64) {
	buf := make([]byte, 0, len(challengeTag)+len(epochSeed)+8*4)
	buf = append(buf, []byte(challengeTag)...)
	buf = append(buf, epochSeed...)
	buf = append(buf, u64be(dealID)...)
	buf = append(buf, u64be(currentGen)...)
	buf = append(buf, u64be(slot)...)
	buf = append(buf, u64be(ordinal)...)
	sum := sha256.Sum256(buf)

	mduOrdinal := binary.BigEndian.Uint64(sum[0:8]) % userMdus
	row := binary.BigEndian.Uint64(sum[8:16]) % rows
	mduIndex = metaMdus + mduOrdinal
	leafIndex = slot*rows + row
	return mduIndex, leafIndex
}

func (k Keeper) recordCreditForProof(ctx sdk.Context, epochID uint64, deal types.Deal, stripe stripeParams, provider string, mduIndex uint64, blobIndex uint64) error {
	var (
		id            []byte
		epochKey      = collections.Join(epochID, deal.Id)
		height        = uint64(ctx.BlockHeight())
		creditsKeyAny any
	)

	metaMdus, _, err := dealMduCounts(deal)
	if err != nil {
		return err
	}
	if mduIndex < metaMdus {
		return nil
	}

	if stripe.mode == 2 {
		slot, ok := providerSlotIndex(deal, provider)
		if !ok {
			return fmt.Errorf("provider not assigned to deal")
		}
		if int(slot) < len(deal.Mode2Slots) {
			slotState := deal.Mode2Slots[slot]
			if slotState != nil && slotState.Status != types.SlotStatus_SLOT_STATUS_ACTIVE {
				return nil
			}
		}
		assignment := assignmentBytesMode2(slot)
		id = computeCreditID(epochID, deal.Id, deal.CurrentGen, assignment, mduIndex, blobIndex)
		creditsKeyAny = collections.Join(epochKey, slot)
	} else {
		assignment, err := assignmentBytesMode1(provider)
		if err != nil {
			return err
		}
		id = computeCreditID(epochID, deal.Id, deal.CurrentGen, assignment, mduIndex, blobIndex)
		creditsKeyAny = collections.Join(epochKey, provider)
	}

	if _, err := k.CreditSeen.Get(ctx, id); err == nil {
		return nil
	} else if !errors.Is(err, collections.ErrNotFound) {
		return fmt.Errorf("failed to load credit seen: %w", err)
	}
	if err := k.CreditSeen.Set(ctx, id, height); err != nil {
		return fmt.Errorf("failed to store credit seen: %w", err)
	}

	if stripe.mode == 2 {
		key := creditsKeyAny.(collections.Pair[collections.Pair[uint64, uint64], uint64])
		current, err := k.EpochCreditsMode2.Get(ctx, key)
		if err != nil && !errors.Is(err, collections.ErrNotFound) {
			return fmt.Errorf("failed to load epoch credits: %w", err)
		}
		if err := k.EpochCreditsMode2.Set(ctx, key, current+1); err != nil {
			return fmt.Errorf("failed to update epoch credits: %w", err)
		}
		return nil
	}

	key := creditsKeyAny.(collections.Pair[collections.Pair[uint64, uint64], string])
	current, err := k.EpochCreditsMode1.Get(ctx, key)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return fmt.Errorf("failed to load epoch credits: %w", err)
	}
	if err := k.EpochCreditsMode1.Set(ctx, key, current+1); err != nil {
		return fmt.Errorf("failed to update epoch credits: %w", err)
	}
	return nil
}

func (k Keeper) recordSyntheticForProof(ctx sdk.Context, epochID uint64, deal types.Deal, stripe stripeParams, provider string, mduIndex uint64, blobIndex uint64) (bool, error) {
	var (
		id          []byte
		epochKey    = collections.Join(epochID, deal.Id)
		height      = uint64(ctx.BlockHeight())
		synthKeyAny any
	)

	if stripe.mode == 2 {
		slot, ok := providerSlotIndex(deal, provider)
		if !ok {
			return false, fmt.Errorf("provider not assigned to deal")
		}
		assignment := assignmentBytesMode2(slot)
		id = computeSyntheticID(epochID, deal.Id, deal.CurrentGen, assignment, mduIndex, blobIndex)
		synthKeyAny = collections.Join(epochKey, slot)
	} else {
		assignment, err := assignmentBytesMode1(provider)
		if err != nil {
			return false, err
		}
		id = computeSyntheticID(epochID, deal.Id, deal.CurrentGen, assignment, mduIndex, blobIndex)
		synthKeyAny = collections.Join(epochKey, provider)
	}

	if _, err := k.SyntheticSeen.Get(ctx, id); err == nil {
		return false, nil
	} else if !errors.Is(err, collections.ErrNotFound) {
		return false, fmt.Errorf("failed to load synthetic seen: %w", err)
	}
	if err := k.SyntheticSeen.Set(ctx, id, height); err != nil {
		return false, fmt.Errorf("failed to store synthetic seen: %w", err)
	}

	if stripe.mode == 2 {
		key := synthKeyAny.(collections.Pair[collections.Pair[uint64, uint64], uint64])
		current, err := k.EpochSyntheticMode2.Get(ctx, key)
		if err != nil && !errors.Is(err, collections.ErrNotFound) {
			return false, fmt.Errorf("failed to load epoch synthetic: %w", err)
		}
		if err := k.EpochSyntheticMode2.Set(ctx, key, current+1); err != nil {
			return false, fmt.Errorf("failed to update epoch synthetic: %w", err)
		}
		return true, nil
	}

	key := synthKeyAny.(collections.Pair[collections.Pair[uint64, uint64], string])
	current, err := k.EpochSyntheticMode1.Get(ctx, key)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return false, fmt.Errorf("failed to load epoch synthetic: %w", err)
	}
	if err := k.EpochSyntheticMode1.Set(ctx, key, current+1); err != nil {
		return false, fmt.Errorf("failed to update epoch synthetic: %w", err)
	}
	return true, nil
}
