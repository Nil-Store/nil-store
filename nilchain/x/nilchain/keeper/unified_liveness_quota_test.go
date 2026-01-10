package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/types"
)

type quotaFixture struct {
	ctx          context.Context
	keeper       Keeper
	addressCodec address.Codec
}

type quotaMockBankKeeper struct{}

func (quotaMockBankKeeper) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins()
}

func (quotaMockBankKeeper) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (quotaMockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (quotaMockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (quotaMockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

type quotaMockAccountKeeper struct{}

func (quotaMockAccountKeeper) AddressCodec() address.Codec {
	return nil
}

func (quotaMockAccountKeeper) GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI {
	return nil
}

func initQuotaFixture(t *testing.T) *quotaFixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig()
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithChainID("quota-test-chain")
	ctx = sdkCtx

	authority := authtypes.NewModuleAddress(types.GovModuleName)
	k := NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		quotaMockBankKeeper{},
		quotaMockAccountKeeper{},
	)

	require.NoError(t, k.Params.Set(ctx, types.DefaultParams()))

	return &quotaFixture{
		ctx:          ctx,
		keeper:       k,
		addressCodec: addressCodec,
	}
}

func TestRequiredBlobsFromSlotBytes_Clamps(t *testing.T) {
	params := types.Params{
		QuotaMinBlobs: 2,
		QuotaMaxBlobs: 5,
	}

	min := requiredBlobsFromSlotBytes(params, 1, uint64(types.BlobSizeBytes))
	require.Equal(t, uint64(2), min)

	max := requiredBlobsFromSlotBytes(params, 10000, uint64(types.BlobSizeBytes)*100)
	require.Equal(t, uint64(5), max)
}

func TestRecordSynthetic_Dedupes(t *testing.T) {
	f := initQuotaFixture(t)
	ctx := sdk.UnwrapSDKContext(f.ctx)

	epochID := uint64(1)
	dealID := uint64(42)
	assignment := []byte{0xAA, 0xBB}
	key := collections.Join(collections.Join(dealID, "provider-1"), epochID)

	counted, err := f.keeper.recordSynthetic(ctx, epochID, dealID, assignment, key, 5, 9)
	require.NoError(t, err)
	require.True(t, counted)

	counted, err = f.keeper.recordSynthetic(ctx, epochID, dealID, assignment, key, 5, 9)
	require.NoError(t, err)
	require.False(t, counted)

	seen, err := f.keeper.Mode1EpochSynthetic.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, uint64(1), seen)
}

func TestRecordCredit_Dedupes(t *testing.T) {
	f := initQuotaFixture(t)
	ctx := sdk.UnwrapSDKContext(f.ctx)

	epochID := uint64(1)
	dealID := uint64(42)
	assignment := []byte{0x01, 0x02}
	key := collections.Join(collections.Join(dealID, "provider-1"), epochID)

	require.NoError(t, f.keeper.recordCredit(ctx, epochID, dealID, assignment, key, 7, 11))
	require.NoError(t, f.keeper.recordCredit(ctx, epochID, dealID, assignment, key, 7, 11))
	require.NoError(t, f.keeper.recordCredit(ctx, epochID, dealID, assignment, key, 7, 12))

	seen, err := f.keeper.Mode1EpochCredits.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, uint64(2), seen)
}

func TestCreditCapBlobs_HotCold(t *testing.T) {
	params := types.DefaultParams()
	params.CreditCapBpsHot = 5000
	params.CreditCapBpsCold = 2500

	quota := uint64(100)
	hotDeal := types.Deal{ServiceHint: "Hot"}
	coldDeal := types.Deal{ServiceHint: "Cold"}

	require.Equal(t, uint64(50), creditCapBlobs(params, hotDeal, quota))
	require.Equal(t, uint64(25), creditCapBlobs(params, coldDeal, quota))
}

func TestCreditCapBlobs_DefaultZero(t *testing.T) {
	params := types.DefaultParams()
	quota := uint64(100)
	deal := types.Deal{ServiceHint: "Hot"}

	require.Equal(t, uint64(0), creditCapBlobs(params, deal, quota))
}

func TestCheckMissedProofs_PruningClearsEpochAccounting(t *testing.T) {
	f := initQuotaFixture(t)
	ctx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(5)

	params := types.DefaultParams()
	params.EpochLenBlocks = 5
	require.NoError(t, f.keeper.Params.Set(ctx, params))

	epochID := uint64(1)
	mode1Key := collections.Join(collections.Join(uint64(1), "provider-1"), epochID)
	mode2Key := collections.Join(collections.Join(uint64(1), uint32(2)), epochID)

	require.NoError(t, f.keeper.Mode1EpochCredits.Set(ctx, mode1Key, 1))
	require.NoError(t, f.keeper.Mode1EpochSynthetic.Set(ctx, mode1Key, 1))
	require.NoError(t, f.keeper.Mode2EpochCredits.Set(ctx, mode2Key, 1))
	require.NoError(t, f.keeper.Mode2EpochSynthetic.Set(ctx, mode2Key, 1))
	require.NoError(t, f.keeper.Mode2EpochSlotServed.Set(ctx, mode2Key, 1))
	require.NoError(t, f.keeper.Mode2EpochDeputyServed.Set(ctx, mode2Key, 1))
	require.NoError(t, f.keeper.CreditSeen.Set(ctx, []byte("credit"), true))
	require.NoError(t, f.keeper.SyntheticSeen.Set(ctx, []byte("synthetic"), true))
	require.NoError(t, f.keeper.DeputySeen.Set(ctx, []byte("deputy"), true))

	require.NoError(t, f.keeper.CheckMissedProofs(ctx))

	_, err := f.keeper.Mode1EpochCredits.Get(ctx, mode1Key)
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = f.keeper.Mode1EpochSynthetic.Get(ctx, mode1Key)
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = f.keeper.Mode2EpochCredits.Get(ctx, mode2Key)
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = f.keeper.Mode2EpochSynthetic.Get(ctx, mode2Key)
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = f.keeper.Mode2EpochSlotServed.Get(ctx, mode2Key)
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = f.keeper.Mode2EpochDeputyServed.Get(ctx, mode2Key)
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = f.keeper.CreditSeen.Get(ctx, []byte("credit"))
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = f.keeper.SyntheticSeen.Get(ctx, []byte("synthetic"))
	require.ErrorIs(t, err, collections.ErrNotFound)
	_, err = f.keeper.DeputySeen.Get(ctx, []byte("deputy"))
	require.ErrorIs(t, err, collections.ErrNotFound)
}
