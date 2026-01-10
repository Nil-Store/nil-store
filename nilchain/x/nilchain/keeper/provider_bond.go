package keeper

import (
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"nilchain/x/nilchain/types"
)

func (k Keeper) buildProviderBondState(ctx sdk.Context, provider string) (types.ProviderBondState, error) {
	addr, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		return types.ProviderBondState{}, err
	}

	spendable := k.BankKeeper.SpendableCoins(ctx, addr).AmountOf(sdk.DefaultBondDenom)
	return types.ProviderBondState{
		Address:            provider,
		BondedAmount:       sdk.NewCoin(sdk.DefaultBondDenom, spendable),
		LockedAmount:       sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
		UnbondingEndHeight: 0,
	}, nil
}

func (k Keeper) providerBondState(ctx sdk.Context, provider string) (types.ProviderBondState, bool, error) {
	state, err := k.ProviderBonds.Get(ctx, strings.TrimSpace(provider))
	if err == nil {
		return state, true, nil
	}
	if errors.Is(err, collections.ErrNotFound) {
		return types.ProviderBondState{}, false, nil
	}
	return types.ProviderBondState{}, false, err
}

func (k Keeper) ensureProviderBondState(ctx sdk.Context, provider string) (types.ProviderBondState, error) {
	state, found, err := k.providerBondState(ctx, provider)
	if err != nil {
		return types.ProviderBondState{}, err
	}
	if !found {
		state, err = k.buildProviderBondState(ctx, provider)
		if err != nil {
			return types.ProviderBondState{}, err
		}
	}

	addr, err := sdk.AccAddressFromBech32(strings.TrimSpace(provider))
	if err != nil {
		return types.ProviderBondState{}, err
	}
	spendable := k.BankKeeper.SpendableCoins(ctx, addr).AmountOf(sdk.DefaultBondDenom)
	state.BondedAmount = sdk.NewCoin(sdk.DefaultBondDenom, spendable)

	if strings.TrimSpace(state.LockedAmount.Denom) != strings.TrimSpace(sdk.DefaultBondDenom) {
		state.LockedAmount = sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())
	}

	return state, nil
}

func providerAvailableBond(state types.ProviderBondState) math.Int {
	if strings.TrimSpace(state.BondedAmount.Denom) != strings.TrimSpace(sdk.DefaultBondDenom) {
		return math.ZeroInt()
	}
	if strings.TrimSpace(state.LockedAmount.Denom) != strings.TrimSpace(sdk.DefaultBondDenom) {
		return state.BondedAmount.Amount
	}
	if state.BondedAmount.Amount.LT(state.LockedAmount.Amount) {
		return math.ZeroInt()
	}
	return state.BondedAmount.Amount.Sub(state.LockedAmount.Amount)
}

func (k Keeper) lockProviderBond(ctx sdk.Context, provider string, amount math.Int) error {
	if !amount.IsPositive() {
		return nil
	}
	state, err := k.ensureProviderBondState(ctx, provider)
	if err != nil {
		return err
	}
	available := providerAvailableBond(state)
	if available.LT(amount) {
		return fmt.Errorf("insufficient available bond for provider %s: need %s, have %s", provider, amount, available)
	}
	state.LockedAmount = sdk.NewCoin(sdk.DefaultBondDenom, state.LockedAmount.Amount.Add(amount))
	return k.ProviderBonds.Set(ctx, strings.TrimSpace(provider), state)
}

func (k Keeper) unlockProviderBond(ctx sdk.Context, provider string, amount math.Int) error {
	if !amount.IsPositive() {
		return nil
	}
	state, err := k.ensureProviderBondState(ctx, provider)
	if err != nil {
		return err
	}
	locked := state.LockedAmount.Amount
	if locked.LT(amount) {
		return fmt.Errorf("provider %s locked bond underflow: have %s, want %s", provider, locked, amount)
	}
	state.LockedAmount = sdk.NewCoin(sdk.DefaultBondDenom, locked.Sub(amount))
	return k.ProviderBonds.Set(ctx, strings.TrimSpace(provider), state)
}

