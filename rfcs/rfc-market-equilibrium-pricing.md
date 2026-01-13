# RFC: Market-Equilibrium Pricing (Bounded On-Chain Controller for Storage + Retrieval)

**Status:** Draft / Normative Candidate
**Scope:** Chain economics (`nilchain/`) — automatic price discovery for storage + retrieval fees
**Motivation:** `spec.md` Appendix B #5; reduce governance overhead and make pricing responsive to on-chain demand/supply signals without oracles
**Depends on:**

* `rfcs/rfc-pricing-and-escrow-accounting.md` (**accounting contract; MUST NOT change**)
* `rfcs/rfc-challenge-derivation-and-quotas.md` (epoch definition and deterministic epoch boundaries)
* `rfcs/rfc-mode2-onchain-state.md` (Mode 2 slot status; `ACTIVE` vs `REPAIRING`)
* `notes/mainnet_policy_resolution_jan2026.md` (baseline defaults + calibration signals)
* (Optional, when enabled) Deputy / proof-of-failure / audit budget workflows in Stage 7 (proxy premium, evidence bond/bounty, audit budget)

---

## 0. Executive Summary

NilStore currently treats key price parameters as **static governance-controlled values**:

* `storage_price` (`LegacyDec`, units: base-denom per byte per block) — used for **lock-in storage deposits** at ingest.
* `retrieval_price_per_blob` (`Coin`, units: base-denom per 128 KiB blob) — used for **retrieval variable fees** locked at session open and settled on completion.
* `base_retrieval_fee` (`Coin`) — burned at retrieval session open as an anti-spam sink.

This RFC introduces an **on-chain, deterministic price-discovery controller** that updates the **spot** storage and retrieval prices automatically over time, while preserving the **frozen accounting rules** for escrow, lock-in pricing, and retrieval settlement.

Key properties:

* **No contract changes:** all accounting formulas in `rfcs/rfc-pricing-and-escrow-accounting.md` remain unchanged.
* **No oracles:** updates rely only on **on-chain measurable signals** (audit budget demand/pressure, repair pressure, deputy/proxy usage, proof-of-failure rate).
* **Deterministic + auditable:** updates occur at deterministic epoch boundaries, using explicit integer/fixed-point arithmetic, bounded deltas, and emitted events.
* **Staged rollout:** metrics-only first; then retrieval pricing; then storage pricing; with governance kill-switches.

### 0.1 “Market-equilibrium” clarification (scope/terminology)

Despite the title, v1 is **not** an auction or order-book market. It is a **bounded feedback controller** that adjusts protocol-wide spot prices to converge toward an operating regime where chosen on-chain signals remain near target values.

In this RFC, “equilibrium” means:

* storage-side: audit-budget demand pressure remains near a target band (e.g., 60% of a reference target),
* retrieval-side: deputy/proxy share and proof-of-failure rate remain near target bands (e.g., 1%),
* and prices remain stable (small deltas) when signals are stable.

---

## 1. Problem Statement and Motivation

Static pricing creates several operational risks:

1. **Governance bottleneck:** tuning storage/retrieval prices by governance requires repeated interventions and may lag fast-changing conditions.
2. **Mispricing under changing load:** when the network becomes stressed (repairs, deputy usage, audit demand), static prices may underprice scarce resources (congestion) or overprice abundant resources (reduced usage).
3. **Protocol subsystem coupling:** deputy/proof-of-failure and audit-budget systems introduce protocol-mediated flows whose effective cost depends on chain conditions. Spot prices should adapt to observed pressure.
4. **User experience:** deterministic, bounded epoch-based drift is easier to communicate and integrate into wallets/UIs than sporadic manual step-changes.

NilStore already has a strong accounting contract for escrow and settlement. What is missing is an **automatic, bounded, deterministic** method for updating the **spot prices** used by that contract.

---

## 2. Design Goals and Non-Goals

### 2.1 Goals

1. **Preserve the accounting contract**

   * No changes to formulas or semantics in `rfcs/rfc-pricing-and-escrow-accounting.md`.
   * Only the *values* of existing price params evolve over time.

2. **Deterministic on-chain computation**

   * Every full node computes identical price updates from identical state.
   * No wall-clock time; only block height / epoch counters.

3. **Auditable dynamics**

   * Store per-epoch pricing inputs and emit explicit events on updates.
   * Provide queries to inspect current prices, last update epoch, baselines, and driving metrics.

4. **Bounded volatility**

   * Hard caps on per-update change.
   * Absolute min/max bounds expressed as multipliers around baselines.
   * **Clarification (normative):** the per-update delta cap bounds the controller’s multiplicative change, but clamp enforcement **MAY** introduce a larger discrete change **only** when the current spot price is already outside the baseline-relative clamp range (typically due to a governance override or misconfiguration).

5. **Manipulation resistance by design**

   * Prefer “costly-to-fake” signals (proof-of-failure requires bonds; audit budget demand arises from protocol work).
   * Avoid raw “successful retrieval volume” as a price-increase signal (wash risk).

6. **Incremental deployment**

   * Stage 0: metric collection only (no price changes)
   * Stage 1: retrieval price updates (optional)
   * Stage 2: storage price updates (optional)
   * Stage 3+: refinements

### 2.2 Non-Goals

1. **No per-deal bidding / auctions in v1**

   * Users do not submit bids; providers do not submit asks.
   * Prices are protocol-updated global spot parameters.

2. **No external price feeds**

   * No oracle dependency for fiat price, hardware cost, or off-chain utilization.

3. **No retroactive repricing**

   * Previously locked-in storage deposits are not repriced.
   * Existing retrieval sessions are never repriced (fees locked at open).

4. **No hot/cold split pricing in v1**

   * Existing escrow accounting references a single `storage_price` and `retrieval_price_per_blob`.
   * Introducing hot/cold spot prices would change the frozen accounting contract and is therefore deferred.
   * This RFC includes forward-compatibility notes for a future hot/cold split RFC.

---

## 3. Terminology and Units

### 3.1 Storage price units (on-chain)

* `Params.storage_price` is a `LegacyDec` representing **base-denom units per byte per block**.

Derived “human” price (GiB-month):

* Define `GiB = 2^30 bytes`.
* Define “month” as `Params.month_len_blocks`.

Then:

```
price_GiBMonth = storage_price * GiB * month_len_blocks
```

### 3.2 Retrieval price units (on-chain)

* `Params.retrieval_price_per_blob` is a `Coin` representing base-denom units per **Blob**.
* `BLOB_SIZE = 128 KiB` is a protocol constant.

Derived “human” price (GiB):

Because `GiB / BLOB_SIZE = 8192`:

```
price_GiBRetrieval = retrieval_price_per_blob * 8192
```

### 3.3 Epoch

This RFC treats epoch boundaries as a consensus-critical primitive and therefore defines epoch numbering explicitly.

Let:

* `h` be the current block height as observed by the application at `BeginBlocker`. Nilchain/Cosmos-SDK heights are **1-indexed** (the first block has `h = 1`).
* `L = Params.epoch_len_blocks` (`uint64`, MUST satisfy `L >= 1`).

Define the epoch id:

```
epoch_id = floor((h - 1) / L)
```

This implies:

* The **first** block of epoch `e` has height: `h = e*L + 1`
* The **last** block of epoch `e` has height: `h = (e+1)*L`

An “epoch boundary” in this RFC means the `BeginBlocker` of the first block of an epoch.

Price updates, metric finalization, and controller state transitions occur only at epoch boundaries as defined above.

### 3.4 Denominations

This RFC uses the term **base denom** for the canonical on-chain denomination used by NilStore escrow accounting, storage rent, and retrieval fees.

**Concrete definition (normative):**

* `base_denom := Params.retrieval_price_per_blob.Denom`

Normative requirements:

* `base_denom` MUST be non-empty.
* `Params.base_retrieval_fee.Denom` MUST equal `base_denom`.
* `MarketPricingState.baseline_retrieval_price_per_blob.Denom` MUST equal `base_denom`.

All `audit_budget_*` coins recorded in `MarketPricingEpochMetrics` MUST use a single denom. v1 REQUIRES that this denom equals `base_denom`. If the audit budget subsystem uses a different denom, storage price updates MUST remain disabled until this RFC is updated with an explicit deterministic conversion rule.

At runtime, if a denom mismatch is detected for a market’s required denoms, the keeper MUST skip the affected market update and MUST emit `EventMarketPricingUpdate` with the corresponding per-market `skip_reason_*=DENOM_MISMATCH`.

---

## 4. Mechanism Overview

At a high level:

1. **Collect on-chain metrics during an epoch** (counts and sums derived from existing tx handlers and epoch hooks).
2. **Compute normalized “pressure” signals** for:

   * the **storage market** (audit-budget demand/pressure and optional repair pressure), and
   * the **retrieval market** (deputy/proxy share and proof-of-failure rate).
3. **Smooth pressure** via an EMA implemented with deterministic fixed-point integer math.
4. **Convert pressure into a capped price delta** (maximum change per update).
5. **Clamp** prices within baseline-relative min/max ranges.
6. **Write new spot prices** back into module params (`storage_price`, `retrieval_price_per_blob`) and emit an event.

---

## 5. Signals That Drive Price Updates

This RFC defines a **minimal viable signal set** that is deterministic, representable on-chain, and relatively manipulation-resistant.

Signals are grouped into **storage** and **retrieval** markets.

### 5.1 Storage market signals

#### S1) Audit budget demand utilization (primary v1 storage signal)

**Critical design constraint:** audit budget minting (Option A) is proportional to `storage_price`. Using `spent/minted` directly as the controller signal creates a tight closed loop where changes in `storage_price` mechanically change the signal denominator.

To prevent a price-induced denominator feedback from dominating the controller, v1 uses a **price-invariant reference denominator** and a **demand-based numerator**:

* **Numerator:** total audit-budget *requested* spend during the epoch (not merely what was successfully paid).
* **Denominator:** a *reference* audit-budget mint computed using the **baseline storage price**, not the current spot `storage_price`.

This isolates the signal from immediate changes to `storage_price` while still tracking real protocol workload demand for audits/evidence.

##### Definitions and types (normative)

All consensus-critical computations in this subsection MUST use only deterministic fixed-point and integer arithmetic. Implementations MUST NOT use floating-point arithmetic.

Let, for a finalized metrics epoch `e`:

* `A_e := metrics.active_slot_bytes` (`sdkmath.Int`, non-negative)
* `E := Params.epoch_len_blocks` (`uint64`, with `E >= 1`)
* `P0 := state.baseline_storage_price` (`LegacyDec`, **exact fixed-point with 18 decimal places**)
* `bps := Params.audit_budget_bps` (`uint64`)
* `cap_bps := Params.audit_budget_cap_bps` (`uint64`)
* `requested_amt := metrics.audit_budget_requested.Amount` (`sdkmath.Int`, non-negative)
* `requested_denom := metrics.audit_budget_requested.Denom` (`string`)

Denom rules (normative):

* `requested_denom` MUST equal `base_denom` (§3.4). If not, the storage signal is unavailable for this epoch boundary and storage updates MUST be skipped with `skip_reason_storage=DENOM_MISMATCH`.

##### Canonical reference mint computation (normative)

Define `DECIMAL_SCALE := 10^18` as `sdkmath.Int`.

Define `P0_atoms := P0 * 10^18` as `sdkmath.Int` with **no rounding**:

* Because `LegacyDec` is stored as an integer scaled by `10^18`, `P0_atoms` is exactly representable and MUST be obtained via the canonical conversion from `LegacyDec` to its scaled integer representation.
* If `P0 < 0` or `P0_atoms < 0`, this is a parameter validity violation; storage updates MUST be skipped and treated as unavailable (see §7.3 for positivity requirements).

Define `E_int := sdkmath.NewIntFromUint64(E)`.

Compute the epoch rent scaled by `10^18`:

```
rent_scaled = P0_atoms * A_e * E_int                // sdk.Int; scaled by 1e18
```

Then compute the Option A-sized mint amounts using `ceil_div_int` (defined in §8.10.3) with an explicit denominator:

