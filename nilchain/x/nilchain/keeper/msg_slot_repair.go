package keeper

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"nilchain/x/nilchain/types"
)

func (k msgServer) StartSlotRepair(goCtx context.Context, msg *types.MsgStartSlotRepair) (*types.MsgStartSlotRepairResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	deal, err := k.Deals.Get(ctx, msg.DealId)
	if err != nil {
		return nil, sdkerrors.ErrNotFound.Wrapf("deal %d not found", msg.DealId)
	}
	if deal.Owner != msg.Creator {
		return nil, sdkerrors.ErrUnauthorized.Wrap("only deal owner can start slot repair")
	}
	if deal.RedundancyMode != 2 {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("slot repair is only supported for Mode 2 deals")
	}
	if deal.Mode2Profile == nil || len(deal.Mode2Slots) == 0 {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("mode2 slot map is not initialized")
	}

	slotIdx := int(msg.Slot)
	if slotIdx < 0 || slotIdx >= len(deal.Mode2Slots) {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("invalid slot %d", msg.Slot)
	}

	slot := deal.Mode2Slots[slotIdx]
	if slot == nil {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("mode2 slot %d is nil", msg.Slot)
	}
	if slot.Status != types.SlotStatus_SLOT_STATUS_ACTIVE {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("slot %d is not active", msg.Slot)
	}

	pending := strings.TrimSpace(msg.PendingProvider)
	if pending == "" {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("pending_provider is required")
	}
	if strings.TrimSpace(slot.Provider) == pending {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("pending_provider must differ from current provider")
	}
	if _, err := k.Providers.Get(ctx, pending); err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, sdkerrors.ErrNotFound.Wrapf("pending provider %q not registered", pending)
		}
		return nil, fmt.Errorf("failed to load pending provider: %w", err)
	}

	requiredBond, err := k.requiredBondPerSlot(ctx, deal)
	if err != nil {
		return nil, err
	}
	if err := k.lockProviderBond(ctx, pending, requiredBond); err != nil {
		return nil, err
	}

	slot.Status = types.SlotStatus_SLOT_STATUS_REPAIRING
	slot.PendingProvider = pending
	slot.StatusSinceHeight = ctx.BlockHeight()
	slot.RepairTargetGen = deal.CurrentGen
	deal.Mode2Slots[slotIdx] = slot

	if err := k.Deals.Set(ctx, deal.Id, deal); err != nil {
		return nil, fmt.Errorf("failed to update deal: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"start_slot_repair",
			sdk.NewAttribute(types.AttributeKeyDealID, fmt.Sprintf("%d", deal.Id)),
			sdk.NewAttribute("slot", fmt.Sprintf("%d", msg.Slot)),
			sdk.NewAttribute("provider", slot.Provider),
			sdk.NewAttribute("pending_provider", slot.PendingProvider),
			sdk.NewAttribute("repair_target_gen", fmt.Sprintf("%d", slot.RepairTargetGen)),
		),
	)

	return &types.MsgStartSlotRepairResponse{Success: true}, nil
}

