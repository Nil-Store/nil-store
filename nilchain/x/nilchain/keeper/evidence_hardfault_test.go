package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/keeper"
	"nilchain/x/nilchain/types"
)

func TestInvalidSystemProofSlashesAndJails(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.EpochLenBlocks = 5
	params.SlashInvalidProofBps = 50
	params.JailInvalidProofEpochs = 3
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	sdkCtx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(11)

	addr := make([]byte, 20)
	addr[19] = 0xA1
	provider, err := f.addressCodec.BytesToString(addr)
	require.NoError(t, err)
	bank.setAccountBalance(sdk.AccAddress(addr), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10_000)))

	_, err = msgServer.RegisterProvider(sdkCtx, &types.MsgRegisterProvider{
		Creator:      provider,
		Capabilities: "General",
		TotalStorage: 1_000_000,
		Endpoints:    testProviderEndpoints,
	})
	require.NoError(t, err)

	deal := types.Deal{
		Id:             1,
		Owner:          provider,
		StartBlock:     1,
		EndBlock:       10_000,
		RedundancyMode: 1,
		Providers:      []string{provider},
		TotalMdus:      3,
		WitnessMdus:    1,
		CurrentGen:     1,
		ServiceHint:    "General",
	}
	require.NoError(t, f.keeper.Deals.Set(sdkCtx, deal.Id, deal))

	res, err := msgServer.ProveLiveness(sdkCtx, &types.MsgProveLiveness{
		Creator:   provider,
		DealId:    deal.Id,
		EpochId:   3,
		ProofType: &types.MsgProveLiveness_SystemProof{SystemProof: nil},
	})
	require.NoError(t, err)
	require.False(t, res.Success)

	remaining := bank.accountBalances[provider].AmountOf(sdk.DefaultBondDenom)
	require.Equal(t, int64(9_950), remaining.Int64())

	endHeight, err := f.keeper.ProviderJails.Get(sdkCtx, provider)
	require.NoError(t, err)
	require.Equal(t, int64(26), endHeight)

	record, err := f.keeper.Providers.Get(sdkCtx, provider)
	require.NoError(t, err)
	require.Equal(t, "Jailed", record.Status)
}

func TestInvalidSystemProofReplayDoesNotSlashTwice(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.EpochLenBlocks = 5
	params.SlashInvalidProofBps = 50
	params.JailInvalidProofEpochs = 0
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	sdkCtx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(11)

	addr := make([]byte, 20)
	addr[19] = 0xB2
	provider, err := f.addressCodec.BytesToString(addr)
	require.NoError(t, err)
	bank.setAccountBalance(sdk.AccAddress(addr), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10_000)))

	_, err = msgServer.RegisterProvider(sdkCtx, &types.MsgRegisterProvider{
		Creator:      provider,
		Capabilities: "General",
		TotalStorage: 1_000_000,
		Endpoints:    testProviderEndpoints,
	})
	require.NoError(t, err)

	deal := types.Deal{
		Id:             2,
		Owner:          provider,
		StartBlock:     1,
		EndBlock:       10_000,
		RedundancyMode: 1,
		Providers:      []string{provider},
		TotalMdus:      3,
		WitnessMdus:    1,
		CurrentGen:     1,
		ServiceHint:    "General",
	}
	require.NoError(t, f.keeper.Deals.Set(sdkCtx, deal.Id, deal))

	for i := 0; i < 2; i++ {
		res, err := msgServer.ProveLiveness(sdkCtx, &types.MsgProveLiveness{
			Creator:   provider,
			DealId:    deal.Id,
			EpochId:   3,
			ProofType: &types.MsgProveLiveness_SystemProof{SystemProof: nil},
		})
		require.NoError(t, err)
		require.False(t, res.Success)
	}

	remaining := bank.accountBalances[provider].AmountOf(sdk.DefaultBondDenom)
	require.Equal(t, int64(9_950), remaining.Int64())
}