```
denom_scaled = sdk.NewIntFromUint64(10_000) * DECIMAL_SCALE

uncapped_amt = ceil_div_int(rent_scaled * bps,     denom_scaled)
cap_amt      = ceil_div_int(rent_scaled * cap_bps, denom_scaled)

ref_minted_amt = min(uncapped_amt, cap_amt)        // sdk.Int
```

Normative constraints for determinism:

* All intermediates in the computation above MUST be `sdkmath.Int` (big.Int-backed) to be overflow-free.
* `ceil_div_int(a,b)` MUST implement the exact rounding defined in §8.10.3 with the precondition `b > 0`.
* `ref_minted_amt` is an **amount** (`sdkmath.Int`), not a `Coin`.

##### Canonical utilization computation (normative)

Define `audit_demand_util_bps_raw` as `sdkmath.Int` (non-negative).

If `ref_minted_amt == 0`:

* `audit_demand_util_bps_raw` MUST be set to `0`, and
* the storage signal MUST be marked unavailable for this epoch boundary (storage EMA MUST NOT advance and storage updates MUST be skipped) with `skip_reason_storage=ZERO_REF_MINT`.

Otherwise (i.e., `ref_minted_amt > 0`):

```
denom = max(sdk.OneInt(), ref_minted_amt)           // sdk.Int; equals ref_minted_amt here
audit_demand_util_bps_raw = floor_div_int(requested_amt * 10_000, denom)
```

Rounding and overflow rules (normative):

* `floor_div_int(a,b)` MUST round down (toward negative infinity) for non-negative operands and MUST use `sdkmath.Int` arithmetic (§8.10.3).
* Because all intermediates are `sdkmath.Int`, overflow does not occur.
* `audit_demand_util_bps_raw` is unbounded above (it may exceed 10,000).

##### Relationship to the accounting contract

This RFC does not change how audit budgets are minted/spent; it only defines a new derived metric for pricing control.

**Implementation requirement:** all protocol flows that attempt to spend from audit budget MUST route through a single keeper helper (e.g., `SpendAuditBudget(ctx, amount, reason)`), which MUST update the per-epoch `requested/spent/denied` counters deterministically.

**Controller assumption (v1; control-stability prerequisite):** this RFC assumes that the *amounts* requested/spent from the audit budget (i.e., `audit_budget_requested.Amount`) are not defined as a direct function of the current spot `Params.storage_price`. In particular, implementations MUST NOT define audit budget spend amounts as `amount = spot_storage_price * (some physical work measure)` in a way that makes `audit_budget_requested` mechanically proportional to `Params.storage_price` each epoch. If such coupling is desired, the storage controller MUST be revised to normalize in physical units (e.g., bytes/blobs/audit-tasks) rather than in token amounts; that redesign is out of scope for v1.

#### S2) Repair pressure (secondary storage signal; v1 optional)

Mode 2 repairs mark slots `REPAIRING`. Repairing slots indicate supply stress (unhealthy assignments and reduced effective capacity).

Let:

* `active := metrics.active_slot_bytes` (`sdkmath.Int`)
* `repairing := metrics.repairing_slot_bytes` (`sdkmath.Int`)

Compute (normative, using `sdkmath.Int` intermediates):

```
total = active + repairing
denom = max(sdk.OneInt(), total)
repair_pressure_bps = floor_div_int(repairing * 10_000, denom)    // sdk.Int in [0, 10_000]
```

Because `repair_pressure_bps <= 10_000`, implementations MAY convert it to `uint64` for events without loss. If conversion is performed, it MUST be exact.

v1 defaults set repair weight to 0 (signal collected but not used for price) unless explicitly enabled by governance in a later stage.

### 5.2 Retrieval market signals

Retrieval pricing reacts to *stress indicators*, not raw retrieval volume.

#### R1) Deputy/proxy-served fraction

A rising proxy share indicates primary-path retrieval stress (provider non-response or routing failures).

**Canonical numerator (v1 normative):** served sessions whose immutable `session.is_proxy == true`, where `session.is_proxy` is derived deterministically at **session open** from on-chain deputy/gateway authorization (§6.4), and counted when the service proof is accepted.

**Canonical denominator:** sessions with an accepted *service proof* (i.e., served sessions), independent of user confirmation.

Define:

* `served_e` = number of accepted `MsgSubmitRetrievalSessionProof` (or equivalent) for epoch `e`.
* `proxy_served_e` = subset of `served_e` for which `session.is_proxy == true` at open (see §6.4 for canonical tagging).

Then:

```
proxy_fraction_bps = floor( 10_000 * proxy_served_e / max(market_pricing_retrieval_rate_denominator_floor, served_e) )
```

#### R2) Proof-of-failure submission rate

Proof-of-failure submissions indicate non-response pressure. These submissions are bonded (evidence bond), which makes large-scale fabrication costly.

Define:

* `opened_e` = number of accepted `MsgOpenRetrievalSession` during epoch `e`.
* `pof_e` = number of accepted proof-of-failure submissions during epoch `e`, as counted by `OnRetrievalProofOfFailureAccepted` (§6.4).

Then:

```
pof_rate_bps = floor( 10_000 * pof_e / max(market_pricing_retrieval_rate_denominator_floor, opened_e) )
```

#### Low-volume / bootstrap guard for retrieval signals

To prevent unstable behavior during low activity epochs:

* retrieval pressure computation MUST use a denominator floor (param) when converting counts to rates; and
* retrieval price updates MUST be gated on minimum sample sizes.

Details are specified in §8.

---

## 6. State Additions (On-Chain Storage)

### 6.1 `MarketPricingState` (singleton)

A new singleton state object stored under the `nilchain` module.

```proto
message MarketPricingState {
  // Epoch at which baselines were captured (initial enable or explicit reset applied at an epoch boundary).
  uint64 baseline_epoch = 1;

  // Last epoch where an automatic update was applied (may be == baseline_epoch).
  uint64 last_update_epoch = 2;

  // Baselines captured at baseline_epoch.
  string baseline_storage_price = 3
    [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false];
  cosmos.base.v1beta1.Coin baseline_retrieval_price_per_blob = 4 [(gogoproto.nullable) = false];

  // Snapshot of month_len_blocks used to compute max delta per update.
  // This prevents unrelated governance changes to month_len_blocks from abruptly changing price volatility.
  uint64 month_len_blocks_snapshot = 5;

  // EMA state for pressure signals, stored as signed fixed-point micro-bps of full-scale pressure.
  // Unit: micro-bps, where 1 bps = 1_000_000 micro-bps.
  // Range: [-10_000 bps, +10_000 bps] scaled by 1e6 => [-1e10, +1e10].
  int64 storage_pressure_ema_microbps = 6;
  int64 retrieval_pressure_ema_microbps = 7;

  // --- Pending baseline reset (epoch-boundary effective; see §10.2.1 and §8.1) ---
  bool pending_baseline_reset = 8;
}
```

### 6.2 `MarketPricingEpochMetrics` (per-epoch, bounded retention)

Keyed by `epoch_id` and retained for `market_pricing_metrics_retention_epochs`.

**Retention and pruning (normative):**

On finalizing metrics for a completed epoch `e` (i.e., immediately after writing `MarketPricingEpochMetrics{epoch_id=e}`), the keeper MUST prune older metrics entries to bound state growth.

Let:

* `ret = market_pricing_metrics_retention_epochs` (`uint64`, MUST satisfy `ret >= 1`)

Define the first epoch to retain (inclusive) using saturating arithmetic:

* If `e + 1 <= ret`, set `keep_from = 0`.
* Else set `keep_from = e - ret + 1`.

Then delete all stored `MarketPricingEpochMetrics` with:

* `epoch_id < keep_from`

This retains exactly the most recent `ret` epochs, **including** epoch `e`.

Example: if `ret = 30` and the most recently finalized metrics are for `e = 120`, retain epochs `91..120` (30 total) and delete epochs `0..90`.

```proto
message MarketPricingEpochMetrics {
  uint64 epoch_id = 1;

  // --- Storage signals ---
  // Slot-responsible bytes are stored as Int to avoid overflow at large network scale.
  string active_slot_bytes = 2
    [(gogoproto.customtype) = "cosmossdk.io/math.Int", (gogoproto.nullable) = false];
  string repairing_slot_bytes = 3
    [(gogoproto.customtype) = "cosmossdk.io/math.Int", (gogoproto.nullable) = false];

  // Audit budget accounting (Option A). All are per-epoch sums.
  cosmos.base.v1beta1.Coin audit_budget_minted = 4 [(gogoproto.nullable) = false];    // actual minted in epoch
  cosmos.base.v1beta1.Coin audit_budget_requested = 5 [(gogoproto.nullable) = false]; // attempted/requested spend
  cosmos.base.v1beta1.Coin audit_budget_spent = 6 [(gogoproto.nullable) = false];     // successfully paid spend
  cosmos.base.v1beta1.Coin audit_budget_denied = 7 [(gogoproto.nullable) = false];    // requested but not paid

  uint64 audit_budget_spend_attempts = 13; // count of recorded audit spend attempts (success or insufficient budget)

  // --- Retrieval signals ---
  uint64 sessions_opened = 8;

  // Canonical “served” count: proofs accepted on-chain (independent of user confirmation).
  uint64 sessions_served = 9;

  // Optional monitoring-only: sessions settled (fees paid out) via MsgConfirmRetrievalSession.
  uint64 sessions_settled = 10;

  uint64 proxy_sessions_served = 11;
  uint64 proofs_of_failure_submitted = 12;
}
```

### 6.3 Slot byte aggregation semantics (normative)

The controller relies on deterministic slot-byte aggregates consistent with audit budget sizing and quota derivation.

**Implementability constraint (normative; DoS prevention):**

* The chain MUST maintain `active_slot_bytes` and `repairing_slot_bytes` as **incrementally updated aggregates** over deal/slot state transitions.
* The epoch boundary MUST NOT require scanning all deals or all slots to compute these sums.

In practice, this means the implementation MUST maintain (in KV state) running totals that are updated in O(1) on each relevant transition (e.g., deal creation, slot promotion, slot status changes `ACTIVE ↔ REPAIRING`, deal termination), and the epoch boundary snapshot MUST read those totals in O(1).

**Define `slot_responsible_bytes(deal)` as:**

* the protocol’s canonical “slot bytes” used for quotas/rent (`rfcs/rfc-challenge-derivation-and-quotas.md`), i.e.:

  * Mode 2: bytes attributable to a single RS slot for the deal at `Deal.current_gen`
  * Mode 1: bytes attributable to a provider assignment (full copy)

**Active vs repairing aggregation (epoch snapshot semantics):**

At an epoch boundary, the keeper MUST snapshot:

* `active_slot_bytes` = sum of `slot_responsible_bytes` over:

  * Mode 2: each `DealSlot` with `status == ACTIVE`, counting one slot per `DealSlot.provider`.

    * MUST NOT include `pending_provider` bytes (pending provider is not accountable until promotion).
    * MUST exclude `status == REPAIRING` slots (per policy: repairing slots are excluded from rent/quota/rewards).
  * Mode 1: each active provider assignment in `Deal.providers[]` (replicas), counting one “slot equivalent” per provider.

    * Mode 1 has no `REPAIRING` state; therefore all active assignments are counted as active.
* `repairing_slot_bytes` = sum of `slot_responsible_bytes` over:

  * Mode 2: each `DealSlot` with `status == REPAIRING` (counting the old accountable provider’s slot; pending provider is not counted).

**Explicit exclusions:**

* Deals with `size_bytes == 0` contribute 0 bytes.
* REPAIRING slots are excluded from `active_slot_bytes` by definition.
* Tombstones are already excluded from `Deal.size_bytes` by definition; therefore they naturally drop out of `slot_responsible_bytes` if it is derived from the committed slab state.

If the implementation does not yet have a single canonical helper for this, it MUST be introduced (e.g., `GetTotalSlotBytesByStatus(ctx) (active, repairing sdkmath.Int)`), and used consistently by:

* audit budget sizing, and
* market pricing metrics.

**Additional requirement (normative):** any helper used at the epoch boundary MUST be implemented as an O(1) read of the incrementally maintained aggregates (not a scan), as required above.

