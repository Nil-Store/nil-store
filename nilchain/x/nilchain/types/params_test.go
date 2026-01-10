package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestDefaultParamsPolicyDefaults(t *testing.T) {
	p := DefaultParams()

	require.Equal(t, sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100)), p.BaseRetrievalFee)
	require.EqualValues(t, 200, p.AuditBudgetBps)
	require.EqualValues(t, 500, p.AuditBudgetCapBps)
	require.EqualValues(t, 2, p.AuditBudgetCarryoverEpochs)
	require.EqualValues(t, 2, p.EvictAfterMissedEpochsHot)
	require.EqualValues(t, 6, p.EvictAfterMissedEpochsCold)
	require.EqualValues(t, 0, p.CreditCapBpsHot)
	require.EqualValues(t, 0, p.CreditCapBpsCold)
	require.False(t, p.RepairOverrideEnabled)
}

func TestParamsValidateRejectsInvalidValues(t *testing.T) {
	p := DefaultParams()

	p.SlashInvalidProofBps = 10001
	require.Error(t, p.Validate())

	p = DefaultParams()
	p.NonresponseThreshold = 0
	require.Error(t, p.Validate())

	p = DefaultParams()
	p.NonresponseWindowEpochs = 0
	require.Error(t, p.Validate())

	p = DefaultParams()
	p.EvidenceBond = sdk.NewCoin("bad", math.NewInt(1))
	require.Error(t, p.Validate())
}
