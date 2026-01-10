package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/keeper"
	"nilchain/x/nilchain/types"
)

func TestSubmitProofOfFailure_ConvictionRefundsAndBounty(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.EpochLenBlocks = 5
	params.NonresponseWindowEpochs = 3
	params.NonresponseThreshold = 2
	params.EvidenceBond = sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100))
	params.FailureBounty = sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(200))
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))
	stored := f.keeper.GetParams(sdk.UnwrapSDKContext(f.ctx))
	require.Equal(t, params.EvidenceBond, stored.EvidenceBond)
	require.Equal(t, params.FailureBounty, stored.FailureBounty)

	target := makeAddr(t, f, "proof_target____")
	deputyA := makeAddr(t, f, "proof_deputy_a")
	deputyB := makeAddr(t, f, "proof_deputy_b")
	deputyC := makeAddr(t, f, "proof_deputy_c")
	owner := makeAddr(t, f, "proof_owner_____")

	require.NoError(t, registerProviderForTest(f.ctx, msgServer, target))
	require.NoError(t, registerProviderForTest(f.ctx, msgServer, deputyA))
	require.NoError(t, registerProviderForTest(f.ctx, msgServer, deputyB))
	require.NoError(t, registerProviderForTest(f.ctx, msgServer, deputyC))

	ownerAddr, err := sdk.AccAddressFromBech32(owner)
	require.NoError(t, err)
	bank.setAccountBalance(ownerAddr, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 2_000_000)))

	resDeal, err := msgServer.CreateDeal(f.ctx, &types.MsgCreateDeal{
		Creator:             owner,
		DurationBlocks:      100,
		ServiceHint:         "General",
		InitialEscrowAmount: math.NewInt(1000000),
		MaxMonthlySpend:     math.NewInt(1000000),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resDeal.AssignedProviders)
	target = resDeal.AssignedProviders[0]

	reporters := pickReporters(t, target, deputyA, deputyB, deputyC)
	deputyA = reporters[0]
	deputyB = reporters[1]

	initialCoins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(20000)))
	setBalance(t, bank, target, initialCoins)
	deputyAAddr := setBalance(t, bank, deputyA, initialCoins)
	deputyBAddr := setBalance(t, bank, deputyB, initialCoins)
	setBalance(t, bank, deputyC, initialCoins)

	proofHashA := make([]byte, 32)
	proofHashA[0] = 0xAA
	proofHashB := make([]byte, 32)
	proofHashB[0] = 0xBB

	resA, err := msgServer.SubmitProofOfFailure(f.ctx, &types.MsgSubmitProofOfFailure{
		Creator:   deputyA,
		DealId:    resDeal.DealId,
		Provider:  target,
		ProofHash: proofHashA,
	})
	require.NoError(t, err)
	require.True(t, resA.Success)

	resB, err := msgServer.SubmitProofOfFailure(f.ctx, &types.MsgSubmitProofOfFailure{
		Creator:   deputyB,
		DealId:    resDeal.DealId,
		Provider:  target,
		ProofHash: proofHashB,
	})
	require.NoError(t, err)
	require.True(t, resB.Success)

	recordA, err := f.keeper.ProofOfFailureRecords.Get(sdk.UnwrapSDKContext(f.ctx), resA.ProofId)
	require.NoError(t, err)
	require.Equal(t, types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_CONVICTED, recordA.Status)

	recordB, err := f.keeper.ProofOfFailureRecords.Get(sdk.UnwrapSDKContext(f.ctx), resB.ProofId)
	require.NoError(t, err)
	require.Equal(t, types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_CONVICTED, recordB.Status)

	updatedA := bank.accountBalances[deputyAAddr.String()]
	require.Equal(t, initialCoins.Add(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(200))), updatedA)
	updatedB := bank.accountBalances[deputyBAddr.String()]
	require.Equal(t, initialCoins.Add(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(200))), updatedB)
}

