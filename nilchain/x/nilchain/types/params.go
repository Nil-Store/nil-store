package types

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyBaseStripeCost        = []byte("BaseStripeCost")
	KeyHalvingInterval       = []byte("HalvingInterval")
	KeyEip712ChainID         = []byte("Eip712ChainId")
	KeyStoragePrice          = []byte("StoragePrice")
	KeyDealCreationFee       = []byte("DealCreationFee")
	KeyMinDurationBlocks     = []byte("MinDurationBlocks")
	KeyBaseRetrievalFee      = []byte("BaseRetrievalFee")
	KeyRetrievalPricePerBlob = []byte("RetrievalPricePerBlob")
	KeyRetrievalBurnBps      = []byte("RetrievalBurnBps")
	KeyMonthLenBlocks        = []byte("MonthLenBlocks")

	KeyEpochLenBlocks         = []byte("EpochLenBlocks")
	KeyQuotaBpsPerEpochHot    = []byte("QuotaBpsPerEpochHot")
	KeyQuotaBpsPerEpochCold   = []byte("QuotaBpsPerEpochCold")
	KeyQuotaMinBlobs          = []byte("QuotaMinBlobs")
	KeyQuotaMaxBlobs          = []byte("QuotaMaxBlobs")
	KeyCreditCapBps           = []byte("CreditCapBps")
	KeyEvictAfterMissedEpochs = []byte("EvictAfterMissedEpochs")

	KeySlashInvalidProofBps        = []byte("SlashInvalidProofBps")
	KeySlashWrongDataBps           = []byte("SlashWrongDataBps")
	KeySlashNonresponseBps         = []byte("SlashNonresponseBps")
	KeyJailInvalidProofEpochs      = []byte("JailInvalidProofEpochs")
	KeyJailWrongDataEpochs         = []byte("JailWrongDataEpochs")
	KeyJailNonresponseEpochs       = []byte("JailNonresponseEpochs")
	KeyNonresponseThreshold        = []byte("NonresponseThreshold")
	KeyNonresponseWindowEpochs     = []byte("NonresponseWindowEpochs")
	KeyMaxStrikesBeforeGlobalJail  = []byte("MaxStrikesBeforeGlobalJail")
	KeyStrikeWindowEpochs          = []byte("StrikeWindowEpochs")
	KeyEvictAfterMissedEpochsHot   = []byte("EvictAfterMissedEpochsHot")
	KeyEvictAfterMissedEpochsCold  = []byte("EvictAfterMissedEpochsCold")
	KeyMinProviderBond             = []byte("MinProviderBond")
	KeyBondMonths                  = []byte("BondMonths")
	KeyProviderUnbondingBlocks     = []byte("ProviderUnbondingBlocks")
	KeyReplacementCooldownBlocks   = []byte("ReplacementCooldownBlocks")
	KeyRepairAttemptsCap           = []byte("RepairAttemptsCap")
	KeyRepairAttemptWindowBlocks   = []byte("RepairAttemptWindowBlocks")
	KeyPremiumBps                  = []byte("PremiumBps")
	KeyEvidenceBond                = []byte("EvidenceBond")
	KeyFailureBounty               = []byte("FailureBounty")
	KeyEvidenceBondBurnBpsOnExpiry = []byte("EvidenceBondBurnBpsOnExpiry")
	KeyProofOfFailureTtlEpochs     = []byte("ProofOfFailureTtlEpochs")
	KeyAuditBudgetBps              = []byte("AuditBudgetBps")
	KeyAuditBudgetCapBps           = []byte("AuditBudgetCapBps")
	KeyAuditBudgetCarryoverEpochs  = []byte("AuditBudgetCarryoverEpochs")
	KeyCreditCapBpsHot             = []byte("CreditCapBpsHot")
	KeyCreditCapBpsCold            = []byte("CreditCapBpsCold")
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance.
func NewParams(
	baseStripeCost uint64,
	halvingInterval uint64,
	eip712ChainID uint64,
	storagePrice math.LegacyDec,
	dealCreationFee sdk.Coin,
	minDurationBlocks uint64,
	baseRetrievalFee sdk.Coin,
	retrievalPricePerBlob sdk.Coin,
	retrievalBurnBps uint64,
	monthLenBlocks uint64,
	epochLenBlocks uint64,
	quotaBpsPerEpochHot uint64,
	quotaBpsPerEpochCold uint64,
	quotaMinBlobs uint64,
	quotaMaxBlobs uint64,
	creditCapBps uint64,
	evictAfterMissedEpochs uint64,
	slashInvalidProofBps uint64,
	slashWrongDataBps uint64,
	slashNonresponseBps uint64,
	jailInvalidProofEpochs uint64,
	jailWrongDataEpochs uint64,
	jailNonresponseEpochs uint64,
	nonresponseThreshold uint64,
	nonresponseWindowEpochs uint64,
	maxStrikesBeforeGlobalJail uint64,
	strikeWindowEpochs uint64,
	evictAfterMissedEpochsHot uint64,
	evictAfterMissedEpochsCold uint64,
	minProviderBond sdk.Coin,
	bondMonths uint64,
	providerUnbondingBlocks uint64,
	replacementCooldownBlocks uint64,
	repairAttemptsCap uint64,
	repairAttemptWindowBlocks uint64,
	premiumBps uint64,
	evidenceBond sdk.Coin,
	failureBounty sdk.Coin,
	evidenceBondBurnBpsOnExpiry uint64,
	proofOfFailureTtlEpochs uint64,
	auditBudgetBps uint64,
	auditBudgetCapBps uint64,
	auditBudgetCarryoverEpochs uint64,
	creditCapBpsHot uint64,
	creditCapBpsCold uint64,
) Params {
	return Params{
		BaseStripeCost:        baseStripeCost,
		HalvingInterval:       halvingInterval,
		Eip712ChainId:         eip712ChainID,
		StoragePrice:          storagePrice,
		DealCreationFee:       dealCreationFee,
		MinDurationBlocks:     minDurationBlocks,
		BaseRetrievalFee:      baseRetrievalFee,
		RetrievalPricePerBlob: retrievalPricePerBlob,
		RetrievalBurnBps:      retrievalBurnBps,
		MonthLenBlocks:        monthLenBlocks,

		EpochLenBlocks:         epochLenBlocks,
		QuotaBpsPerEpochHot:    quotaBpsPerEpochHot,
		QuotaBpsPerEpochCold:   quotaBpsPerEpochCold,
		QuotaMinBlobs:          quotaMinBlobs,
		QuotaMaxBlobs:          quotaMaxBlobs,
		CreditCapBps:           creditCapBps,
		EvictAfterMissedEpochs: evictAfterMissedEpochs,

		SlashInvalidProofBps:        slashInvalidProofBps,
		SlashWrongDataBps:           slashWrongDataBps,
		SlashNonresponseBps:         slashNonresponseBps,
		JailInvalidProofEpochs:      jailInvalidProofEpochs,
		JailWrongDataEpochs:         jailWrongDataEpochs,
		JailNonresponseEpochs:       jailNonresponseEpochs,
		NonresponseThreshold:        nonresponseThreshold,
		NonresponseWindowEpochs:     nonresponseWindowEpochs,
		MaxStrikesBeforeGlobalJail:  maxStrikesBeforeGlobalJail,
		StrikeWindowEpochs:          strikeWindowEpochs,
		EvictAfterMissedEpochsHot:   evictAfterMissedEpochsHot,
		EvictAfterMissedEpochsCold:  evictAfterMissedEpochsCold,
		MinProviderBond:             minProviderBond,
		BondMonths:                  bondMonths,
		ProviderUnbondingBlocks:     providerUnbondingBlocks,
		ReplacementCooldownBlocks:   replacementCooldownBlocks,
		RepairAttemptsCap:           repairAttemptsCap,
		RepairAttemptWindowBlocks:   repairAttemptWindowBlocks,
		PremiumBps:                  premiumBps,
		EvidenceBond:                evidenceBond,
		FailureBounty:               failureBounty,
		EvidenceBondBurnBpsOnExpiry: evidenceBondBurnBpsOnExpiry,
		ProofOfFailureTtlEpochs:     proofOfFailureTtlEpochs,
		AuditBudgetBps:              auditBudgetBps,
		AuditBudgetCapBps:           auditBudgetCapBps,
		AuditBudgetCarryoverEpochs:  auditBudgetCarryoverEpochs,
		CreditCapBpsHot:             creditCapBpsHot,
		CreditCapBpsCold:            creditCapBpsCold,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(
		10,                   // BaseStripeCost
		1000,                 // HalvingInterval
		31337,                // EIP712ChainId (MetaMask localhost default)
		math.LegacyNewDec(0), // StoragePrice
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(0)), // DealCreationFee
		10, // MinDurationBlocks
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100)), // BaseRetrievalFee (0.0001 NIL)
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1)),   // RetrievalPricePerBlob (provisional devnet default)
		500,  // RetrievalBurnBps (5%)
		1000, // MonthLenBlocks (devnet-friendly "month")
		100,  // EpochLenBlocks (devnet-friendly "epoch")
		100,  // QuotaBpsPerEpochHot (1%)
		50,   // QuotaBpsPerEpochCold (0.5%)
		1,    // QuotaMinBlobs
		64,   // QuotaMaxBlobs
		0,    // CreditCapBps (legacy fallback, devnet caps disabled)
		3,    // EvictAfterMissedEpochs (legacy fallback)

		50,  // SlashInvalidProofBps (0.5%)
		500, // SlashWrongDataBps (5%)
		100, // SlashNonresponseBps (1%)
		3,   // JailInvalidProofEpochs
		30,  // JailWrongDataEpochs
		10,  // JailNonresponseEpochs
		3,   // NonresponseThreshold
		6,   // NonresponseWindowEpochs
		10,  // MaxStrikesBeforeGlobalJail
		100, // StrikeWindowEpochs
		2,   // EvictAfterMissedEpochsHot
		6,   // EvictAfterMissedEpochsCold
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100)), // MinProviderBond
		2,      // BondMonths
		1000,   // ProviderUnbondingBlocks (default to MonthLenBlocks)
		604800, // ReplacementCooldownBlocks (7 days at 1s blocks)
		3,      // RepairAttemptsCap
		1000,   // RepairAttemptWindowBlocks (default to MonthLenBlocks)
		2000,   // PremiumBps (20%)
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(10000)), // EvidenceBond (0.01 NIL)
		sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(20000)), // FailureBounty (0.02 NIL)
		5000, // EvidenceBondBurnBpsOnExpiry (50%)
		6,    // ProofOfFailureTtlEpochs (default to nonresponse window)
		200,  // AuditBudgetBps (2%)
		500,  // AuditBudgetCapBps (5%)
		2,    // AuditBudgetCarryoverEpochs
		0,    // CreditCapBpsHot
		0,    // CreditCapBpsCold
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyBaseStripeCost, &p.BaseStripeCost, validateBaseStripeCost),
		paramtypes.NewParamSetPair(KeyHalvingInterval, &p.HalvingInterval, validateHalvingInterval),
		paramtypes.NewParamSetPair(KeyEip712ChainID, &p.Eip712ChainId, validateEip712ChainID),
		paramtypes.NewParamSetPair(KeyStoragePrice, &p.StoragePrice, validateStoragePrice),
		paramtypes.NewParamSetPair(KeyDealCreationFee, &p.DealCreationFee, validateDealCreationFee),
		paramtypes.NewParamSetPair(KeyMinDurationBlocks, &p.MinDurationBlocks, validateMinDurationBlocks),
		paramtypes.NewParamSetPair(KeyBaseRetrievalFee, &p.BaseRetrievalFee, validateBaseRetrievalFee),
		paramtypes.NewParamSetPair(KeyRetrievalPricePerBlob, &p.RetrievalPricePerBlob, validateRetrievalPricePerBlob),
		paramtypes.NewParamSetPair(KeyRetrievalBurnBps, &p.RetrievalBurnBps, validateRetrievalBurnBps),
		paramtypes.NewParamSetPair(KeyMonthLenBlocks, &p.MonthLenBlocks, validateMonthLenBlocks),

		paramtypes.NewParamSetPair(KeyEpochLenBlocks, &p.EpochLenBlocks, validateEpochLenBlocks),
		paramtypes.NewParamSetPair(KeyQuotaBpsPerEpochHot, &p.QuotaBpsPerEpochHot, validateQuotaBpsPerEpoch),
		paramtypes.NewParamSetPair(KeyQuotaBpsPerEpochCold, &p.QuotaBpsPerEpochCold, validateQuotaBpsPerEpoch),
		paramtypes.NewParamSetPair(KeyQuotaMinBlobs, &p.QuotaMinBlobs, validateQuotaMinBlobs),
		paramtypes.NewParamSetPair(KeyQuotaMaxBlobs, &p.QuotaMaxBlobs, validateQuotaMaxBlobs),
		paramtypes.NewParamSetPair(KeyCreditCapBps, &p.CreditCapBps, validateCreditCapBps),
		paramtypes.NewParamSetPair(KeyEvictAfterMissedEpochs, &p.EvictAfterMissedEpochs, validateEvictAfterMissedEpochs),
		paramtypes.NewParamSetPair(KeySlashInvalidProofBps, &p.SlashInvalidProofBps, validateSlashInvalidProofBps),
		paramtypes.NewParamSetPair(KeySlashWrongDataBps, &p.SlashWrongDataBps, validateSlashWrongDataBps),
		paramtypes.NewParamSetPair(KeySlashNonresponseBps, &p.SlashNonresponseBps, validateSlashNonresponseBps),
		paramtypes.NewParamSetPair(KeyJailInvalidProofEpochs, &p.JailInvalidProofEpochs, validateJailInvalidProofEpochs),
		paramtypes.NewParamSetPair(KeyJailWrongDataEpochs, &p.JailWrongDataEpochs, validateJailWrongDataEpochs),
		paramtypes.NewParamSetPair(KeyJailNonresponseEpochs, &p.JailNonresponseEpochs, validateJailNonresponseEpochs),
		paramtypes.NewParamSetPair(KeyNonresponseThreshold, &p.NonresponseThreshold, validateNonresponseThreshold),
		paramtypes.NewParamSetPair(KeyNonresponseWindowEpochs, &p.NonresponseWindowEpochs, validateNonresponseWindowEpochs),
		paramtypes.NewParamSetPair(KeyMaxStrikesBeforeGlobalJail, &p.MaxStrikesBeforeGlobalJail, validateMaxStrikesBeforeGlobalJail),
		paramtypes.NewParamSetPair(KeyStrikeWindowEpochs, &p.StrikeWindowEpochs, validateStrikeWindowEpochs),
		paramtypes.NewParamSetPair(KeyEvictAfterMissedEpochsHot, &p.EvictAfterMissedEpochsHot, validateEvictAfterMissedEpochsHot),
		paramtypes.NewParamSetPair(KeyEvictAfterMissedEpochsCold, &p.EvictAfterMissedEpochsCold, validateEvictAfterMissedEpochsCold),
		paramtypes.NewParamSetPair(KeyMinProviderBond, &p.MinProviderBond, validateMinProviderBond),
		paramtypes.NewParamSetPair(KeyBondMonths, &p.BondMonths, validateBondMonths),
		paramtypes.NewParamSetPair(KeyProviderUnbondingBlocks, &p.ProviderUnbondingBlocks, validateProviderUnbondingBlocks),
		paramtypes.NewParamSetPair(KeyReplacementCooldownBlocks, &p.ReplacementCooldownBlocks, validateReplacementCooldownBlocks),
		paramtypes.NewParamSetPair(KeyRepairAttemptsCap, &p.RepairAttemptsCap, validateRepairAttemptsCap),
		paramtypes.NewParamSetPair(KeyRepairAttemptWindowBlocks, &p.RepairAttemptWindowBlocks, validateRepairAttemptWindowBlocks),
		paramtypes.NewParamSetPair(KeyPremiumBps, &p.PremiumBps, validatePremiumBps),
		paramtypes.NewParamSetPair(KeyEvidenceBond, &p.EvidenceBond, validateEvidenceBond),
		paramtypes.NewParamSetPair(KeyFailureBounty, &p.FailureBounty, validateFailureBounty),
		paramtypes.NewParamSetPair(KeyEvidenceBondBurnBpsOnExpiry, &p.EvidenceBondBurnBpsOnExpiry, validateEvidenceBondBurnBpsOnExpiry),
		paramtypes.NewParamSetPair(KeyProofOfFailureTtlEpochs, &p.ProofOfFailureTtlEpochs, validateProofOfFailureTtlEpochs),
		paramtypes.NewParamSetPair(KeyAuditBudgetBps, &p.AuditBudgetBps, validateAuditBudgetBps),
		paramtypes.NewParamSetPair(KeyAuditBudgetCapBps, &p.AuditBudgetCapBps, validateAuditBudgetCapBps),
		paramtypes.NewParamSetPair(KeyAuditBudgetCarryoverEpochs, &p.AuditBudgetCarryoverEpochs, validateAuditBudgetCarryoverEpochs),
		paramtypes.NewParamSetPair(KeyCreditCapBpsHot, &p.CreditCapBpsHot, validateCreditCapBpsHot),
		paramtypes.NewParamSetPair(KeyCreditCapBpsCold, &p.CreditCapBpsCold, validateCreditCapBpsCold),
	}
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if err := validateBaseStripeCost(p.BaseStripeCost); err != nil {
		return err
	}
	if err := validateEip712ChainID(p.Eip712ChainId); err != nil {
		return err
	}
	if err := validateHalvingInterval(p.HalvingInterval); err != nil {
		return err
	}
	if err := validateStoragePrice(p.StoragePrice); err != nil {
		return err
	}
	if err := validateDealCreationFee(p.DealCreationFee); err != nil {
		return err
	}
	if err := validateMinDurationBlocks(p.MinDurationBlocks); err != nil {
		return err
	}
	if err := validateBaseRetrievalFee(p.BaseRetrievalFee); err != nil {
		return err
	}
	if err := validateRetrievalPricePerBlob(p.RetrievalPricePerBlob); err != nil {
		return err
	}
	if err := validateRetrievalBurnBps(p.RetrievalBurnBps); err != nil {
		return err
	}
	if err := validateMonthLenBlocks(p.MonthLenBlocks); err != nil {
		return err
	}
	if err := validateEpochLenBlocks(p.EpochLenBlocks); err != nil {
		return err
	}
	if err := validateQuotaBpsPerEpoch(p.QuotaBpsPerEpochHot); err != nil {
		return err
	}
	if err := validateQuotaBpsPerEpoch(p.QuotaBpsPerEpochCold); err != nil {
		return err
	}
	if err := validateQuotaMinBlobs(p.QuotaMinBlobs); err != nil {
		return err
	}
	if err := validateQuotaMaxBlobs(p.QuotaMaxBlobs); err != nil {
		return err
	}
	if p.QuotaMaxBlobs != 0 && p.QuotaMinBlobs > p.QuotaMaxBlobs {
		return fmt.Errorf("quota_min_blobs must be <= quota_max_blobs (got %d > %d)", p.QuotaMinBlobs, p.QuotaMaxBlobs)
	}
	if err := validateCreditCapBps(p.CreditCapBps); err != nil {
		return err
	}
	if err := validateEvictAfterMissedEpochs(p.EvictAfterMissedEpochs); err != nil {
		return err
	}
	if err := validateSlashInvalidProofBps(p.SlashInvalidProofBps); err != nil {
		return err
	}
	if err := validateSlashWrongDataBps(p.SlashWrongDataBps); err != nil {
		return err
	}
	if err := validateSlashNonresponseBps(p.SlashNonresponseBps); err != nil {
		return err
	}
	if err := validateJailInvalidProofEpochs(p.JailInvalidProofEpochs); err != nil {
		return err
	}
	if err := validateJailWrongDataEpochs(p.JailWrongDataEpochs); err != nil {
		return err
	}
	if err := validateJailNonresponseEpochs(p.JailNonresponseEpochs); err != nil {
		return err
	}
	if err := validateNonresponseThreshold(p.NonresponseThreshold); err != nil {
		return err
	}
	if err := validateNonresponseWindowEpochs(p.NonresponseWindowEpochs); err != nil {
		return err
	}
	if err := validateMaxStrikesBeforeGlobalJail(p.MaxStrikesBeforeGlobalJail); err != nil {
		return err
	}
	if err := validateStrikeWindowEpochs(p.StrikeWindowEpochs); err != nil {
		return err
	}
	if err := validateEvictAfterMissedEpochsHot(p.EvictAfterMissedEpochsHot); err != nil {
		return err
	}
	if err := validateEvictAfterMissedEpochsCold(p.EvictAfterMissedEpochsCold); err != nil {
		return err
	}
	if err := validateMinProviderBond(p.MinProviderBond); err != nil {
		return err
	}
	if err := validateBondMonths(p.BondMonths); err != nil {
		return err
	}
	if err := validateProviderUnbondingBlocks(p.ProviderUnbondingBlocks); err != nil {
		return err
	}
	if err := validateReplacementCooldownBlocks(p.ReplacementCooldownBlocks); err != nil {
		return err
	}
	if err := validateRepairAttemptsCap(p.RepairAttemptsCap); err != nil {
		return err
	}
	if err := validateRepairAttemptWindowBlocks(p.RepairAttemptWindowBlocks); err != nil {
		return err
	}
	if err := validatePremiumBps(p.PremiumBps); err != nil {
		return err
	}
	if err := validateEvidenceBond(p.EvidenceBond); err != nil {
		return err
	}
	if err := validateFailureBounty(p.FailureBounty); err != nil {
		return err
	}
	if err := validateEvidenceBondBurnBpsOnExpiry(p.EvidenceBondBurnBpsOnExpiry); err != nil {
		return err
	}
	if err := validateProofOfFailureTtlEpochs(p.ProofOfFailureTtlEpochs); err != nil {
		return err
	}
	if err := validateAuditBudgetBps(p.AuditBudgetBps); err != nil {
		return err
	}
	if err := validateAuditBudgetCapBps(p.AuditBudgetCapBps); err != nil {
		return err
	}
	if err := validateAuditBudgetCarryoverEpochs(p.AuditBudgetCarryoverEpochs); err != nil {
		return err
	}
	if err := validateCreditCapBpsHot(p.CreditCapBpsHot); err != nil {
		return err
	}
	if err := validateCreditCapBpsCold(p.CreditCapBpsCold); err != nil {
		return err
	}
	return nil
}

