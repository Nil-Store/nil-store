package keeper

import (
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"nilchain/x/nilchain/types"
)

func (k Keeper) mintAuditBudget(ctx sdk.Context, params types.Params) error {
	epochID := epochIDAtHeight(ctx.BlockHeight(), params.EpochLenBlocks)
	if epochID == 0 {
		return nil
	}

	totalActiveSlotBytes, err := k.totalActiveSlotBytes(ctx)
	if err != nil {
		return err
	}

	rentDec := math.LegacyNewDec(0)
	if totalActiveSlotBytes > 0 && params.StoragePrice.IsPositive() && params.EpochLenBlocks > 0 {
		rentDec = params.StoragePrice
		rentDec = rentDec.MulInt(math.NewIntFromUint64(totalActiveSlotBytes))
		rentDec = rentDec.MulInt(math.NewIntFromUint64(params.EpochLenBlocks))
	}

	budget := auditBudgetFromRent(rentDec, params.AuditBudgetBps)
	cap := auditBudgetFromRent(rentDec, params.AuditBudgetCapBps)
	minted := budget
	capBound := false
	if cap.IsPositive() && minted.GT(cap) {
		minted = cap
		capBound = true
	}

	if minted.IsPositive() {
		coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, minted))
		if err := k.BankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
			return fmt.Errorf("failed to mint audit budget: %w", err)
		}
		if err := k.AuditBudgetByEpoch.Set(ctx, epochID, minted); err != nil {
			return fmt.Errorf("failed to record audit budget: %w", err)
		}
	} else {
		if err := k.AuditBudgetByEpoch.Remove(ctx, epochID); err != nil && !errors.Is(err, collections.ErrNotFound) {
			return fmt.Errorf("failed to clear audit budget entry: %w", err)
		}
	}

	oldestAllowed := uint64(1)
	if params.AuditBudgetCarryoverEpochs > 0 && epochID > params.AuditBudgetCarryoverEpochs {
		oldestAllowed = epochID - params.AuditBudgetCarryoverEpochs + 1
	}

	available := math.ZeroInt()
	expiredTotal := math.ZeroInt()
	err = k.AuditBudgetByEpoch.Walk(ctx, nil, func(id uint64, amount math.Int) (bool, error) {
		if id < oldestAllowed {
			if amount.IsPositive() {
				expiredTotal = expiredTotal.Add(amount)
			}
			if err := k.AuditBudgetByEpoch.Remove(ctx, id); err != nil {
				return true, err
			}
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.TypeAuditBudgetExpire,
					sdk.NewAttribute(types.AttributeKeyEpochID, fmt.Sprintf("%d", epochID)),
					sdk.NewAttribute(types.AttributeKeyExpiredEpoch, fmt.Sprintf("%d", id)),
					sdk.NewAttribute(types.AttributeKeyExpiredAmount, amount.String()),
				),
			)
			return false, nil
		}
		if amount.IsPositive() {
			available = available.Add(amount)
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	if expiredTotal.IsPositive() {
		coins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, expiredTotal))
		if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
			return fmt.Errorf("failed to burn expired audit budget: %w", err)
		}
	}

	if err := k.AuditBudgetAvailable.Set(ctx, available); err != nil {
		return fmt.Errorf("failed to store audit budget availability: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeAuditBudgetRent,
			sdk.NewAttribute(types.AttributeKeyEpochID, fmt.Sprintf("%d", epochID)),
			sdk.NewAttribute(types.AttributeKeyEpochSlotRent, rentDec.String()),
			sdk.NewAttribute(types.AttributeKeyTotalSlotBytes, fmt.Sprintf("%d", totalActiveSlotBytes)),
		),
	)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeAuditBudgetMint,
			sdk.NewAttribute(types.AttributeKeyEpochID, fmt.Sprintf("%d", epochID)),
			sdk.NewAttribute(types.AttributeKeyAuditBudgetMint, minted.String()),
			sdk.NewAttribute(types.AttributeKeyAuditBudgetCap, cap.String()),
			sdk.NewAttribute(types.AttributeKeyCapBound, fmt.Sprintf("%t", capBound)),
		),
	)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeAuditBudgetCarryover,
			sdk.NewAttribute(types.AttributeKeyEpochID, fmt.Sprintf("%d", epochID)),
			sdk.NewAttribute(types.AttributeKeyCarryoverEpochs, fmt.Sprintf("%d", params.AuditBudgetCarryoverEpochs)),
			sdk.NewAttribute(types.AttributeKeyAuditBudgetAvail, available.String()),
		),
	)

	return nil
}