### 6.4 Canonical retrieval counters (normative)

The controller uses retrieval-side stress signals derived from on-chain retrieval session state transitions. To avoid bias from missing user confirmations, denominators MUST be based on objectively accepted on-chain proofs, not user acknowledgements.

#### Required hook points (normative)

Implementations MUST increment metrics via canonical keeper hook points. Exact message names may differ across branches, but the following semantics are REQUIRED:

* `OnRetrievalSessionOpened(ctx, session_id, opener_addr)`
  Called exactly once when a retrieval session is created and accepted on-chain.

* `OnRetrievalSessionServiceProofAccepted(ctx, session_id)`
  Called exactly once when a service proof for the session is accepted on-chain (the session is “served”).

* `OnRetrievalSessionSettled(ctx, session_id)`
  Called when a user confirmation/settlement message is accepted (monitoring only).

* `OnRetrievalProofOfFailureAccepted(ctx, session_id)`
  Called exactly once when the *first* proof-of-failure for a session is accepted on-chain.

The nilchain implementation MAY call these hooks directly inside message handlers, but MUST preserve the “exactly once” semantics below.

#### Counting semantics and invariants (normative)

For a given epoch `e`, counters record events that occur in epoch `e`:

* `sessions_opened` MUST count unique `session_id` values for which `OnRetrievalSessionOpened` executed in epoch `e`.
* `sessions_served` MUST count unique `session_id` values for which `OnRetrievalSessionServiceProofAccepted` executed in epoch `e`.
* `sessions_settled` SHOULD count unique `session_id` values for which `OnRetrievalSessionSettled` executed in epoch `e`, but MUST NOT be used as a denominator for retrieval pressure.
* `proofs_of_failure_submitted` MUST count unique `session_id` values for which `OnRetrievalProofOfFailureAccepted` executed in epoch `e`.

Uniqueness rules (consensus-critical):

* Each retrieval session MUST contribute **at most 1** to each of:

  * `sessions_opened`
  * `sessions_served`
  * `sessions_settled`
  * `proofs_of_failure_submitted`
    over the lifetime of that session.
* Implementations MUST enforce uniqueness by session state (e.g., boolean flags) such that replays / duplicate submissions do not increment counters.

Mutual exclusion rule (consensus-critical):

* A retrieval session MUST NOT be both “served” and have a proof-of-failure accepted.

  * If a proof-of-failure is accepted for a session, any subsequent service proof submissions for that session MUST be rejected.
  * If a service proof is accepted for a session, any proof-of-failure submissions for that session MUST be rejected.
* Enforcement MUST occur in the consensus-validated message handlers/state machine prior to invoking the corresponding hooks, so that counters and state transitions remain consistent.

Proxy subset invariant (consensus-critical):

* `proxy_sessions_served` MUST count the subset of `sessions_served` in epoch `e` whose `session.is_proxy == true` (see Proxy classification below).
* For each epoch `e`, the keeper MUST ensure:

  * `proxy_sessions_served <= sessions_served`

Note on epoch windows (non-normative): `proofs_of_failure_submitted` in epoch `e` may refer to sessions opened in prior epochs, so it is not guaranteed to be `<= sessions_opened` for the same epoch. Rate computation and pressure mapping therefore MUST be robust to numerator > denominator (see §8.5 and §8.7.2).

#### Proxy classification rule (v1 normative)

The retrieval session state MUST include an immutable boolean `is_proxy` set at session open.

* The open path MUST NOT allow arbitrary callers to set `is_proxy=true`. Instead, the keeper MUST derive `session.is_proxy` deterministically from on-chain deputy/gateway authority state and the opener address.

  * **v1 normative rule:** `session.is_proxy == true` iff the session opener is an **authorized deputy/gateway** at the time of open, as determined by an on-chain registry/allowlist owned by the deputy/gateway authority.
  * If the opener is not authorized, `session.is_proxy` MUST be `false`. If the chain exposes a dedicated “proxy open” message/path, that message/path MUST reject unauthorized callers.
* At service proof acceptance (`OnRetrievalSessionServiceProofAccepted`), the keeper MUST read `session.is_proxy` and MUST increment `proxy_sessions_served` iff it is `true`.
* `session.is_proxy` MUST NOT change after session open; any state transition or message that would mutate it MUST be rejected.

If the session type does not yet contain `is_proxy` **or** there is no on-chain deputy/gateway authorization registry to derive it from, retrieval pricing updates MUST remain disabled (Stage 0 metrics-only) until this field and authorization rule are implemented and enforced.

**Legacy session handling (normative):** retrieval sessions created before the upgrade that introduces `session.is_proxy` (or before proxy authorization is enforceable) MUST be treated as `is_proxy=false` for the purposes of `proxy_sessions_served` accounting.

### 6.5 Audit budget counters (normative)

`audit_budget_requested`, `audit_budget_spent`, and `audit_budget_denied` MUST be derived from a single keeper helper that is the exclusive debit path for the audit budget account.

#### Required helper (normative)

All protocol flows that attempt to spend from audit budget MUST call:

* `SpendAuditBudget(ctx, amount, reason)` (or an equivalent helper with identical semantics)

#### Validation (normative)

For a given `amount`:

* `amount` MUST be strictly positive (`amount > 0`).
* `amount.Denom` MUST equal `base_denom` (see §3.4).

If validation fails, the helper MUST return an error and MUST NOT mutate any of:

* `audit_budget_requested`
* `audit_budget_spent`
* `audit_budget_denied`
* `audit_budget_spend_attempts`

#### “Insufficient balance” detection (normative; determinism safety)

The helper MUST detect “insufficient audit budget balance” using a stable error identity (a module-defined sentinel error or a stable SDK error code/type), and MUST NOT use string matching.

Normative requirement:

* The audit budget debit path MUST return (or wrap) a dedicated sentinel error value, herein referred to as `ErrAuditBudgetInsufficientFunds`, when and only when the debit fails solely due to insufficient audit budget balance.
* `SpendAuditBudget` MUST classify the “denied” path **only** when `errors.Is(err, ErrAuditBudgetInsufficientFunds)` is true (or equivalent stable identity check).
* Any other error MUST be treated as “other error” and MUST NOT be classified as denied.

#### Counter update semantics (normative)

For a validated `amount`, the helper MUST attempt the debit and update per-epoch counters deterministically as follows:

1. **On success**:

   * increment `audit_budget_requested += amount`
   * increment `audit_budget_spent += amount`
   * increment `audit_budget_spend_attempts += 1`

2. **On failure solely due to insufficient audit budget balance** (as defined above):

   * increment `audit_budget_requested += amount`
   * increment `audit_budget_denied += amount`
   * increment `audit_budget_spend_attempts += 1`

3. **On any other error** (internal module error, invariant violation, etc.):

   * the helper MUST return that error, and
   * MUST NOT mutate any of the counters above.

`audit_budget_denied` MUST represent the amount requested but not paid due to insufficient balance. Implementations MUST ensure, for each epoch `e`:

* `audit_budget_requested == audit_budget_spent + audit_budget_denied`

This invariant is checked at metrics finalization (§6.6), and violations are handled fail-closed (no chain halt) as specified below.

`audit_budget_minted` MUST be the actual minted amount for the epoch as recorded by the audit budget subsystem (Option A).

**Canonical measurement point (normative):** the audit budget subsystem MUST call a chain hook (name illustrative) exactly once per epoch immediately after minting the audit budget for that epoch:

* `RecordAuditBudgetMint(ctx, epoch_id, amount)`

This hook MUST set `MarketPricingEpochMetrics{epoch_id}.audit_budget_minted = amount` deterministically. The epoch metrics entry MAY initialize `audit_budget_minted` to zero as a placeholder; `RecordAuditBudgetMint` MUST overwrite it exactly once per epoch after minting completes.

### 6.6 Metrics lifecycle and immutability (normative)

This RFC requires deterministic per-epoch metrics that are unambiguous about which epoch they describe.

Define:

* `e = current_epoch` per §3.3.

`MarketPricingEpochMetrics{epoch_id=e}` describes epoch `e` and has two classes of fields:

* **Snapshot-at-epoch-start fields** (written once at the epoch boundary into epoch `e`):

  * `active_slot_bytes`
  * `repairing_slot_bytes`
  * `audit_budget_minted` (the amount minted *for epoch e*)
* **Accumulators-over-the-epoch fields** (updated during epoch `e`):

  * `audit_budget_requested`, `audit_budget_spent`, `audit_budget_denied`, `audit_budget_spend_attempts`
  * `sessions_opened`, `sessions_served`, `sessions_settled`, `proxy_sessions_served`, `proofs_of_failure_submitted`

Lifecycle (normative):

1. **Initialization (epoch boundary into epoch `e`)**

   * The keeper MUST create or reset a **mutable** metrics entry for `epoch_id=e`.

   * It MUST set snapshot-at-epoch-start fields as follows:

     * `active_slot_bytes` and `repairing_slot_bytes` MUST be obtained at the epoch boundary using the incrementally maintained aggregates defined in §6.3 (MUST NOT scan all deals/slots).
     * `audit_budget_minted` MUST be initialized to zero as a placeholder and then set to the actual amount minted for epoch `e` by the audit budget subsystem (Option A) via `RecordAuditBudgetMint(ctx, epoch_id=e, amount)` exactly once per epoch.

       * The epoch-`e` metrics entry MUST exist before `RecordAuditBudgetMint` is invoked. The application MUST ensure this by module ordering (market pricing metrics initialization runs before audit budget minting), or by defining `RecordAuditBudgetMint` to create the epoch metrics entry if absent (implementation choice).
       * If audit budget minting depends on `Params.storage_price`, minting for epoch `e` MUST run **after** any market-pricing update to `storage_price` for epoch `e` has been applied (see also §8.1).

   * It MUST zero all accumulator-over-the-epoch fields.

2. **Accumulation (during epoch `e`)**

   * Transaction handlers and keeper helpers MUST update **only** the mutable metrics entry for `epoch_id=e`.
   * Implementations MUST ensure counters are **unique per retrieval session id** where required (see §6.4), and MUST NOT use wall-clock time.

3. **Finalization (epoch boundary into epoch `e+1`)**

   * The metrics entry for `epoch_id=e` becomes **finalized** and MUST NOT be mutated thereafter.
   * During finalization, the keeper MUST evaluate the following invariants for epoch `e`:

     * Audit budget arithmetic invariant:

       * `audit_budget_requested == audit_budget_spent + audit_budget_denied`
     * Denom invariants for audit budget coins:

       * `audit_budget_requested.Denom == audit_budget_spent.Denom == audit_budget_denied.Denom == base_denom`
     * Retrieval proxy subset invariant:

       * `proxy_sessions_served <= sessions_served`

   **Liveness / fail-closed handling (normative):**

   * Invariant failures in this subsection MUST NOT halt block execution in production configurations.
   * If an invariant failure is detected for epoch `e`, the keeper MUST treat the corresponding market signal(s) as unavailable for the subsequent epoch-boundary controller computation (§8.1), MUST NOT advance the affected EMA(s), and MUST skip applying the affected market updates with `skip_reason_*=INVARIANT_VIOLATION`.
   * The chain MAY panic in debug/test configurations (non-normative), but such panics MUST NOT be relied upon for consensus safety.

4. **Controller input**

   * EMA and price updates applied at the epoch boundary into epoch `e+1` MUST use only finalized metrics for epoch `e` (never partial-epoch metrics).

Implementation note (non-normative): a keeper may implement the mutable entry as in-place updates of the KV entry keyed by `epoch_id=e`. Immutability is then enforced by never writing to past epoch ids.

---

## 7. Params Additions, Reserved Field Numbers, and Defaults

This RFC extends `nilchain/nilchain/v1/params.proto`.

### 7.1 Param field numbering (implementation safety)

As of the `Params` definition included in the repo assets, `params.proto` uses field numbers **1–46**, with `repair_override_enabled = 46`. This RFC proposes adding market pricing fields beginning at **47** and reserving a contiguous range for future additions.

