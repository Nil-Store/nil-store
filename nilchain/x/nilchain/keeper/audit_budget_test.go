package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/keeper"
	module "nilchain/x/nilchain/module"
	niltypes "nilchain/x/nilchain/types"
)

type auditBankKeeper struct {
	minted sdk.Coins
	burned sdk.Coins
}

func (b *auditBankKeeper) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1_000_000))
}

func (b *auditBankKeeper) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	b.minted = b.minted.Add(amt...)
	return nil
}

func (b *auditBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (b *auditBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (b *auditBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	b.burned = b.burned.Add(amt...)
	return nil
}

type auditFixture struct {
	ctx    context.Context
	keeper keeper.Keeper
	bank   *auditBankKeeper
}

func initAuditFixture(t *testing.T) *auditFixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(niltypes.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithChainID("test-chain")
	ctx = sdkCtx

	authority := authtypes.NewModuleAddress(niltypes.GovModuleName)
	bank := &auditBankKeeper{}
	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		bank,
		MockAccountKeeper{},
	)

	if err := k.Params.Set(ctx, niltypes.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &auditFixture{
		ctx:    ctx,
		keeper: k,
		bank:   bank,
	}
}

func TestAuditBudgetMint_Ceil(t *testing.T) {
	f := initAuditFixture(t)
	params := niltypes.DefaultParams()
	params.EpochLenBlocks = 1
	params.StoragePrice = math.LegacyMustNewDecFromStr("0.1")
	params.AuditBudgetBps = 200
	params.AuditBudgetCapBps = 500
	params.AuditBudgetCarryoverEpochs = 2
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	deal := niltypes.Deal{
		Id:             1,
		Owner:          "owner",
		StartBlock:     1,
		EndBlock:       100,
		TotalMdus:      2,
		WitnessMdus:    0,
		RedundancyMode: 2,
		Mode2Profile: &niltypes.StripeReplicaProfile{
			K: 64,
			M: 1,
		},
		Mode2Slots: []*niltypes.DealSlot{
			{
				Slot:     0,
				Provider: "provider-1",
				Status:   niltypes.SlotStatus_SLOT_STATUS_ACTIVE,
			},
		},
	}
	require.NoError(t, f.keeper.Deals.Set(f.ctx, deal.Id, deal))

	ctx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(1)
	require.NoError(t, f.keeper.EndBlock(ctx))

	slotBytes := uint64(niltypes.BlobSizeBytes)
	rentDec := params.StoragePrice.
		MulInt(math.NewIntFromUint64(slotBytes)).
		MulInt(math.NewIntFromUint64(params.EpochLenBlocks))
	expected := rentDec.
		MulInt(math.NewIntFromUint64(params.AuditBudgetBps)).
		QuoInt(math.NewInt(10000)).
		Ceil().
		TruncateInt()

	minted := f.bank.minted.AmountOf(sdk.DefaultBondDenom)
	require.Equal(t, expected, minted)

	available, err := f.keeper.AuditBudgetAvailable.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, available)
}

func TestAuditBudgetMint_Cap(t *testing.T) {
	f := initAuditFixture(t)
	params := niltypes.DefaultParams()
	params.EpochLenBlocks = 1
	params.StoragePrice = math.LegacyMustNewDecFromStr("0.1")
	params.AuditBudgetBps = 500
	params.AuditBudgetCapBps = 200
	params.AuditBudgetCarryoverEpochs = 2
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	deal := niltypes.Deal{
		Id:             1,
		Owner:          "owner",
		StartBlock:     1,
		EndBlock:       100,
		TotalMdus:      2,
		WitnessMdus:    0,
		RedundancyMode: 2,
		Mode2Profile: &niltypes.StripeReplicaProfile{
			K: 64,
			M: 1,
		},
		Mode2Slots: []*niltypes.DealSlot{
			{
				Slot:     0,
				Provider: "provider-1",
				Status:   niltypes.SlotStatus_SLOT_STATUS_ACTIVE,
			},
		},
	}
	require.NoError(t, f.keeper.Deals.Set(f.ctx, deal.Id, deal))

	ctx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(1)
	require.NoError(t, f.keeper.EndBlock(ctx))

	slotBytes := uint64(niltypes.BlobSizeBytes)
	rentDec := params.StoragePrice.
		MulInt(math.NewIntFromUint64(slotBytes)).
		MulInt(math.NewIntFromUint64(params.EpochLenBlocks))
	expectedCap := rentDec.
		MulInt(math.NewIntFromUint64(params.AuditBudgetCapBps)).
		QuoInt(math.NewInt(10000)).
		Ceil().
		TruncateInt()

	minted := f.bank.minted.AmountOf(sdk.DefaultBondDenom)
	require.Equal(t, expectedCap, minted)
}

func TestAuditBudgetCarryover_ExpiresAfterTwoEpochs(t *testing.T) {
	f := initAuditFixture(t)
	params := niltypes.DefaultParams()
	params.EpochLenBlocks = 1
	params.StoragePrice = math.LegacyMustNewDecFromStr("1")
	params.AuditBudgetBps = 10000
	params.AuditBudgetCapBps = 10000
	params.AuditBudgetCarryoverEpochs = 2
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	deal := niltypes.Deal{
		Id:             1,
		Owner:          "owner",
		StartBlock:     1,
		EndBlock:       100,
		TotalMdus:      2,
		WitnessMdus:    0,
		RedundancyMode: 2,
		Mode2Profile: &niltypes.StripeReplicaProfile{
			K: 64,
			M: 1,
		},
		Mode2Slots: []*niltypes.DealSlot{
			{
				Slot:     0,
				Provider: "provider-1",
				Status:   niltypes.SlotStatus_SLOT_STATUS_ACTIVE,
			},
		},
	}
	require.NoError(t, f.keeper.Deals.Set(f.ctx, deal.Id, deal))

	slotBytes := uint64(niltypes.BlobSizeBytes)
	expectedMint := math.NewIntFromUint64(slotBytes)

	ctx1 := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(1)
	require.NoError(t, f.keeper.EndBlock(ctx1))

	ctx2 := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(2)
	require.NoError(t, f.keeper.EndBlock(ctx2))

	available, err := f.keeper.AuditBudgetAvailable.Get(ctx2)
	require.NoError(t, err)
	require.Equal(t, expectedMint.MulRaw(2), available)

	ctx3 := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(3)
	require.NoError(t, f.keeper.EndBlock(ctx3))

	available, err = f.keeper.AuditBudgetAvailable.Get(ctx3)
	require.NoError(t, err)
	require.Equal(t, expectedMint.MulRaw(2), available)

	require.Equal(t, expectedMint, f.bank.burned.AmountOf(sdk.DefaultBondDenom))
	_, err = f.keeper.AuditBudgetByEpoch.Get(ctx3, 1)
	require.Error(t, err)
}

func TestAuditBudgetExcludesRepairingSlots(t *testing.T) {
	f := initAuditFixture(t)
	params := niltypes.DefaultParams()
	params.EpochLenBlocks = 1
	params.StoragePrice = math.LegacyMustNewDecFromStr("1")
	params.AuditBudgetBps = 10000
	params.AuditBudgetCapBps = 10000
	params.AuditBudgetCarryoverEpochs = 2
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	deal := niltypes.Deal{
		Id:             1,
		Owner:          "owner",
		StartBlock:     1,
		EndBlock:       100,
		TotalMdus:      2,
		WitnessMdus:    0,
		RedundancyMode: 2,
		Mode2Profile: &niltypes.StripeReplicaProfile{
			K: 1,
			M: 1,
		},
		Mode2Slots: []*niltypes.DealSlot{
			{
				Slot:     0,
				Provider: "provider-1",
				Status:   niltypes.SlotStatus_SLOT_STATUS_ACTIVE,
			},
			{
				Slot:     1,
				Provider: "provider-2",
				Status:   niltypes.SlotStatus_SLOT_STATUS_REPAIRING,
			},
		},
	}
	require.NoError(t, f.keeper.Deals.Set(f.ctx, deal.Id, deal))

	ctx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(1)
	require.NoError(t, f.keeper.EndBlock(ctx))

	rows := uint64(64)
	slotBytes := rows * uint64(niltypes.BlobSizeBytes)
	expectedMint := math.NewIntFromUint64(slotBytes)

	minted := f.bank.minted.AmountOf(sdk.DefaultBondDenom)
	require.Equal(t, expectedMint, minted)
}