func TestSubmitProofOfFailure_ExpiryBurnRefund(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.EpochLenBlocks = 5
	params.ProofOfFailureTtlEpochs = 1
	params.EvidenceBond = sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100))
	params.EvidenceBondBurnBpsOnExpiry = 5000
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	target := makeAddr(t, f, "expiry_target___")
	deputy := makeAddr(t, f, "expiry_deputy__")
	extra := makeAddr(t, f, "expiry_extra___")
	owner := makeAddr(t, f, "expiry_owner____")

	require.NoError(t, registerProviderForTest(f.ctx, msgServer, target))
	require.NoError(t, registerProviderForTest(f.ctx, msgServer, deputy))
	require.NoError(t, registerProviderForTest(f.ctx, msgServer, extra))

	ownerAddr, err := sdk.AccAddressFromBech32(owner)
	require.NoError(t, err)
	bank.setAccountBalance(ownerAddr, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 2_000_000)))

	resDeal, err := msgServer.CreateDeal(f.ctx, &types.MsgCreateDeal{
		Creator:             owner,
		DurationBlocks:      100,
		ServiceHint:         "General",
		InitialEscrowAmount: math.NewInt(1000000),
		MaxMonthlySpend:     math.NewInt(1000000),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resDeal.AssignedProviders)
	target = resDeal.AssignedProviders[0]
	if target == deputy {
		deputy = extra
	}

	initialCoins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(20000)))
	setBalance(t, bank, target, initialCoins)
	deputyAddr := setBalance(t, bank, deputy, initialCoins)
	setBalance(t, bank, extra, initialCoins)

	proofHash := make([]byte, 32)
	proofHash[0] = 0xCC

	ctxEpoch1 := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(1)
	res, err := msgServer.SubmitProofOfFailure(ctxEpoch1, &types.MsgSubmitProofOfFailure{
		Creator:   deputy,
		DealId:    resDeal.DealId,
		Provider:  target,
		ProofHash: proofHash,
	})
	require.NoError(t, err)
	require.True(t, res.Success)

	ctxEpoch3 := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(11)
	require.NoError(t, f.keeper.BeginBlock(ctxEpoch3))

	record, err := f.keeper.ProofOfFailureRecords.Get(ctxEpoch3, res.ProofId)
	require.NoError(t, err)
	require.Equal(t, types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_EXPIRED, record.Status)

	updated := bank.accountBalances[deputyAddr.String()]
	expected := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(19950)))
	require.Equal(t, expected, updated)
}

func TestSubmitProofOfFailure_RejectsDuplicateDeputyInWindow(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.EpochLenBlocks = 5
	params.NonresponseWindowEpochs = 3
	params.EvidenceBond = sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100))
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	target := makeAddr(t, f, "dupe_target____")
	deputy := makeAddr(t, f, "dupe_deputy___")
	extra := makeAddr(t, f, "dupe_extra____")
	owner := makeAddr(t, f, "dupe_owner_____")

	require.NoError(t, registerProviderForTest(f.ctx, msgServer, target))
	require.NoError(t, registerProviderForTest(f.ctx, msgServer, deputy))
	require.NoError(t, registerProviderForTest(f.ctx, msgServer, extra))

	ownerAddr, err := sdk.AccAddressFromBech32(owner)
	require.NoError(t, err)
	bank.setAccountBalance(ownerAddr, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 2_000_000)))

	resDeal, err := msgServer.CreateDeal(f.ctx, &types.MsgCreateDeal{
		Creator:             owner,
		DurationBlocks:      100,
		ServiceHint:         "General",
		InitialEscrowAmount: math.NewInt(1000000),
		MaxMonthlySpend:     math.NewInt(1000000),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resDeal.AssignedProviders)
	target = resDeal.AssignedProviders[0]
	if target == deputy {
		deputy = extra
	}

	initialCoins := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(20000)))
	setBalance(t, bank, target, initialCoins)
	setBalance(t, bank, deputy, initialCoins)
	setBalance(t, bank, extra, initialCoins)

	proofHash := make([]byte, 32)
	proofHash[0] = 0xDD
	_, err = msgServer.SubmitProofOfFailure(f.ctx, &types.MsgSubmitProofOfFailure{
		Creator:   deputy,
		DealId:    resDeal.DealId,
		Provider:  target,
		ProofHash: proofHash,
	})
	require.NoError(t, err)

	proofHash2 := make([]byte, 32)
	proofHash2[0] = 0xEE
	_, err = msgServer.SubmitProofOfFailure(f.ctx, &types.MsgSubmitProofOfFailure{
		Creator:   deputy,
		DealId:    resDeal.DealId,
		Provider:  target,
		ProofHash: proofHash2,
	})
	require.Error(t, err)
}

func registerProviderForTest(ctx context.Context, msgServer types.MsgServer, addr string) error {
	_, err := msgServer.RegisterProvider(ctx, &types.MsgRegisterProvider{
		Creator:      addr,
		Capabilities: "General",
		TotalStorage: 100000000000,
		Endpoints:    testProviderEndpoints,
	})
	return err
}

func makeAddr(t *testing.T, f *fixture, seed string) string {
	t.Helper()
	addrBz := make([]byte, 20)
	copy(addrBz, []byte(seed))
	addr, err := f.addressCodec.BytesToString(addrBz)
	require.NoError(t, err)
	return addr
}

func pickReporters(t *testing.T, target string, candidates ...string) []string {
	t.Helper()
	reporters := make([]string, 0, 2)
	for _, candidate := range candidates {
		if candidate == target {
			continue
		}
		reporters = append(reporters, candidate)
		if len(reporters) == 2 {
			break
		}
	}
	require.Len(t, reporters, 2, "need two reporters distinct from target")
	return reporters
}

func setBalance(t *testing.T, bank *trackingBankKeeper, addr string, coins sdk.Coins) sdk.AccAddress {
	t.Helper()
	acc, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	bank.setAccountBalance(acc, coins)
	return acc
}
