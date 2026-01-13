# RFC: Market-Equilibrium Pricing (Bounded On-Chain Controller for Storage + Retrieval)

**Status:** Draft / Normative Candidate  
**Scope:** Chain economics (`nilchain/`) — automatic price discovery for storage and retrieval fees  
**Motivation:** `spec.md` Appendix B #5; reduce governance overhead and make pricing responsive to on-chain demand/supply signals without oracles  
**Depends on:**
- `rfcs/rfc-pricing-and-escrow-accounting.md` (**accounting contract; MUST NOT change**)
- `rfcs/rfc-challenge-derivation-and-quotas.md` (epoch definition and deterministic epoch boundaries)
- `rfcs/rfc-mode2-onchain-state.md` (Mode 2 slot status; `ACTIVE` vs `REPAIRING`)
- `notes/mainnet_policy_resolution_jan2026.md` (baseline defaults + calibration signals)
- (Optional, when enabled) Deputy / proof-of-failure / audit budget workflows in Stage 7 (proxy premium, evidence bond/bounty, audit budget)

---

## 0. Executive Summary

NilStore currently treats key price parameters as **static governance-controlled values**:

- `storage_price` (`LegacyDec`, units: base-denom per byte per block) — used for **lock-in storage deposits** at ingest.
- `retrieval_price_per_blob` (`Coin`, units: base-denom per 128 KiB blob) — used for **retrieval variable fees** locked at session open and settled on completion.
- `base_retrieval_fee` (`Coin`) — burned at retrieval session open as an anti-spam sink.

This RFC introduces an **on-chain, deterministic price-discovery controller** that updates the **spot** storage and retrieval prices automatically over time, while preserving the **frozen accounting rules** for escrow, lock-in pricing, and retrieval settlement.

Key properties:

- **No contract changes:** all accounting formulas in `rfcs/rfc-pricing-and-escrow-accounting.md` remain unchanged.
- **No oracles:** updates rely only on **on-chain measurable signals** (audit budget demand/pressure, repair pressure, deputy/proxy usage, proof-of-failure rate).
- **Deterministic + auditable:** updates occur at deterministic epoch boundaries, using explicit integer/fixed-point arithmetic, bounded deltas, and emitted events.
- **Staged rollout:** metrics-only first; then retrieval pricing; then storage pricing; with governance kill-switches.

### 0.1 “Market-equilibrium” clarification (scope/terminology)

Despite the title, v1 is **not** an auction or order-book market. It is a **bounded feedback controller** that adjusts protocol-wide spot prices to converge toward an operating regime where chosen on-chain signals remain near target values.

In this RFC, “equilibrium” means:
- storage-side: audit-budget demand pressure remains near a target band (e.g., 60% of a reference target),
- retrieval-side: deputy/proxy share and proof-of-failure rate remain near target bands (e.g., 1%),
- and prices remain stable (small deltas) when signals are stable.

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
   - No changes to formulas or semantics in `rfcs/rfc-pricing-and-escrow-accounting.md`.
   - Only the *values* of existing price params evolve over time.

2. **Deterministic on-chain computation**
   - Every full node computes identical price updates from identical state.
   - No wall-clock time; only block height / epoch counters.

3. **Auditable dynamics**
   - Store per-epoch pricing inputs and emit explicit events on updates.
   - Provide queries to inspect current prices, last update epoch, baselines, and driving metrics.

4. **Bounded volatility**
   - Hard caps on per-update change.
   - Absolute min/max bounds expressed as multipliers around baselines.

5. **Manipulation resistance by design**
   - Prefer “costly-to-fake” signals (proof-of-failure requires bonds; audit budget demand arises from protocol work).
   - Avoid raw “successful retrieval volume” as a price-increase signal (wash risk).

6. **Incremental deployment**
   - Stage 0: metric collection only (no price changes)
   - Stage 1: retrieval price updates (optional)
   - Stage 2: storage price updates (optional)
   - Stage 3+: refinements

### 2.2 Non-Goals

1. **No per-deal bidding / auctions in v1**
   - Users do not submit bids; providers do not submit asks.
   - Prices are protocol-updated global spot parameters.

2. **No external price feeds**
   - No oracle dependency for fiat price, hardware cost, or off-chain utilization.

3. **No retroactive repricing**
   - Previously locked-in storage deposits are not repriced.
   - Existing retrieval sessions are never repriced (fees locked at open).

4. **No hot/cold split pricing in v1**
   - Existing escrow accounting references a single `storage_price` and `retrieval_price_per_blob`.
   - Introducing hot/cold spot prices would change the frozen accounting contract and is therefore deferred.
   - This RFC includes forward-compatibility notes for a future hot/cold split RFC.

---

## 3. Terminology and Units

### 3.1 Storage price units (on-chain)

- `Params.storage_price` is a `LegacyDec` representing **base-denom units per byte per block**.

Derived “human” price (GiB-month):

- Define `GiB = 2^30 bytes`.
- Define “month” as `Params.month_len_blocks`.

Then:

```
price_GiBMonth = storage_price * GiB * month_len_blocks
```

### 3.2 Retrieval price units (on-chain)

- `Params.retrieval_price_per_blob` is a `Coin` representing base-denom units per **Blob**.
- `BLOB_SIZE = 128 KiB` is a protocol constant.

Derived “human” price (GiB):

Because `GiB / BLOB_SIZE = 8192`:

```
price_GiBRetrieval = retrieval_price_per_blob * 8192
```

### 3.3 Epoch

This RFC treats epoch boundaries as a consensus-critical primitive and therefore defines epoch numbering explicitly.

Let:
- `h` be the current block height as observed by the application at `BeginBlocker`. Nilchain/Cosmos-SDK heights are **1-indexed** (the first block has `h = 1`).
- `L = Params.epoch_len_blocks` (`uint64`, MUST satisfy `L >= 1`).

Define the epoch id:

```
epoch_id = floor((h - 1) / L)
```

This implies:

- The **first** block of epoch `e` has height: `h = e*L + 1`
- The **last** block of epoch `e` has height: `h = (e+1)*L`

An “epoch boundary” in this RFC means the `BeginBlocker` of the first block of an epoch.

Price updates, metric finalization, and controller state transitions occur only at epoch boundaries as defined above.

### 3.4 Denominations

This RFC uses the term **base denom** for the canonical on-chain denomination used by NilStore escrow accounting, storage rent, and retrieval fees.

Normative requirements:

- `Params.retrieval_price_per_blob.Denom` MUST equal the base denom.
- `Params.base_retrieval_fee.Denom` MUST equal the base denom.
- `MarketPricingState.baseline_retrieval_price_per_blob.Denom` MUST equal the base denom.

All `audit_budget_*` coins recorded in `MarketPricingEpochMetrics` MUST use a single denom. v1 REQUIRES that this denom equals the base denom. If the audit budget subsystem uses a different denom, storage price updates MUST remain disabled until this RFC is updated with an explicit deterministic conversion rule.

At runtime, if a denom mismatch is detected, the keeper MUST skip the affected market update and MUST emit `EventMarketPricingUpdate` with `skip_reason=DENOM_MISMATCH`.


## 4. Mechanism Overview

At a high level:

1. **Collect on-chain metrics during an epoch** (counts and sums derived from existing tx handlers and epoch hooks).
2. **Compute normalized “pressure” signals** for:
   - the **storage market** (audit-budget demand pressure and optional repair pressure), and
   - the **retrieval market** (deputy/proxy share and proof-of-failure rate).
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

