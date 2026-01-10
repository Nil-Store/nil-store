package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/keeper"
	"nilchain/x/nilchain/types"
)

func TestProviderBondQueryUsesSpendableBalance(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	addrBytes := make([]byte, 20)
	addrBytes[0] = 0xA1
	creator, err := f.addressCodec.BytesToString(addrBytes)
	require.NoError(t, err)

	bank.setAccountBalance(sdk.AccAddress(addrBytes), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 150)))

	_, err = msgServer.RegisterProvider(f.ctx, &types.MsgRegisterProvider{
		Creator:      creator,
		Capabilities: "General",
		TotalStorage: 1000,
		Endpoints:    testProviderEndpoints,
	})
	require.NoError(t, err)

	stored, err := f.keeper.ProviderBonds.Get(f.ctx, creator)
	require.NoError(t, err)
	require.Equal(t, int64(150), stored.BondedAmount.Amount.Int64())
	require.Equal(t, sdk.DefaultBondDenom, stored.BondedAmount.Denom)

	queryServer := keeper.NewQueryServerImpl(f.keeper)
	resp, err := queryServer.GetProviderBond(f.ctx, &types.QueryGetProviderBondRequest{Address: creator})
	require.NoError(t, err)
	require.Equal(t, int64(150), resp.Bond.BondedAmount.Amount.Int64())
	require.Equal(t, sdk.DefaultBondDenom, resp.Bond.BondedAmount.Denom)
	require.Equal(t, int64(0), resp.Bond.LockedAmount.Amount.Int64())
}

func TestAssignProvidersRespectsMinProviderBond(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.MinProviderBond = sdk.NewInt64Coin(sdk.DefaultBondDenom, 100)
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	eligible := make(map[string]struct{})
	for i := 0; i < int(types.DealBaseReplication); i++ {
		addrBytes := make([]byte, 20)
		addrBytes[0] = byte(0x10 + i)
		creator, err := f.addressCodec.BytesToString(addrBytes)
		require.NoError(t, err)

		amount := int64(50)
		if i%2 == 0 {
			amount = 200
			eligible[creator] = struct{}{}
		}
		bank.setAccountBalance(sdk.AccAddress(addrBytes), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, amount)))

		_, err = msgServer.RegisterProvider(f.ctx, &types.MsgRegisterProvider{
			Creator:      creator,
			Capabilities: "General",
			TotalStorage: 1000,
			Endpoints:    testProviderEndpoints,
		})
		require.NoError(t, err)
	}

	assigned, err := f.keeper.AssignProviders(sdk.UnwrapSDKContext(f.ctx), 1, []byte("seed"), "Hot", types.DealBaseReplication)
	require.NoError(t, err)
	require.Len(t, assigned, len(eligible))
	for _, addr := range assigned {
		_, ok := eligible[addr]
		require.True(t, ok)
	}
}