func validateBaseStripeCost(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateHalvingInterval(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("halving interval must be non-zero")
	}
	return nil
}

func validateEip712ChainID(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("eip712_chain_id must be non-zero")
	}
	return nil
}

func validateStoragePrice(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v.IsNegative() {
		return fmt.Errorf("storage price cannot be negative: %s", v)
	}
	return nil
}

func validateDealCreationFee(i interface{}) error {
	v, ok := i.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if !v.IsValid() {
		return fmt.Errorf("invalid deal creation fee: %s", v)
	}
	if strings.TrimSpace(v.Denom) != strings.TrimSpace(sdk.DefaultBondDenom) {
		return fmt.Errorf("deal creation fee denom must be %q (got %q)", sdk.DefaultBondDenom, v.Denom)
	}
	return nil
}

func validateMinDurationBlocks(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("min duration blocks must be non-zero")
	}
	return nil
}

func validateBaseRetrievalFee(i interface{}) error {
	v, ok := i.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if !v.IsValid() {
		return fmt.Errorf("invalid base retrieval fee: %s", v)
	}
	if strings.TrimSpace(v.Denom) != strings.TrimSpace(sdk.DefaultBondDenom) {
		return fmt.Errorf("base retrieval fee denom must be %q (got %q)", sdk.DefaultBondDenom, v.Denom)
	}
	return nil
}