- **Numerator:** total audit-budget *requested* spend during the epoch (not merely what was successfully paid).
- **Denominator:** a *reference* audit-budget mint computed using the **baseline storage price**, not the current spot `storage_price`.

This isolates the signal from immediate changes to `storage_price` while still tracking real protocol workload demand for audits/evidence.

##### Definitions

Let, for epoch `e`:

- `A_e` = `active_slot_bytes` (slot-responsible bytes in `ACTIVE` state; excludes repairing)
- `E` = `epoch_len_blocks`
- `P0` = `baseline_storage_price` (Dec, base-denom per byte per block)
- `bps` = `Params.audit_budget_bps`
- `cap_bps` = `Params.audit_budget_cap_bps`

Define reference epoch slot rent:

```
ref_epoch_slot_rent = P0 * A_e * E
```

Define reference audit budget mint (Option A sizing, using baseline price):

```
ref_mint_uncapped = ceil( bps / 10_000 * ref_epoch_slot_rent )
ref_mint_cap      = ceil( cap_bps / 10_000 * ref_epoch_slot_rent )
ref_minted        = min(ref_mint_uncapped, ref_mint_cap)
```

Define:
- `requested_e` = total audit budget amount *requested* via the audit-budget spend helper during epoch `e` (Coin, base denom).
- `spent_e`     = total audit budget amount *successfully debited* during epoch `e` (Coin, base denom).
- `denied_e`    = total audit budget amount requested but not paid due to insufficient audit budget balance during epoch `e` (Coin, base denom).

`denied_e` is recorded directly by the audit-budget spend helper (§6.5) and MUST satisfy:

```
requested_e == spent_e + denied_e
```

Define **audit demand utilization** (basis points):

```
audit_demand_util_bps = floor( 10_000 * requested_e / max(1, ref_minted) )
```

Notes:
- `requested_e` is recorded regardless of whether the audit budget had sufficient funds to pay.
- `ref_minted` is recomputed deterministically from state at the epoch boundary.
- This signal does not mechanically change when `storage_price` changes, because it uses `P0` (baseline).

##### Relationship to the accounting contract

This RFC does not change how audit budgets are minted/spent; it only defines a new derived metric for pricing control.

**Implementation requirement:** all protocol flows that attempt to spend from audit budget MUST route through a single keeper helper (e.g., `SpendAuditBudget(ctx, amount, reason)`), which MUST update the per-epoch `requested/spent/denied` counters deterministically.

#### S2) Repair pressure (secondary storage signal; v1 optional)

Mode 2 repairs mark slots `REPAIRING`. Repairing slots indicate supply stress (unhealthy assignments and reduced effective capacity).

Define:

```
repair_pressure_bps = floor( 10_000 * repairing_slot_bytes / max(1, active_slot_bytes + repairing_slot_bytes) )
```

v1 defaults set repair weight to 0 (signal collected but not used for price) unless explicitly enabled by governance in a later stage.

### 5.2 Retrieval market signals

Retrieval pricing reacts to *stress indicators*, not raw retrieval volume.

#### R1) Deputy/proxy-served fraction

A rising proxy share indicates primary-path retrieval stress (provider non-response or routing failures).

**Canonical numerator:** sessions whose *service proof* indicates they were served by deputy/proxy.
**Canonical denominator:** sessions with an accepted *service proof* (i.e., served sessions), independent of user confirmation.

Define:

- `served_e` = number of accepted `MsgSubmitRetrievalSessionProof` (or equivalent) for epoch `e`.
- `proxy_served_e` = subset of `served_e` that are deputy/proxy-served (see §6.4 for canonical tagging).

Then:

```
proxy_fraction_bps = floor( 10_000 * proxy_served_e / max(market_pricing_retrieval_rate_denominator_floor, served_e) )
```

#### R2) Proof-of-failure submission rate

Proof-of-failure submissions indicate non-response pressure. These submissions are bonded (evidence bond), which makes large-scale fabrication costly.

Define:

- `opened_e` = number of accepted `MsgOpenRetrievalSession` during epoch `e`.
- `pof_e` = number of accepted proof-of-failure submissions during epoch `e`, as counted by `OnRetrievalProofOfFailureAccepted` (§6.4).

Then:

```
pof_rate_bps = floor( 10_000 * pof_e / max(market_pricing_retrieval_rate_denominator_floor, opened_e) )
```

#### Low-volume / bootstrap guard for retrieval signals

To prevent unstable behavior during low activity epochs:
- retrieval pressure computation MUST use a denominator floor (param) when converting counts to rates; and
- retrieval price updates MUST be gated on minimum sample sizes.

Details are specified in §8.

---

## 6. State Additions (On-Chain Storage)

### 6.1 `MarketPricingState` (singleton)

A new singleton state object stored under the `nilchain` module.

```proto
message MarketPricingState {
  // Epoch at which baselines were captured (initial enable or explicit reset).
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
}
```

### 6.2 `MarketPricingEpochMetrics` (per-epoch, bounded retention)

Keyed by `epoch_id` and retained for `market_pricing_metrics_retention_epochs`.

**Retention and pruning (normative):**

On finalizing metrics for a completed epoch `e` (i.e., immediately after writing `MarketPricingEpochMetrics{epoch_id=e}`), the keeper MUST prune older metrics entries to bound state growth.

Let:
- `ret = market_pricing_metrics_retention_epochs` (`uint64`, MUST satisfy `ret >= 1`)

Define the first epoch to retain (inclusive) using saturating arithmetic:

- If `e + 1 <= ret`, set `keep_from = 0`.
- Else set `keep_from = e - ret + 1`.

Then delete all stored `MarketPricingEpochMetrics` with:

- `epoch_id < keep_from`

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

**Define `slot_responsible_bytes(deal)` as:**
- the protocol’s canonical “slot bytes” used for quotas/rent (`rfcs/rfc-challenge-derivation-and-quotas.md`), i.e.:
  - Mode 2: bytes attributable to a single RS slot for the deal at `Deal.current_gen`
  - Mode 1: bytes attributable to a provider assignment (full copy)

**Active vs repairing aggregation (epoch snapshot):**

At the epoch boundary, the keeper MUST compute:

- `active_slot_bytes` = sum of `slot_responsible_bytes` over:
  - Mode 2: each `DealSlot` with `status == ACTIVE`, counting one slot per `DealSlot.provider`.
    - MUST NOT include `pending_provider` bytes (pending provider is not accountable until promotion).
    - MUST exclude `status == REPAIRING` slots (per policy: repairing slots are excluded from rent/quota/rewards).
  - Mode 1: each active provider assignment in `Deal.providers[]` (replicas), counting one “slot equivalent” per provider.
    - Mode 1 has no `REPAIRING` state; therefore all active assignments are counted as active.
- `repairing_slot_bytes` = sum of `slot_responsible_bytes` over:
  - Mode 2: each `DealSlot` with `status == REPAIRING` (counting the old accountable provider’s slot; pending provider is not counted).

**Explicit exclusions:**
- Deals with `size_bytes == 0` contribute 0 bytes.
- REPAIRING slots are excluded from `active_slot_bytes` by definition.
- Tombstones are already excluded from `Deal.size_bytes` by definition; therefore they naturally drop out of `slot_responsible_bytes` if it is derived from the committed slab state.

If the implementation does not yet have a single canonical helper for this, it MUST be introduced (e.g., `GetTotalSlotBytesByStatus(ctx) (active, repairing sdkmath.Int)`), and used consistently by:
- audit budget sizing, and
- market pricing metrics.

### 6.4 Canonical retrieval counters (normative)