func (k Keeper) requiredBondPerSlot(ctx sdk.Context, deal types.Deal) (math.Int, error) {
	params := k.GetParams(ctx)
	if params.BondMonths == 0 || params.MonthLenBlocks == 0 {
		return math.ZeroInt(), nil
	}
	if !params.StoragePrice.IsPositive() {
		return math.ZeroInt(), nil
	}

	in, ok := slabInputs(deal)
	if !ok {
		return math.ZeroInt(), nil
	}
	slotBytes, err := slotBytesForDeal(deal, in)
	if err != nil || slotBytes == 0 {
		return math.ZeroInt(), err
	}

	costDec := params.StoragePrice
	costDec = costDec.MulInt(math.NewIntFromUint64(slotBytes))
	costDec = costDec.MulInt(math.NewIntFromUint64(params.MonthLenBlocks))
	costDec = costDec.MulInt(math.NewIntFromUint64(params.BondMonths))
	return costDec.Ceil().TruncateInt(), nil
}

func slotBytesForDeal(deal types.Deal, in quotaInputs) (uint64, error) {
	if in.userMdus == 0 {
		return 0, nil
	}
	if deal.RedundancyMode == 2 {
		stripe, err := stripeParamsForDeal(deal)
		if err != nil {
			return 0, err
		}
		perMdu, overflow := mulUint64(stripe.rows, types.BlobSizeBytes)
		if overflow {
			return 0, fmt.Errorf("slot bytes overflow for deal %d", deal.Id)
		}
		slotBytes, overflow := mulUint64(in.userMdus, perMdu)
		if overflow {
			return 0, fmt.Errorf("slot bytes overflow for deal %d", deal.Id)
		}
		return slotBytes, nil
	}

	slotBytes, overflow := mulUint64(in.userMdus, uint64(types.MDU_SIZE))
	if overflow {
		return 0, fmt.Errorf("slot bytes overflow for deal %d", deal.Id)
	}
	return slotBytes, nil
}

func dealSlotAssignments(deal types.Deal) map[string]uint64 {
	assignments := make(map[string]uint64)
	if deal.RedundancyMode == 2 && len(deal.Mode2Slots) > 0 {
		for _, slot := range deal.Mode2Slots {
			if slot == nil {
				continue
			}
			active := strings.TrimSpace(slot.Provider)
			if active != "" {
				assignments[active]++
			}
			pending := strings.TrimSpace(slot.PendingProvider)
			if pending != "" {
				assignments[pending]++
			}
		}
		return assignments
	}
	for _, provider := range deal.Providers {
		addr := strings.TrimSpace(provider)
		if addr == "" {
			continue
		}
		assignments[addr]++
	}
	return assignments
}

func (k Keeper) lockBondForAssignments(ctx sdk.Context, deal types.Deal, providers []string) error {
	required, err := k.requiredBondPerSlot(ctx, deal)
	if err != nil || !required.IsPositive() {
		return err
	}
	for _, provider := range providers {
		if err := k.lockProviderBond(ctx, provider, required); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) lockBondForDealGrowth(ctx sdk.Context, prev types.Deal, next types.Deal) error {
	oldRequired, err := k.requiredBondPerSlot(ctx, prev)
	if err != nil {
		return err
	}
	newRequired, err := k.requiredBondPerSlot(ctx, next)
	if err != nil {
		return err
	}
	if !newRequired.IsPositive() || newRequired.LTE(oldRequired) {
		return nil
	}
	delta := newRequired.Sub(oldRequired)
	for provider, slots := range dealSlotAssignments(next) {
		if slots == 0 {
			continue
		}
		deltaTotal := math.NewIntFromUint64(slots).Mul(delta)
		if err := k.lockProviderBond(ctx, provider, deltaTotal); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) providerMeetsMinBond(ctx sdk.Context, provider types.Provider, params types.Params) bool {
	min := params.MinProviderBond
	if !min.IsValid() || !min.IsPositive() {
		return true
	}

	state, found, err := k.providerBondState(ctx, provider.Address)
	if err != nil {
		return false
	}
	if !found {
		state, err = k.buildProviderBondState(ctx, provider.Address)
		if err != nil {
			return false
		}
	}

	bonded := state.BondedAmount
	if strings.TrimSpace(bonded.Denom) != strings.TrimSpace(sdk.DefaultBondDenom) {
		bonded = sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())
	}

	return bonded.Amount.GTE(min.Amount)
}

func (k Keeper) providerHasAvailableBond(ctx sdk.Context, provider types.Provider, required math.Int) bool {
	if !required.IsPositive() {
		return true
	}
	state, err := k.ensureProviderBondState(ctx, provider.Address)
	if err != nil {
		return false
	}
	return providerAvailableBond(state).GTE(required)
}
