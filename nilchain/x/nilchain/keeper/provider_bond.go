package keeper

import (
	"errors"
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