The controller uses retrieval-side stress signals derived from on-chain retrieval session state transitions. To avoid bias from missing user confirmations, denominators MUST be based on objectively accepted on-chain proofs, not user acknowledgements.

#### Required hook points (normative)

Implementations MUST increment metrics via canonical keeper hook points. Exact message names may differ across branches, but the following semantics are REQUIRED:

- `OnRetrievalSessionOpened(ctx, session_id, opener_addr)`  
  Called exactly once when a retrieval session is created and accepted on-chain.

- `OnRetrievalSessionServiceProofAccepted(ctx, session_id)`  
  Called exactly once when a service proof for the session is accepted on-chain (the session is “served”).

- `OnRetrievalSessionSettled(ctx, session_id)`  
  Called when a user confirmation/settlement message is accepted (monitoring only).

- `OnRetrievalProofOfFailureAccepted(ctx, session_id)`  
  Called exactly once when the *first* proof-of-failure for a session is accepted on-chain.

The nilchain implementation MAY call these hooks directly inside message handlers, but MUST preserve the “exactly once” semantics below.

#### Counting semantics and invariants (normative)

For a given epoch `e`, counters record events that occur in epoch `e`:

- `sessions_opened` MUST count unique `session_id` values for which `OnRetrievalSessionOpened` executed in epoch `e`.
- `sessions_served` MUST count unique `session_id` values for which `OnRetrievalSessionServiceProofAccepted` executed in epoch `e`.
- `sessions_settled` SHOULD count unique `session_id` values for which `OnRetrievalSessionSettled` executed in epoch `e`, but MUST NOT be used as a denominator for retrieval pressure.
- `proofs_of_failure_submitted` MUST count unique `session_id` values for which `OnRetrievalProofOfFailureAccepted` executed in epoch `e`.

Uniqueness rules (consensus-critical):

- Each retrieval session MUST contribute **at most 1** to each of:
  - `sessions_opened`
  - `sessions_served`
  - `sessions_settled`
  - `proofs_of_failure_submitted`
  over the lifetime of that session.
- Implementations MUST enforce uniqueness by session state (e.g., boolean flags) such that replays / duplicate submissions do not increment counters.

Proxy subset invariant (consensus-critical):

- `proxy_sessions_served` MUST count the subset of `sessions_served` in epoch `e` whose `session.is_proxy == true` (see Proxy classification below).
- For each epoch `e`, the keeper MUST ensure:

  - `proxy_sessions_served <= sessions_served`

Note on epoch windows (non-normative): `proofs_of_failure_submitted` in epoch `e` may refer to sessions opened in prior epochs, so it is not guaranteed to be `<= sessions_opened` for the same epoch. Rate computation and pressure mapping therefore MUST be robust to numerator > denominator (see §8.5 and §8.7.2).

#### Proxy classification rule (v1 normative)

The retrieval session state MUST include an immutable boolean `is_proxy` set at session open.

- The open path MUST NOT allow arbitrary callers to set `is_proxy=true`. Instead, the keeper MUST derive `session.is_proxy` deterministically from on-chain deputy/gateway authority state and the opener address.
  - **v1 normative rule:** `session.is_proxy == true` iff the session opener is an **authorized deputy/gateway** at the time of open, as determined by an on-chain registry/allowlist owned by the deputy/gateway authority.
  - If the opener is not authorized, `session.is_proxy` MUST be `false`. If the chain exposes a dedicated “proxy open” message/path, that message/path MUST reject unauthorized callers.
- At service proof acceptance (`OnRetrievalSessionServiceProofAccepted`), the keeper MUST read `session.is_proxy` and MUST increment `proxy_sessions_served` iff it is `true`.
- `session.is_proxy` MUST NOT change after session open; any state transition or message that would mutate it MUST be rejected.

If the session type does not yet contain `is_proxy` **or** there is no on-chain deputy/gateway authorization registry to derive it from, retrieval pricing updates MUST remain disabled (Stage 0 metrics-only) until this field and authorization rule are implemented and enforced.


### 6.5 Audit budget counters (normative)

`audit_budget_requested`, `audit_budget_spent`, and `audit_budget_denied` MUST be derived from a single keeper helper that is the exclusive debit path for the audit budget account.

#### Required helper (normative)

All protocol flows that attempt to spend from audit budget MUST call:

- `SpendAuditBudget(ctx, amount, reason)` (or an equivalent helper with identical semantics)

#### Validation (normative)

For a given `amount`:

- `amount` MUST be strictly positive (`amount > 0`).
- `amount.Denom` MUST equal the base denom (see §3.4).

If validation fails, the helper MUST return an error and MUST NOT mutate any of:

- `audit_budget_requested`
- `audit_budget_spent`
- `audit_budget_denied`
- `audit_budget_spend_attempts`

#### Counter update semantics (normative)

For a validated `amount`, the helper MUST attempt the debit and update per-epoch counters deterministically as follows:

1. **On success**:
   - increment `audit_budget_requested += amount`
   - increment `audit_budget_spent += amount`
   - increment `audit_budget_spend_attempts += 1`

2. **On failure solely due to insufficient audit budget balance**:
   - increment `audit_budget_requested += amount`
   - increment `audit_budget_denied += amount`
   - increment `audit_budget_spend_attempts += 1`

3. **On any other error** (internal module error, invariant violation, etc.):
   - the helper MUST return that error, and
   - MUST NOT mutate any of the counters above.

`audit_budget_denied` MUST represent the amount requested but not paid due to insufficient balance. Implementations MUST ensure, for each epoch `e`:

- `audit_budget_requested == audit_budget_spent + audit_budget_denied`

This invariant is enforced at metrics finalization (§6.6).

`audit_budget_minted` MUST be the actual minted amount for the epoch as recorded by the audit budget subsystem (Option A). The keeper MUST record `audit_budget_minted` exactly once per epoch during epoch initialization (§6.6).


### 6.6 Metrics lifecycle and immutability (normative)

This RFC requires deterministic per-epoch metrics that are unambiguous about which epoch they describe.

Define:
- `e = current_epoch` per §3.3.

`MarketPricingEpochMetrics{epoch_id=e}` describes epoch `e` and has two classes of fields:

- **Snapshot-at-epoch-start fields** (written once at initialization of epoch `e`):
  - `active_slot_bytes`
  - `repairing_slot_bytes`
  - `audit_budget_minted` (the amount minted *for epoch e*)
- **Accumulators-over-the-epoch fields** (updated during epoch `e`):
  - `audit_budget_requested`, `audit_budget_spent`, `audit_budget_denied`, `audit_budget_spend_attempts`
  - `sessions_opened`, `sessions_served`, `sessions_settled`, `proxy_sessions_served`, `proofs_of_failure_submitted`

Lifecycle (normative):

1. **Initialization (epoch boundary into epoch `e`)**
   - The keeper MUST create or reset a **mutable** metrics entry for `epoch_id=e`.
   - It MUST set snapshot-at-epoch-start fields as follows:
     - `active_slot_bytes` and `repairing_slot_bytes` MUST be computed at the epoch boundary using §6.3.
     - `audit_budget_minted` MUST be recorded as the actual amount minted for epoch `e` by the audit budget subsystem (Option A). If minting occurs in the same epoch-boundary hook, recording MUST occur after minting completes. If audit budget minting depends on `Params.storage_price`, minting for epoch `e` MUST run **after** any market-pricing update to `storage_price` for epoch `e` has been applied, so the minted budget corresponds to the storage price effective for that epoch.
   - It MUST zero all accumulator-over-the-epoch fields.