func validateRetrievalPricePerBlob(i interface{}) error {
	v, ok := i.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if !v.IsValid() {
		return fmt.Errorf("invalid retrieval price per blob: %s", v)
	}
	if strings.TrimSpace(v.Denom) != strings.TrimSpace(sdk.DefaultBondDenom) {
		return fmt.Errorf("retrieval price per blob denom must be %q (got %q)", sdk.DefaultBondDenom, v.Denom)
	}
	return nil
}

func validateRetrievalBurnBps(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v > 10000 {
		return fmt.Errorf("retrieval burn bps must be <= 10000 (got %d)", v)
	}
	return nil
}

func validateMonthLenBlocks(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("month_len_blocks must be non-zero")
	}
	return nil
}

func validateEpochLenBlocks(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("epoch_len_blocks must be non-zero")
	}
	return nil
}

func validateQuotaBpsPerEpoch(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v > 10000 {
		return fmt.Errorf("quota bps must be <= 10000 (got %d)", v)
	}
	return nil
}

func validateQuotaMinBlobs(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("quota_min_blobs must be non-zero")
	}
	return nil
}

func validateQuotaMaxBlobs(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("quota_max_blobs must be non-zero")
	}
	return nil
}

func validateCreditCapBps(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v > 10000 {
		return fmt.Errorf("credit cap bps must be <= 10000 (got %d)", v)
	}
	return nil
}

