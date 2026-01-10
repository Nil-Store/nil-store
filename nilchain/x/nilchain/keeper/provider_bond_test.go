package keeper_test

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
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

	assigned, err := f.keeper.AssignProviders(sdk.UnwrapSDKContext(f.ctx), 1, []byte("seed"), "Hot", types.DealBaseReplication, math.ZeroInt())
	require.NoError(t, err)
	require.Len(t, assigned, len(eligible))
	for _, addr := range assigned {
		_, ok := eligible[addr]
		require.True(t, ok)
	}
}

func TestAssignProvidersRespectsAvailableBond(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.MinProviderBond = sdk.NewInt64Coin(sdk.DefaultBondDenom, 0)
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	addrA := []byte("provider_avail_a")
	addrB := []byte("provider_avail_b")
	providerA, _ := f.addressCodec.BytesToString(addrA)
	providerB, _ := f.addressCodec.BytesToString(addrB)

	bank.setAccountBalance(sdk.AccAddress(addrA), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 100)))
	bank.setAccountBalance(sdk.AccAddress(addrB), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 100)))

	_, err := msgServer.RegisterProvider(f.ctx, &types.MsgRegisterProvider{
		Creator:      providerA,
		Capabilities: "General",
		TotalStorage: 1000,
		Endpoints:    testProviderEndpoints,
	})
	require.NoError(t, err)
	_, err = msgServer.RegisterProvider(f.ctx, &types.MsgRegisterProvider{
		Creator:      providerB,
		Capabilities: "General",
		TotalStorage: 1000,
		Endpoints:    testProviderEndpoints,
	})
	require.NoError(t, err)

	stateA, err := f.keeper.ProviderBonds.Get(f.ctx, providerA)
	require.NoError(t, err)
	stateA.LockedAmount = sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(80))
	require.NoError(t, f.keeper.ProviderBonds.Set(f.ctx, providerA, stateA))

	required := math.NewInt(50)
	assigned, err := f.keeper.AssignProviders(sdk.UnwrapSDKContext(f.ctx), 2, []byte("seed"), "Hot", types.DealBaseReplication, required)
	require.NoError(t, err)
	require.Len(t, assigned, 1)
	require.Equal(t, providerB, assigned[0])
}

func TestUpdateDealContentLocksBondForGrowth(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.StoragePrice = math.LegacyNewDec(1)
	params.BondMonths = 1
	params.MonthLenBlocks = 1
	params.MinDurationBlocks = 1
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	providerAddr := []byte("bond_growth_provider")
	provider, _ := f.addressCodec.BytesToString(providerAddr)
	bank.setAccountBalance(sdk.AccAddress(providerAddr), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 20_000_000)))

	_, err := msgServer.RegisterProvider(f.ctx, &types.MsgRegisterProvider{
		Creator:      provider,
		Capabilities: "General",
		TotalStorage: 1000,
		Endpoints:    testProviderEndpoints,
	})
	require.NoError(t, err)

	userAddr := []byte("bond_growth_user____")
	user, _ := f.addressCodec.BytesToString(userAddr)
	bank.setAccountBalance(sdk.AccAddress(userAddr), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 50_000_000)))

	createResp, err := msgServer.CreateDeal(f.ctx, &types.MsgCreateDeal{
		Creator:             user,
		DurationBlocks:      10,
		ServiceHint:         "General",
		MaxMonthlySpend:     math.NewInt(0),
		InitialEscrowAmount: math.NewInt(0),
	})
	require.NoError(t, err)

	manifestHex := strings.Repeat("11", 48)
	_, err = msgServer.UpdateDealContent(f.ctx, &types.MsgUpdateDealContent{
		Creator:     user,
		DealId:      createResp.DealId,
		Cid:         manifestHex,
		Size_:       1024,
		TotalMdus:   2,
		WitnessMdus: 0,
	})
	require.NoError(t, err)

	bondState, err := f.keeper.ProviderBonds.Get(f.ctx, provider)
	require.NoError(t, err)
	expected := int64(types.MDU_SIZE)
	require.Equal(t, expected, bondState.LockedAmount.Amount.Int64())
}

func TestUpdateDealContentRejectsInsufficientProviderBond(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.StoragePrice = math.LegacyNewDec(1)
	params.BondMonths = 1
	params.MonthLenBlocks = 1
	params.MinDurationBlocks = 1
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	providerAddr := []byte("bond_small_provider")
	provider, _ := f.addressCodec.BytesToString(providerAddr)
	bank.setAccountBalance(sdk.AccAddress(providerAddr), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000)))

	_, err := msgServer.RegisterProvider(f.ctx, &types.MsgRegisterProvider{
		Creator:      provider,
		Capabilities: "General",
		TotalStorage: 1000,
		Endpoints:    testProviderEndpoints,
	})
	require.NoError(t, err)

	userAddr := []byte("bond_small_user____")
	user, _ := f.addressCodec.BytesToString(userAddr)
	bank.setAccountBalance(sdk.AccAddress(userAddr), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 50_000_000)))

	createResp, err := msgServer.CreateDeal(f.ctx, &types.MsgCreateDeal{
		Creator:             user,
		DurationBlocks:      10,
		ServiceHint:         "General",
		MaxMonthlySpend:     math.NewInt(0),
		InitialEscrowAmount: math.NewInt(0),
	})
	require.NoError(t, err)

	manifestHex := strings.Repeat("22", 48)
	_, err = msgServer.UpdateDealContent(f.ctx, &types.MsgUpdateDealContent{
		Creator:     user,
		DealId:      createResp.DealId,
		Cid:         manifestHex,
		Size_:       1024,
		TotalMdus:   2,
		WitnessMdus: 0,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient available bond")
}