2. **Accumulation (during epoch `e`)**
   - Transaction handlers and keeper helpers MUST update **only** the mutable metrics entry for `epoch_id=e`.
   - Implementations MUST ensure counters are **unique per retrieval session id** where required (see §6.4), and MUST NOT use wall-clock time.

3. **Finalization (epoch boundary into epoch `e+1`)**
   - The metrics entry for `epoch_id=e` becomes **finalized** and MUST NOT be mutated thereafter.
   - During finalization, the keeper MUST assert the following invariant for epoch `e`:

     - `audit_budget_requested == audit_budget_spent + audit_budget_denied`

     and all three coins MUST have identical denoms.

   - If the invariant check fails, the chain MUST treat it as a consensus-critical invariant violation (panic / abort block execution).

4. **Controller input**
   - EMA and price updates applied at the epoch boundary into epoch `e+1` MUST use only finalized metrics for epoch `e` (never partial-epoch metrics).

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

- `market_pricing_update_interval_epochs >= 1`
- `market_pricing_min_epochs_before_update >= 0` (always true for `uint64`; retained for clarity)
- `market_pricing_ema_alpha_bps <= 10_000`
- `market_pricing_max_delta_bps_per_month <= 10_000`

- Denoms:
  - `Params.retrieval_price_per_blob.Denom` MUST be non-empty.
  - `Params.base_retrieval_fee.Denom` MUST equal `Params.retrieval_price_per_blob.Denom` (base denom; see §3.4).

- Clamp multipliers:
  - `0 < market_pricing_storage_floor_mult <= 1`
  - `market_pricing_storage_ceil_mult >= 1`
  - `0 < market_pricing_retrieval_floor_mult <= 1`
  - `market_pricing_retrieval_ceil_mult >= 1`

- All target/alert bps params MUST satisfy `<= 10_000`.

- For each banded signal:
  - `alert_low <= target <= alert_high`

- Retrieval denominator parameters:
  - `market_pricing_retrieval_rate_denominator_floor >= 1`
  - `market_pricing_min_retrieval_denominator_per_epoch` is a `uint64` (may be 0 to disable sample-size gating), but SHOULD be bounded (recommended `<= 100_000`).

- Metrics retention:
  - `market_pricing_metrics_retention_epochs >= 1`
  - `market_pricing_metrics_retention_epochs` SHOULD be bounded (recommended `<= 365`).

- `market_pricing_min_audit_spend_attempts_per_epoch` is a `uint64` (may be 0 to disable sample-size gating), but SHOULD be bounded (recommended `<= 100_000`).


### 7.4 Defaults (safe posture)

- Market pricing is **off by default**:
  - `market_pricing_enabled = false`
- Even if enabled, automatic updates are **paused by default**:
  - `market_pricing_updates_paused = true`
- Per-market updates are **disabled by default**:
  - `market_pricing_update_storage_price = false`
  - `market_pricing_update_retrieval_price = false`

- v1 defaults prevent below-baseline retrieval price decreases unless explicitly enabled by governance:
  - `market_pricing_retrieval_floor_mult = 1.00` (retrieval price cannot fall below the captured baseline unless governance lowers the floor multiplier and resets baselines)

This enables Stage 0 “metrics only” without further code toggles:
- Set `market_pricing_enabled=true` while keeping `market_pricing_updates_paused=true`.

---

## 8. Update Cadence, Bounds, and Exact Algorithm

This section is normative and implementation-oriented.

### 8.1 When updates happen (epoch boundary rule)

Define:

- `e = current_epoch` computed at the epoch boundary using §3.3.
- `update_period_epochs = market_pricing_update_interval_epochs` (>= 1).

**Determinism rule:** prices for epoch `e` MUST be computed only from metrics observed during the fully-completed previous epoch `e-1`, and applied exactly once at the epoch boundary into epoch `e` (i.e., `BeginBlocker` of the first block of epoch `e`).

This ensures:
- every tx in epoch `e` observes a single consistent price vector, and
- there is no intra-epoch price drift.

#### Epoch-boundary procedure (normative)

At the epoch boundary into epoch `e`:

1. **Disabled fast-path**
   - If `Params.market_pricing_enabled == false`:
     - the keeper MUST delete `MarketPricingState` if it exists, and
     - the keeper MUST delete any mutable “current epoch” metrics entry (if it exists), and
     - the keeper MUST NOT collect metrics or apply any price/EMA updates for this RFC.
     - Return.

2. **Initialization on first enable**
   - If `Params.market_pricing_enabled == true` and `MarketPricingState` does not exist:
     - The keeper MUST initialize `MarketPricingState` as specified in §8.3, with:
       - `baseline_epoch = e`
       - `last_update_epoch = e`
     - The keeper MUST initialize the mutable metrics entry for epoch `e` per §6.6.
     - No price update is applied at this boundary (the controller enters warm-up).
     - Return.

3. **Finalize previous epoch metrics**
   - Let `prev = e - 1` (only defined if `e > 0`).
   - If `e > 0`, the keeper MUST finalize metrics for epoch `prev` per §6.6 (i.e., the metrics entry for `prev` becomes immutable).
   - Immediately after finalizing epoch `prev`, the keeper MUST prune metrics according to §6.2.

4. **Compute pressures and advance EMA (even if updates are paused)**
   - Using only finalized metrics for epoch `prev`, the keeper MUST:
     - compute storage and retrieval pressure inputs (§5, §8.5–§8.7), and
     - advance both EMAs (§8.8), except as noted below for denom mismatches.
   - The keeper MUST treat the storage signal as unavailable for this epoch boundary and MUST NOT modify `state.storage_pressure_ema_microbps` if any storage gating condition fails (§8.6), including denom mismatch, `ref_minted == 0`, `active_slot_bytes == 0`, or insufficient audit spend attempts.
   - EMA advancement MUST occur whenever finalized metrics are available and `market_pricing_enabled == true`, regardless of whether price updates are paused or interval-gated. (This avoids “latent jumps” when unpausing.) Storage signals that fail gating are treated as unavailable as specified above.

5. **Global update eligibility gates**
   - Price updates at the epoch boundary into `e` are globally eligible only if all of the following hold:
     - `Params.market_pricing_updates_paused == false`
     - warm-up is satisfied (§8.2)
     - the per-update cap was computed successfully (§8.4) (i.e., no required overflow check failed)
     - interval gate is satisfied:

       ```
       (e - state.last_update_epoch) >= update_period_epochs
       ```

     - If the interval gate is not satisfied, `state.last_update_epoch` MUST NOT change.

6. **Per-market updates**
   - Storage price update for epoch `e` MAY be applied only if:
     - global eligibility holds, and
     - `Params.market_pricing_update_storage_price == true`, and
     - storage gating holds (§8.6).
   - Retrieval price update for epoch `e` MAY be applied only if:
     - global eligibility holds, and
     - `Params.market_pricing_update_retrieval_price == true`, and
     - retrieval gating holds (§8.5).

7. **`last_update_epoch` semantics**
   - If at least one of (storage update, retrieval update) is applied at this boundary, the keeper MUST set:

     - `state.last_update_epoch = e`

   - Otherwise, `state.last_update_epoch` MUST NOT change.

8. **Initialize metrics for epoch `e`**
   - The keeper MUST initialize the mutable metrics entry for epoch `e` per §6.6 (snapshot + zeroed counters).


### 8.2 Warm-up counter semantics

`market_pricing_min_epochs_before_update` MUST be counted from `MarketPricingState.baseline_epoch`:

- Let `baseline_epoch` be the epoch when baselines were captured (initial enable or explicit reset).
- Automatic updates MUST NOT be applied unless:

