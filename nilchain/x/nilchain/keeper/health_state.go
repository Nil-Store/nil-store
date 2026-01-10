package keeper

import (
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"nilchain/x/nilchain/types"
)

func (k Keeper) getProviderHealthState(ctx sdk.Context, dealID uint64, provider string) (types.HealthState, bool, error) {
	key := collections.Join(dealID, provider)
	state, err := k.DealProviderHealth.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.HealthState{}, false, nil
		}
		return types.HealthState{}, false, err
	}
	return state, true, nil
}

func (k Keeper) setProviderHealthState(ctx sdk.Context, dealID uint64, provider string, state types.HealthState) error {
	key := collections.Join(dealID, provider)
	if state.MissedEpochs == 0 && state.HardFailures == 0 {
		if err := k.DealProviderHealth.Remove(ctx, key); err != nil && !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		return nil
	}
	return k.DealProviderHealth.Set(ctx, key, state)
}

func (k Keeper) getSlotHealthState(ctx sdk.Context, dealID uint64, slot uint32) (types.HealthState, bool, error) {
	key := collections.Join(dealID, slot)
	state, err := k.DealSlotHealth.Get(ctx, key)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.HealthState{}, false, nil
		}
		return types.HealthState{}, false, err
	}
	return state, true, nil
}

func (k Keeper) setSlotHealthState(ctx sdk.Context, dealID uint64, slot uint32, state types.HealthState) error {
	key := collections.Join(dealID, slot)
	if state.MissedEpochs == 0 && state.HardFailures == 0 {
		if err := k.DealSlotHealth.Remove(ctx, key); err != nil && !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		return nil
	}
	return k.DealSlotHealth.Set(ctx, key, state)
}