// SpendAuditBudget debits the available audit budget and transfers funds to the recipient.
func (k Keeper) SpendAuditBudget(ctx sdk.Context, recipient sdk.AccAddress, amount sdk.Coin, reason string) error {
	if !amount.IsValid() {
		return sdkerrors.ErrInvalidRequest.Wrapf("invalid audit budget amount: %s", amount)
	}
	if !amount.Amount.IsPositive() {
		return nil
	}
	if amount.Denom != sdk.DefaultBondDenom {
		return sdkerrors.ErrInvalidRequest.Wrapf("invalid audit budget denom: %s", amount.Denom)
	}

	available, err := k.AuditBudgetAvailable.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		available = math.ZeroInt()
	}

	if available.LT(amount.Amount) {
		return sdkerrors.ErrInsufficientFunds.Wrapf("audit budget %s < spend %s", available.String(), amount.Amount.String())
	}

	coins := sdk.NewCoins(amount)
	if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, coins); err != nil {
		return fmt.Errorf("failed to spend audit budget: %w", err)
	}

	available = available.Sub(amount.Amount)
	if err := k.AuditBudgetAvailable.Set(ctx, available); err != nil {
		return fmt.Errorf("failed to update audit budget availability: %w", err)
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "unspecified"
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.TypeAuditBudgetSpend,
			sdk.NewAttribute(types.AttributeKeyAuditBudgetSpend, amount.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyAuditBudgetAvail, available.String()),
			sdk.NewAttribute(types.AttributeKeyReason, reason),
		),
	)

	return nil
}

func auditBudgetFromRent(rent math.LegacyDec, bps uint64) math.Int {
	if bps == 0 || !rent.IsPositive() {
		return math.ZeroInt()
	}
	portion := rent.MulInt(math.NewIntFromUint64(bps)).QuoInt(math.NewInt(10000))
	return portion.Ceil().TruncateInt()
}

func (k Keeper) totalActiveSlotBytes(ctx sdk.Context) (uint64, error) {
	height := uint64(ctx.BlockHeight())
	total := uint64(0)

	err := k.Deals.Walk(ctx, nil, func(dealID uint64, deal types.Deal) (bool, error) {
		if height < deal.StartBlock || height > deal.EndBlock {
			return false, nil
		}
		in, ok := slabInputs(deal)
		if !ok {
			return false, nil
		}
		slotBytes, err := slotBytesForDeal(deal, in)
		if err != nil || slotBytes == 0 {
			return false, err
		}

		slotCount := uint64(0)
		if deal.RedundancyMode == 2 && len(deal.Mode2Slots) > 0 {
			for _, slot := range deal.Mode2Slots {
				if slot == nil {
					continue
				}
				if slot.Status != types.SlotStatus_SLOT_STATUS_ACTIVE {
					continue
				}
				if strings.TrimSpace(slot.Provider) == "" {
					continue
				}
				slotCount++
			}
		} else {
			for _, provider := range deal.Providers {
				if strings.TrimSpace(provider) == "" {
					continue
				}
				slotCount++
			}
		}

		if slotCount == 0 {
			return false, nil
		}
		add, overflow := mulUint64(slotBytes, slotCount)
		if overflow {
			return true, fmt.Errorf("total active slot bytes overflow for deal %d", dealID)
		}
		next, overflow := addUint64(total, add)
		if overflow {
			return true, fmt.Errorf("total active slot bytes overflow")
		}
		total = next
		return false, nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}