**Requirement:** during implementation, engineers MUST re-check `nilchain/proto/nilchain/nilchain/v1/params.proto` on the target branch and ensure the chosen field numbers do not collide. The implementation SHOULD reserve a range (e.g., `47–80`) for market pricing to prevent later accidental collisions.

### 7.2 New params (v1)

```proto
// --- Market-equilibrium pricing (v1) ---
bool market_pricing_enabled = 47;        // default: false

// Master pause: freezes automatic updates but allows metrics collection if enabled.
bool market_pricing_updates_paused = 48; // default: true

// Fine-grained toggles for staged rollout.
bool market_pricing_update_storage_price = 49;   // default: false
bool market_pricing_update_retrieval_price = 50; // default: false

// Update cadence.
uint64 market_pricing_update_interval_epochs = 51; // default: 1

// Warm-up counter: counted from baseline_epoch (not genesis).
uint64 market_pricing_min_epochs_before_update = 52; // default: 10

// EMA smoothing factor (alpha = bps / 10_000).
uint64 market_pricing_ema_alpha_bps = 53; // default: 2000 (0.20)

// Max multiplicative change per month (converted to per-update cap using month_len_blocks_snapshot).
uint64 market_pricing_max_delta_bps_per_month = 54; // default: 1000 (10% / month)

// Baseline-relative clamps (multipliers).
string market_pricing_storage_floor_mult = 55
  [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false]; // default: 0.25
string market_pricing_storage_ceil_mult = 56
  [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false]; // default: 4.00

string market_pricing_retrieval_floor_mult = 57
  [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false]; // default: 1.00
string market_pricing_retrieval_ceil_mult = 58
  [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false]; // default: 4.00

// Storage target band (applies to audit_demand_util_bps).
uint64 market_pricing_target_audit_util_bps = 59;      // default: 6000 (60%)
uint64 market_pricing_alert_audit_util_low_bps = 60;   // default: 1000 (10%)
uint64 market_pricing_alert_audit_util_high_bps = 61;  // default: 9500 (95%)

// Retrieval target band.
uint64 market_pricing_target_proxy_frac_bps = 62;      // default: 100 (1%)
uint64 market_pricing_alert_proxy_frac_low_bps = 63;   // default: 0 (0%)
uint64 market_pricing_alert_proxy_frac_high_bps = 64;  // default: 500 (5%)

uint64 market_pricing_target_pof_rate_bps = 65;        // default: 100 (1%)
uint64 market_pricing_alert_pof_rate_low_bps = 66;     // default: 0 (0%)
uint64 market_pricing_alert_pof_rate_high_bps = 67;    // default: 300 (3%)

// Minimum sample sizes + rate denominator floor for retrieval updates.
uint64 market_pricing_min_retrieval_denominator_per_epoch = 68; // default: 100
uint64 market_pricing_retrieval_rate_denominator_floor = 69;    // default: 100
// Metrics retention horizon.
uint64 market_pricing_metrics_retention_epochs = 70;   // default: 30

// Minimum sample size for storage updates (audit budget spend attempts).
uint64 market_pricing_min_audit_spend_attempts_per_epoch = 71; // default: 10
```

### 7.3 Validation rules

* `market_pricing_update_interval_epochs >= 1`

* `market_pricing_min_epochs_before_update >= 0` (always true for `uint64`; retained for clarity)

* `market_pricing_ema_alpha_bps <= 10_000`

* `market_pricing_max_delta_bps_per_month <= 10_000`

* Denoms:

  * `Params.retrieval_price_per_blob.Denom` MUST be non-empty.
  * `Params.base_retrieval_fee.Denom` MUST equal `Params.retrieval_price_per_blob.Denom` (base denom; see §3.4).

* Non-degeneracy (normative):

  * `Params.month_len_blocks >= 1`
  * `Params.retrieval_price_per_blob.Amount > 0`
  * `Params.storage_price > 0` (strictly positive `LegacyDec`)

* Clamp multipliers:

  * `0 < market_pricing_storage_floor_mult <= 1`
  * `market_pricing_storage_ceil_mult >= 1`
  * `0 < market_pricing_retrieval_floor_mult <= 1`
  * `market_pricing_retrieval_ceil_mult >= 1`

* All target/alert bps params MUST satisfy `<= 10_000`.

* For each banded signal:

  * `alert_low <= target <= alert_high`

* Retrieval denominator parameters:

  * `market_pricing_retrieval_rate_denominator_floor >= 1`
  * `market_pricing_min_retrieval_denominator_per_epoch` is a `uint64` (may be 0 to disable sample-size gating), but SHOULD be bounded (recommended `<= 100_000`).

* Metrics retention:

  * `market_pricing_metrics_retention_epochs >= 1`
  * `market_pricing_metrics_retention_epochs` SHOULD be bounded (recommended `<= 365`).

* `market_pricing_min_audit_spend_attempts_per_epoch` is a `uint64` (may be 0 to disable sample-size gating), but SHOULD be bounded (recommended `<= 100_000`).

**Epoch length mutability constraint (normative; consensus safety):**

* While `Params.market_pricing_enabled == true`, `Params.epoch_len_blocks` MUST NOT change. Any governance/authority mechanism that sets params MUST enforce this by rejecting a params update that changes `epoch_len_blocks` while market pricing remains enabled.
* Governance MUST disable market pricing (`market_pricing_enabled=false`) prior to changing `epoch_len_blocks`, and MUST re-enable market pricing (capturing baselines at the next epoch boundary) after such a change.

### 7.4 Defaults (safe posture)

* Market pricing is **off by default**:

  * `market_pricing_enabled = false`

* Even if enabled, automatic updates are **paused by default**:

  * `market_pricing_updates_paused = true`

* Per-market updates are **disabled by default**:

  * `market_pricing_update_storage_price = false`
  * `market_pricing_update_retrieval_price = false`

* v1 defaults prevent below-baseline retrieval price decreases unless explicitly enabled by governance:

  * `market_pricing_retrieval_floor_mult = 1.00` (retrieval price cannot fall below the captured baseline unless governance lowers the floor multiplier and resets baselines)

This enables Stage 0 “metrics only” without further code toggles:

* Set `market_pricing_enabled=true` while keeping `market_pricing_updates_paused=true`.

---

## 8. Update Cadence, Bounds, and Exact Algorithm

This section is normative and implementation-oriented.

### 8.1 When updates happen (epoch boundary rule)

Define:

* `e = current_epoch` computed at the epoch boundary using §3.3.
* `update_period_epochs = market_pricing_update_interval_epochs` (>= 1).

**Determinism rule:** prices for epoch `e` MUST be computed only from metrics observed during the fully-completed previous epoch `e-1`, and applied exactly once at the epoch boundary into epoch `e` (i.e., `BeginBlocker` of the first block of epoch `e`).

This ensures:

* every tx in epoch `e` observes a single consistent price vector, and
* there is no intra-epoch price drift.

#### Epoch-boundary procedure (normative)

At the epoch boundary into epoch `e`:

0. **Event requirement (normative)**

   * If `Params.market_pricing_enabled == true`, the keeper MUST emit exactly one `EventMarketPricingUpdate` for this epoch boundary (§8.11), even if:

     * this is the first boundary after enable/baseline capture,
     * warm-up is not satisfied,
     * metrics are missing, or
     * updates are paused/interval-gated.
   * `EventMarketPricingEpochMetrics` is emitted only when a prior epoch’s metrics are successfully finalized (§8.11).

1. **Disabled fast-path**

   * If `Params.market_pricing_enabled == false`:

     * the keeper MUST delete `MarketPricingState` if it exists, and
     * the keeper MUST delete any mutable “current epoch” metrics entry (if it exists), and
     * the keeper MUST NOT collect metrics or apply any price/EMA updates for this RFC.
     * No events are required in this RFC when disabled.
     * Return.

2. **Initialization on first enable (baseline capture)**

   * If `Params.market_pricing_enabled == true` and `MarketPricingState` does not exist:

     * The keeper MUST initialize `MarketPricingState` as specified in §8.3, with:

       * `baseline_epoch = e`
       * `last_update_epoch = e`
       * `pending_baseline_reset = false`
     * The keeper MUST perform epoch-`e` metrics initialization (§6.6, snapshot + zeroed counters).
     * No price update or EMA advancement is applied at this boundary (the controller enters warm-up).
     * The keeper MUST emit `EventMarketPricingUpdate` with:

       * `warmup_satisfied=false`
       * `applied_storage=false`, `skip_reason_storage=WARMUP`
       * `applied_retrieval=false`, `skip_reason_retrieval=WARMUP`
       * and prices unchanged (next == previous).
     * Return.

3. **Finalize previous epoch metrics**

   * Let `prev = e - 1` (only defined if `e > 0`).
   * If `e > 0`, the keeper MUST attempt to load the mutable metrics entry for epoch `prev`.

     * If the entry does not exist:

       * the keeper MUST treat this as `MISSING_METRICS` for this boundary:

         * MUST NOT advance any EMAs,
         * MUST NOT apply any price updates,
         * MUST still perform epoch-`e` metrics initialization (§6.6),
         * MUST emit `EventMarketPricingUpdate` with `skip_reason_storage=MISSING_METRICS` and `skip_reason_retrieval=MISSING_METRICS`,
         * and Return.
     * Else (metrics for `prev` exist):

       * the keeper MUST finalize metrics for epoch `prev` per §6.6, including invariant evaluation.
       * Immediately after finalizing epoch `prev`, the keeper MUST prune metrics according to §6.2.
       * The keeper MUST emit `EventMarketPricingEpochMetrics` for epoch `prev` (§8.11).

4. **Apply pending baseline reset (epoch-boundary effective)**

   * If `state.pending_baseline_reset == true`, the keeper MUST apply the pending reset **at this epoch boundary** before any EMA or price logic:

     * set `state.baseline_epoch = e`
     * set `state.last_update_epoch = e`
     * set `state.baseline_storage_price = Params.storage_price`
     * set `state.baseline_retrieval_price_per_blob = Params.retrieval_price_per_blob`
     * set `state.month_len_blocks_snapshot = Params.month_len_blocks`
     * set `state.storage_pressure_ema_microbps = 0`
     * set `state.retrieval_pressure_ema_microbps = 0`
     * set `state.pending_baseline_reset = false`
   * After applying a pending reset at this boundary:

     * the keeper MUST NOT advance EMAs at this boundary, and
     * MUST NOT apply any price updates at this boundary (warm-up restarts from `baseline_epoch=e`).
   * The keeper MUST still perform epoch-`e` metrics initialization (§6.6) and MUST emit `EventMarketPricingUpdate` (with `skip_reason_*=WARMUP` and unchanged prices).
   * Return.

5. **Compute pressures and advance EMAs (metrics-driven; even if updates are paused)**

   * Using only finalized metrics for epoch `prev`, the keeper MUST compute storage and retrieval pressure inputs (§5, §8.5–§8.7) and update EMAs (§8.8), subject to signal availability and invariant-failure rules:

     * Storage signal availability is governed by §8.6. If unavailable, the keeper MUST:

       * set `storage_signal_available=false`,
       * set `p_storage_bps=0` for observability, and
       * MUST NOT modify `state.storage_pressure_ema_microbps` (storage EMA freeze).
     * Retrieval signal availability is governed by §8.5. If a retrieval invariant failure is detected at finalization (§6.6), the keeper MUST treat the retrieval market as unavailable for this boundary: set `retrieval_signal_available=false`, set `p_retrieval_bps=0` for observability, and MUST NOT modify `state.retrieval_pressure_ema_microbps`. If no invariant failure is present, then if both retrieval signals are unavailable, retrieval updates are skipped, but EMA advancement still occurs using neutral contributions as specified in §8.7.2.
   * EMA advancement MUST occur whenever finalized metrics are available and `market_pricing_enabled == true`, regardless of whether price updates are paused or interval-gated. (This avoids “latent jumps” when unpausing.) Storage EMA is the exception when the storage signal is unavailable (freeze; see also §8.6 and §8.11).

6. **Compute per-update cap**

   * The keeper MUST compute `max_delta_bps_per_update` per §8.4.
   * If §8.4 yields an overflow condition (`OVERFLOW`), the keeper MUST treat global eligibility as false for this boundary (no price updates), but MAY still have advanced EMAs in step 5.