func validateEvictAfterMissedEpochs(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("evict_after_missed_epochs must be non-zero")
	}
	return nil
}

func validateSlashInvalidProofBps(i interface{}) error {
	return validateBps(i, "slash_invalid_proof_bps")
}

func validateSlashWrongDataBps(i interface{}) error {
	return validateBps(i, "slash_wrong_data_bps")
}

func validateSlashNonresponseBps(i interface{}) error {
	return validateBps(i, "slash_nonresponse_bps")
}

func validateJailInvalidProofEpochs(i interface{}) error {
	return validateUint64NonZero(i, "jail_invalid_proof_epochs")
}

func validateJailWrongDataEpochs(i interface{}) error {
	return validateUint64NonZero(i, "jail_wrong_data_epochs")
}

func validateJailNonresponseEpochs(i interface{}) error {
	return validateUint64NonZero(i, "jail_nonresponse_epochs")
}

func validateNonresponseThreshold(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v < 1 {
		return fmt.Errorf("nonresponse_threshold must be >= 1 (got %d)", v)
	}
	return nil
}

func validateNonresponseWindowEpochs(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v < 1 {
		return fmt.Errorf("nonresponse_window_epochs must be >= 1 (got %d)", v)
	}
	return nil
}

func validateMaxStrikesBeforeGlobalJail(i interface{}) error {
	return validateUint64NonZero(i, "max_strikes_before_global_jail")
}

