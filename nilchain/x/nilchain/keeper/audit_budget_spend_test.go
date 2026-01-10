package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"nilchain/x/nilchain/types"
)

func TestSpendAuditBudget_DebitsAvailable(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	ctx := sdk.UnwrapSDKContext(f.ctx)

	recipient := makeAddr(t, f, "audit_recipient")
	recipientAddr, err := sdk.AccAddressFromBech32(recipient)
	require.NoError(t, err)

	bank.setAccountBalance(recipientAddr, sdk.NewCoins())
	bank.moduleBalances[types.ModuleName] = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100)))
	require.NoError(t, f.keeper.AuditBudgetAvailable.Set(ctx, math.NewInt(100)))

	err = f.keeper.SpendAuditBudget(ctx, recipientAddr, sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(60)), "test")
	require.NoError(t, err)

	available, err := f.keeper.AuditBudgetAvailable.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(40), available)
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(60))), bank.accountBalances[recipientAddr.String()])
}

func TestSpendAuditBudget_Insufficient(t *testing.T) {
	bank := newTrackingBankKeeper()
	f := initFixtureWithBankKeeper(t, bank)
	ctx := sdk.UnwrapSDKContext(f.ctx)

	recipient := makeAddr(t, f, "audit_recipient")
	recipientAddr, err := sdk.AccAddressFromBech32(recipient)
	require.NoError(t, err)

	bank.setAccountBalance(recipientAddr, sdk.NewCoins())
	bank.moduleBalances[types.ModuleName] = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(10)))
	require.NoError(t, f.keeper.AuditBudgetAvailable.Set(ctx, math.NewInt(10)))

	err = f.keeper.SpendAuditBudget(ctx, recipientAddr, sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(20)), "test")
	require.Error(t, err)

	available, err := f.keeper.AuditBudgetAvailable.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(10), available)
	require.Equal(t, sdk.NewCoins(), bank.accountBalances[recipientAddr.String()])
}