```
current_epoch >= baseline_epoch + market_pricing_min_epochs_before_update
```

This resolves ambiguity about whether the warm-up is from genesis vs enablement.

### 8.3 Enable/disable behavior (baseline capture + reset)

#### State existence as the enable-edge detector (normative)

To avoid reliance on an in-memory “previous param value”, this RFC defines the enable edge deterministically using on-chain state:

- When `Params.market_pricing_enabled == false`, the keeper MUST delete `MarketPricingState` and any mutable “current epoch” metrics entry at the next epoch boundary (§8.1, step 1).
- Therefore, when pricing is later enabled, the *absence* of `MarketPricingState` is the canonical indicator that baselines MUST be captured.

#### Baseline capture on enable (normative)

At the epoch boundary into epoch `e`, if `Params.market_pricing_enabled == true` and `MarketPricingState` is absent, the keeper MUST initialize `MarketPricingState` as follows:

- `baseline_epoch = e`
- `last_update_epoch = e`
- `baseline_storage_price = Params.storage_price`
- `baseline_retrieval_price_per_blob = Params.retrieval_price_per_blob`
- `month_len_blocks_snapshot = Params.month_len_blocks`
- `storage_pressure_ema_microbps = 0`
- `retrieval_pressure_ema_microbps = 0`

This initialization is the only time baselines are captured automatically.

#### Mid-epoch governance changes (normative)

Governance MAY toggle params mid-epoch. This RFC defines:

- Baseline capture and metrics initialization occur only at epoch boundaries. If `market_pricing_enabled` becomes true mid-epoch, the controller MUST remain inactive until the next epoch boundary, at which point baselines are captured for that next epoch.
- If `market_pricing_enabled` becomes false mid-epoch, the controller MUST stop collecting metrics immediately. Any partial-epoch mutable metrics for that epoch MUST be discarded at the next epoch boundary as part of the disabled fast-path (§8.1, step 1).

#### Explicit reset

Baseline reset via `MsgResetMarketPricingBaselines` is specified in §10.2.1 and MUST override any existing state.


### 8.4 Per-update max delta (monthly cap converted using a snapshot)

Let:

- `blocks_per_update = Params.epoch_len_blocks * update_period_epochs`
- `month_len = state.month_len_blocks_snapshot`

`blocks_per_update` and the subsequent cap computation are consensus-critical and MUST be overflow-safe.

#### Computation (normative)

1. Compute `blocks_per_update` using checked `uint64` multiplication.
   - If overflow occurs, the keeper MUST treat this as a non-fatal configuration error for the epoch:
     - skip applying **all** price updates at this boundary, and
     - emit `EventMarketPricingUpdate` with `skip_reason=OVERFLOW`.

2. Compute the raw per-update cap (basis points):

```
max_delta_bps_per_update_raw =
  ceil_div_u64( market_pricing_max_delta_bps_per_month * blocks_per_update, max(1, month_len) )
```

where `ceil_div_u64(a,b) = (a + b - 1) / b` for `b > 0`, using checked arithmetic.

3. Clamp to a safe bound:

```
max_delta_bps_per_update = min(max_delta_bps_per_update_raw, 10_000)
```

This ensures the per-update cap is never greater than 100% (10,000 bps). (Decreases are additionally clamped to avoid `k = 0`; see §8.9 and §8.10.2.)

**Volatility freeze rule (normative):** `state.month_len_blocks_snapshot` MUST NOT change unless baselines are reset (by enable initialization or `MsgResetMarketPricingBaselines`). This prevents unrelated governance changes to `Params.month_len_blocks` from abruptly changing price volatility.


### 8.5 Retrieval update gating and rate denominator floor

Retrieval gating is evaluated in §8.1 (step 6) and applies only to the retrieval market. Global gates (enabled/paused/warm-up/interval) are handled separately in §8.1.

Define:
- `min_den = market_pricing_min_retrieval_denominator_per_epoch`
- `floor_den = market_pricing_retrieval_rate_denominator_floor`

#### Signal availability (normative)

For epoch `prev` metrics:

- The proxy-fraction signal is **available** iff:

  - `sessions_served >= min_den`

- The proof-of-failure signal is **available** iff:

  - `sessions_opened >= min_den`

Retrieval price update MUST be skipped if **both** signals are unavailable (i.e., `sessions_served < min_den` AND `sessions_opened < min_den`).

If exactly one of the two signals is available, the retrieval market MAY still be updated using the available signal; the unavailable signal contributes neutral pressure (`0 bps`) in §8.7.2.

#### Rate denominator floor (normative)

When computing rates, denominators MUST be floored to reduce low-volume spikes:

```
denom_opened = max(sessions_opened, floor_den)
denom_served = max(sessions_served, floor_den)
```

#### Rate computation (normative)

Rates MUST be computed using overflow-safe integer arithmetic (e.g., `sdkmath.Int` / `big.Int` intermediates). Implementations MUST NOT allow `10_000 * count` to overflow a native integer type.

Compute:

```
proxy_fraction_bps_raw = floor(10_000 * proxy_sessions_served / denom_served)
pof_rate_bps_raw       = floor(10_000 * proofs_of_failure_submitted / denom_opened)
```

Notes:
- `proxy_fraction_bps_raw` is guaranteed to be `<= 10_000` if `proxy_sessions_served <= sessions_served` (§6.4).
- `pof_rate_bps_raw` MAY exceed `10_000` if proofs-of-failure are submitted for sessions opened in prior epochs. The controller MUST therefore saturate/clamp in the pressure mapping step (§8.7.2).


### 8.6 Storage update gating

Storage gating is evaluated in §8.1 (step 6) and applies only to the storage market. Global gates (enabled/paused/warm-up/interval) are handled separately in §8.1.

Storage price update MUST be skipped unless all of the following hold for epoch `prev` metrics:

- Sample size:
  - `audit_budget_spend_attempts >= market_pricing_min_audit_spend_attempts_per_epoch`
- Audit budget subsystem is active:
  - audit budget Option A is enabled (`Params.audit_budget_bps > 0`)
- Non-degenerate supply snapshot:
  - `active_slot_bytes > 0`
- Reference mint is well-defined:
  - the reference minted amount computed in §5.1 for epoch `prev` satisfies `ref_minted > 0`
- Denoms are consistent:
  - all `audit_budget_*` coins in metrics for epoch `prev` MUST use the base denom (§3.4)
  - otherwise, the storage update MUST be skipped with `skip_reason=DENOM_MISMATCH` and the storage EMA MUST NOT be advanced at this boundary (treat the storage signal as unavailable).

If any condition fails, the storage pressure may still be computed for observability, but storage price MUST NOT be updated.


### 8.7 Pressure normalization (integer bps)

For both markets, compute a normalized pressure as **signed basis points**:

- `p_bps ∈ [-10_000, +10_000]`
  - `+10_000` means “maximum upward pressure”
  - `0` means neutral
  - `-10_000` means “maximum downward pressure”

All computations in this section MUST be done using integer arithmetic with explicit truncation rules to avoid consensus divergence.

#### 8.7.1 Storage pressure (v1)

Compute `audit_demand_util_bps_raw` per §5.1.

Let:
- `u_raw = audit_demand_util_bps_raw`
- `t = market_pricing_target_audit_util_bps`
- `lo = market_pricing_alert_audit_util_low_bps`
- `hi = market_pricing_alert_audit_util_high_bps`

For pressure mapping only (not for observability), define:

```
u = clamp_u64(u_raw, lo, hi)
```

This clamps the signal into the alert band so the mapping saturates cleanly and avoids any risk of `int64` overflow.