7. **Global update eligibility gates**

   * Price updates at the epoch boundary into `e` are globally eligible only if all of the following hold:

     * `Params.market_pricing_updates_paused == false`

     * warm-up is satisfied (§8.2)

     * the per-update cap was computed successfully (§8.4) (i.e., no required overflow check failed)

     * interval gate is satisfied:

       ```
       (e - state.last_update_epoch) >= update_period_epochs
       ```

     * If the interval gate is not satisfied, `state.last_update_epoch` MUST NOT change.

8. **Per-market updates**

   * Storage price update for epoch `e` MAY be applied only if:

     * global eligibility holds, and
     * `Params.market_pricing_update_storage_price == true`, and
     * storage gating holds (§8.6).
   * Retrieval price update for epoch `e` MAY be applied only if:

     * global eligibility holds, and
     * `Params.market_pricing_update_retrieval_price == true`, and
     * retrieval gating holds (§8.5).

9. **`last_update_epoch` semantics**

   * If at least one of (storage update, retrieval update) is applied at this boundary, the keeper MUST set:

     * `state.last_update_epoch = e`

   * Otherwise, `state.last_update_epoch` MUST NOT change.

10. **Audit budget mint ordering (epoch `e`)**

* If the audit budget subsystem mints the audit budget for epoch `e` at epoch boundaries and that mint amount depends on `Params.storage_price`, then the chain MUST ensure the following ordering within the epoch boundary into `e`:

  * the `storage_price` update for epoch `e` (if applied in step 8) is committed **before** the audit budget minting logic for epoch `e` executes, and
  * after minting for epoch `e` completes, the audit budget subsystem MUST call `RecordAuditBudgetMint(ctx, epoch_id=e, amount)` (see §6.5/§6.6) to record `MarketPricingEpochMetrics{epoch_id=e}.audit_budget_minted`.

* The application MUST enforce this ordering via BeginBlocker module order or by orchestrating the audit budget mint call such that it occurs after step 8 and after step 11 has initialized the epoch-`e` metrics entry.

11. **Initialize metrics for epoch `e`**

* The keeper MUST initialize the mutable metrics entry for epoch `e` per §6.6 (snapshot + zeroed counters).
* As part of initialization, `audit_budget_minted` MUST be set to zero and then updated exactly once via `RecordAuditBudgetMint(ctx, epoch_id=e, amount)` after the audit budget subsystem mints for epoch `e` (step 10).

12. **Emit `EventMarketPricingUpdate`**

* The keeper MUST emit exactly one `EventMarketPricingUpdate` for this boundary (§8.11), reflecting:

  * gate outcomes,
  * signal availability (including `storage_signal_available`),
  * skip reasons, and
  * previous/next prices and EMAs.

### 8.2 Warm-up counter semantics

`market_pricing_min_epochs_before_update` MUST be counted from `MarketPricingState.baseline_epoch`:

* Let `baseline_epoch` be the epoch when baselines were captured (initial enable or explicit reset applied).
* Automatic updates MUST NOT be applied unless:

```
current_epoch >= baseline_epoch + market_pricing_min_epochs_before_update
```

This resolves ambiguity about whether the warm-up is from genesis vs enablement.

### 8.3 Enable/disable behavior (baseline capture + reset)

#### State existence as the enable-edge detector (normative)

To avoid reliance on an in-memory “previous param value”, this RFC defines the enable edge deterministically using on-chain state:

* When `Params.market_pricing_enabled == false`, the keeper MUST delete `MarketPricingState` and any mutable “current epoch” metrics entry at the next epoch boundary (§8.1, step 1).
* Therefore, when pricing is later enabled, the *absence* of `MarketPricingState` is the canonical indicator that baselines MUST be captured.

#### Baseline capture on enable (normative)

At the epoch boundary into epoch `e`, if `Params.market_pricing_enabled == true` and `MarketPricingState` is absent, the keeper MUST initialize `MarketPricingState` as follows:

* `baseline_epoch = e`
* `last_update_epoch = e`
* `baseline_storage_price = Params.storage_price`
* `baseline_retrieval_price_per_blob = Params.retrieval_price_per_blob`
* `month_len_blocks_snapshot = Params.month_len_blocks`
* `storage_pressure_ema_microbps = 0`
* `retrieval_pressure_ema_microbps = 0`
* `pending_baseline_reset = false`

This initialization is the only time baselines are captured automatically.

#### Mid-epoch governance changes (normative)

Governance MAY toggle params mid-epoch. This RFC defines:

* Baseline capture and metrics initialization occur only at epoch boundaries. If `market_pricing_enabled` becomes true mid-epoch, the controller MUST remain inactive until the next epoch boundary, at which point baselines are captured for that next epoch.
* If `market_pricing_enabled` becomes false mid-epoch, the controller MUST stop collecting metrics immediately. Any partial-epoch mutable metrics for that epoch MUST be discarded at the next epoch boundary as part of the disabled fast-path (§8.1, step 1).
* `MsgResetMarketPricingBaselines` is epoch-boundary effective via the pending reset mechanism (§10.2.1); it MUST NOT change `P0` mid-epoch.

#### Explicit reset

Baseline reset via `MsgResetMarketPricingBaselines` is specified in §10.2.1 and MUST override any existing state via the pending reset mechanism.

### 8.4 Per-update max delta (monthly cap converted using a snapshot)

Let:

* `blocks_per_update = Params.epoch_len_blocks * update_period_epochs`
* `month_len = state.month_len_blocks_snapshot`

**Snapshot validity (normative):** if `state.month_len_blocks_snapshot == 0` (e.g., due to a bad genesis/migration), the keeper MUST use `Params.month_len_blocks` for this boundary’s cap computation instead. If both are `0`, the keeper MUST fail-closed for this boundary (skip all updates; emit `skip_reason_*=INVARIANT_VIOLATION`).

All computations in this subsection are consensus-critical and MUST be overflow-safe.

#### Computation (normative)

1. Compute `blocks_per_update` using checked `uint64` multiplication:

   * `blocks_per_update, overflow := checkedMulU64(Params.epoch_len_blocks, update_period_epochs)`
   * If `overflow == true`, the keeper MUST:

     * set `overflow=true` for the boundary,
     * skip applying **all** price updates at this boundary, and
     * emit `EventMarketPricingUpdate` with `skip_reason_storage=OVERFLOW` and `skip_reason_retrieval=OVERFLOW`.
     * (EMA advancement may still occur per §8.1 step 5.)

2. Compute the numerator using checked multiplication:

   * `numer, overflow := checkedMulU64(Params.market_pricing_max_delta_bps_per_month, blocks_per_update)`
   * If `overflow == true`, the keeper MUST take the same overflow handling path as above (skip all updates; emit OVERFLOW).

3. Compute the raw per-update cap (basis points) using checked ceil division:

```
den = max(1, month_len)
max_delta_bps_per_update_raw = ceil_div_u64_checked(numer, den)
```

Where:

* `ceil_div_u64_checked(a,b)` implements `ceil_div_u64(a,b) = (a + b - 1) / b` with `b > 0`, and MUST treat overflow in `a + b - 1` as an overflow condition for this boundary (skip all updates; emit OVERFLOW).

4. Clamp to a safe bound:

```
max_delta_bps_per_update = min(max_delta_bps_per_update_raw, 10_000)
```

This ensures the per-update cap is never greater than 100% (10,000 bps). (Decreases are additionally clamped to avoid `k = 0`; see §8.9 and §8.10.2.)

**Volatility freeze rule (normative):** `state.month_len_blocks_snapshot` MUST NOT change unless baselines are reset (by enable initialization or the pending reset mechanism). This prevents unrelated governance changes to `Params.month_len_blocks` from abruptly changing price volatility.

### 8.5 Retrieval update gating and rate denominator floor

Retrieval gating is evaluated in §8.1 (step 8) and applies only to the retrieval market. Global gates (enabled/paused/warm-up/interval) are handled separately in §8.1.

Define:

* `min_den = market_pricing_min_retrieval_denominator_per_epoch`
* `floor_den = market_pricing_retrieval_rate_denominator_floor`

#### Signal availability (normative)

For epoch `prev` finalized metrics:

* The proxy-fraction signal is **available** iff:

  * `sessions_served >= min_den`

* The proof-of-failure signal is **available** iff:

  * `sessions_opened >= min_den`

Retrieval price update MUST be skipped if **both** signals are unavailable (i.e., `sessions_served < min_den` AND `sessions_opened < min_den`).

If exactly one of the two signals is available, the retrieval market MAY still be updated using the available signal; the unavailable signal contributes neutral pressure (`0 bps`) in §8.7.2.

#### Rate denominator floor (normative)

When computing rates, denominators MUST be floored to reduce low-volume spikes:

```
denom_opened = max(sessions_opened, floor_den)
denom_served = max(sessions_served, floor_den)
```

#### Rate computation (normative)

Rates MUST be computed using overflow-free intermediates (`sdkmath.Int`) and explicit rounding. Implementations MUST NOT allow `10_000 * count` to overflow a native integer type.

Let:

* `served := sdk.NewIntFromUint64(sessions_served)`
* `proxy_served := sdk.NewIntFromUint64(proxy_sessions_served)`
* `opened := sdk.NewIntFromUint64(sessions_opened)`
* `pof := sdk.NewIntFromUint64(proofs_of_failure_submitted)`
* `denom_served_int := sdk.NewIntFromUint64(denom_served)` (>= 1)
* `denom_opened_int := sdk.NewIntFromUint64(denom_opened)` (>= 1)

Compute raw rates as `sdkmath.Int`:

```
proxy_fraction_bps_raw = floor_div_int(proxy_served * 10_000, denom_served_int)
pof_rate_bps_raw       = floor_div_int(pof * 10_000,        denom_opened_int)
```

Notes:

* `proxy_fraction_bps_raw` is expected to be `<= 10_000` if `proxy_sessions_served <= sessions_served` (§6.4). If the invariant is violated, retrieval signal availability MUST be treated as false for this boundary with `skip_reason_retrieval=INVARIANT_VIOLATION` (§6.6).
* `pof_rate_bps_raw` MAY exceed `10_000` if proofs-of-failure are submitted for sessions opened in prior epochs. Pressure mapping MUST clamp/saturate as specified in §8.7.2.
* Event representability for raw rates is defined in §8.11.

### 8.6 Storage update gating

Storage gating is evaluated in §8.1 (step 8) and applies only to the storage market. Global gates (enabled/paused/warm-up/interval) are handled separately in §8.1.

#### Storage signal availability (normative)

Storage signal availability for epoch `prev` is **true** if and only if all of the following hold:

1. **Sample size**

   * `audit_budget_spend_attempts >= market_pricing_min_audit_spend_attempts_per_epoch`

2. **Audit budget subsystem is active**

   * audit budget Option A is enabled (`Params.audit_budget_bps > 0`)

3. **Non-degenerate supply snapshot**

   * `active_slot_bytes > 0`

4. **Audit budget metrics invariants for `prev` hold**

   * `audit_budget_requested == audit_budget_spent + audit_budget_denied`, and
   * all three denoms are equal to `base_denom`
   * If any fail: storage signal is unavailable with `skip_reason_storage=INVARIANT_VIOLATION` (or `DENOM_MISMATCH` where applicable).

5. **Reference mint is well-defined**

   * The keeper MUST compute `ref_minted_amt` for epoch `prev` using the canonical function in §5.1 (baseline price `P0 = state.baseline_storage_price` and `A_prev = metrics.active_slot_bytes`).
   * If `ref_minted_amt == 0`, the storage signal is unavailable with `skip_reason_storage=ZERO_REF_MINT`, and `audit_demand_util_bps_raw` MUST be recorded as `0` for this boundary (no `requested/1` observability artifact).

6. **Denoms are consistent**

   * `metrics.audit_budget_requested.Denom == base_denom` (and similarly for spent/denied, per invariant checks).

If any condition fails, storage price update MUST be skipped.

#### Storage EMA behavior when unavailable (normative)

