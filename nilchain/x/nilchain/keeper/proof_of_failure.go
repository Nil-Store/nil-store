package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"nilchain/x/nilchain/types"
)

var proofOfFailureSeedTag = []byte("nilstore/proof_of_failure/v1")

func deriveProofOfFailureID(provider string, reporter string, dealID uint64, proofHash []byte) [32]byte {
	buf := make([]byte, 0, len(proofOfFailureSeedTag)+len(provider)+len(reporter)+8+len(proofHash))
	buf = append(buf, proofOfFailureSeedTag...)
	buf = append(buf, []byte(provider)...)
	buf = append(buf, []byte(reporter)...)
	buf = binary.BigEndian.AppendUint64(buf, dealID)
	buf = append(buf, proofHash...)
	return sha256.Sum256(buf)
}

func nonresponseWindowStart(currentEpoch uint64, window uint64) uint64 {
	if window == 0 {
		return currentEpoch
	}
	if currentEpoch <= window {
		return 1
	}
	return currentEpoch - window + 1
}

func (k Keeper) expireProofsOfFailure(ctx sdk.Context, currentEpoch uint64) error {
	var expired []types.ProofOfFailure

	err := k.ProofOfFailureByProvider.Walk(ctx, nil, func(key collections.Pair[string, []byte], _ uint64) (bool, error) {
		proofID := key.K2()
		record, err := k.ProofOfFailureRecords.Get(ctx, proofID)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				_ = k.ProofOfFailureByProvider.Remove(ctx, key)
				return false, nil
			}
			return true, err
		}
		if record.Status != types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_OPEN {
			_ = k.ProofOfFailureByProvider.Remove(ctx, key)
			return false, nil
		}
		if currentEpoch > record.ExpiresAtEpoch {
			expired = append(expired, record)
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	if len(expired) == 0 {
		return nil
	}
	params := k.GetParams(ctx)
	for _, record := range expired {
		if err := k.settleProofOfFailure(ctx, record, types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_EXPIRED, params); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) settleProofOfFailure(ctx sdk.Context, record types.ProofOfFailure, status types.ProofOfFailureStatus, params types.Params) error {
	if status != types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_CONVICTED &&
		status != types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_EXPIRED {
		return nil
	}

	reporter := strings.TrimSpace(record.Reporter)
	if reporter == "" {
		return fmt.Errorf("invalid reporter for proof of failure settlement")
	}
	reporterAddr, err := sdk.AccAddressFromBech32(reporter)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrap("invalid reporter address")
	}

	bond := record.BondAmount
	if bond.IsValid() && bond.Amount.IsPositive() {
		switch status {
		case types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_CONVICTED:
			coins := sdk.NewCoins(bond)
			if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, reporterAddr, coins); err != nil {
				return fmt.Errorf("failed to refund evidence bond: %w", err)
			}
		case types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_EXPIRED:
			burnBps := params.EvidenceBondBurnBpsOnExpiry
			bps := math.NewIntFromUint64(burnBps)
			bpsDiv := math.NewInt(10000)
			bpsCeil := math.NewInt(9999)
			burnAmt := bond.Amount.Mul(bps).Add(bpsCeil).Quo(bpsDiv)
			if burnAmt.GT(bond.Amount) {
				burnAmt = bond.Amount
			}
			refundAmt := bond.Amount.Sub(burnAmt)
			if burnAmt.IsPositive() {
				burnCoins := sdk.NewCoins(sdk.NewCoin(bond.Denom, burnAmt))
				if err := k.BankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
					return fmt.Errorf("failed to burn evidence bond: %w", err)
				}
			}
			if refundAmt.IsPositive() {
				refundCoins := sdk.NewCoins(sdk.NewCoin(bond.Denom, refundAmt))
				if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, reporterAddr, refundCoins); err != nil {
					return fmt.Errorf("failed to refund evidence bond: %w", err)
				}
			}
		}
	}

	if status == types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_CONVICTED {
		bounty := params.FailureBounty
		if bounty.IsValid() && bounty.Amount.IsPositive() {
			if err := k.SpendAuditBudget(ctx, reporterAddr, bounty, "failure_bounty"); err != nil {
				return fmt.Errorf("failed to pay failure bounty: %w", err)
			}
		}
	}

	record.Status = status
	if err := k.ProofOfFailureRecords.Set(ctx, record.ProofId, record); err != nil {
		return err
	}
	if err := k.ProofOfFailureByProvider.Remove(ctx, collections.Join(record.Provider, record.ProofId)); err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	return nil
}