func (k msgServer) CompleteSlotRepair(goCtx context.Context, msg *types.MsgCompleteSlotRepair) (*types.MsgCompleteSlotRepairResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	deal, err := k.Deals.Get(ctx, msg.DealId)
	if err != nil {
		return nil, sdkerrors.ErrNotFound.Wrapf("deal %d not found", msg.DealId)
	}
	if deal.RedundancyMode != 2 {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("slot repair is only supported for Mode 2 deals")
	}
	if deal.Mode2Profile == nil || len(deal.Mode2Slots) == 0 {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("mode2 slot map is not initialized")
	}

	slotIdx := int(msg.Slot)
	if slotIdx < 0 || slotIdx >= len(deal.Mode2Slots) {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("invalid slot %d", msg.Slot)
	}

	slot := deal.Mode2Slots[slotIdx]
	if slot == nil {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("mode2 slot %d is nil", msg.Slot)
	}
	if slot.Status != types.SlotStatus_SLOT_STATUS_REPAIRING || strings.TrimSpace(slot.PendingProvider) == "" {
		return nil, sdkerrors.ErrInvalidRequest.Wrapf("slot %d has no pending repair", msg.Slot)
	}
	if slot.RepairTargetGen != deal.CurrentGen {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("slot repair target generation is stale")
	}

	creator := strings.TrimSpace(msg.Creator)
	if creator == "" {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("creator is required")
	}
	if creator != deal.Owner && creator != strings.TrimSpace(slot.PendingProvider) {
		return nil, sdkerrors.ErrUnauthorized.Wrap("only deal owner or pending provider can complete slot repair")
	}

	ready, err := k.slotRepairReady(ctx, deal, msg.Slot)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, sdkerrors.ErrInvalidRequest.Wrap("slot repair not ready (quota not met)")
	}

	oldProvider := slot.Provider
	slot.Provider = slot.PendingProvider
	slot.PendingProvider = ""
	slot.Status = types.SlotStatus_SLOT_STATUS_ACTIVE
	slot.StatusSinceHeight = ctx.BlockHeight()
	slot.RepairTargetGen = 0
	deal.Mode2Slots[slotIdx] = slot

	// Keep legacy providers[] aligned with the canonical slots map when possible.
	if slotIdx >= 0 && slotIdx < len(deal.Providers) {
		deal.Providers[slotIdx] = slot.Provider
	}

	// Rotate the deterministic challenge set after replacement so a failing provider
	// cannot keep replaying historical proofs.
	deal.CurrentGen++

	requiredBond, err := k.requiredBondPerSlot(ctx, deal)
	if err != nil {
		return nil, err
	}
	if err := k.unlockProviderBond(ctx, oldProvider, requiredBond); err != nil {
		return nil, err
	}

	if err := k.Deals.Set(ctx, deal.Id, deal); err != nil {
		return nil, fmt.Errorf("failed to update deal: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"complete_slot_repair",
			sdk.NewAttribute(types.AttributeKeyDealID, fmt.Sprintf("%d", deal.Id)),
			sdk.NewAttribute("slot", fmt.Sprintf("%d", msg.Slot)),
			sdk.NewAttribute("old_provider", oldProvider),
			sdk.NewAttribute("new_provider", slot.Provider),
			sdk.NewAttribute("current_gen", fmt.Sprintf("%d", deal.CurrentGen)),
		),
	)

	return &types.MsgCompleteSlotRepairResponse{Success: true}, nil
}

func (k msgServer) slotRepairReady(ctx sdk.Context, deal types.Deal, slot uint32) (bool, error) {
	params := k.GetParams(ctx)
	if params.EpochLenBlocks == 0 {
		return false, sdkerrors.ErrInvalidRequest.Wrap("epoch_len_blocks is 0 (liveness disabled)")
	}

	epochID := epochIDAtHeight(ctx.BlockHeight(), params.EpochLenBlocks)
	if epochID == 0 {
		return false, sdkerrors.ErrInvalidRequest.Wrap("epoch id is 0 (liveness disabled)")
	}

	in, ok := slabInputs(deal)
	if !ok {
		return true, nil
	}

	stripe, err := stripeParamsForDeal(deal)
	if err != nil {
		return false, sdkerrors.ErrInvalidRequest.Wrapf("invalid stripe params: %s", err.Error())
	}
	if stripe.mode != 2 {
		return false, sdkerrors.ErrInvalidRequest.Wrap("slot repair readiness only applies to Mode 2 deals")
	}

	quota := requiredBlobsMode2(params, deal, stripe, in)
	if quota == 0 {
		return true, nil
	}

	keyEpoch := mode2EpochKey(deal.Id, slot, epochID)
	creditsRaw, err := k.Mode2EpochCredits.Get(ctx, keyEpoch)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return false, err
	}
	synth, err := k.Mode2EpochSynthetic.Get(ctx, keyEpoch)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return false, err
	}

	creditCap := creditCapBlobs(params, deal, quota)
	credits := creditsRaw
	if creditCap < credits {
		credits = creditCap
	}

	return credits+synth >= quota, nil
}