If storage signal availability is false for this boundary:

* `storage_signal_available` MUST be emitted as `false` (§8.11),
* `p_storage_bps` MUST be treated as `0` for observability, and
* `state.storage_pressure_ema_microbps` MUST NOT change (storage EMA freeze).

**Warning (normative, observability):** because storage EMA freezes when the storage signal is unavailable, the stored EMA value may become stale if signal unavailability persists for multiple epochs; when availability returns, the first resumed update may appear as a step-change relative to time. Indexers and UIs MUST use `storage_signal_available` and `skip_reason_storage` to interpret storage behavior correctly (§8.11).

### 8.7 Pressure normalization (integer bps)

For both markets, compute a normalized pressure as **signed basis points**:

* `p_bps ∈ [-10_000, +10_000]`

  * `+10_000` means “maximum upward pressure”
  * `0` means neutral
  * `-10_000` means “maximum downward pressure”

All computations in this section MUST be done using integer arithmetic with explicit truncation rules to avoid consensus divergence.

#### 8.7.1 Storage pressure (v1)

If `storage_signal_available == false` (§8.6):

* set `p_storage_bps = 0` and do not advance storage EMA (§8.8).

Else:

Compute `audit_demand_util_bps_raw` per §5.1 as `sdkmath.Int` (non-negative).

Let:

* `u_raw_int = audit_demand_util_bps_raw` (`sdkmath.Int`)
* `t = market_pricing_target_audit_util_bps` (`uint64`)
* `lo = market_pricing_alert_audit_util_low_bps` (`uint64`)
* `hi = market_pricing_alert_audit_util_high_bps` (`uint64`)

For pressure mapping only (not for observability), define `u` as a `uint64` obtained by clamping `u_raw_int` into `[lo, hi]` deterministically:

* If `u_raw_int <= lo`, set `u = lo`.
* Else if `u_raw_int >= hi`, set `u = hi`.
* Else (so `lo < u_raw_int < hi`), `u_raw_int` fits in `uint64` and MUST be converted exactly: `u = u_raw_int.Uint64()`.

Define signed error:

```
err = int64(u) - int64(t)
```

Compute:

* if `err >= 0` (util above target):

```
den = max(1, hi - t)
p_storage_bps = min(10_000, 10_000 * err / int64(den))
```

* else (util below target):

```
den = max(1, t - lo)
p_storage_bps = max(-10_000, 10_000 * err / int64(den))   // err is negative => p negative
```

Rounding and overflow rules (normative):

* Division of `int64` MUST truncate toward zero (Go semantics).
* Intermediate products here are bounded (|err| <= 10_000 and multipliers <= 10_000), so int64 overflow is not expected. If an implementation uses narrower types, it MUST use checked arithmetic and MUST fail-closed (treat signal unavailable and skip updates) on overflow.

Optional repair pressure can be integrated later with explicit weights; v1 defaults to audit-only.

#### 8.7.2 Retrieval pressure (v1)

Compute the raw rates per §8.5 as `sdkmath.Int`:

* `proxy_fraction_bps_raw`
* `pof_rate_bps_raw`

Also compute signal availability per §8.5:

* `proxy_available = (sessions_served >= market_pricing_min_retrieval_denominator_per_epoch)`
* `pof_available   = (sessions_opened >= market_pricing_min_retrieval_denominator_per_epoch)`

If a signal is unavailable, its pressure contribution MUST be set to `0 bps` for this epoch. The combination rule below MUST ensure an unavailable signal does not mask an available signal (notably when the available signal is negative).

##### Proxy fraction pressure

If `proxy_available == false`:

* set `p_proxy_bps = 0`.

Else:

Let:

* `u_raw_int = proxy_fraction_bps_raw` (`sdkmath.Int`)
* `t = market_pricing_target_proxy_frac_bps`
* `lo = market_pricing_alert_proxy_frac_low_bps`
* `hi = market_pricing_alert_proxy_frac_high_bps`

For pressure mapping only, define `u` as the clamp of `u_raw_int` into `[lo, hi]` using the same deterministic clamp/conversion rule as §8.7.1.

Compute `p_proxy_bps` using the same piecewise linear mapping (target-centered):

* if `u >= t`:

```
den = max(1, hi - t)
p_proxy_bps = min(10_000, 10_000 * int64(u - t) / int64(den))
```

* else:

```
den = max(1, t - lo)
p_proxy_bps = max(-10_000, -10_000 * int64(t - u) / int64(den))
```

##### Proof-of-failure pressure

If `pof_available == false`:

* set `p_pof_bps = 0`.

Else:

Let:

* `u_raw_int = pof_rate_bps_raw` (`sdkmath.Int`; may exceed 10_000)
* `t = market_pricing_target_pof_rate_bps`
* `lo = market_pricing_alert_pof_rate_low_bps`
* `hi = market_pricing_alert_pof_rate_high_bps`

For pressure mapping only, define `u` as the clamp of `u_raw_int` into `[lo, hi]` using the same deterministic clamp/conversion rule as §8.7.1. (Because `hi <= 10_000`, conversion is always exact once clamped.)

Compute `p_pof_bps` analogously.

##### Combine retrieval pressure

Combine (normative):

```
if proxy_available && pof_available:
  p_retrieval_bps = max(p_proxy_bps, p_pof_bps)
else if proxy_available:
  p_retrieval_bps = p_proxy_bps
else if pof_available:
  p_retrieval_bps = p_pof_bps
else:
  p_retrieval_bps = 0
```

Rationale:

* any strong positive stress signal dominates (price increases),
* negative pressure is allowed when the available signal(s) are below target, and
* if exactly one signal is available, the update uses that signal rather than masking it with a neutral `0`.

### 8.8 EMA update (deterministic fixed-point)

Define:

* `alpha_bps = market_pricing_ema_alpha_bps` (0..10_000)
* `S = 1_000_000` (micro-bps per bps)

Let:

* `ema_prev = state.*_pressure_ema_microbps` (int64)
* `p = p_*_bps` (int64, bps in [-10_000, +10_000])

Compute (normative, int64 arithmetic):

```
ema_next = (ema_prev*(10_000 - alpha_bps) + (p*S)*alpha_bps) / 10_000
```

**Rounding rule (normative):** signed integer division MUST truncate toward zero (Go default). This is deterministic.

After computing `ema_next`, the keeper MUST clamp it into the representable range:

```
ema_next = clamp_int64(-10_000*S, +10_000*S, ema_next)
```

Overflow rule (normative):

* Given the specified bounds on `ema_prev`, `p`, and `alpha_bps`, all intermediate products fit in signed 64-bit range. If an implementation uses narrower types or cannot guarantee the bound, it MUST use checked arithmetic and MUST fail-closed (treat signal unavailable and skip affected EMA update) on overflow.

### 8.9 Convert EMA to capped per-update delta (integer bps)

Given:

* `ema_next` in micro-bps,
* `max_delta = max_delta_bps_per_update` (uint64, clamped to `<= 10_000` in §8.4),
* `S = 1_000_000`,

Compute signed delta:

```
delta_bps = trunc_toward_zero( ema_next * int64(max_delta) / (10_000 * S) )
```

Then clamp to the configured cap:

```
delta_bps = clamp_int64(-int64(max_delta), +int64(max_delta), delta_bps)
```

Finally, enforce a strict positivity guard on the multiplicative factor `k = 10_000 + delta_bps` used in §8.10:

```
delta_bps = max(delta_bps, -9_999)
```

This guarantees `k >= 1` (i.e., price multipliers never reach zero).

### 8.10 Apply price updates + clamps (deterministic rounding)

#### 8.10.1 Storage price

Let:

* `sp = Params.storage_price` (`LegacyDec`, units: base-denom per byte per block)
* `m = 10_000 + delta_storage_bps` (int64)

By §8.9, `delta_storage_bps >= -9_999`, therefore `m >= 1` and the multiplier is strictly positive.

Compute (normative):

```
sp_next = sp.MulInt64(m).QuoInt64(10_000)
```

This specifies the exact operation order and avoids any float conversions.

Clamp:

* `floor = baseline_storage_price * market_pricing_storage_floor_mult`
* `ceil  = baseline_storage_price * market_pricing_storage_ceil_mult`

Then:

```
sp_next = clamp_dec(floor, ceil, sp_next)
```

**Strict positivity (normative):** `sp_next` MUST be strictly positive. Because `LegacyDec` division may truncate small values to zero, after clamping the keeper MUST enforce:

* `sp_next = max(sp_next, 1e-18)`

where `1e-18` is the smallest non-zero value representable in `LegacyDec` (1 atom at 18 decimal places).

**Bounded-volatility clarification (normative):** if `sp` is already outside `[floor, ceil]` (e.g., due to a governance override), then clamp enforcement MAY change the price by more than the per-update delta cap; this is the only case where a larger discrete change is permitted (§2.1, §10.2.2).

#### 8.10.2 Retrieval price per blob

Let:

* `rp = Params.retrieval_price_per_blob` (`Coin`)
* `rp_amt = rp.Amount` (`sdk.Int`, non-negative)
* `k = 10_000 + delta_retrieval_bps` (int64)

By §8.9, `delta_retrieval_bps >= -9_999`, therefore:

* `1 <= k <= 20_000`

#### Denom consistency (normative)

If `rp.Denom != state.baseline_retrieval_price_per_blob.Denom`, the keeper MUST skip the retrieval update and emit `EventMarketPricingUpdate` with `skip_reason_retrieval=DENOM_MISMATCH`.

#### Update (normative)

Compute using `sdk.Int` intermediates (no overflow):

* if `delta_retrieval_bps >= 0`: **ceil division** (avoid undercharging on increases)

```
rp_next = ceil_div_int( rp_amt * k, 10_000 )
```

* else: **floor division** (ensure decreases actually decrease when possible)

```
rp_next = floor_div_int( rp_amt * k, 10_000 )
```

#### Clamp bounds (baseline-relative)

Let `baseline_rp = state.baseline_retrieval_price_per_blob`.

Compute bounds using deterministic `LegacyDec` math:

* `floor_amt = ceil( baseline_rp.Amount * market_pricing_retrieval_floor_mult )` (round up)
* `ceil_amt  = floor( baseline_rp.Amount * market_pricing_retrieval_ceil_mult )` (round down)

Then clamp:

```
rp_next = clamp_int(floor_amt, ceil_amt, rp_next)
```

**Strict positivity (normative):** `rp_next` MUST be `>= 1` in base denom units. After clamping, the keeper MUST enforce:

* `rp_next = max(rp_next, sdk.OneInt())`

Finally set:

```
Params.retrieval_price_per_blob = Coin{Denom: rp.Denom, Amount: rp_next}
```

**Bounded-volatility clarification (normative):** if the current `rp_amt` is already outside `[floor_amt, ceil_amt]` (e.g., due to a governance override), then clamp enforcement MAY change the price by more than the per-update delta cap; this is the only case where a larger discrete change is permitted (§2.1, §10.2.2).

#### 8.10.3 Deterministic integer primitives and helper definitions

All consensus-critical arithmetic in this RFC MUST be implemented using integer or fixed-point types with deterministic rounding. Implementations MUST NOT use floats.

Definitions (normative):

* For `uint64` checked arithmetic:

  * `checkedMulU64(a,b) -> (uint64, overflow)` returns `overflow=true` iff `a*b` exceeds `math.MaxUint64`; otherwise returns the product and `overflow=false`.

  * `checkedAddU64(a,b) -> (uint64, overflow)` returns `overflow=true` iff `a+b` exceeds `math.MaxUint64`; otherwise returns the sum and `overflow=false`.

* For `uint64` ceil division with overflow checks:

  * `ceil_div_u64_checked(a,b)` requires `b > 0` and computes:

    ```
    t, overflow := checkedAddU64(a, b-1)
    if overflow { overflow condition }
    return t / b
    ```

  * If overflow condition occurs, the keeper MUST fail-closed for this boundary: skip all price updates and emit `OVERFLOW` (§8.4).