Define signed error:

```
err = int64(u) - int64(t)
```

Compute:

- if `err >= 0` (util above target):

```
den = max(1, hi - t)
p_storage_bps = min(10_000, 10_000 * err / den)
```

- else (util below target):

```
den = max(1, t - lo)
p_storage_bps = max(-10_000, 10_000 * err / den)   // err is negative => p negative
```

Optional repair pressure can be integrated later with explicit weights; v1 defaults to audit-only.


#### 8.7.2 Retrieval pressure (v1)

Compute the raw rates per §8.5:

- `proxy_fraction_bps_raw`
- `pof_rate_bps_raw`

Also compute signal availability per §8.5:

- `proxy_available = (sessions_served >= market_pricing_min_retrieval_denominator_per_epoch)`
- `pof_available   = (sessions_opened >= market_pricing_min_retrieval_denominator_per_epoch)`

If a signal is unavailable, its pressure contribution MUST be treated as neutral (`0 bps`) for this epoch.

This allows retrieval pressure (and therefore retrieval price updates) to be driven by whichever signal(s) are available in the epoch, while treating missing signals conservatively as neutral.

##### Proxy fraction pressure

If `proxy_available == false`:

- set `p_proxy_bps = 0`.

Else:

Let:
- `u_raw = proxy_fraction_bps_raw`
- `t = market_pricing_target_proxy_frac_bps`
- `lo = market_pricing_alert_proxy_frac_low_bps`
- `hi = market_pricing_alert_proxy_frac_high_bps`

For pressure mapping only, define:

```
u = clamp_u64(u_raw, lo, hi)
```

Compute `p_proxy_bps` using the same piecewise linear mapping as storage (target-centered):

- if `u >= t`:

```
den = max(1, hi - t)
p_proxy_bps = min(10_000, 10_000 * int64(u - t) / int64(den))
```

- else:

```
den = max(1, t - lo)
p_proxy_bps = max(-10_000, -10_000 * int64(t - u) / int64(den))
```

##### Proof-of-failure pressure

If `pof_available == false`:

- set `p_pof_bps = 0`.

Else:

Let:
- `u_raw = pof_rate_bps_raw` (may exceed `10_000`)
- `t = market_pricing_target_pof_rate_bps`
- `lo = market_pricing_alert_pof_rate_low_bps`
- `hi = market_pricing_alert_pof_rate_high_bps`

For pressure mapping only, define:

```
u = clamp_u64(u_raw, lo, hi)
```

Compute `p_pof_bps` analogously.

##### Combine retrieval pressure

Combine:

```
p_retrieval_bps = max(p_proxy_bps, p_pof_bps)
```

Rationale:
- any strong positive stress signal dominates (price increases),
- negative pressure applies only when available signals are below target, and
- unavailable signals contribute `0` (neutral), which is intentionally conservative.


### 8.8 EMA update (deterministic fixed-point)

Define:
- `alpha_bps = market_pricing_ema_alpha_bps` (0..10_000)
- `S = 1_000_000` (micro-bps per bps)

Let:
- `ema_prev = state.*_pressure_ema_microbps` (int64)
- `p = p_*_bps` (int64, bps in [-10_000, +10_000])

Compute:

```
ema_next = (ema_prev*(10_000 - alpha_bps) + (p*S)*alpha_bps) / 10_000
```

**Rounding rule (normative):** signed integer division MUST truncate toward zero (Go default). This is deterministic.

After computing `ema_next`, the keeper MUST clamp it into the representable range:

```
ema_next = clamp_int64(-10_000*S, +10_000*S, ema_next)
```

This clamp is defensive: it ensures later computations remain bounded even if upstream signals are misconfigured.


### 8.9 Convert EMA to capped per-update delta (integer bps)

Given:
- `ema_next` in micro-bps,
- `max_delta = max_delta_bps_per_update` (uint64, clamped to `<= 10_000` in §8.4),
- `S = 1_000_000`,

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
- `sp = Params.storage_price` (`LegacyDec`, units: base-denom per byte per block)
- `m = 10_000 + delta_storage_bps` (int64)

By §8.9, `delta_storage_bps >= -9_999`, therefore `m >= 1` and the multiplier is strictly positive.

Compute (normative):

```
sp_next = sp.MulInt64(m).QuoInt64(10_000)
```

This specifies the exact operation order and avoids any float conversions.

Clamp:

- `floor = baseline_storage_price * market_pricing_storage_floor_mult`
- `ceil  = baseline_storage_price * market_pricing_storage_ceil_mult`

Then:

```
sp_next = clamp_dec(floor, ceil, sp_next)
```


#### 8.10.2 Retrieval price per blob

Let:
- `rp = Params.retrieval_price_per_blob` (`Coin`)
- `rp_amt = rp.Amount` (`sdk.Int`, non-negative)
- `k = 10_000 + delta_retrieval_bps` (int64)

By §8.9, `delta_retrieval_bps >= -9_999`, therefore:

- `1 <= k <= 20_000`

#### Denom consistency (normative)

If `rp.Denom != state.baseline_retrieval_price_per_blob.Denom`, the keeper MUST skip the retrieval update and emit `EventMarketPricingUpdate` with `skip_reason=DENOM_MISMATCH`.

#### Update (normative)

Compute:

- if `delta_retrieval_bps >= 0`: **ceil division** (avoid undercharging on increases)

```
rp_next = ceil_div_int( rp_amt * k, 10_000 )
```

- else: **floor division** (ensure decreases actually decrease when possible)

```
rp_next = floor_div_int( rp_amt * k, 10_000 )
```

`rp_amt * k` MUST be computed using `sdk.Int` / `big.Int` intermediates (no overflow).

#### Clamp bounds (baseline-relative)

Let `baseline_rp = state.baseline_retrieval_price_per_blob`.

Compute bounds using deterministic `LegacyDec` math:

- `floor_amt = ceil( baseline_rp.Amount * market_pricing_retrieval_floor_mult )` (round up)
- `ceil_amt  = floor( baseline_rp.Amount * market_pricing_retrieval_ceil_mult )` (round down)

Then clamp:

```
rp_next = clamp_int(floor_amt, ceil_amt, rp_next)
```

Finally set:

```
Params.retrieval_price_per_blob = Coin{Denom: rp.Denom, Amount: rp_next}
```


#### 8.10.3 Deterministic integer primitives and helper definitions

All consensus-critical arithmetic in this RFC MUST be implemented using integer or fixed-point types with deterministic rounding. Implementations MUST NOT use floats.

Definitions (normative):

- For `uint64` with `b > 0`:

  ```
  ceil_div_u64(a, b) = (a + b - 1) / b
  ```

  This MUST be implemented with overflow checks on `a + b - 1`.

- For non-negative `sdk.Int` (arbitrary-precision integer) with `b > 0`:

  - `floor_div_int(a, b)` is integer division rounding down.
  - `ceil_div_int(a, b)` is:

    ```
    ceil_div_int(a, b) = floor_div_int(a + (b - 1), b)
    ```

    where the `+ (b - 1)` is done in `sdk.Int` space.

- `trunc_toward_zero(x / y)` for signed integers is defined as the language-default signed integer division that truncates toward zero (Go semantics).

- `clamp_u64(x, lo, hi)` returns:
  - `lo` if `x < lo`
  - `hi` if `x > hi`
  - otherwise `x`

- `clamp_int64(lo, hi, x)` returns:
  - `lo` if `x < lo`
  - `hi` if `x > hi`
  - otherwise `x`