func validateStrikeWindowEpochs(i interface{}) error {
	return validateUint64NonZero(i, "strike_window_epochs")
}

func validateEvictAfterMissedEpochsHot(i interface{}) error {
	return validateUint64NonZero(i, "evict_after_missed_epochs_hot")
}

func validateEvictAfterMissedEpochsCold(i interface{}) error {
	return validateUint64NonZero(i, "evict_after_missed_epochs_cold")
}

func validateMinProviderBond(i interface{}) error {
	return validateCoinWithDenom(i, "min_provider_bond")
}

func validateBondMonths(i interface{}) error {
	return validateUint64NonZero(i, "bond_months")
}

func validateProviderUnbondingBlocks(i interface{}) error {
	return validateUint64NonZero(i, "provider_unbonding_blocks")
}

func validateReplacementCooldownBlocks(i interface{}) error {
	return validateUint64NonZero(i, "replacement_cooldown_blocks")
}

func validateRepairAttemptsCap(i interface{}) error {
	return validateUint64NonZero(i, "repair_attempts_cap")
}

func validateRepairAttemptWindowBlocks(i interface{}) error {
	return validateUint64NonZero(i, "repair_attempt_window_blocks")
}

func validatePremiumBps(i interface{}) error {
	return validateBps(i, "premium_bps")
}