* For non-negative `sdk.Int` (arbitrary-precision integer) with `b > 0`:

  * `floor_div_int(a, b)` is integer division rounding down.
  * `ceil_div_int(a, b)` is:

    ```
    ceil_div_int(a, b) = floor_div_int(a + (b - 1), b)
    ```

    where the addition is done in `sdk.Int` space.

* `trunc_toward_zero(x / y)` for signed integers is defined as the language-default signed integer division that truncates toward zero (Go semantics).

* `clamp_u64(x, lo, hi)` returns:

  * `lo` if `x < lo`
  * `hi` if `x > hi`
  * otherwise `x`

* `clamp_int64(lo, hi, x)` returns:

  * `lo` if `x < lo`
  * `hi` if `x > hi`
  * otherwise `x`

* `clamp_int(lo, hi, x)` for `sdk.Int` returns:

  * `lo` if `x < lo`
  * `hi` if `x > hi`
  * otherwise `x`

* `clamp_dec(lo, hi, x)` for `LegacyDec` returns:

  * `lo` if `x < lo`
  * `hi` if `x > hi`
  * otherwise `x`

### 8.11 Events and observability

On each epoch boundary where market pricing is enabled, the keeper MUST emit events sufficient to audit both the inputs and the update decision.

#### Metrics event (normative)

After finalizing metrics for epoch `prev = e-1` (when `e > 0` and the metrics entry exists), emit:

* `EventMarketPricingEpochMetrics` containing:

  * `epoch_id = prev`
  * the full persisted `MarketPricingEpochMetrics{epoch_id=prev}`

If metrics for `prev` are missing (§8.1 step 3), this event MUST NOT be emitted (and the boundary is handled via `MISSING_METRICS`).

#### Update decision event (normative)

At every epoch boundary where `market_pricing_enabled == true`, emit exactly one `EventMarketPricingUpdate` containing (at minimum):

* `epoch_id = e`

* `prev_epoch_id = prev` (if `e > 0` and `prev` metrics existed; otherwise omit or set to an explicit sentinel per implementation)

* `baseline_epoch`

* `last_update_epoch_before`, `last_update_epoch_after`

* Global gate flags:

  * `updates_paused`
  * `warmup_satisfied`
  * `interval_satisfied`
  * `overflow` (true if any required overflow check failed this epoch)

* `max_delta_bps_per_update`

Per-market decision fields:

* `applied_storage` (bool), `skip_reason_storage` (string/enum)
* `applied_retrieval` (bool), `skip_reason_retrieval` (string/enum)

Signal availability fields:

* `storage_signal_available` (bool; §8.6)
* (optional) `retrieval_proxy_signal_available` and `retrieval_pof_signal_available` (bools), if implemented

Recommended `skip_reason_*` values (string/enum):

* `APPLIED`
* `PAUSED`
* `WARMUP`
* `INTERVAL`
* `INSUFFICIENT_SAMPLE`
* `DENOM_MISMATCH`
* `ZERO_REF_MINT`
* `OVERFLOW`
* `MISSING_METRICS`
* `INVARIANT_VIOLATION`

Computed signals and controller state:

* raw rates (as computed in §5 and §8.5):

  * `audit_demand_util_bps_raw`
  * `proxy_fraction_bps_raw`
  * `pof_rate_bps_raw`
* pressure values:

  * `p_storage_bps`
  * `p_retrieval_bps`
* EMA values:

  * previous and next EMA micro-bps for storage and retrieval
* deltas:

  * `delta_storage_bps`, `delta_retrieval_bps`

Prices:

* previous and next prices for storage and retrieval (if not applied, next MUST equal previous).

##### Raw-rate representability (normative)

Raw-rate fields can exceed typical fixed-width integer ranges (notably `audit_demand_util_bps_raw` and `pof_rate_bps_raw`). Therefore:

* `audit_demand_util_bps_raw`, `proxy_fraction_bps_raw`, and `pof_rate_bps_raw` MUST be emitted as decimal strings representing non-negative integers (i.e., `sdkmath.Int` base-10 encoding), ensuring representability without overflow.
* If an implementation also emits a fixed-width numeric variant (e.g., `uint64`), it MUST deterministically cap the numeric variant to its maximum representable value (e.g., `math.MaxUint64`) and MUST NOT rely on wraparound.

Additionally, when `ref_minted_amt == 0`, `audit_demand_util_bps_raw` MUST be emitted as `"0"` and `storage_signal_available=false` with `skip_reason_storage=ZERO_REF_MINT` (no misleading `requested/1` artifact).

---

## 9. Interaction With Escrow, Spend Windows, and Retrieval Settlement

This RFC intentionally preserves all invariants in `rfcs/rfc-pricing-and-escrow-accounting.md`.

### 9.1 Storage lock-in deposits

Lock-in storage deposit at `MsgUpdateDealContent*` remains:

```
storage_cost = ceil(storage_price * delta_bytes * duration_blocks)
```

Dynamic pricing effects:

* Only **new delta bytes** are charged at the current spot `storage_price`.
* Previously committed bytes are not repriced.

### 9.2 Retrieval session open and settlement

Retrieval fees remain:

* At open:

  * burn `base_retrieval_fee` (non-refundable)
  * lock `variable_fee = retrieval_price_per_blob * blob_count`
* At completion:

  * burn `ceil(variable_fee * retrieval_burn_bps / 10_000)`
  * pay the remainder to the provider
* On cancel/expiry:

  * refund only the locked variable fee to `Deal.escrow_balance`

Dynamic pricing effects:

* Only **new sessions** use the new spot `retrieval_price_per_blob`.
* Existing sessions are unaffected because they store `session.locked_fee`.

### 9.3 Spend windows and escrow predictability

* Elasticity spend caps use `base_stripe_cost` and the spend window; this RFC does not modify them.
* To avoid destabilizing user escrow UX:

  * price updates are epoch-based (no intra-epoch drift),
  * per-update deltas are capped,
  * clamp bounds prevent runaway.

Wallets/UIs can compute the worst-case next-epoch drift as `max_delta_bps_per_update` and recommend an escrow buffer accordingly.

---

## 10. Governance Control Surface

Governance (module authority) retains full control and can override or disable market pricing.

### 10.1 Standard controls (params)

Authority MAY:

* enable metrics + controller via `market_pricing_enabled`
* pause/resume updates via `market_pricing_updates_paused`
* enable each market independently:

  * `market_pricing_update_retrieval_price`
  * `market_pricing_update_storage_price`
* tune cadence, caps, clamps, targets, and sample-size guards

### 10.2 Baseline reset semantics (required for safe governance overrides)

Baselines are critical because:

* clamps are defined as multipliers around baselines, and
* storage controller normalization uses the baseline storage price.

Therefore, this RFC defines an explicit baseline reset mechanism.

#### 10.2.1 Authority message: `MsgResetMarketPricingBaselines`

Add a new authority-only message:

* `MsgResetMarketPricingBaselines { string authority }`

**Epoch-boundary effective semantics (normative):** this message MUST NOT change baselines mid-epoch.

On acceptance of the message at any height within an epoch:

* The keeper MUST set `MarketPricingState.pending_baseline_reset = true`.

If multiple reset messages are accepted before the next epoch boundary, the latest accepted message wins (the pending flag remains true).

At the next epoch boundary into epoch `e` after the message is accepted, the keeper MUST apply the pending reset as specified in §8.1 step 4:

* set `baseline_epoch = e`
* set `last_update_epoch = e`
* set baselines to the current `Params.storage_price`, `Params.retrieval_price_per_blob`, and `Params.month_len_blocks` as observed at the epoch boundary
* reset EMAs to zero
* clear `pending_baseline_reset`

**Normative consequence:** `P0 = baseline_storage_price` used in §5.1 is constant within an epoch and cannot change mid-epoch due to reset.

#### 10.2.2 Interaction with governance spot price overrides

* Governance can set `storage_price` and/or `retrieval_price_per_blob` at any time via params update.
* If the new spot price lies outside the current baseline-relative clamps, the next automatic update will clamp the price back into range.

**Operational rule:** if governance intends a lasting step-change to the price regime (or volatility regime via month length), governance SHOULD:

1. set the new spot prices (and any clamp multipliers if needed), then
2. call `MsgResetMarketPricingBaselines` (which will apply at the next epoch boundary).

**Bounded-volatility clarification:** clamp enforcement may “snap” a price back into range from an out-of-range value due to governance override/misconfiguration; such a snap may exceed the per-update delta cap, which otherwise bounds controller-driven changes (§2.1, §8.10).

### 10.3 Emergency kill switch

Setting `market_pricing_updates_paused = true` MUST immediately stop automatic updates at the next epoch boundary without affecting:

* escrow balances,
* retrieval settlement, or
* in-flight sessions.

---

## 11. Security and Manipulation Analysis

### 11.1 Storage-side manipulation

**Threat:** attacker tries to manipulate `audit_budget_requested` to move `storage_price`.

Observations/mitigations:

* `audit_budget_requested` increments only when protocol subsystems attempt audit-budget spends (audit retrieval traffic, evidence incentives if funded from audit budget).
* Most such spends are gated by:

  * bonded evidence (proof-of-failure) and/or
  * protocol-driven audit scheduling.
* Even if an attacker can create additional demand, the controller is bounded:

  * EMA smoothing limits impact of short spikes,
  * per-update max delta caps price movement,
  * clamps cap long-term movement without governance action.

**Implementation note:** the RFC requires `requested/spent/denied` to be recorded in the audit budget helper, making manipulation analysis auditable by inspecting spend reasons.

### 11.2 Retrieval wash / Sybil manipulation

**Threat A (upward):** attacker inflates retrieval stress to push price up.

* Controller does not use raw successful volume; it uses proxy share and proof-of-failure rate.
* Proof-of-failure is bonded (`evidence_bond`), and repeated submissions can be penalized; therefore large-scale fabrication is costly.

**Threat B (downward):** attacker tries to push price down by generating many “healthy” served sessions (low proxy and low pof rates).

* Downward pressure is conservative (`max(p_proxy, p_pof)`), and update deltas are capped.
* The controller’s denominators are **proof-accepted** counts (`sessions_served`, `sessions_opened`), not necessarily **settled** sessions.
* Because the retrieval variable fee locked at open can be refunded on cancel/expiry (§9.2), an attacker influencing “served”-based signals may incur **only**:

  * `base_retrieval_fee` per opened session (burned; non-refundable), and
  * any off-chain resource/opportunity costs (bandwidth, provider cooperation),
    while still shaping `sessions_served`-based and `sessions_opened`-based rates.
* Manipulation resistance for Threat B therefore relies on:

  * the non-refundable base fee as a per-session cost,
  * the difficulty/cost of producing on-chain accepted service proofs at scale without genuine service,
  * and (for the PoF-based signal) bonded submissions and mutual-exclusion rules (§6.4).

### 11.3 Low-volume epochs / noisy denominators

Risk:

* If denominators are small, single proxy/pof events can produce extreme rates.

Mitigations:

* Retrieval updates are gated on a minimum denominator.
* Rate denominators are floored for computation (stable floor), preventing 100% spikes from small N.

### 11.4 Oscillation and controller stability

Risks:

* oscillation if signals lag or if multiple controllers interact.

Mitigations:

* EMA smoothing with explicit alpha
* capped delta per update
* warm-up gating after baseline reset/enable
* price-invariant storage-side normalization prevents denominator feedback loops from dominating dynamics

---

## 12. Backward Compatibility and Migration

### 12.1 Backward compatibility

* New params and new state are additive.
* Default values keep market pricing disabled and updates paused.
* Existing deals and sessions are unaffected:

  * storage deposits already paid remain in escrow
  * retrieval session fees already locked remain unchanged

### 12.2 Migration strategy

On upgrade that introduces this RFC:

1. Add new params with safe defaults (`market_pricing_enabled=false`).
2. Add new stores for `MarketPricingState` and `MarketPricingEpochMetrics`.
3. If market pricing is enabled at genesis (not recommended), initialize `MarketPricingState` at genesis epoch using genesis params.

When governance later enables market pricing:

* baselines are captured at the enable edge as defined in §8.3.

---

## 13. Implementation Plan (Staged MVP)

### Stage 0 — Metrics only (no price updates)

