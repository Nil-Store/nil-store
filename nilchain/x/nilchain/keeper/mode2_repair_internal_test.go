package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"testing"

	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/math"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/types"
)

type repairMockBankKeeper struct{}

func (m repairMockBankKeeper) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1_000_000))
}

func (m repairMockBankKeeper) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m repairMockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m repairMockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (m repairMockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

type repairMockAccountKeeper struct{}

func (m repairMockAccountKeeper) AddressCodec() address.Codec {
	return nil
}

func (m repairMockAccountKeeper) GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return nil
}

func initRepairFixture(t *testing.T) (sdk.Context, Keeper, address.Codec) {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig()
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithChainID("test-chain")

	authority := authtypes.NewModuleAddress(types.GovModuleName)
	keeper := NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		repairMockBankKeeper{},
		repairMockAccountKeeper{},
	)

	params := types.DefaultParams()
	params.StoragePrice = math.LegacyNewDec(0)
	params.BondMonths = 0
	params.MinProviderBond = sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt())
	require.NoError(t, keeper.Params.Set(sdkCtx, params))

	return sdkCtx, keeper, addressCodec
}

func TestMode2ReplacementSelectionDeterministicWithAttempt(t *testing.T) {
	ctx, keeper, addressCodec := initRepairFixture(t)

	mkAddr := func(tag byte) string {
		addr := make([]byte, 20)
		addr[19] = tag
		out, err := addressCodec.BytesToString(addr)
		require.NoError(t, err)
		return out
	}

	providers := []string{
		mkAddr(0xA1),
		mkAddr(0xB2),
		mkAddr(0xC3),
		mkAddr(0xD4),
		mkAddr(0xE5),
		mkAddr(0xF6),
	}

	for _, addr := range providers {
		require.NoError(t, keeper.Providers.Set(ctx, addr, types.Provider{
			Address:      addr,
			Capabilities: "General",
			Status:       "Active",
		}))
	}

	deal := types.Deal{
		Id:             1,
		Owner:          mkAddr(0xEE),
		RedundancyMode: 2,
		ServiceHint:    "General",
		CurrentGen:     7,
		Mode2Profile:   &types.StripeReplicaProfile{K: 2, M: 1},
		Mode2Slots: []*types.DealSlot{
			{Slot: 0, Provider: providers[0], Status: types.SlotStatus_SLOT_STATUS_ACTIVE},
			{Slot: 1, Provider: providers[1], Status: types.SlotStatus_SLOT_STATUS_ACTIVE},
			{Slot: 2, Provider: providers[2], Status: types.SlotStatus_SLOT_STATUS_ACTIVE},
		},
	}

	candidates := []string{providers[3], providers[4], providers[5]}
	epochID := uint64(1)

	expectedSelection := func(attempt uint64) string {
		seed := keeper.getEpochSeed(ctx, epochID)
		buf := make([]byte, 0, 32+8+4+8+8)
		buf = append(buf, seed[:]...)
		buf = append(buf, sdk.Uint64ToBigEndian(deal.Id)...)
		var slotBytes [4]byte
		binary.BigEndian.PutUint32(slotBytes[:], 0)
		buf = append(buf, slotBytes[:]...)
		buf = append(buf, sdk.Uint64ToBigEndian(deal.CurrentGen)...)
		buf = append(buf, sdk.Uint64ToBigEndian(attempt)...)
		sum := sha256.Sum256(buf)

		sorted := append([]string(nil), candidates...)
		sort.Strings(sorted)
		idx := int(binary.BigEndian.Uint64(sum[:8]) % uint64(len(sorted)))
		return sorted[idx]
	}

	var attemptA uint64 = 1
	expectA := expectedSelection(attemptA)
	var attemptB uint64
	var expectB string
	for i := uint64(2); i <= 50; i++ {
		expectB = expectedSelection(i)
		if expectB != expectA {
			attemptB = i
			break
		}
	}
	require.NotZero(t, attemptB)

	actualA, err := keeper.selectMode2ReplacementProvider(ctx, deal, 0, epochID, attemptA)
	require.NoError(t, err)
	require.Equal(t, expectA, actualA)

	actualB, err := keeper.selectMode2ReplacementProvider(ctx, deal, 0, epochID, attemptB)
	require.NoError(t, err)
	require.Equal(t, expectB, actualB)
	require.NotEqual(t, actualA, actualB)
}