func TestWrongDataEvidenceSlashesAndJails(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	params := types.DefaultParams()
	params.EpochLenBlocks = 5
	params.SlashWrongDataBps = 500
	params.JailWrongDataEpochs = 30
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	sdkCtx := sdk.UnwrapSDKContext(f.ctx).WithBlockHeight(10)

	addr := make([]byte, 20)
	addr[19] = 0xC3
	provider, err := f.addressCodec.BytesToString(addr)
	require.NoError(t, err)
	bank.setAccountBalance(sdk.AccAddress(addr), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10_000)))

	ownerAddr := make([]byte, 20)
	ownerAddr[19] = 0xD4
	owner, err := f.addressCodec.BytesToString(ownerAddr)
	require.NoError(t, err)

	_, err = msgServer.RegisterProvider(sdkCtx, &types.MsgRegisterProvider{
		Creator:      provider,
		Capabilities: "General",
		TotalStorage: 1_000_000,
		Endpoints:    testProviderEndpoints,
	})
	require.NoError(t, err)

	manifestRoot := make([]byte, 48)
	for i := range manifestRoot {
		manifestRoot[i] = byte(i + 1)
	}
	deal := types.Deal{
		Id:             3,
		Owner:          owner,
		StartBlock:     1,
		EndBlock:       10_000,
		RedundancyMode: 1,
		Providers:      []string{provider},
		TotalMdus:      3,
		WitnessMdus:    1,
		CurrentGen:     1,
		ServiceHint:    "General",
		ManifestRoot:   manifestRoot,
	}
	require.NoError(t, f.keeper.Deals.Set(sdkCtx, deal.Id, deal))

	ownerAcc, err := sdk.AccAddressFromBech32(owner)
	require.NoError(t, err)
	providerAcc, err := sdk.AccAddressFromBech32(provider)
	require.NoError(t, err)

	sessionID, err := types.HashRetrievalSessionID(
		ownerAcc.Bytes(),
		deal.Id,
		providerAcc.Bytes(),
		manifestRoot,
		0,
		0,
		1,
		1,
		0,
	)
	require.NoError(t, err)

	session := types.RetrievalSession{
		SessionId:      sessionID,
		DealId:         deal.Id,
		Owner:          owner,
		Provider:       provider,
		ManifestRoot:   manifestRoot,
		StartMduIndex:  0,
		StartBlobIndex: 0,
		BlobCount:      1,
		TotalBytes:     types.BlobSizeBytes,
		Nonce:          1,
		ExpiresAt:      0,
		OpenedHeight:   sdkCtx.BlockHeight(),
		UpdatedHeight:  sdkCtx.BlockHeight(),
		Status:         types.RetrievalSessionStatus_RETRIEVAL_SESSION_STATUS_OPEN,
		LockedFee:      math.ZeroInt(),
	}
	require.NoError(t, f.keeper.RetrievalSessions.Set(sdkCtx, sessionID, session))

	res, err := msgServer.SubmitRetrievalSessionProof(sdkCtx, &types.MsgSubmitRetrievalSessionProof{
		Creator:   provider,
		SessionId: sessionID,
		Proofs:    []types.ChainedProof{{}},
	})
	require.NoError(t, err)
	require.False(t, res.Success)

	remaining := bank.accountBalances[provider].AmountOf(sdk.DefaultBondDenom)
	require.Equal(t, int64(9_500), remaining.Int64())

	endHeight, err := f.keeper.ProviderJails.Get(sdkCtx, provider)
	require.NoError(t, err)
	require.Equal(t, int64(160), endHeight)

	updated, err := f.keeper.RetrievalSessions.Get(sdkCtx, sessionID)
	require.NoError(t, err)
	require.Equal(t, types.RetrievalSessionStatus_RETRIEVAL_SESSION_STATUS_OPEN, updated.Status)
}
