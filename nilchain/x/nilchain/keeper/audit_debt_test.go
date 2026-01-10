package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/keeper"
	"nilchain/x/nilchain/types"
)

func TestAuditDebtTrackingAndQuery(t *testing.T) {
	f := initFixture(t)
	provider := makeAddr(t, f, "audit_provider")
	ctx := sdk.UnwrapSDKContext(f.ctx)

	require.NoError(t, f.keeper.IncrementAuditDebtRequired(ctx, provider, 5))
	require.NoError(t, f.keeper.IncrementAuditDebtCompleted(ctx, provider, 2))

	qs := keeper.NewQueryServerImpl(f.keeper)
	resp, err := qs.GetAuditDebt(f.ctx, &types.QueryGetAuditDebtRequest{Provider: provider})
	require.NoError(t, err)
	require.EqualValues(t, 5, resp.State.Required)
	require.EqualValues(t, 2, resp.State.Completed)
	require.EqualValues(t, 3, resp.Outstanding)

	list, err := qs.ListAuditDebt(f.ctx, &types.QueryListAuditDebtRequest{})
	require.NoError(t, err)

	found := false
	for _, entry := range list.Entries {
		if entry.Provider == provider {
			found = true
			require.EqualValues(t, 5, entry.State.Required)
			require.EqualValues(t, 2, entry.State.Completed)
			break
		}
	}
	require.True(t, found)
}