func validateEvidenceBond(i interface{}) error {
	return validateCoinWithDenom(i, "evidence_bond")
}

func validateFailureBounty(i interface{}) error {
	return validateCoinWithDenom(i, "failure_bounty")
}

func validateEvidenceBondBurnBpsOnExpiry(i interface{}) error {
	return validateBps(i, "evidence_bond_burn_bps_on_expiry")
}

func validateProofOfFailureTtlEpochs(i interface{}) error {
	return validateUint64NonZero(i, "proof_of_failure_ttl_epochs")
}

func validateAuditBudgetBps(i interface{}) error {
	return validateBps(i, "audit_budget_bps")
}

func validateAuditBudgetCapBps(i interface{}) error {
	return validateBps(i, "audit_budget_cap_bps")
}

func validateAuditBudgetCarryoverEpochs(i interface{}) error {
	return validateUint64NonZero(i, "audit_budget_carryover_epochs")
}

func validateCreditCapBpsHot(i interface{}) error {
	return validateBps(i, "credit_cap_bps_hot")
}

func validateCreditCapBpsCold(i interface{}) error {
	return validateBps(i, "credit_cap_bps_cold")
}

func validateBps(i interface{}, name string) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v > 10000 {
		return fmt.Errorf("%s must be <= 10000 (got %d)", name, v)
	}
	return nil
}

func validateUint64NonZero(i interface{}, name string) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("%s must be non-zero", name)
	}
	return nil
}

func validateCoinWithDenom(i interface{}, name string) error {
	v, ok := i.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if !v.IsValid() {
		return fmt.Errorf("invalid %s: %s", name, v)
	}
	if strings.TrimSpace(v.Denom) != strings.TrimSpace(sdk.DefaultBondDenom) {
		return fmt.Errorf("%s denom must be %q (got %q)", name, sdk.DefaultBondDenom, v.Denom)
	}
	return nil
}
