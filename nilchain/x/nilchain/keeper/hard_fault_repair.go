package keeper

import (
	"encoding/binary"
	"strconv"
	"strings"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"nilchain/x/nilchain/types"
)

func (k Keeper) triggerHardFaultRepair(ctx sdk.Context, dealID uint64, provider string, reason string) error {
	deal, err := k.Deals.Get(ctx, dealID)
	if err != nil {
		return err
	}
	if deal.RedundancyMode != 2 || deal.Mode2Profile == nil || len(deal.Mode2Slots) == 0 {
		return nil
	}
	slotIdxU64, ok := providerSlotIndex(deal, provider)
	if !ok || int(slotIdxU64) >= len(deal.Mode2Slots) {
		return nil
	}
	slot := uint32(slotIdxU64)
	entry := deal.Mode2Slots[slotIdxU64]
	if entry == nil || entry.Status != types.SlotStatus_SLOT_STATUS_ACTIVE {
		return nil
	}
	if strings.TrimSpace(entry.PendingProvider) != "" {
		return nil
	}

	params := k.GetParams(ctx)
	epochID := epochIDAtHeight(ctx.BlockHeight(), params.EpochLenBlocks)

	attempt, windowStart, err := k.prepareMode2RepairStart(ctx, dealID, slot)
	if err != nil {
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.TypeRepairRejected,
			sdk.NewAttribute(types.AttributeKeyDealID, strconv.FormatUint(dealID, 10)),
			sdk.NewAttribute(types.AttributeKeySlot, strconv.FormatUint(slotIdxU64, 10)),
			sdk.NewAttribute(types.AttributeKeyReason, err.Error()),
		))
		return nil
	}

	pending, err := k.selectMode2ReplacementProvider(ctx, deal, slot, epochID, attempt)
	if err != nil {
		return err
	}

	requiredBond, err := k.requiredBondPerSlot(ctx, deal)
	if err != nil {
		return err
	}
	if err := k.lockProviderBond(ctx, pending, requiredBond); err != nil {
		return err
	}

	entry.Status = types.SlotStatus_SLOT_STATUS_REPAIRING
	entry.PendingProvider = strings.TrimSpace(pending)
	entry.StatusSinceHeight = ctx.BlockHeight()
	entry.RepairTargetGen = deal.CurrentGen
	deal.Mode2Slots[slotIdxU64] = entry

	if err := k.Deals.Set(ctx, dealID, deal); err != nil {
		return err
	}
	if err := k.recordMode2RepairStart(ctx, dealID, slot, attempt, windowStart); err != nil {
		return err
	}
	_ = k.Mode2MissedEpochs.Remove(ctx, collections.Join(dealID, slot))

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.TypeHealthRepairStarted,
		sdk.NewAttribute(types.AttributeKeyDealID, strconv.FormatUint(dealID, 10)),
		sdk.NewAttribute(types.AttributeKeyEpochID, strconv.FormatUint(epochID, 10)),
		sdk.NewAttribute(types.AttributeKeySlot, strconv.FormatUint(slotIdxU64, 10)),
		sdk.NewAttribute(types.AttributeKeyProvider, provider),
		sdk.NewAttribute(types.AttributeKeyPendingProvider, entry.PendingProvider),
		sdk.NewAttribute(types.AttributeKeyReason, reason),
	))

	extra := make([]byte, 0, 4)
	extra = binary.BigEndian.AppendUint32(extra, slot)
	eid := deriveEvidenceID("hard_fault_repair_started", dealID, epochID, extra)
	if err := k.recordEvidenceSummary(ctx, dealID, provider, "hard_fault_repair_started", eid[:], "chain", false); err != nil {
		ctx.Logger().Error("failed to record evidence summary", "error", err)
	}

	return nil
}
