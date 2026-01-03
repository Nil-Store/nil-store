package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "nilchain"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// GovModuleName duplicates the gov module's name to avoid a dependency with x/gov.
	// It should be synced with the gov module's name if it is ever changed.
	// See: https://github.com/cosmos/cosmos-sdk/blob/v0.52.0-beta.2/x/gov/types/keys.go#L9
	GovModuleName = "gov"
)

// ParamsKey is the prefix to retrieve all Params
var ParamsKey = collections.NewPrefix("p_nilchain")

var (
	ProofCountKey = collections.NewPrefix("ProofCount/value/")
	ProofsKey     = collections.NewPrefix("Proofs/value/")

	DealCountKey                   = collections.NewPrefix("DealCount/value/")
	DealsKey                       = collections.NewPrefix("Deals/value/")
	ProvidersKey                   = collections.NewPrefix("Providers/value/")
	DealProviderStatusKey          = collections.NewPrefix("DealProviderStatus/value/")
	DealProviderFailuresKey        = collections.NewPrefix("DealProviderFailures/value/")
	ProviderRewardsKey             = collections.NewPrefix("ProviderRewards/value/")
	ReceiptNonceKey                = collections.NewPrefix("ReceiptNonce/value/")
	ReceiptNonceDealFileKey        = collections.NewPrefix("ReceiptNonceDealFile/value/")
	EvmNonceKey                    = collections.NewPrefix("EvmNonce/value/")
	DealHeatStateKey               = collections.NewPrefix("DealHeatState/value/")
	EpochSeedsKey                  = collections.NewPrefix("EpochSeeds/value/")
	CreditSeenKey                  = collections.NewPrefix("CreditSeen/value/")
	SyntheticSeenKey               = collections.NewPrefix("SyntheticSeen/value/")
	EpochCreditsMode1Key           = collections.NewPrefix("EpochCreditsMode1/value/")
	EpochCreditsMode2Key           = collections.NewPrefix("EpochCreditsMode2/value/")
	EpochSyntheticMode1Key         = collections.NewPrefix("EpochSyntheticMode1/value/")
	EpochSyntheticMode2Key         = collections.NewPrefix("EpochSyntheticMode2/value/")
	MissedEpochsMode1Key           = collections.NewPrefix("MissedEpochsMode1/value/")
	MissedEpochsMode2Key           = collections.NewPrefix("MissedEpochsMode2/value/")
	RetrievalSessionsKey           = collections.NewPrefix("RetrievalSessions/value/")
	RetrievalSessionsByOwnerKey    = collections.NewPrefix("RetrievalSessionsByOwner/value/")
	RetrievalSessionsByProviderKey = collections.NewPrefix("RetrievalSessionsByProvider/value/")
	RetrievalSessionNonceKey       = collections.NewPrefix("RetrievalSessionNonce/value/")
)