* Add `MarketPricingState` and `MarketPricingEpochMetrics`.
* Wire tx hooks:

  * `MsgOpenRetrievalSession` → `sessions_opened++`
  * `MsgSubmitRetrievalSessionProof` → `sessions_served++` and proxy tagging
  * `MsgConfirmRetrievalSession` → `sessions_settled++` (monitoring-only)
  * proof-of-failure submission → `proofs_of_failure_submitted++` (with mutual exclusion vs served)
  * audit budget spend helper → requested/spent/denied counters (with sentinel insufficient-funds classification)
* Maintain incrementally updated aggregates for slot bytes (§6.3) and snapshot them at epoch boundary.
* Epoch boundary hook finalizes prior epoch metrics (fail-closed on invariant failures), prunes retention, emits events.
* Add queries:

  * `QueryMarketPricingState`
  * `QueryMarketPricingMetrics(epoch_id)` and/or window query

**Exit gate:** unit tests + e2e that assert metrics determinism across nodes.

### Stage 1 — Retrieval price updates

* Enable retrieval controller under explicit param:

  * `market_pricing_enabled=true`
  * `market_pricing_updates_paused=false`
  * `market_pricing_update_retrieval_price=true`
* Keep storage controller disabled:

  * `market_pricing_update_storage_price=false`

**Exit gate:** e2e in a multi-SP setup where proxy retrieval + proof-of-failure paths are exercised and retrieval price changes are observed across epochs.

### Stage 2 — Storage price updates

* Enable storage controller:

  * `market_pricing_update_storage_price=true`
* Ensure audit budget Option A and audit budget spend helper instrumentation are active.
* Ensure audit budget mint ordering relative to storage price update is correct (§8.1 step 10).

**Exit gate:** e2e where audit budget demand is generated (e.g., audit debt spending) and `storage_price` adjusts without oscillation or snapback artifacts.

### Stage 3 — Refinements (optional)

* Add repair pressure weighting with an explicit param and simulations.
* Add additional manipulation-resistant signals if needed (e.g., audit debt backlog) once on-chain.

---

## 14. Testing Plan

### 14.1 Unit tests

1. **Deterministic fixed-point math**

   * EMA update matches reference integer arithmetic.
   * delta_bps conversion is deterministic with truncation toward zero.

2. **Volatility cap freeze**

   * changing `Params.month_len_blocks` does not change `max_delta_bps_per_update` until baseline reset is applied.
   * baseline reset pending and application update `month_len_blocks_snapshot`.

3. **Gating correctness**

   * warm-up counted from baseline_epoch.
   * retrieval updates require denominators; low-volume epochs skip updates.
   * storage updates require `audit_budget_spend_attempts >= market_pricing_min_audit_spend_attempts_per_epoch`, `ref_minted_amt > 0`, and `active_slot_bytes > 0`.
   * storage signal unavailability freezes storage EMA and emits `storage_signal_available=false`.

4. **Invariant handling (fail-closed)**

   * audit budget requested/spent/denied mismatch does not halt chain; instead yields `skip_reason_storage=INVARIANT_VIOLATION` and no storage EMA advancement or update.
   * `proxy_sessions_served > sessions_served` yields `skip_reason_retrieval=INVARIANT_VIOLATION` and no retrieval update.

5. **Clamp enforcement**

   * prices never exceed baseline-relative min/max after update.
   * clamp snap behavior is permitted when current price is already out of range.

6. **Rounding**

   * retrieval price uses ceil on increases and floor on decreases.
   * baseline-derived floor/ceil bounds apply correct rounding directions.

7. **Baseline reset semantics**

   * reset is pending and applied only at an epoch boundary.
   * applying a reset clears EMAs and restarts warm-up, and does not apply updates at that boundary.

8. **Overflow handling**

   * per-update cap computation overflows cause deterministic skip with `OVERFLOW`.

### 14.2 End-to-end tests

Extend econ e2e scripts:

* run with small `epoch_len_blocks`
* create deal, ingest content, open/serve/settle retrieval sessions
* exercise proxy and pof submission paths (including mutual exclusion)
* verify:

  * price params change only at epoch boundaries
  * escrow accounting invariants hold (same formulas)
  * in-flight sessions settle correctly regardless of later price changes
  * baseline reset message is epoch-boundary effective
  * EventMarketPricingUpdate is emitted even on first enable and on skip paths

### 14.3 Simulation gate (stability / adversarial)

Add a deterministic sim harness (fixed RNG seed) that runs for `N` epochs and simulates:

* varying proxy share
* varying proof-of-failure rates
* varying audit-budget requested demand

Assert invariants:

* prices remain within clamps
* max delta per update respected (except clamp snap from out-of-range overrides)
* controller converges under constant signals (EMA stabilizes, deltas go to 0)
* no oscillation beyond configured bounds for reasonable alpha/caps

---

## 15. Calibration Signals and Alert Thresholds

Expose these metrics (events + queries):

* `price_GiBMonth` (derived)
* `price_GiBRetrieval` (derived)
* `audit_demand_util_bps_raw` (requested vs reference mint; string-encoded integer)
* `audit_budget_denied` (budget binding indicator)
* `proxy_fraction_bps_raw`
* `pof_rate_bps_raw`
* `repair_pressure_bps` (even if not used for price)
* `storage_signal_available` and `skip_reason_storage` (to interpret storage EMA freeze)

Recommended alert thresholds (consistent with policy notes):

* Audit demand utilization:

  * alert if `> 95%` (sustained demand pressure)
  * alert if `< 10%` sustained (overprovisioned or unused subsystem)
* Audit budget denied amount:

  * alert if non-zero for `> X` epochs (budget binding)
* Deputy/proxy share:

  * target `< 1%`, alert `> 5%`
* Proof-of-failure rate:

  * target `< 1%`, alert `> 3%`
* Clamp saturation:

  * alert if a price hits floor or ceiling for > `X` epochs (controller saturated; governance tuning needed)

---

## 16. Open Questions and Dependencies

If any of these are unresolved on the target branch, they MUST be treated as implementation blockers or explicitly deferred.

1. **Canonical proof-of-failure acceptance hook**

   * The retrieval subsystem MUST have a single canonical state transition for “PoF accepted” that can enforce *first-PoF-only* semantics per session, mutual exclusion vs served proofs (§6.4), and invoke `OnRetrievalProofOfFailureAccepted`.

2. **Canonical proxy classification (`is_proxy`)**

   * Retrieval sessions MUST include an immutable `is_proxy` flag derived deterministically at session open from an on-chain deputy/gateway authorization registry (§6.4).
   * Until this is implemented and enforced, retrieval pricing updates MUST remain disabled.

3. **Audit budget denomination**

   * v1 REQUIRES audit budget mint/spend to use `base_denom` (§3.4). If the audit budget subsystem uses a different denom, storage pricing updates MUST remain disabled until a deterministic conversion rule is specified.

4. **Audit budget insufficient-funds sentinel**

   * The audit budget debit path MUST expose a stable error identity for “insufficient audit budget balance” (§6.5); string matching MUST NOT be used.

5. **Incremental slot-byte aggregates**

   * The chain MUST maintain incrementally updated aggregates for `active_slot_bytes` and `repairing_slot_bytes` (§6.3). Epoch boundaries MUST NOT scan all deals/slots.

6. **Epoch-boundary ordering**

   * If audit budget minting depends on `Params.storage_price`, the epoch-boundary ordering MUST ensure that any market-pricing updates to `storage_price` for epoch `e` (if applied) occur before minting for epoch `e`, and before recording `audit_budget_minted` in the epoch-`e` metrics snapshot (§8.1 step 10).

7. **Module wiring / keeper patterns**

   * Confirm the epoch-boundary hook location and ordering for: finalize metrics → prune → apply pending reset (if any) → compute pressure/EMA → (optional) apply updates → audit budget mint ordering → initialize next epoch metrics → emit events (per §8.1 and §6.6).

---

---

## Change Summary (based on )

**Finding 1 — Applied change**

* **Sections changed:** §5.1, §8.6, §8.7.1, §8.11, §8.10.3
* **Change:** Made `ref_minted_amt` and `audit_demand_util_bps_raw` fully deterministic with explicit `sdk.Int` types, integer 1e18 scaling for `P0`, canonical ceil/floor div primitives, and `ZERO_REF_MINT` unavailability semantics.

**Finding 2 — Applied change**

* **Sections changed:** §8.4, §8.10.3
* **Change:** Extended overflow safety to all multiplications/additions in per-update cap computation using checked `uint64` ops and a deterministic `OVERFLOW` skip-all-updates path.

**Finding 3 — Applied change**

* **Sections changed:** §6.1, §8.1, §8.3, §10.2.1
* **Change:** Made baseline reset epoch-boundary effective via pending-reset fields in `MarketPricingState` and applied at the next epoch boundary before any EMA/price logic.

**Finding 4 — Applied change**

* **Sections changed:** §8.1 (step 10), §6.6
* **Change:** Promoted audit budget mint ordering into the epoch-boundary procedure: storage price update for epoch `e` (if applied) must occur before audit budget mint for `e`, and minted amount recorded afterward.

**Finding 5 — Applied change**

* **Sections changed:** §5.2 (R1), §6.4
* **Change:** Canonically defined “proxy-served” as served sessions with immutable `session.is_proxy==true` derived at open from on-chain authorization, consistent across sections.

**Finding 6 — Applied change**

* **Sections changed:** §6.4, §6.6, §8.5
* **Change:** Added served-vs-PoF mutual exclusion enforcement, added finalization-time `proxy_sessions_served <= sessions_served` invariant handling, and clarified “exactly once on successful transition” semantics.

**Finding 7 — Applied change**

* **Sections changed:** §6.6, §8.6, §8.1, §8.11
* **Change:** Replaced panic/abort on metrics-only invariant failures with deterministic fail-closed behavior (skip affected market updates, freeze affected EMA, emit `INVARIANT_VIOLATION`).

**Finding 8 — Applied change**

* **Sections changed:** §6.5
* **Change:** Required a canonical, stable insufficient-funds error identity (`ErrAuditBudgetInsufficientFunds`) and explicitly forbade string matching for classification.

**Finding 9 — Applied change**

* **Sections changed:** §5.1, §8.5, §8.11
* **Change:** Defined representability rules for raw-rate fields (string-encoded integers; deterministic capping if numeric variants exist) and enforced non-misleading `ref_minted_amt==0` behavior (`util_raw=0`, signal unavailable).

**Finding 10 — Applied change**

* **Sections changed:** §3.4, §7.3
* **Change:** Defined `base_denom` concretely as `Params.retrieval_price_per_blob.Denom`, enforced denom consistency, and added explicit non-degeneracy validation (`month_len_blocks>=1`, `retrieval_price_per_blob.Amount>0`, `storage_price>0`).

**Finding 11 — Applied change**

* **Sections changed:** §8.6, §8.1 (step 5), §8.11
* **Change:** Kept storage EMA freeze behavior but made it explicit, added normative warning, and added `storage_signal_available` to the update event for observability.

**Finding 12 — Applied change**

* **Sections changed:** §8.1, §8.11
* **Change:** Required `EventMarketPricingUpdate` emission even on first enable/baseline capture and defined deterministic handling for missing prior metrics (`MISSING_METRICS`, no EMA/price updates, still initialize current metrics).

**Finding 13 — Applied change**

* **Sections changed:** §11.2
* **Change:** Corrected Threat B analysis to reflect that served-based denominators need not imply variable-fee burn (variable fee may be refunded), so downward manipulation cost may be primarily `base_retrieval_fee` plus resource/opportunity costs.

**Finding 14 — Applied change**

* **Sections changed:** §6.3, §6.6
* **Change:** Added a normative incremental-maintenance requirement for `active_slot_bytes`/`repairing_slot_bytes` aggregates and forbade epoch-boundary scanning of all deals/slots.

**Finding 15 — Applied change**

* **Sections changed:** §2.1, §8.10, §10.2.2
* **Change:** Clarified that bounded-volatility caps apply to controller deltas, while clamp enforcement may snap prices back into range by more than the cap only when prices are already out of clamp range (e.g., governance override/misconfig).