- `clamp_int(lo, hi, x)` for `sdk.Int` returns:
  - `lo` if `x < lo`
  - `hi` if `x > hi`
  - otherwise `x`

- `clamp_dec(lo, hi, x)` for `LegacyDec` returns:
  - `lo` if `x < lo`
  - `hi` if `x > hi`
  - otherwise `x`


### 8.11 Events and observability

On each epoch boundary where market pricing is enabled, the keeper MUST emit events sufficient to audit both the inputs and the update decision.

#### Metrics event (normative)

After finalizing metrics for epoch `prev = e-1` (when `e > 0`), emit:

- `EventMarketPricingEpochMetrics` containing:
  - `epoch_id = prev`
  - the full persisted `MarketPricingEpochMetrics{epoch_id=prev}`

#### Update decision event (normative)

At the same epoch boundary, emit exactly one `EventMarketPricingUpdate` containing (at minimum):

- `epoch_id = e`
- `prev_epoch_id = prev` (if `e > 0`)
- `baseline_epoch`
- `last_update_epoch_before`, `last_update_epoch_after`
- Global gate flags:
  - `updates_paused`
  - `warmup_satisfied`
  - `interval_satisfied`
  - `overflow` (true if any required overflow check failed this epoch)
- `max_delta_bps_per_update`

Per-market decision fields:

- `applied_storage` (bool), `skip_reason_storage` (string/enum)
- `applied_retrieval` (bool), `skip_reason_retrieval` (string/enum)

Recommended `skip_reason_*` values:

- `APPLIED`
- `DISABLED`
- `PAUSED`
- `WARMUP`
- `INTERVAL`
- `INSUFFICIENT_SAMPLE`
- `DENOM_MISMATCH`
- `ZERO_REF_MINT`
- `OVERFLOW`
- `MISSING_METRICS`

Computed signals and controller state:

- raw rates (as computed in §5 and §8.5):
  - `audit_demand_util_bps_raw`
  - `proxy_fraction_bps_raw`
  - `pof_rate_bps_raw`
- pressure values:
  - `p_storage_bps`
  - `p_retrieval_bps`
- EMA values:
  - previous and next EMA micro-bps for storage and retrieval
- deltas:
  - `delta_storage_bps`, `delta_retrieval_bps`

Prices:

- previous and next prices for storage and retrieval (if not applied, next MUST equal previous).


## 9. Interaction With Escrow, Spend Windows, and Retrieval Settlement

This RFC intentionally preserves all invariants in `rfcs/rfc-pricing-and-escrow-accounting.md`.

### 9.1 Storage lock-in deposits

Lock-in storage deposit at `MsgUpdateDealContent*` remains:

```
storage_cost = ceil(storage_price * delta_bytes * duration_blocks)
```

Dynamic pricing effects:
- Only **new delta bytes** are charged at the current spot `storage_price`.
- Previously committed bytes are not repriced.

### 9.2 Retrieval session open and settlement

Retrieval fees remain:

- At open:
  - burn `base_retrieval_fee` (non-refundable)
  - lock `variable_fee = retrieval_price_per_blob * blob_count`
- At completion:
  - burn `ceil(variable_fee * retrieval_burn_bps / 10_000)`
  - pay the remainder to the provider
- On cancel/expiry:
  - refund only the locked variable fee to `Deal.escrow_balance`

Dynamic pricing effects:
- Only **new sessions** use the new spot `retrieval_price_per_blob`.
- Existing sessions are unaffected because they store `session.locked_fee`.

### 9.3 Spend windows and escrow predictability

- Elasticity spend caps use `base_stripe_cost` and the spend window; this RFC does not modify them.
- To avoid destabilizing user escrow UX:
  - price updates are epoch-based (no intra-epoch drift),
  - per-update deltas are capped,
  - clamp bounds prevent runaway.

Wallets/UIs can compute the worst-case next-epoch drift as `max_delta_bps_per_update` and recommend an escrow buffer accordingly.

---

## 10. Governance Control Surface

Governance (module authority) retains full control and can override or disable market pricing.

### 10.1 Standard controls (params)

Authority MAY:
- enable metrics + controller via `market_pricing_enabled`
- pause/resume updates via `market_pricing_updates_paused`
- enable each market independently:
  - `market_pricing_update_retrieval_price`
  - `market_pricing_update_storage_price`
- tune cadence, caps, clamps, targets, and sample-size guards

### 10.2 Baseline reset semantics (required for safe governance overrides)

Baselines are critical because:
- clamps are defined as multipliers around baselines, and
- storage controller normalization uses the baseline storage price.

Therefore, this RFC defines an explicit baseline reset mechanism.

#### 10.2.1 Authority message: `MsgResetMarketPricingBaselines`

Add a new authority-only message:

- `MsgResetMarketPricingBaselines { string authority }`

On success, it MUST:
- set `MarketPricingState.baseline_epoch = current_epoch`
- set `MarketPricingState.last_update_epoch = current_epoch`
- set `baseline_storage_price = Params.storage_price`
- set `baseline_retrieval_price_per_blob = Params.retrieval_price_per_blob`
- set `month_len_blocks_snapshot = Params.month_len_blocks`
- reset EMAs to zero

#### 10.2.2 Interaction with governance spot price overrides

- Governance can set `storage_price` and/or `retrieval_price_per_blob` at any time via params update.
- If the new spot price lies outside the current baseline-relative clamps, the next automatic update will clamp the price back into range.

**Operational rule:** if governance intends a lasting step-change to the price regime (or volatility regime via month length), governance SHOULD:
1. set the new spot prices (and any clamp multipliers if needed), then
2. call `MsgResetMarketPricingBaselines`.

This prevents “snap back” to old baselines.

### 10.3 Emergency kill switch

Setting `market_pricing_updates_paused = true` MUST immediately stop automatic updates at the next epoch boundary without affecting:
- escrow balances,
- retrieval settlement, or
- in-flight sessions.

---

## 11. Security and Manipulation Analysis

### 11.1 Storage-side manipulation

**Threat:** attacker tries to manipulate `audit_budget_requested` to move `storage_price`.

Observations/mitigations:
- `audit_budget_requested` increments only when protocol subsystems attempt audit-budget spends (audit retrieval traffic, evidence incentives if funded from audit budget).
- Most such spends are gated by:
  - bonded evidence (proof-of-failure) and/or
  - protocol-driven audit scheduling.
- Even if an attacker can create additional demand, the controller is bounded:
  - EMA smoothing limits impact of short spikes,
  - per-update max delta caps price movement,
  - clamps cap long-term movement without governance action.

**Implementation note:** the RFC requires `requested/spent/denied` to be recorded in the audit budget helper, making manipulation analysis auditable by inspecting spend reasons.

### 11.2 Retrieval wash / Sybil manipulation

**Threat A (upward):** attacker inflates retrieval stress to push price up.

- Controller does not use raw successful volume; it uses proxy share and proof-of-failure rate.
- Proof-of-failure is bonded (`evidence_bond`), and repeated submissions can be penalized; therefore large-scale fabrication is costly.

**Threat B (downward):** attacker tries to push price down by generating many “healthy” served sessions (low proxy and low pof rates).

- Downward pressure is conservative (`max(p_proxy, p_pof)`), and update deltas are capped.
- Any attempt to create large numbers of served sessions still pays:
  - `base_retrieval_fee` (burned), and
  - the variable-fee burn cut on settled sessions.
  This makes large-scale manipulation expensive even if colluding with a provider to recoup payouts.

### 11.3 Low-volume epochs / noisy denominators

Risk:
- If denominators are small, single proxy/pof events can produce extreme rates.

