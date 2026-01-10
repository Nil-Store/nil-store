package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/keeper"
	"nilchain/x/nilchain/types"
)

func TestGetDealProviderHealthQuery(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	dealID := uint64(42)
	provider := "provider-one"
	state := types.HealthState{MissedEpochs: 2, HardFailures: 1, LastUpdateHeight: 11}
	require.NoError(t, f.keeper.DealProviderHealth.Set(f.ctx, collections.Join(dealID, provider), state))

	resp, err := qs.GetDealProviderHealth(f.ctx, &types.QueryGetDealProviderHealthRequest{
		DealId:   dealID,
		Provider: provider,
	})
	require.NoError(t, err)
	require.Equal(t, state, resp.State)

	missing, err := qs.GetDealProviderHealth(f.ctx, &types.QueryGetDealProviderHealthRequest{
		DealId:   dealID,
		Provider: "missing-provider",
	})
	require.NoError(t, err)
	require.Equal(t, types.HealthState{}, missing.State)
}

func TestListDealProviderHealthQuery(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	dealID := uint64(7)
	otherDealID := uint64(8)
	require.NoError(t, f.keeper.DealProviderHealth.Set(f.ctx, collections.Join(dealID, "provider-a"), types.HealthState{MissedEpochs: 1}))
	require.NoError(t, f.keeper.DealProviderHealth.Set(f.ctx, collections.Join(dealID, "provider-b"), types.HealthState{MissedEpochs: 2}))
	require.NoError(t, f.keeper.DealProviderHealth.Set(f.ctx, collections.Join(otherDealID, "provider-c"), types.HealthState{MissedEpochs: 3}))

	first, err := qs.ListDealProviderHealth(f.ctx, &types.QueryListDealProviderHealthRequest{
		DealId: dealID,
		Pagination: &query.PageRequest{
			Limit: 1,
		},
	})
	require.NoError(t, err)
	require.Len(t, first.Entries, 1)
	require.NotEmpty(t, first.Pagination.NextKey)
	require.Equal(t, dealID, first.Entries[0].DealId)

	second, err := qs.ListDealProviderHealth(f.ctx, &types.QueryListDealProviderHealthRequest{
		DealId: dealID,
		Pagination: &query.PageRequest{
			Key: first.Pagination.NextKey,
		},
	})
	require.NoError(t, err)
	require.Len(t, second.Entries, 1)
	require.Empty(t, second.Pagination.NextKey)
	require.Equal(t, dealID, second.Entries[0].DealId)
	require.NotEqual(t, first.Entries[0].Provider, second.Entries[0].Provider)
}

func TestDealSlotHealthQueries(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	dealID := uint64(9)
	otherDealID := uint64(10)
	slot := uint32(2)
	state := types.HealthState{MissedEpochs: 4, LastUpdateHeight: 22}
	require.NoError(t, f.keeper.DealSlotHealth.Set(f.ctx, collections.Join(dealID, slot), state))
	require.NoError(t, f.keeper.DealSlotHealth.Set(f.ctx, collections.Join(otherDealID, uint32(1)), types.HealthState{MissedEpochs: 1}))

	resp, err := qs.GetDealSlotHealth(f.ctx, &types.QueryGetDealSlotHealthRequest{
		DealId: dealID,
		Slot:   slot,
	})
	require.NoError(t, err)
	require.Equal(t, state, resp.State)

	missing, err := qs.GetDealSlotHealth(f.ctx, &types.QueryGetDealSlotHealthRequest{
		DealId: dealID,
		Slot:   99,
	})
	require.NoError(t, err)
	require.Equal(t, types.HealthState{}, missing.State)

	list, err := qs.ListDealSlotHealth(f.ctx, &types.QueryListDealSlotHealthRequest{
		DealId: dealID,
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	})
	require.NoError(t, err)
	require.Len(t, list.Entries, 1)
	require.Equal(t, dealID, list.Entries[0].DealId)
	require.Equal(t, slot, list.Entries[0].Slot)
}