func (k Keeper) aggregateProofOfFailure(ctx sdk.Context, provider string, currentEpoch uint64) error {
	params := k.GetParams(ctx)
	if params.NonresponseThreshold == 0 {
		return nil
	}
	windowStart := nonresponseWindowStart(currentEpoch, params.NonresponseWindowEpochs)
	convictionKey := collections.Join(provider, windowStart)
	if _, err := k.ProofOfFailureConvictions.Get(ctx, convictionKey); err == nil {
		return nil
	} else if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}

	deputyProofs := make(map[string]types.ProofOfFailure)
	err := k.ProofOfFailureByProvider.Walk(ctx, nil, func(key collections.Pair[string, []byte], _ uint64) (bool, error) {
		if key.K1() != provider {
			return false, nil
		}
		proofID := key.K2()
		record, err := k.ProofOfFailureRecords.Get(ctx, proofID)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				_ = k.ProofOfFailureByProvider.Remove(ctx, key)
				return false, nil
			}
			return true, err
		}
		if record.Status != types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_OPEN {
			_ = k.ProofOfFailureByProvider.Remove(ctx, key)
			return false, nil
		}
		if record.EpochId < windowStart {
			return false, nil
		}
		reporter := strings.TrimSpace(record.Reporter)
		if reporter == "" {
			return false, nil
		}
		if _, ok := deputyProofs[reporter]; !ok {
			deputyProofs[reporter] = record
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	if uint64(len(deputyProofs)) < params.NonresponseThreshold {
		return nil
	}

	if err := k.ProofOfFailureConvictions.Set(ctx, convictionKey, uint64(ctx.BlockHeight())); err != nil {
		return err
	}
	if _, err := k.slashProviderBond(ctx, provider, params.SlashNonresponseBps); err != nil {
		return err
	}
	if _, err := k.jailProvider(ctx, provider, params.JailNonresponseEpochs); err != nil {
		return err
	}

	for _, record := range deputyProofs {
		if err := k.settleProofOfFailure(ctx, record, types.ProofOfFailureStatus_PROOF_OF_FAILURE_STATUS_CONVICTED, params); err != nil {
			return err
		}
		if err := k.recordEvidenceSummary(ctx, record.DealId, record.Provider, "proof_of_failure_conviction", record.ProofId, "chain", false); err != nil {
			ctx.Logger().Error("failed to record proof-of-failure evidence summary", "error", err)
		}
		k.incrementProviderHardFailure(ctx, record.DealId, record.Provider)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"proof_of_failure_convicted",
			sdk.NewAttribute(types.AttributeKeyProvider, provider),
			sdk.NewAttribute(types.AttributeKeyEpochID, fmt.Sprintf("%d", currentEpoch)),
			sdk.NewAttribute(types.AttributeKeyThreshold, fmt.Sprintf("%d", params.NonresponseThreshold)),
		),
	)

	return nil
}

func (k Keeper) incrementProviderHardFailure(ctx sdk.Context, dealID uint64, provider string) {
	state, _, err := k.getProviderHealthState(ctx, dealID, provider)
	if err != nil {
		ctx.Logger().Error("failed to get provider health state", "deal", dealID, "provider", provider, "error", err)
		return
	}
	state.HardFailures++
	state.LastUpdateHeight = ctx.BlockHeight()
	if err := k.setProviderHealthState(ctx, dealID, provider, state); err != nil {
		ctx.Logger().Error("failed to update provider health state", "deal", dealID, "provider", provider, "error", err)
	}
}