Mitigations:
- Retrieval updates are gated on a minimum denominator.
- Rate denominators are floored for computation (stable floor), preventing 100% spikes from small N.

### 11.4 Oscillation and controller stability

Risks:
- oscillation if signals lag or if multiple controllers interact.

Mitigations:
- EMA smoothing with explicit alpha
- capped delta per update
- warm-up gating after baseline reset/enable
- price-invariant storage-side normalization prevents denominator feedback loops from dominating dynamics

---

## 12. Backward Compatibility and Migration

### 12.1 Backward compatibility

- New params and new state are additive.
- Default values keep market pricing disabled and updates paused.
- Existing deals and sessions are unaffected:
  - storage deposits already paid remain in escrow
  - retrieval session fees already locked remain unchanged

### 12.2 Migration strategy

On upgrade that introduces this RFC:

1. Add new params with safe defaults (`market_pricing_enabled=false`).
2. Add new stores for `MarketPricingState` and `MarketPricingEpochMetrics`.
3. If market pricing is enabled at genesis (not recommended), initialize `MarketPricingState` at genesis epoch using genesis params.

When governance later enables market pricing:
- baselines are captured at the enable edge as defined in §8.3.

---

## 13. Implementation Plan (Staged MVP)

### Stage 0 — Metrics only (no price updates)

- Add `MarketPricingState` and `MarketPricingEpochMetrics`.
- Wire tx hooks:
  - `MsgOpenRetrievalSession` → `sessions_opened++`
  - `MsgSubmitRetrievalSessionProof` → `sessions_served++` and proxy tagging
  - `MsgConfirmRetrievalSession` → `sessions_settled++` (monitoring-only)
  - proof-of-failure submission → `proofs_of_failure_submitted++`
  - audit budget spend helper → requested/spent/denied counters
- Epoch boundary hook snapshots slot bytes and minted amount, writes metrics, emits event.
- Add queries:
  - `QueryMarketPricingState`
  - `QueryMarketPricingMetrics(epoch_id)` and/or window query

**Exit gate:** unit tests + e2e that assert metrics determinism across nodes.

### Stage 1 — Retrieval price updates

- Enable retrieval controller under explicit param:
  - `market_pricing_enabled=true`
  - `market_pricing_updates_paused=false`
  - `market_pricing_update_retrieval_price=true`
- Keep storage controller disabled:
  - `market_pricing_update_storage_price=false`

**Exit gate:** e2e in a multi-SP setup where proxy retrieval + proof-of-failure paths are exercised and retrieval price changes are observed across epochs.

### Stage 2 — Storage price updates

- Enable storage controller:
  - `market_pricing_update_storage_price=true`
- Ensure audit budget Option A and audit budget spend helper instrumentation are active.

**Exit gate:** e2e where audit budget demand is generated (e.g., audit debt spending) and `storage_price` adjusts without oscillation or snapback artifacts.

### Stage 3 — Refinements (optional)

- Add repair pressure weighting with an explicit param and simulations.
- Add additional manipulation-resistant signals if needed (e.g., audit debt backlog) once on-chain.

---

## 14. Testing Plan

### 14.1 Unit tests

1. **Deterministic fixed-point math**
   - EMA update matches reference integer arithmetic.
   - delta_bps conversion is deterministic with truncation toward zero.

2. **Volatility cap freeze**
   - changing `Params.month_len_blocks` does not change `max_delta_bps_per_update` until baseline reset.
   - baseline reset updates `month_len_blocks_snapshot`.

3. **Gating correctness**
   - warm-up counted from baseline_epoch.
   - retrieval updates require denominators; low-volume epochs skip updates.
   - storage updates require `audit_budget_spend_attempts >= market_pricing_min_audit_spend_attempts_per_epoch`, `ref_minted > 0`, and `active_slot_bytes > 0`.

4. **Clamp enforcement**
   - prices never exceed baseline-relative min/max after update.

5. **Rounding**
   - retrieval price uses ceil on increases and floor on decreases.
   - baseline-derived floor/ceil bounds apply correct rounding directions.

6. **Baseline reset semantics**
   - governance override without reset can be clamped back.
   - reset updates baselines and clears EMA.

### 14.2 End-to-end tests

Extend econ e2e scripts:

- run with small `epoch_len_blocks`
- create deal, ingest content, open/serve/settle retrieval sessions
- exercise proxy and pof submission paths
- verify:
  - price params change only at epoch boundaries
  - escrow accounting invariants hold (same formulas)
  - in-flight sessions settle correctly regardless of later price changes
  - baseline reset message behaves as specified

### 14.3 Simulation gate (stability / adversarial)

Add a deterministic sim harness (fixed RNG seed) that runs for `N` epochs and simulates:

- varying proxy share
- varying proof-of-failure rates
- varying audit-budget requested demand

Assert invariants:

- prices remain within clamps
- max delta per update respected
- controller converges under constant signals (EMA stabilizes, deltas go to 0)
- no oscillation beyond configured bounds for reasonable alpha/caps

---

## 15. Calibration Signals and Alert Thresholds

Expose these metrics (events + queries):

- `price_GiBMonth` (derived)
- `price_GiBRetrieval` (derived)
- `audit_demand_util_bps` (requested vs reference mint)
- `audit_budget_denied` (budget binding indicator)
- `proxy_fraction_bps`
- `pof_rate_bps`
- `repair_pressure_bps` (even if not used for price)

Recommended alert thresholds (consistent with policy notes):

- Audit demand utilization:
  - alert if `> 95%` (sustained demand pressure)
  - alert if `< 10%` sustained (overprovisioned or unused subsystem)
- Audit budget denied amount:
  - alert if non-zero for `> X` epochs (budget binding)
- Deputy/proxy share:
  - target `< 1%`, alert `> 5%`
- Proof-of-failure rate:
  - target `< 1%`, alert `> 3%`
- Clamp saturation:
  - alert if a price hits floor or ceiling for > `X` epochs (controller saturated; governance tuning needed)

---

## 16. Open Questions and Dependencies

If any of these are unresolved on the target branch, they MUST be treated as implementation blockers or explicitly deferred.

1. **Canonical proof-of-failure acceptance hook**
   - The retrieval subsystem MUST have a single canonical state transition for “PoF accepted” that can enforce *first-PoF-only* semantics per session and invoke `OnRetrievalProofOfFailureAccepted` (§6.4).

2. **Canonical proxy classification (`is_proxy`)**
   - Retrieval sessions MUST include an immutable `is_proxy` flag derived deterministically at session open from an on-chain deputy/gateway authorization registry (§6.4).
   - Until this is implemented and enforced, retrieval pricing updates MUST remain disabled.

3. **Audit budget denomination**
   - v1 REQUIRES audit budget mint/spend to use the base denom (§3.4). If the audit budget subsystem uses a different denom, storage pricing updates MUST remain disabled until a deterministic conversion rule is specified.

4. **Exact `slot_responsible_bytes` computation**
   - The chain MUST provide a single canonical helper used by quotas, audit budget sizing, and this RFC (§6.3).
   - Ensure Mode 2 `REPAIRING` exclusions match policy.

5. **Epoch-boundary ordering**
   - If audit budget minting depends on `Params.storage_price`, the epoch-boundary ordering MUST ensure that market-pricing updates to `storage_price` (if applied) occur before minting for that epoch (§6.6).

6. **Module wiring / keeper patterns**
   - Confirm the epoch-boundary hook location and ordering for: finalize metrics → prune → compute pressure/EMA → (optional) apply updates → initialize next epoch metrics (per §8.1 and §6.6).
