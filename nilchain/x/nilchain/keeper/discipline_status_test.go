package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/keeper"
	"nilchain/x/nilchain/types"
)

func TestAssignProviders_ExcludesOfflineAndJailedByDiscipline(t *testing.T) {
	f := initFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	sdkCtx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(5)

	params := types.DefaultParams()
	params.EpochLenBlocks = 5
	params.EvictAfterMissedEpochs = 1
	require.NoError(t, f.keeper.Params.Set(sdkCtx, params))

	mkAddr := func(tag byte) string {
		addr := make([]byte, 20)
		addr[19] = tag
		out, err := f.addressCodec.BytesToString(addr)
		require.NoError(t, err)
		return out
	}

	providerA := mkAddr(0xA1)
	providerB := mkAddr(0xB2)
	providerC := mkAddr(0xC3)

	for _, addr := range []string{providerA, providerB, providerC} {
		_, err := msgServer.RegisterProvider(sdkCtx, &types.MsgRegisterProvider{
			Creator:      addr,
			Capabilities: "General",
			TotalStorage: 1_000_000_000,
			Endpoints:    testProviderEndpoints,
		})
		require.NoError(t, err)
	}

	// providerA should move to Offline (threshold=1), providerB to Jailed (threshold=2).
	require.NoError(t, f.keeper.ProviderDisciplineWindowEpoch.Set(sdkCtx, providerA, 1))
	require.NoError(t, f.keeper.ProviderDisciplineTotal.Set(sdkCtx, providerA, 1))
	require.NoError(t, f.keeper.ProviderDisciplineQuotaMiss.Set(sdkCtx, providerA, 1))

	require.NoError(t, f.keeper.ProviderDisciplineWindowEpoch.Set(sdkCtx, providerB, 1))
	require.NoError(t, f.keeper.ProviderDisciplineTotal.Set(sdkCtx, providerB, 2))
	require.NoError(t, f.keeper.ProviderDisciplineInvalidProof.Set(sdkCtx, providerB, 2))

	assigned, err := f.keeper.AssignProviders(sdkCtx, 42, []byte("blockhash"), "General", 3)
	require.NoError(t, err)
	require.Equal(t, []string{providerC}, assigned)

	pA, err := f.keeper.Providers.Get(sdkCtx, providerA)
	require.NoError(t, err)
	require.Equal(t, "Offline", pA.Status)
	pB, err := f.keeper.Providers.Get(sdkCtx, providerB)
	require.NoError(t, err)
	require.Equal(t, "Jailed", pB.Status)
	pC, err := f.keeper.Providers.Get(sdkCtx, providerC)
	require.NoError(t, err)
	require.Equal(t, "Active", pC.Status)
}

func TestAssignProviders_DisciplineDecayRecoversProviderToActive(t *testing.T) {
	f := initFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.EpochLenBlocks = 5
	params.EvictAfterMissedEpochs = 1
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	mkAddr := func(tag byte) string {
		addr := make([]byte, 20)
		addr[19] = tag
		out, err := f.addressCodec.BytesToString(addr)
		require.NoError(t, err)
		return out
	}

	providerA := mkAddr(0xA1)
	providerB := mkAddr(0xB2)

	for _, addr := range []string{providerA, providerB} {
		_, err := msgServer.RegisterProvider(f.ctx, &types.MsgRegisterProvider{
			Creator:      addr,
			Capabilities: "General",
			TotalStorage: 1_000_000_000,
			Endpoints:    testProviderEndpoints,
		})
		require.NoError(t, err)
	}

	ctxEpoch1 := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(5)
	require.NoError(t, f.keeper.ProviderDisciplineWindowEpoch.Set(ctxEpoch1, providerA, 1))
	require.NoError(t, f.keeper.ProviderDisciplineTotal.Set(ctxEpoch1, providerA, 1))
	require.NoError(t, f.keeper.ProviderDisciplineNonResponse.Set(ctxEpoch1, providerA, 1))

	_, err := f.keeper.AssignProviders(ctxEpoch1, 7, []byte("h1"), "General", 1)
	require.NoError(t, err)
	pA, err := f.keeper.Providers.Get(ctxEpoch1, providerA)
	require.NoError(t, err)
	require.Equal(t, "Offline", pA.Status)

	// Move to a much later epoch; linear decay drops total to 0 and status recovers to Active.
	ctxEpoch4 := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(20)
	assigned, err := f.keeper.AssignProviders(ctxEpoch4, 8, []byte("h2"), "General", 2)
	require.NoError(t, err)
	require.Len(t, assigned, 2)
	require.Contains(t, assigned, providerA)
	require.Contains(t, assigned, providerB)

	pA, err = f.keeper.Providers.Get(ctxEpoch4, providerA)
	require.NoError(t, err)
	require.Equal(t, "Active", pA.Status)
	total, err := f.keeper.ProviderDisciplineTotal.Get(ctxEpoch4, providerA)
	require.NoError(t, err)
	require.Equal(t, uint64(0), total)
}
