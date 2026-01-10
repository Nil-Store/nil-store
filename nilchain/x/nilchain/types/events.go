package types

// Event types
const (
	TypeMsgRegisterProvider = "register_provider"
	TypeMsgCreateDeal       = "create_deal"
	TypeMsgProveLiveness    = "prove_liveness"
	TypeMsgSignalSaturation = "signal_saturation"
	TypeLockInDeposit       = "lock_in_deposit"
	TypeMsgOpenRetrievalSession = "open_retrieval_session"

	AttributeKeyProvider          = "provider"
	AttributeKeyCapabilities      = "capabilities"
	AttributeKeyTotalStorage      = "total_storage"
	AttributeKeyDealID            = "deal_id"
	AttributeKeyCID               = "cid"
	AttributeKeyOwner             = "owner"
	AttributeKeySize              = "size"
	AttributeKeyHint              = "service_hint"
	AttributeKeyAssignedProviders = "assigned_providers"
	AttributeKeySuccess           = "success"
	AttributeKeyTier              = "tier"
	AttributeKeyRewardAmount      = "reward_amount"
	AttributeKeyDeltaBytes        = "delta_bytes"
	AttributeKeyStorageCost       = "storage_cost"
	AttributeKeyDurationBlocks    = "duration_blocks"
	AttributeKeySessionID         = "session_id"
	AttributeKeyBaseFee           = "base_fee"
	AttributeKeyVariableFee       = "variable_fee"
	AttributeKeyTotalFee          = "total_fee"
	AttributeKeyBlobCount         = "blob_count"
)
