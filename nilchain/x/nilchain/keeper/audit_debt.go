package keeper

import (
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"nilchain/x/nilchain/types"
)

func (k Keeper) getAuditDebtState(ctx sdk.Context, provider string) (types.AuditDebtState, error) {
	state, err := k.AuditDebtStates.Get(ctx, provider)
	if err == nil {
		return state, nil
	}
	if errors.Is(err, collections.ErrNotFound) {
		return types.AuditDebtState{}, nil
	}
	return types.AuditDebtState{}, err
}

// IncrementAuditDebtRequired records additional audit obligations for a provider.
func (k Keeper) IncrementAuditDebtRequired(ctx sdk.Context, provider string, delta uint64) error {
	provider = strings.TrimSpace(provider)
	if provider == "" || delta == 0 {
		return nil
	}

	state, err := k.getAuditDebtState(ctx, provider)
	if err != nil {
		return err
	}

	next, overflow := addUint64(state.Required, delta)
	if overflow {
		return fmt.Errorf("audit debt required overflow")
	}
	state.Required = next

	return k.AuditDebtStates.Set(ctx, provider, state)
}

// IncrementAuditDebtCompleted records completed audit work for a provider.
func (k Keeper) IncrementAuditDebtCompleted(ctx sdk.Context, provider string, delta uint64) error {
	provider = strings.TrimSpace(provider)
	if provider == "" || delta == 0 {
		return nil
	}

	state, err := k.getAuditDebtState(ctx, provider)
	if err != nil {
		return err
	}

	next, overflow := addUint64(state.Completed, delta)
	if overflow {
		return fmt.Errorf("audit debt completed overflow")
	}
	state.Completed = next

	return k.AuditDebtStates.Set(ctx, provider, state)
}

func auditDebtOutstanding(state types.AuditDebtState) uint64 {
	if state.Required <= state.Completed {
		return 0
	}
	return state.Required - state.Completed
}
