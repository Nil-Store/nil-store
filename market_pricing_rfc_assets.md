```AGENTS_MAINNET_PARITY.md
# AGENTS_MAINNET_PARITY.md

## 0) Header

This file is the **Codex-executable** execution punch list for completing remaining **Mainnet econ/security parity** work (plus the devnet/testnet launch-critical pieces) across `nilchain/`, `nil_gateway/`, and `nil_p2p/`. It is derived from the staged checklist and the frozen/approved economic and repair policies; tasks are written to be **low ambiguity**, **test-gated**, and small enough to land in **1–3 commits** each.

### How to run locally

```bash
# (1) Start the multi-SP devnet stack (router + multiple providers)
./scripts/run_devnet_alpha_multi_sp.sh start

# (2) Run the CI-style multi-SP gateway retrieval regression (start → test → stop)
./scripts/ci_e2e_gateway_retrieval_multi_sp.sh

# (3) Run the econ lifecycle E2E (create deal → upload/commit → retrieve)
./scripts/e2e_lifecycle.sh

# (4) Run chain unit tests (params/keeper logic)
go test ./nilchain/...
```

---

## 1) Progress Log

Append-only. Do not edit prior entries.

**Template:**

* `YYYY-MM-DD | TASK <ID> | <status> | <notes> | <commit> | <PR link (optional)>`

* `2026-01-09 | TASK P0-PARAMS-001 | in progress | start params/proto/defaults/overrides work | - | -`
* `2026-01-09 | TASK P0-PARAMS-001 | done | params proto/defaults/validation + overrides + tests | d35eba4 | -`
* `2026-01-09 | TASK P0-ECON-LOCKIN-001 | in progress | start lock-in pricing on UpdateDealContent | - | -`
* `2026-01-09 | TASK P0-ECON-LOCKIN-001 | done | lock-in deposit accounting + tests | 7ff3382 | -`
* `2026-01-09 | TASK P0-ECON-SPEND-002 | in progress | start deterministic spend window + elasticity debits | - | -`
* `2026-01-09 | TASK P0-ECON-SPEND-002 | done | spend window reset + cap enforcement tests | cb6c0c9 | -`
* `2026-01-09 | TASK P0-RETRIEVAL-FEES-001 | in progress | start session open fee burn + lock | - | -`
* `2026-01-09 | TASK P0-RETRIEVAL-FEES-001 | done | burn base fee + lock variable fee on open | 65d0ba7 | -`
* `2026-01-09 | TASK P0-RETRIEVAL-SETTLE-002 | in progress | start settlement burn/payout + cancel/expiry paths | - | -`
* `2026-01-09 | TASK P0-RETRIEVAL-SETTLE-002 | done | settle burn/payout and expiry refund handling + tests | f4912fa | -`
* `2026-01-09 | TASK P0-ECON-E2E-001 | in progress | start econ accounting regression script | - | -`
* `2026-01-09 | TASK P0-ECON-E2E-001 | done | econ parity E2E + confirm CLI + tx polling | 898f141 | -`
* `2026-01-09 | TASK P0-QUOTAS-001 | in progress | start deterministic challenge derivation + repairing exclusions | - | -`
* `2026-01-09 | TASK P0-QUOTAS-001 | done | enforce synthetic challenge exclusions + derivation tests | 1adb0b9 | -`
* `2026-01-09 | TASK P0-QUOTAS-002 | in progress | start quota accounting + synthetic tracking cleanup | - | -`
* `2026-01-09 | TASK P0-QUOTAS-002 | done | prune quota accounting + add quota clamp/dedup tests | 67cab32 | -`
* `2026-01-09 | TASK P0-QUOTAS-SIM-003 | in progress | start adversarial sim gate for challenge derivation | - | -`
* `2026-01-09 | TASK P0-QUOTAS-SIM-003 | done | add deterministic challenge derivation sim test | 62492e7 | -`
* `2026-01-09 | TASK P0-HEALTH-001 | in progress | start HealthState updates + eviction thresholds | - | -`
* `2026-01-09 | TASK P0-HEALTH-001 | done | add health state tracking + hot/cold eviction tests | 5f63784 | -`
* `2026-01-09 | TASK P0-MODE2-MBB-001 | in progress | start make-before-break repair state machine | - | -`
* `2026-01-09 | TASK P0-MODE2-MBB-001 | done | gate slot repair promotion on quota readiness + tests | 0746542 | -`
* `2026-01-09 | TASK P0-HEALTH-002 | in progress | start health observability queries + events | - | -`
* `2026-01-09 | TASK P0-HEALTH-002 | done | add health queries/events + tests | 06fb3f9 | -`
* `2026-01-10 | TASK P0-BOND-001 | in progress | start provider bond baseline | - | -`
* `2026-01-10 | TASK P0-BOND-001 | done | provider bond state + min bond gating + tests | b3cbab2 | -`
* `2026-01-10 | TASK P0-BOND-002 | in progress | start assignment collateral lock + unbonding guard | - | -`
* `2026-01-10 | TASK P0-BOND-002 | done | lock assignment collateral + bond availability tests | fc5e9c8 | -`
* `2026-01-10 | TASK P0-REPAIR-001 | in progress | start replacement selection cooldown/cap | - | -`
* `2026-01-10 | TASK P0-REPAIR-001 | done | deterministic replacement selection nonce + cooldown/cap enforcement | 858abf4 | -`
* `2026-01-10 | TASK P0-MODE2-ROUTING-002 | in progress | start Mode2 routing guard against REPAIRING slots | - | -`
* `2026-01-10 | TASK P0-MODE2-ROUTING-002 | done | avoid routing reads to repairing slots + active-only selection | 7a030e2 | -`
* `2026-01-10 | TASK P0-MODE2-REWARD-003 | in progress | start excluding repairing slots from rewards/challenges | - | -`
* `2026-01-10 | TASK P0-MODE2-REWARD-003 | done | suppress rewards for repairing slots + reward eligibility test | 23ea526 | -`
* `2026-01-10 | TASK P0-REPAIR-E2E-002 | in progress | start multi-SP repair e2e script | - | -`
* `2026-01-10 | TASK P0-REPAIR-E2E-002 | blocked | repair e2e still failing: pending provider fetch 502; slab catch-up incomplete; session proof mismatch on multi-blob attempts | - | -`
* `2026-01-10 | TASK P0-EVIDENCE-001 | in progress | start hard-fault evidence slash/jail + repair wiring | - | -`
* `2026-01-10 | TASK P0-EVIDENCE-001 | done | hard-fault evidence penalties + jail tracking + tests | fc890ef | -`
* `2026-01-10 | TASK P0-EVIDENCE-E2E-002 | in progress | start evidence e2e script for wrong-data slash/jail/repair | - | -`
* `2026-01-10 | TASK P0-EVIDENCE-E2E-002 | done | add wrong-data evidence E2E script | 21d013a | -`
* `2026-01-10 | TASK P0-DEPUTY-001 | in progress | start proxy retrieval premium lock/payout accounting | - | -`
* `2026-01-10 | TASK P0-DEPUTY-001 | done | lock proxy premium fees + payout on deputy success | 4f6a07a | -`
* `2026-01-10 | TASK P0-DEPUTY-002 | in progress | start proof-of-failure aggregation + bond/bounty + expiry handling | - | -`
* `2026-01-10 | TASK P0-DEPUTY-002 | done | proof-of-failure submit + aggregation + expiry handling + tests | 550f123 | -`
* `2026-01-10 | TASK P0-AUDIT-001 | in progress | start audit budget minting + carryover logic | - | -`
* `2026-01-10 | TASK P0-AUDIT-001 | done | audit budget minting + carryover/expiry + tests | c318180 | -`
* `2026-01-10 | TASK P0-AUDIT-002 | in progress | start audit debt tracking + budget spend helper | - | -`
* `2026-01-10 | TASK P0-AUDIT-002 | done | audit debt state/query + audit budget spend helper + tests | 2368aa9 | -`
* `2026-01-10 | TASK P0-DEPUTY-003 | in progress | start AskForProxy request/response + gateway proxy fallback integration | - | -`
* `2026-01-10 | TASK P0-DEPUTY-003 | done | AskForProxy http bridge + gateway proxy fallback + proof-of-failure CLI/tests | 6106e7a | -`
* `2026-01-10 | TASK P0-DEPUTY-E2E-002 | in progress | start ghosting-provider deputy E2E gate | - | -`
* `2026-01-10 | TASK P0-DEPUTY-E2E-002 | done | deputy ghosting CI script + premium/escrow assertions | cb6bdea | -`
* `2026-01-10 | TASK P0-REPAIR-E2E-002 | in progress | resume repair e2e debugging: pending provider fetch + proof mismatch | - | -`
* `2026-01-10 | TASK P0-REPAIR-E2E-002 | done | stabilize repair E2E + mode2 reconstruction and proof submission flow | da2c9b7 | -`
* `2026-01-10 | TASK P1-REPAIR-OVERRIDE-001 | in progress | start authority-only repair override posture | - | -`
* `2026-01-10 | TASK P1-REPAIR-OVERRIDE-001 | done | authority-only repair override param + msg + tests | 50e64a4 | -`
* `2026-01-10 | TASK P1-CREDITS-001 | in progress | add organic credit cap and dedupe coverage | - | -`
* `2026-01-10 | TASK P1-CREDITS-001 | done | add organic credit dedupe + cap tests | 3d9e848 | -`

---

## 2) Working Rules

* **One task at a time:** do not start a new task until the current task’s **Test gate** has been run.
* **No aggressive git commands:** do not run `git clean`, `git reset --hard`, or similar destructive commands.
* **Run test gate before marking done:** a task cannot be marked **done** without running its specified test gate(s).
* **Update this file as you go:**

  * When you begin a task, set **Status → in progress** and add a Progress Log entry.
  * When you finish a task, set **Status → done** and add a Progress Log entry with the commit hash.
  * Keep the progress log append-only. Never delete tasks; only add new tasks if required (use `TASK P0-...` / `TASK P1-...`).
* If this repo uses multiple git remotes, follow the repo’s agent protocol (see `AGENTS.md`).

---

## 3) Task Board

Organized by Stage 0–7 (per `MAINNET_ECON_PARITY_CHECKLIST.md`). Each task must meet its DoD and pass its test gate before being marked done.

---

### Stage 0 — Policy freeze → params + interfaces

#### TASK P0-PARAMS-001 — Encode final policy params (B1/B2/B4/B5/B6) + validation + devnet override plumbing

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/proto/nilchain/nilchain/v1/params.proto`, `nilchain/x/nilchain/types/`, `nilchain/x/nilchain/keeper/`, `scripts/run_devnet_alpha_multi_sp.sh`
* **Depends on:** (none)
* **Context:**

  * `notes/mainnet_policy_resolution_jan2026.md` — **Final defaults** to encode, including:

    * `base_retrieval_fee`: dev/test `0.0001 NIL`, mainnet `0.0002 NIL`
    * Audit budget Option A: `audit_budget_bps`, `audit_budget_cap_bps`, carryover ≤2 epochs, and `epoch_slot_rent` formula
    * Credits phase-in: devnet caps = 0; testnet hot/cold caps 25%/10%; mainnet caps = 0 at launch → later enable
    * Trusted override posture: dev/test enabled **if implemented**; mainnet disabled by default (governance-emergency only)
  * `nilchain/proto/nilchain/nilchain/v1/params.proto` currently ends at field `evict_after_missed_epochs = 17;` (see existing file).
  * `scripts/run_devnet_alpha_multi_sp.sh` has a python `overrides = {...}` block that currently overrides: `month_len_blocks`, `epoch_len_blocks`, `quota_*`, `credit_cap_bps`, `evict_after_missed_epochs`.
  * `rfcs/rfc-challenge-derivation-and-quotas.md` §4–§5 (quota + credits), §7 (state additions).
* **Work plan:**

  1. Extend `Params` in `nilchain/proto/nilchain/nilchain/v1/params.proto` by adding new fields **after** the existing ones (use new field numbers ≥ 18). Do not renumber existing fields.
  2. Add params required to encode the approved policy surfaces:

     * **B1 Slashing/jailing ladder + non-response windowing**

       * `slash_invalid_proof_bps`, `slash_wrong_data_bps`, `slash_nonresponse_bps`
       * `jail_invalid_proof_epochs`, `jail_wrong_data_epochs`, `jail_nonresponse_epochs`
       * `nonresponse_threshold`, `nonresponse_window_epochs`
       * `max_strikes_before_global_jail`, `strike_window_epochs`
       * `evict_after_missed_epochs_hot`, `evict_after_missed_epochs_cold`
     * **B2 Bonding**

       * `min_provider_bond` (Coin), `bond_months` (uint64), `provider_unbonding_blocks` (uint64)
     * **B4 Replacement**

       * `replacement_cooldown_blocks`, `repair_attempts_cap`, `repair_attempt_window_blocks`
     * **B5 Deputy + audit budget**

       * `premium_bps`
       * `evidence_bond`, `failure_bounty`
       * `evidence_bond_burn_bps_on_expiry`
       * `proof_of_failure_ttl_epochs` (default = `nonresponse_window_epochs` unless explicitly set)
       * `audit_budget_bps`, `audit_budget_cap_bps`, `audit_budget_carryover_epochs`
     * **B6 Credits phase-in**

       * `credit_cap_bps_hot`, `credit_cap_bps_cold`
  3. Maintain backwards compatibility where needed:

     * Keep `credit_cap_bps` and `evict_after_missed_epochs` as legacy defaults, but update keeper code to **prefer hot/cold split** if present.
  4. Regenerate protobuf bindings using the repo’s existing proto generation workflow (do not invent new tooling).
  5. Update `Params` defaults and validation (`nilchain/x/nilchain/types/`):

     * Enforce all `*_bps <= 10_000`
     * Enforce epoch/month lengths > 0
     * Enforce `nonresponse_threshold >= 1`, `nonresponse_window_epochs >= 1`
     * Enforce coin denoms match `sdk.DefaultBondDenom` and are non-negative
     * Encode **approved defaults** (per `notes/mainnet_policy_resolution_jan2026.md`), especially:

       * `base_retrieval_fee` defaults (dev/test `0.0001`, mainnet `0.0002` in NIL units; expressed in base denom units)
       * Audit budget defaults: dev/test `audit_budget_bps=200`, `audit_budget_cap_bps=500`, carryover `2`; mainnet `100/200/2`
       * Credit cap defaults: devnet hot/cold = `0/0`; testnet `2500/1000`; mainnet launch `0/0`
  6. Update `scripts/run_devnet_alpha_multi_sp.sh` to support overriding the new params via env vars (follow existing `NIL_*` pattern). Ensure overrides write **stringified** values for uint64 fields (Cosmos JSON convention).
  7. Add/extend unit tests that:

     * parse default params
     * validate params
     * confirm the presence of new fields and that validation rejects obvious invalid values (bps > 10_000, negative coins, etc.).
* **Artifacts:**

  * `nilchain/proto/nilchain/nilchain/v1/params.proto`
  * generated proto outputs under `nilchain/` (wherever this repo keeps `*.pb.go`)
  * `nilchain/x/nilchain/types/` (params defaults + validation)
  * `nilchain/x/nilchain/keeper/` (param accessors, if required)
  * `scripts/run_devnet_alpha_multi_sp.sh`
  * `nilchain/` unit tests for params validation
* **DoD:**

  * New params are present in proto and in generated Go bindings.
  * `Params.Validate()` (or equivalent) enforces the new constraints deterministically.
  * `DefaultParams()` (or equivalent) includes the new fields with values consistent with `notes/mainnet_policy_resolution_jan2026.md` (network-specific differences are documented/handled via genesis/script overrides).
  * Devnet script can override the new params through environment variables without breaking existing overrides.
  * Unit tests cover (at minimum) `base_retrieval_fee`, audit budget bps/cap/carryover, hot/cold eviction thresholds, and nonresponse threshold/window validation.
* **Test gate:**

  * `go test ./nilchain/...`
  * (Optional smoke) `./scripts/run_devnet_alpha_multi_sp.sh start` and ensure chain boots with updated genesis overrides.
* **Notes / gotchas:**

  * Coin params are integers in base denom units; represent `0.0001 NIL` / `0.0002 NIL` consistently with existing denom precision in this repo.
  * Jail durations should be **epoch-based in policy** but stored/enforced as `jail_end_height` (block height) to avoid ambiguity if epoch length changes later.
  * Avoid non-deterministic map iteration when serializing or hashing params/state.

---

### Stage 1 — Storage lock-in pricing + escrow accounting

#### TASK P0-ECON-LOCKIN-001 — Implement pay-at-ingest lock-in pricing on UpdateDealContent* (storage cost deposit)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, (Deal update message handlers), `rfcs/rfc-pricing-and-escrow-accounting.md`
* **Depends on:** `P0-PARAMS-001`
* **Context:**

  * `rfcs/rfc-pricing-and-escrow-accounting.md` §4.1 “Storage lock-in pricing”
  * Deal fields used by the RFC: `size_bytes`, `start_block`, `end_block`, `escrow_balance`
* **Work plan:**

  1. Locate the chain handler(s) that finalize content ingestion (`UpdateDealContent*` variants, including any EVM intent path) and route them through **one shared accounting function**.
  2. Implement **delta-only** charging:

     * `delta_bytes = max(0, new_size_bytes - old_size_bytes)`
     * no repricing and no refunds on shrink.
  3. Compute storage lock-in deposit:

     * `duration_blocks = deal.end_block - deal.start_block`
     * `storage_cost = ceil(storage_price * delta_bytes * duration_blocks)`
  4. Transfer `storage_cost` from deal owner → `nilchain` module account.
  5. Update bookkeeping:

     * `deal.escrow_balance += storage_cost`
  6. Emit an event with `deal_id`, `delta_bytes`, `storage_cost`, `duration_blocks`.
  7. Add unit tests covering:

     * increasing size charges once and is deterministic
     * shrinking size charges 0 (no refund)
     * same size charges 0 (idempotency)
     * ceil rounding behavior on boundary cases
* **Artifacts:**

  * `nilchain/x/nilchain/keeper/` (deal update handlers + shared accounting)
  * `nilchain/x/nilchain/types/` (if deal fields/helpers need updates)
  * new/updated keeper unit tests
* **DoD:**

  * Storage lock-in deposit is charged **only on growth** and is deterministic across runs.
  * The module account receives the deposited funds and `deal.escrow_balance` increases by the same amount.
  * Unit tests validate correct arithmetic and idempotency.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Use deterministic rounding (`ceil`) for `Dec * uint64 * uint64`; avoid float conversions.
  * Protect against overflow by using the repo’s canonical math types for large multiplications.

#### TASK P0-ECON-SPEND-002 — Deterministic spend window reset + deterministic elasticity debits

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, `rfcs/rfc-pricing-and-escrow-accounting.md`
* **Depends on:** `P0-PARAMS-001`, `P0-ECON-LOCKIN-001`
* **Context:**

  * `rfcs/rfc-pricing-and-escrow-accounting.md` §6.1–§6.2 “Elasticity caps” and spend windows
  * `params.proto` includes `base_stripe_cost` and `month_len_blocks`
* **Work plan:**

  1. Identify the elasticity trigger path (e.g., `MsgSignalSaturation` or equivalent) that currently enforces `max_monthly_spend` but does not do deterministic debits.
  2. Ensure deal state includes spend window fields:

     * `spend_window_start_height`
     * `spend_window_spent`
     * If missing, add them to the deal type (and wire migrations if needed).
  3. Implement deterministic window reset:

     * If `height >= spend_window_start_height + month_len_blocks`, set `spend_window_start_height = height` and `spend_window_spent = 0`.
  4. Compute elasticity cost deterministically (RFC): `cost = base_stripe_cost * delta_replication` (or the repo’s equivalent unit).
  5. Enforce caps and available escrow:

     * fail if `spend_window_spent + cost > max_monthly_spend`
     * fail if `escrow_balance < cost`
  6. Apply deterministic debit:

     * `escrow_balance -= cost`
     * `spend_window_spent += cost`
  7. Emit an event with `deal_id`, `delta_replication`, `cost`, and window fields.
  8. Add unit tests for:

     * reset boundary correctness
     * cap enforcement
     * escrow debit correctness
* **Artifacts:**

  * `nilchain/x/nilchain/keeper/` (elasticity handler)
  * `nilchain/x/nilchain/types/` (deal state)
  * new/updated unit tests
* **DoD:**

  * Elasticity actions deterministically debit escrow and track window spend.
  * Window reset is height-based and deterministic.
  * Unit tests cover debit, reset, and cap failure cases.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Ensure the debit is replay-safe (same tx cannot be applied twice).
  * Avoid any time-based logic; use heights only.

---

### Stage 2 — Retrieval session economics

#### TASK P0-RETRIEVAL-FEES-001 — Enforce session open: burn base fee + lock variable fee; reject insufficient escrow

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, `rfcs/rfc-pricing-and-escrow-accounting.md`
* **Depends on:** `P0-PARAMS-001`, `P0-ECON-LOCKIN-001`
* **Context:**

  * `rfcs/rfc-pricing-and-escrow-accounting.md` §5.1 “Open session”
  * Approved defaults include **lower** `base_retrieval_fee` (dev/test `0.0001 NIL`, mainnet `0.0002 NIL`)
* **Work plan:**

  1. Locate `MsgOpenRetrievalSession` (and any equivalent path) and ensure it has access to: `deal_id`, `manifest_root`, `start_blob`, `blob_count`, `provider`.
  2. Compute fees deterministically:

     * `base_fee = params.base_retrieval_fee`
     * `variable_fee = params.retrieval_price_per_blob * blob_count`
     * `total = base_fee + variable_fee`
  3. Validate:

     * `manifest_root` matches pinned `deal.manifest_root`
     * `deal.escrow_balance >= total`
  4. Accounting at open (RFC):

     * Burn `base_fee` from the **module account** (non-refundable).
     * Decrement `deal.escrow_balance -= (base_fee + variable_fee)`.
     * Store `session.locked_fee = variable_fee`.
  5. Add events for session open with fee breakdown.
  6. Add unit tests:

     * insufficient escrow fails
     * base fee is burned at open (and not refunded later)
     * locked_fee equals computed variable_fee and is stored once
* **Artifacts:**

  * `nilchain/x/nilchain/keeper/` (session open handler)
  * `nilchain/x/nilchain/types/` (session state)
  * unit tests for retrieval open
* **DoD:**

  * Session open burns the base fee and locks the variable fee per RFC.
  * Insufficient escrow prevents session creation.
  * Unit tests confirm burn/lock behavior and determinism.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Burning must reduce actual supply / module balance (not just bookkeeping).
  * Ensure denom consistency for base and variable fees.

#### TASK P0-RETRIEVAL-SETTLE-002 — Enforce settlement: burn cut + provider payout; cancel/expiry refunds locked fee only

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, `rfcs/rfc-pricing-and-escrow-accounting.md`
* **Depends on:** `P0-RETRIEVAL-FEES-001`
* **Context:**

  * `rfcs/rfc-pricing-and-escrow-accounting.md` §5.2 “Completion” and §5.3 “Cancel/expire”
  * `retrieval_burn_bps` defines the burn cut on completion (dev/test lower than mainnet, per policy defaults)
* **Work plan:**

  1. Implement completion path (`MsgConfirmRetrievalSession` or equivalent):

     * verify session is OPEN and proof is valid
  2. Compute settlement:

     * `burn_cut = ceil(locked_fee * retrieval_burn_bps / 10_000)`
     * `payout = locked_fee - burn_cut`
  3. Apply settlement:

     * burn `burn_cut` from module account
     * transfer `payout` from module → provider
     * mark session COMPLETED and zero out locked amount (or mark “settled”)
  4. Implement cancel/expiry path:

     * refund only `locked_fee` back into `deal.escrow_balance`
     * base fee remains burned and is never refunded
     * mark session CANCELLED/EXPIRED and zero out locked amount
  5. Add unit tests:

     * open→complete burn/payout math
     * open→cancel refunds locked only
     * expiry path is deterministic and idempotent
* **Artifacts:**

  * `nilchain/x/nilchain/keeper/` (settlement handlers)
  * `nilchain/x/nilchain/types/` (session state)
  * unit tests for settlement/cancel/expiry
* **DoD:**

  * Completion burns the configured cut and pays provider the remainder.
  * Cancel/expiry refunds only the locked variable portion.
  * Unit tests validate accounting and idempotency.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Avoid double-settlement; enforce a strict session state machine.
  * Use deterministic rounding (`ceil`) for burn cut.

#### TASK P0-ECON-E2E-001 — End-to-end econ accounting regression suite (escrow + burns + payouts + refunds)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `scripts/`, `tests/` (if present), chain queries/CLI flows
* **Depends on:** `P0-ECON-LOCKIN-001`, `P0-ECON-SPEND-002`, `P0-RETRIEVAL-FEES-001`, `P0-RETRIEVAL-SETTLE-002`
* **Context:**

  * `scripts/e2e_lifecycle.sh` is the baseline lifecycle E2E.
  * `scripts/ci_e2e_gateway_retrieval_multi_sp.sh` is the CI-style script pattern (start/stop stack, deterministic asserts).
  * `MAINNET_GAP_TRACKER.md` P0-ECON-001 requires E2E coverage for escrow and session economics.
* **Work plan:**

  1. Extend `scripts/e2e_lifecycle.sh` **or** add a new econ-specific script in `scripts/` that:

     * creates a deal
     * uploads/commits (triggers lock-in deposit)
     * opens a retrieval session and completes it
     * opens another session and cancels/expires it
  2. Add deterministic assertions by querying chain state:

     * `deal.escrow_balance` changes as expected:

       * increases on ingest by `storage_cost`
       * decreases on open by `base_fee + variable_fee`
       * increases on cancel by `locked_fee` refund
     * module account balance reflects burns/payouts
  3. Support at least two param regimes via existing env overrides (fast blocks / cheap fees for CI).
  4. Make failures actionable: print before/after values and computed expected deltas.
* **Artifacts:**

  * `scripts/e2e_lifecycle.sh` and/or a new `scripts/e2e_econ_parity.sh`
  * optionally a `scripts/ci_*` wrapper matching existing CI style
* **DoD:**

  * Script exits non-zero on mismatch.
  * Script validates escrow delta, base fee burn, burn cut, provider payout, and cancel refund.
  * Script is stable (bounded retries/timeouts; no infinite waits).
* **Test gate:**

  * `./scripts/e2e_lifecycle.sh` (or the new econ script)
  * (If added) the new `./scripts/ci_*` wrapper
* **Notes / gotchas:**

  * Prefer polling for state transitions over fixed sleeps.
  * Ensure the script uses the same denom expected by the chain (devnet uses `sdk.DefaultBondDenom`, typically `stake`).

---

### Stage 3 — Deterministic challenge derivation + quotas + synthetic fill

#### TASK P0-QUOTAS-001 — Deterministic challenge derivation (Mode1 + Mode2) with REPAIRING exclusions

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, `rfcs/rfc-challenge-derivation-and-quotas.md`
* **Depends on:** `P0-PARAMS-001`
* **Context:**

  * `rfcs/rfc-challenge-derivation-and-quotas.md` §3.1–§3.4 (epoch randomness and derivation)
  * Policy requirement: **REPAIRING slots excluded** from synthetic challenges
* **Work plan:**

  1. Implement epoch randomness `R_e` derivation exactly per RFC:

     * `R_e = SHA256("nilstore/epoch/v1" || chain_id || epoch_id || block_hash(epoch_start_height))`
     * store `block_hash(epoch_start_height)` deterministically at epoch boundary if required.
  2. Implement derivation functions for:

     * Mode1: `(provider, deal_id, i) → mdu_index, blob_index`
     * Mode2: `(slot, current_gen, deal_id, i) → leaf_index → (row, mdu_ordinal) → mdu_index, blob_index`
  3. Enforce exclusions:

     * do not target metadata MDUs: `mdu_index >= meta_mdus` where `meta_mdus = 1 + witness_mdus`
     * skip Mode2 slots with status != ACTIVE (REPAIRING excluded)
  4. Add membership-check helper for “is this proof one of the derived synthetic challenges for epoch e?”
  5. Add unit tests for determinism and exclusions.
* **Artifacts:**

  * `nilchain/x/nilchain/keeper/` derivation helpers
  * `nilchain/x/nilchain/types/` helper funcs/constants
  * unit tests for determinism/exclusion
* **DoD:**

  * Challenge derivation matches RFC structure and is deterministic.
  * Unit tests prove metadata is never targeted and REPAIRING slots are excluded.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Avoid iterating over maps to build challenge sets; use sorted lists for any provider/slot enumeration.
  * Keep derivation code pure and easily testable.

#### TASK P0-QUOTAS-002 — Quota accounting + synthetic fill tracking + end-of-epoch evaluation (quota shortfall is HealthState-only)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, `rfcs/rfc-challenge-derivation-and-quotas.md`
* **Depends on:** `P0-QUOTAS-001`
* **Context:**

  * `rfcs/rfc-challenge-derivation-and-quotas.md` §4.1–§4.2 (quota computation), §6.2–§6.3 (enforcement)
  * Policy decision: **quota shortfall does not slash**; it drives HealthState only
* **Work plan:**

  1. Implement quota computation per RFC:

     * compute `slot_bytes` for each assignment
     * compute `quota_blobs` using `quota_bps_per_epoch_*`, min/max clamps
  2. Track synthetic satisfaction deterministically:

     * maintain a `SyntheticSeen` uniqueness set (challenge-id keyed) to prevent double-counting
     * increment `synthetic_satisfied_blobs` only for valid, in-set proofs
  3. At epoch end, evaluate:

     * if `synthetic_satisfied_blobs < quota_blobs` → record quota miss (soft failure) and emit event
     * do not slash for quota shortfall
  4. Provide a hook or callout for Stage 4 HealthState updates (soft failure path).
  5. Add unit tests for quota calc, clamps, dedup, and epoch evaluation.
* **Artifacts:**

  * keeper quota accounting + storage keys
  * unit tests for quota accounting
* **DoD:**

  * Quota miss results in recorded soft failure and event emission.
  * No slashing occurs from quota shortfall.
  * Uniqueness set prevents double counting.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Ensure `SyntheticSeen` state is pruned/TTL’d to prevent unbounded growth.
  * Avoid heavy per-epoch O(N * quota_blobs) loops; respect `quota_max_blobs`.

#### TASK P1-CREDITS-001 — Organic retrieval credits: accrual rules, caps, and phase-in defaults (devnet off, testnet limited, mainnet off at launch)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, `rfcs/rfc-challenge-derivation-and-quotas.md`
* **Depends on:** `P0-QUOTAS-002`, `P0-RETRIEVAL-SETTLE-002`, `P0-PARAMS-001`
* **Context:**

  * `rfcs/rfc-challenge-derivation-and-quotas.md` §5 (organic credits)
  * Phase-in (policy): devnet caps = 0; testnet caps hot/cold = 25%/10%; mainnet launch caps = 0
* **Work plan:**

  1. Implement credit-id derivation and uniqueness set (`CreditSeen`) per RFC §5.1–§5.2 with TTL pruning.
  2. Accrue credits on successful “organic” retrieval proofs/receipts.
  3. Apply caps using hot/cold split params:

     * `credit_cap_hot = ceil(quota_blobs * credit_cap_bps_hot / 10_000)`
     * `credit_cap_cold = ceil(quota_blobs * credit_cap_bps_cold / 10_000)`
  4. Integrate into synthetic fill:

     * `synthetic_needed = max(0, quota_blobs - min(credits_unique, credit_cap))`
  5. Add unit tests for uniqueness, caps, and synthetic reduction.
* **Artifacts:**

  * keeper credit accounting
  * credit uniqueness TTL/pruning
  * unit tests
* **DoD:**

  * Credits accrue only once per credit-id (uniqueness enforced).
  * Caps apply correctly for hot/cold deals.
  * Devnet defaults yield no quota reduction (caps = 0), but accounting code exists.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Credit state growth must be bounded; TTL pruning is mandatory.
  * Ensure credits cannot reduce quota beyond policy caps in any mode.

#### TASK P0-QUOTAS-SIM-003 — Adversarial sim / determinism gate for anti-grind properties

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `scripts/`, `performance/` (if present), or `nilchain/` property tests
* **Depends on:** `P0-QUOTAS-001`
* **Context:**

  * `MAINNET_ECON_PARITY_CHECKLIST.md` Stage 3 requires an adversarial sim test gate.
* **Work plan:**

  1. Add a deterministic property test (or a small sim harness) that generates:

     * multiple epochs
     * multiple deals (Mode1 + Mode2)
     * multiple assignments and slot statuses (including REPAIRING)
       and asserts invariants (bounds, exclusions, determinism).
  2. Provide a single command to run it (either `go test ...` or a wrapper script in `scripts/`).
  3. Keep runtime short and deterministic (fixed RNG seed).
* **Artifacts:**

  * sim test code (under `nilchain/` or a script wrapper)
  * documentation comment in the sim explaining invariants
* **DoD:**

  * Sim gate exists, runs deterministically, and fails on invariant violation.
* **Test gate:**

  * The command you add (e.g., `go test ./nilchain/... -run TestChallengeDerivationSim` or `./scripts/...`)
* **Notes / gotchas:**

  * Do not assert probabilistic distribution properties with tight thresholds; focus on correctness invariants.

---

### Stage 4 — HealthState + eviction curve

#### TASK P0-HEALTH-001 — HealthState per (deal, provider/slot): updates from hard/soft failures + hot/cold eviction thresholds

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`
* **Depends on:** `P0-QUOTAS-002`, `P0-PARAMS-001`
* **Context:**

  * `MAINNET_ECON_PARITY_CHECKLIST.md` Stage 4
  * Policy: eviction thresholds differ by service class: hot vs cold (params `evict_after_missed_epochs_hot/cold`)
* **Work plan:**

  1. Define/store HealthState keyed by `(deal_id, assignment)` where assignment is:

     * Mode1: provider address
     * Mode2: `(slot_index)` or `(provider, slot_index)` as required by existing schema
  2. Implement soft failure update hook from quota evaluation:

     * increment missed epochs
     * emit event
  3. Implement hard failure update hook (fed by Stage 6 evidence outcomes):

     * mark as hard-failed
     * emit event
  4. Implement eviction trigger:

     * on soft failure, compare against hot/cold threshold and trigger repair start (Stage 5) once
     * on hard failure, trigger immediate repair start (Stage 5)
  5. Unit tests for:

     * hot/cold threshold differences (recommended: hot=2, cold=6 per policy)
     * single-trigger behavior (no repeated starts)
* **Artifacts:**

  * health state structs + store keys
  * keeper hooks from quota/evidence paths
  * unit tests
* **DoD:**

  * HealthState updates occur for both soft and hard failures.
  * Eviction triggers follow hot/cold thresholds and do not spam-trigger.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Ensure HealthState updates are idempotent; the same epoch’s miss should not be counted twice.
  * Avoid “repair thrash” by recording last repair-trigger epoch/height.

#### TASK P0-HEALTH-002 — Health observability: queries + events suitable for testnet monitoring

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, query layer
* **Depends on:** `P0-HEALTH-001`
* **Context:**

  * `notes/mainnet_policy_resolution_jan2026.md` includes testnet monitoring signals for slashing/jailing, repair rates, etc.
* **Work plan:**

  1. Emit explicit events for:

     * soft miss recorded
     * eviction threshold crossed
     * repair started (include reason: soft vs hard)
  2. Add query endpoints to fetch HealthState:

     * by deal+assignment
     * list for a deal (paginated)
  3. Add unit tests for query correctness and event emission.
* **Artifacts:**

  * query proto/service updates (existing query files)
  * keeper query handlers
  * unit tests
* **DoD:**

  * HealthState is queryable and events are emitted with stable fields.
  * Query pagination prevents unbounded responses.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Do not embed large binary blobs in events; keep events indexable and lightweight.

---

### Stage 5 — Mode 2 repair + make-before-break replacement

#### TASK P0-MODE2-MBB-001 — Make-before-break replacement state machine (slot status, catch-up, promotion)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, `rfcs/rfc-mode2-onchain-state.md`
* **Depends on:** `P0-PARAMS-001`, `P0-HEALTH-001`
* **Context:**

  * `rfcs/rfc-mode2-onchain-state.md` §3.3 (repair workflow) and slot fields (`pending_provider`, `repair_target_gen`, `status_since_height`)
* **Work plan:**

  1. Locate existing Mode2 slot state and confirm it matches RFC concepts (`ACTIVE`, `REPAIRING`, `pending_provider`, `repair_target_gen`).
  2. Implement `StartRepair` transition:

     * only from `ACTIVE`
     * set `status=REPAIRING`, `pending_provider=candidate`, `repair_target_gen=current_gen`, `status_since_height=height`
  3. Implement `Promote` transition:

     * only from `REPAIRING`
     * require a deterministic readiness proof that candidate caught up to `repair_target_gen`
     * on success, swap provider, clear pending fields, return to `ACTIVE`
  4. Ensure legacy compatibility: if `deal.providers[]` exists, keep it synced with slot providers.
  5. Add unit tests for correct state transitions and invalid-state rejection.
* **Artifacts:**

  * Mode2 slot keeper logic
  * state transition tests
* **DoD:**

  * State transitions implement make-before-break semantics (no “break-before-make” window).
  * Promotion requires an objective readiness condition.
  * Unit tests cover start/promotion and failure cases.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Promotion must be replay-safe; no double swaps.
  * Read routing must not depend on pending provider until promotion.

#### TASK P0-BOND-001 — Provider bonding baseline: provider bond state + min_provider_bond enforcement

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, provider registry/activation path
* **Depends on:** `P0-PARAMS-001`
* **Context:**

  * `notes/mainnet_policy_resolution_jan2026.md` B2 “Bonding: base bond + assignment collateral”
* **Work plan:**

  1. Identify how storage providers are represented/registered in this repo (and where eligibility is checked).
  2. Implement or extend provider bond state:

     * `bonded_amount`
     * `locked_amount` (reserved for assignments)
     * `unbonding_end_height` (if unbonding exists)
  3. Enforce `min_provider_bond`:

     * providers below min are ineligible for new assignments and deputy duties.
  4. Add query for provider bond state (required for operator UX and debugging).
  5. Unit tests for min bond enforcement.
* **Artifacts:**

  * provider bond keeper/types
  * query support
  * unit tests
* **DoD:**

  * Providers below `min_provider_bond` are deterministically rejected by assignment selection.
  * Provider bond state is queryable.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * If the repo already uses staking for provider collateral, do not duplicate; map “bond” to the existing stake source and document it.

#### TASK P0-BOND-002 — Assignment collateral: bond_months * storage_price * month_len_blocks * slot_bytes (locked) + unbonding lock

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`
* **Depends on:** `P0-BOND-001`, `P0-ECON-LOCKIN-001`, `P0-MODE2-MBB-001`
* **Context:**

  * Policy formula (B2): `required_bond = ceil(bond_months * storage_price * month_len_blocks * slot_bytes)`
* **Work plan:**

  1. Implement deterministic `slot_bytes` computation (reuse quota slot_bytes logic if possible).
  2. Implement required collateral calculation and locking:

     * lock additional bond when provider is assigned a slot (or becomes pending_provider in REPAIRING)
     * unlock when provider is removed/unassigned (including replacement/promotion)
  3. Enforce unbonding lock:

     * prevent unbonding/withdrawal that would drop provider below `bonded_amount - locked_amount`
     * enforce `provider_unbonding_blocks` if unbonding workflow exists
  4. Unit tests for lock/unlock and rejection of insufficient collateral.
* **Artifacts:**

  * keeper bond locking logic
  * unit tests
* **DoD:**

  * Assignment collateral is locked deterministically and scales with `slot_bytes`.
  * Providers cannot escape required collateral via unbonding while assigned.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Use ceil rounding for required bond to avoid under-collateralization.

#### TASK P0-REPAIR-001 — Deterministic replacement selection + churn/griefing controls (cooldown + attempt caps)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`
* **Depends on:** `P0-MODE2-MBB-001`, `P0-BOND-001`, `P0-BOND-002`, `P0-HEALTH-001`
* **Context:**

  * `notes/mainnet_policy_resolution_jan2026.md` B4 “Replacement selection”
  * Required controls: `replacement_cooldown_blocks`, `repair_attempts_cap`, `repair_attempt_window_blocks`
* **Work plan:**

  1. Implement deterministic candidate ranking seeded by epoch randomness (B4):

     * seed includes `R_e`, `deal_id`, `slot`, `current_gen`, and an attempt nonce
  2. Define eligibility:

     * not jailed
     * meets min bond + assignment collateral availability
     * not the current slot provider
  3. Implement cooldown enforcement:

     * do not start a new repair for the same slot if the last start was within `replacement_cooldown_blocks`
  4. Implement attempt caps:

     * track attempts per slot in a rolling window of `repair_attempt_window_blocks`
     * after `repair_attempts_cap`, enter backoff state (document exact behavior)
  5. Emit events for selection decisions and reasons for rejection.
  6. Add unit tests for determinism, cooldown, and cap behavior.
* **Artifacts:**

  * keeper candidate selection + state counters
  * unit tests
* **DoD:**

  * Candidate selection is deterministic and repeatable given the same inputs.
  * Cooldown and attempt caps prevent repair churn/grief.
  * Eligibility checks include bond and jail status.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Never iterate over an unsorted provider set when ranking; sort before hashing.
  * Keep attempt counters bounded; prune by window.

#### TASK P0-MODE2-ROUTING-002 — Ensure reads avoid REPAIRING slots (gateway/router)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nil_gateway/`
* **Depends on:** `P0-MODE2-MBB-001`, `P0-REPAIR-001`
* **Context:**

  * Stage 5 requirement: reads succeed throughout repair (router must avoid `REPAIRING`)
* **Work plan:**

  1. Locate the Mode2 provider selection logic in `nil_gateway/`.
  2. Ensure gateway fetches slot **status** (not just provider addresses). If the existing chain query does not include status, extend it or add a new query.
  3. Update routing:

     * choose `K` slots among those with `status=ACTIVE`
     * retry using other ACTIVE slots if a provider fails
  4. Add a regression test hook in the repair E2E that asserts retrieval succeeds while one slot is REPAIRING.
* **Artifacts:**

  * gateway routing code changes
  * any chain query usage updates needed by gateway
* **DoD:**

  * Gateway never routes reads to REPAIRING slots unless there are insufficient ACTIVE slots (then it fails fast with an explicit error).
  * Repair E2E confirms no outage during repair.
* **Test gate:**

  * `./scripts/ci_e2e_gateway_retrieval_multi_sp.sh`
  * (After Stage 5 E2E exists) run `P0-REPAIR-E2E-002` gate
* **Notes / gotchas:**

  * Avoid infinite retry loops; cap retries and log tried providers.

#### TASK P0-MODE2-REWARD-003 — Repairing slots earn no rewards and are ignored by synthetic challenges

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`
* **Depends on:** `P0-QUOTAS-001`, `P0-QUOTAS-002`, `P0-MODE2-MBB-001`
* **Context:**

  * Stage 5 requirement: synthetic challenges ignore REPAIRING; repairing slots do not earn rewards
* **Work plan:**

  1. Verify/ensure challenge derivation excludes non-ACTIVE slots (from `P0-QUOTAS-001`).
  2. Ensure quota accounting does not require proofs for REPAIRING slots.
  3. If the chain pays any liveness/retrieval rewards, ensure REPAIRING slots are rejected or paid 0.
  4. Add unit tests that:

     * derived challenges never target repairing slots
     * quota for repairing slots is effectively 0 / excluded
* **Artifacts:**

  * keeper logic updates
  * unit tests
* **DoD:**

  * Repairing slots are excluded from challenges, quotas, and rewards.
  * Unit tests validate exclusion end-to-end at the keeper level.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Separate “repair readiness proofs” from “liveness proofs” so you don’t accidentally pay for repair traffic.

#### TASK P0-REPAIR-E2E-002 — Multi-SP repair e2e: slot failure → catch-up → promotion; reads succeed throughout

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `scripts/`, `nil_gateway/`, (devnet stack)
* **Depends on:** `P0-MODE2-MBB-001`, `P0-REPAIR-001`, `P0-MODE2-ROUTING-002`
* **Context:**

  * `scripts/ci_e2e_gateway_retrieval_multi_sp.sh` demonstrates the preferred CI gate shape.
  * Stage 5 requires an E2E gate validating “replacement without read outage”.
* **Work plan:**

  1. Add a CI-friendly script (new or extension) that:

     * starts multi-SP stack
     * creates a Mode2 deal + uploads data
     * kills/ghosts one provider process corresponding to a slot
     * waits for repair start (or triggers it through chain state if already exposed)
     * verifies gateway retrieval succeeds during repair
     * completes catch-up + triggers promotion
     * verifies on-chain slot provider changed and slot returns ACTIVE
  2. Add assertions (exit non-zero on failure) and bounded polling loops.
  3. If needed, add a `scripts/ci_e2e_mode2_repair_multi_sp.sh` wrapper mirroring the existing `ci_` script style.
* **Artifacts:**

  * new or updated script(s) under `scripts/`
* **DoD:**

  * E2E demonstrates make-before-break repair with no read outage.
  * Script is stable (timeouts, logs, deterministic).
* **Test gate:**

  * the new/updated repair E2E script command
* **Notes / gotchas:**

  * Avoid flakiness by polling chain state and gateway health endpoints rather than sleeping.

#### TASK P1-REPAIR-OVERRIDE-001 — Trusted repair override posture (dev/test enabled if implemented; mainnet disabled by default)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`
* **Depends on:** `P0-MODE2-MBB-001`, `P0-REPAIR-001`
* **Context:**

  * Policy posture requirement (explicit): dev/test enabled **if implemented**; mainnet disabled by default, governance-emergency only.
* **Work plan:**

  1. If implementing the override, do it as an **authority-only** (governance) message, not a user tx.
  2. Gate functionality behind a boolean param (default: enabled in dev/test genesis; disabled in mainnet genesis).
  3. Ensure override actions emit explicit events and are auditable.
  4. Unit tests verifying:

     * unauthorized cannot call
     * mainnet default disables
* **Artifacts:**

  * optional new msg + keeper handler
  * unit tests
* **DoD:**

  * Override cannot be invoked by non-authority.
  * Mainnet default is disabled; dev/test default is enabled (if feature exists).
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Keep this as an emergency tool only; do not allow silent bypass of bonding/slashing unless explicitly authorized.

---

### Stage 6 — Evidence / fraud proofs pipeline

#### TASK P0-EVIDENCE-001 — Evidence taxonomy + verification + replay protection + slash/jail/evict wiring (B1)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`, evidence/proof proto handlers
* **Depends on:** `P0-PARAMS-001`, `P0-HEALTH-001`, `P0-MODE2-MBB-001`, `P0-REPAIR-001`
* **Context:**

  * `MAINNET_ECON_PARITY_CHECKLIST.md` Stage 6
  * Policy ladder (B1):

    * invalid proof: slash 0.5% (50 bps), jail 3 epochs
    * wrong data: slash 5% (500 bps), jail 30 epochs
    * non-response: handled in Stage 7 aggregation; this task handles hard-fault proofs
* **Work plan:**

  1. Enumerate evidence types required for hard faults:

     * invalid proof (cryptographic verification fails)
     * wrong data (provable mismatch against commitment/manifest)
  2. Implement verification paths for both:

     * accept only if verifiable on-chain
     * compute stable `evidence_id` hash and enforce replay protection
  3. Wire penalties using B1 params:

     * slash provider bond by bps (deterministic rounding)
     * jail provider for configured epochs (store/enforce as end-height)
  4. Integrate with repair start:

     * on conviction, immediately trigger Mode2 repair start (hard failure)
  5. Add unit tests per evidence type:

     * verify acceptance, replay rejection
     * verify slash/jail/repair triggered exactly once
* **Artifacts:**

  * evidence message/handler implementation
  * keeper slashing/jailing primitives
  * unit tests
* **DoD:**

  * Evidence is verified on-chain, replay-protected, and applies the configured slash/jail.
  * Hard-fault evidence triggers repair start for Mode2.
  * Unit tests cover correctness and idempotency.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Do not allow multiple evidence submissions to cause repeated slashes; store a “penalized” marker keyed by `evidence_id` or `(deal, slot, epoch)`.
  * Jail duration should be policy epoch-based but enforced at height granularity.

#### TASK P0-EVIDENCE-E2E-002 — E2E evidence gate: proven wrong data → slash + jail + repair start

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `scripts/`, provider/gateway test hooks
* **Depends on:** `P0-EVIDENCE-001`, `P0-REPAIR-E2E-002` (or at least `P0-REPAIR-001`)
* **Context:**

  * Stage 6 requires an E2E demonstrating slash on proven bad data.
* **Work plan:**

  1. Add a script that:

     * sets up a deal and uploads content
     * causes a provider to return provably wrong data for a known shard (test hook or controlled corruption)
     * submits wrong-data evidence tx
     * asserts: provider slashed, jailed, and slot enters REPAIRING
  2. Keep it gated and deterministic (explicit provider index, bounded polling).
* **Artifacts:**

  * new script under `scripts/`
  * (if needed) provider test hook guarded by env var
* **DoD:**

  * The script deterministically demonstrates slash/jail/repair transition from wrong-data evidence.
* **Test gate:**

  * the new evidence E2E script command
* **Notes / gotchas:**

  * Corruption should be opt-in test mode only; never default-enable in normal runs.

---

### Stage 7 — Deputy market + proxy retrieval + audit debt

#### TASK P0-DEPUTY-001 — Proxy retrieval economics (chain): premium lock + premium payout on success

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`
* **Depends on:** `P0-PARAMS-001`, `P0-RETRIEVAL-SETTLE-002`
* **Context:**

  * Policy (B5): proxy premium defaults: dev/test 20% (`premium_bps=2000`), mainnet 10% (`premium_bps=1000`)
  * Proxy semantics: user pays market rate + premium; provider paid as normal; deputy gets premium on success
* **Work plan:**

  1. Extend retrieval session accounting to represent proxy sessions (reuse existing session type if possible; otherwise add a dedicated proxy session type).
  2. On proxy session open:

     * burn base fee (same as normal)
     * lock variable fee and lock premium fee
     * decrement escrow by `base + variable + premium`
  3. On completion:

     * settle variable fee (burn cut + provider payout) as normal
     * pay premium to deputy (no burn unless explicitly specified elsewhere)
  4. Add unit tests for:

     * premium calculation and payout
     * premium paid only on success and only to the deputy
* **Artifacts:**

  * chain session logic updates
  * unit tests
* **DoD:**

  * Proxy sessions correctly lock/payout premium.
  * Premium is not paid without a successful proof-based completion.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Ensure deputy cannot be the same entity as the failing provider for the same session (or document/guard if allowed).

#### TASK P0-DEPUTY-002 — Proof-of-failure aggregation + evidence incentives (bond/bounty + partial burn on TTL expiry)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`
* **Depends on:** `P0-DEPUTY-001`, `P0-PARAMS-001`, `P0-EVIDENCE-001`
* **Context:**

  * Policy (B1): `nonresponse_threshold=3`, `nonresponse_window_epochs=6`, `slash_nonresponse_bps=100`, `jail_nonresponse_epochs=10`
  * Policy (B5): `evidence_bond=0.01 NIL`, `failure_bounty=0.02 NIL`, burn 50% on TTL expiry (`evidence_bond_burn_bps_on_expiry=5000`)
* **Work plan:**

  1. Add a proof-of-failure submission message and store:

     * replay-protect proof-of-failure ids
     * lock `evidence_bond` on submission (module holds funds)
     * set expiry epoch = now + `proof_of_failure_ttl_epochs` (default = `nonresponse_window_epochs`)
  2. Maintain an aggregation window keyed by target provider (and optionally deal/slot):

     * count distinct deputies within `nonresponse_window_epochs`
     * convict when count >= `nonresponse_threshold`
  3. On conviction:

     * apply slash/jail for non-response
     * refund evidence bonds for proofs contributing to conviction
     * pay `failure_bounty` to deputies (define funding source; prefer audit budget module once implemented)
  4. On expiry without conviction:

     * burn `evidence_bond_burn_bps_on_expiry` portion
     * refund remainder
  5. Unit tests for windowing, distinct deputy counting, conviction idempotency, and bond burn/refund.
* **Artifacts:**

  * on-chain proof-of-failure storage + handlers
  * bond/bounty settlement logic
  * unit tests
* **DoD:**

  * Non-response conviction triggers at the configured threshold/window.
  * Evidence bond is locked and either refunded+bountied (on conviction) or partially burned (on expiry).
  * Unit tests cover all state transitions deterministically.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Defend against Sybil reporters via the bond and by ensuring deputies are selected by the gateway/p2p layer (not arbitrary self-assigned).

#### TASK P0-AUDIT-001 — Audit budget minting (Option A): audit_budget_bps/cap + carryover ≤2 epochs + epoch_slot_rent formula

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`
* **Depends on:** `P0-PARAMS-001`, `P0-MODE2-MBB-001`, `P0-MODE2-REWARD-003`
* **Context:**

  * Policy (Option A, closed):

    * `epoch_slot_rent = storage_price * total_active_slot_bytes * epoch_len_blocks`
    * `audit_budget_mint = ceil(audit_budget_bps/10_000 * epoch_slot_rent)` capped by `audit_budget_cap_bps`
    * carryover unused budget for ≤2 epochs
* **Work plan:**

  1. Implement deterministic computation of `total_active_slot_bytes`:

     * include ACTIVE Mode2 slots only
     * include Mode1 assignments (if applicable)
     * exclude REPAIRING slots
  2. At epoch boundary, compute `epoch_slot_rent` and `audit_budget_mint` with cap.
  3. Mint budget into the designated module account and track spendable budget with bounded carryover (≤2 epochs).
  4. Emit events for rent, mint, cap binding, carryover, and expirations/burns.
  5. Unit tests:

     * mint math correctness (ceil + cap)
     * carryover expiry after 2 epochs
* **Artifacts:**

  * epoch boundary keeper logic
  * audit budget state storage
  * unit tests
* **DoD:**

  * Audit budget mints deterministically per epoch and honors cap + carryover ≤2 epochs.
  * REPAIRING slots are excluded from rent computation.
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Avoid full chain scans each epoch if it’s too costly; prefer maintaining an incrementally updated aggregate, but correctness comes first.

#### TASK P0-AUDIT-002 — Audit debt tracking + budget spend path (MVP)

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nilchain/x/nilchain/keeper/`, `nilchain/x/nilchain/types/`
* **Depends on:** `P0-AUDIT-001`, `P0-DEPUTY-002`
* **Context:**

  * `MAINNET_GAP_TRACKER.md` P0-P2P-001 expects “audit debt tasks assignable/trackable” and budget utilization monitoring.
* **Work plan:**

  1. Implement minimal audit debt state:

     * per-provider counters for “audit required” and “audit completed”
     * query endpoints for outstanding debt
  2. Define a minimal “spend from audit budget” primitive used by:

     * paying `failure_bounty` on conviction
     * paying for audit retrieval traffic (if/when integrated)
  3. Unit tests for deterministic debt updates and budget spend accounting.
* **Artifacts:**

  * audit debt state + query
  * budget spend helper
  * unit tests
* **DoD:**

  * Audit debt is stored and queryable.
  * Audit budget can be debited deterministically for bounties (and later audits).
* **Test gate:**

  * `go test ./nilchain/...`
* **Notes / gotchas:**

  * Keep MVP tight: trackability first; full audit task scheduling can iterate later.

#### TASK P0-DEPUTY-003 — Deputy routing (gateway + p2p): AskForProxy → deputy serves → chain settlement

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `nil_p2p/`, `nil_gateway/`
* **Depends on:** `P0-DEPUTY-001`, `P0-DEPUTY-002`, `P0-AUDIT-001`
* **Context:**

  * `MAINNET_GAP_TRACKER.md` P0-P2P-001: `nil_p2p` has an `AskForProxy` stub that must be implemented.
  * Stage 7 requires end-to-end proxy retrieval (selection, routing, settlement).
* **Work plan:**

  1. Implement `AskForProxy` request/response flow in `nil_p2p/`:

     * request identifies deal/range/provider and premium offer
     * response provides a deputy endpoint/identity
  2. Gateway integrates fallback logic:

     * on primary failure, call `AskForProxy` and fetch from deputy
     * on success, submit proxy settlement to chain
     * on failure, submit proof-of-failure to chain (locks bond)
  3. Add logging/metrics hooks aligned with monitoring signals (proxy success rate, deputy fraction).
* **Artifacts:**

  * `nil_p2p/` AskForProxy implementation
  * `nil_gateway/` fallback routing integration
* **DoD:**

  * Gateway can retrieve via deputy when primary fails and settle premium correctly on chain.
  * Gateway can submit proof-of-failure when appropriate.
* **Test gate:**

  * Run the ghosting-provider E2E (next task).
* **Notes / gotchas:**

  * Protect against deputy spam: rate-limit and require deputy eligibility (min bond) if available.

#### TASK P0-DEPUTY-E2E-002 — Ghosting-provider E2E: proxy retrieval succeeds + evidence recorded

* **Status:** `[ ] not started  [ ] in progress  [ ] blocked  [x] done`
* **Owner:**
* **Area:** `scripts/`
* **Depends on:** `P0-DEPUTY-003`, `P0-DEPUTY-001`, `P0-DEPUTY-002`
* **Context:**

  * `MAINNET_ECON_PARITY_CHECKLIST.md` Stage 7 requires “ghosting-provider deputy e2e”.
* **Work plan:**

  1. Add a CI-style script (mirroring `scripts/ci_e2e_gateway_retrieval_multi_sp.sh`) that:

     * starts multi-SP stack
     * creates a deal + uploads content
     * forces the primary provider to ghost
     * performs retrieval and validates it succeeds via deputy
     * asserts proxy premium paid to deputy
  2. (Optional extension) Trigger multiple proof-of-failure submissions and verify:

     * bond locked
     * conviction triggers at threshold
     * bond refund/burn works on conviction/expiry
  3. Ensure stable polling, explicit timeouts, and clear logs.
* **Artifacts:**

  * new `scripts/ci_e2e_deputy_ghosting.sh` (or similar, if you add it)
* **DoD:**

  * Script deterministically validates deputy fallback retrieval and on-chain settlement.
  * Script exits non-zero on failure.
* **Test gate:**

  * run the new CI-style ghosting script locally
* **Notes / gotchas:**

  * Keep the test deterministic by selecting a specific provider index to kill/ghost.

---

## 4) Global Test Gates

These are the canonical “stop-the-line” gates used to claim parity. Tasks should reference one or more of these.

* **Stage 0 (params/interfaces):**

  * `go test ./nilchain/...`
  * (Optional) `./scripts/run_devnet_alpha_multi_sp.sh start` (boot smoke)
* **Stage 1–2 (econ + retrieval session lifecycle):**

  * `./scripts/e2e_lifecycle.sh`
  * `go test ./nilchain/...`
* **Stage 3 (challenges/quotas):**

  * `go test ./nilchain/...`
  * Challenge sim gate from `TASK P0-QUOTAS-SIM-003`
* **Stage 5 (repair + no outage):**

  * `./scripts/ci_e2e_gateway_retrieval_multi_sp.sh`
  * Repair E2E from `TASK P0-REPAIR-E2E-002`
* **Stage 6 (evidence):**

  * `go test ./nilchain/...`
  * Evidence E2E from `TASK P0-EVIDENCE-E2E-002`
* **Stage 7 (deputy + audit debt):**

  * Ghosting-provider E2E from `TASK P0-DEPUTY-E2E-002`

---

## 5) Open Decisions

(Empty — add here only if a decision is truly unresolved and blocks implementation.)
```

```MAINNET_GAP_TRACKER.md
# Mainnet Gap Tracker (NilStore)

This document tracks **what is missing** between the current implementation in this repo and the **long‑term Mainnet plan** described by `spec.md` (canonical), `rfcs/`, and `notes/`.

**Sources (ordered):**
- `spec.md` (canonical protocol spec; v2.4 at time of writing)
- `rfcs/` (design proposals / deep dives; check header status)
- `notes/roadmap_milestones_strategic.md` (milestone sequencing)
- `notes/mainnet_policy_resolution_jan2026.md` (proposal: concrete defaults for remaining econ/repair/deputy policies)
- `AGENTS_MAINNET_PARITY.md` (codex-ready agents punch list derived from checklist + policy defaults)

## How To Use

- Keep items **small enough to ship** (1–5 PRs each).
- Every epic should have a **test gate** (unit/e2e/script) before it can be marked “Done”.
- Prefer tracking **code ownership** by directory:
  - Chain: `nilchain/`
  - Gateway/SP: `nil_gateway/`
  - Core crypto/WASM: `nil_core/`
  - CLI automation: `nil_cli/`
  - P2P: `nil_p2p/`
  - Web UX: `nil-website/`

## Status Legend

- **DONE**: implemented + tested in CI and/or e2e scripts
- **PARTIAL (DEVNET)**: exists, but incomplete vs spec/mainnet hardening (often “devnet convenience”)
- **MISSING**: not implemented
- **RFC / UNSPECIFIED**: explicitly underspecified in `spec.md` Appendix B; needs policy finalization
- **SPECIFIED (RFC)**: policy/interfaces frozen in RFCs, but implementation is still missing

## Critical Path (P0) — Mainnet Blocking

### P0-CHAIN-001 — Mode 2 generations + repair mode + make‑before‑break replacement
- **Status:** PARTIAL (DEVNET)
- **Spec:** `spec.md` §8.4, §5.3, Appendix B (2, 4, 6)
- **Current state:** the chain now tracks typed Mode 2 slots + a first-pass `current_gen` and per-slot repair state, but make-before-break replacement, append-safe repair coordination, and read routing around repairing slots are not fully implemented.
- **DoD:** Chain has explicit generation + slot status; repairs are observable; replacement is make‑before‑break; reads route around repairing slots; append-only commit rules enforced.
- **Test gate:** new e2e (multi-SP) that simulates slot failure → repair catch-up → slot rejoin without breaking reads.

### P0-CHAIN-002 — Challenge derivation + proof demand policy + quota enforcement
- **Status:** SPECIFIED (RFC)
- **Spec:** `spec.md` §7.6, Appendix B (3, 4); `rfcs/rfc-challenge-derivation-and-quotas.md`
- **Current state:** sessions/proofs exist; deterministic quota + synthetic fill policy is now specified, but not implemented in keeper state machines.
- **DoD:** deterministic challenge derivation from chain state + epoch randomness; quota accounting; penalties for non-compliance distinct from invalid proofs.
- **Test gate:** keeper unit tests + adversarial sim tests for challenge determinism and anti-grind properties.

### P0-CHAIN-003 — Fraud proofs / evidence taxonomy (wrong data, non-response, etc.)
- **Status:** PARTIAL (DEVNET)
- **Spec:** `spec.md` §7.5
- **Current state:** session-based flows exist, but the full evidence/fraud proof pipeline and policy-level outcomes (slash/evict) aren’t complete.
- **DoD:** on-chain evidence messages/types + verification; slashing/jailing/eviction integration; replay protections; clear invariants.
- **Test gate:** unit tests for each evidence type + e2e that demonstrates slash on proven bad data.

### P0-P2P-001 — Deputy system + proxy retrieval market + audit debt
- **Status:** PARTIAL (stub only)
- **Spec:** `spec.md` §7.7–§7.8; `rfcs/rfc-retrieval-validation.md`; Appendix B (7)
- **Current state:** `nil_p2p` has an `AskForProxy` message stub, but no end-to-end deputy selection, relay, compensation, or evidence.
- **DoD:** proxy retrieval works when an SP “ghosts”; failure evidence is produced and aggregated; audit debt tasks are assignable/trackable; griefing mitigations.
- **Test gate:** e2e “ghosting provider” scenario that still retrieves via deputy and records evidence.

### P0-PERF-001 — High-throughput KZG (GPU) + parallel ingest pipeline
- **Status:** PARTIAL (DEVNET)
- **Spec/Notes:** `notes/kzg_upload_bottleneck_report.md`, `notes/kzg_gpu_design.md`, `notes/roadmap_milestones_strategic.md` (Milestone 2)
- **Current state:** CPU KZG works and the gateway ingest pipeline is parallelized by default; GPU-class acceleration is still missing for mainnet target throughput.
- **DoD:** CUDA (server) and/or WebGPU (client) path that materially raises sustained throughput; pipeline parallelism is default.
- **Test gate:** reproducible perf benchmark suite (CI “doesn’t regress”) + local benchmark script with thresholds.

### P0-CORE-001 — “One core” migration (NilFS + crypto single source of truth)
- **Status:** PARTIAL (DEVNET)
- **Spec/Notes:** `notes/roadmap_milestones_strategic.md` (Milestone 1)
- **Current state:** `nil_gateway` contains NilFS/layout logic in Go, while the browser uses `nil_core` WASM for crypto; risk of drift.
- **DoD:** NilFS builder/layout + commitment logic live in `nil_core` with WASM + CGO bindings; browser + gateway agree on commitments deterministically.
- **Test gate:** parity tests that compare browser vs gateway roots/commitments for the same file set.

### P0-ECON-001 — Mainnet escrow accounting + lock-in pricing (pay-at-ingest)
- **Status:** PARTIAL (DEVNET)
- **Spec:** `spec.md` §6.1–§6.2, §7.2.1; Appendix B (5); `rfcs/rfc-pricing-and-escrow-accounting.md`
- **Current state:** deal escrow exists and retrieval fees exist; lock-in + fee settlement is partially implemented, but spend windows and deterministic elasticity debits remain incomplete.
- **DoD:** clear accounting rules for storage rent + bandwidth; enforce max spend caps; elasticity debits are deterministic and replay-safe.
- **Test gate:** chain-level econ e2e (create deal → upload → retrieve → check balances/fees/burns) for multiple parameter sets.

### P0-OPS-001 — Mainnet-grade security + audits + threat model closure
- **Status:** MISSING
- **Spec/Notes:** `spec.md` §5, Appendix B (8, 9)
- **Current state:** devnet-grade hardening exists (auth tokens, strict parsing in many places), but audit posture is not “mainnet ready”.
- **DoD:** external audits (crypto + chain + gateway), hardening issues resolved, incident response plan, secure defaults.
- **Test gate:** security test suite + documented audit scope and “must-fix” checklist.

## Domain Backlog (P1/P2) — Organized By Subsystem

### Chain / Protocol (`nilchain/`)

#### CHAIN-101 — Explicit Mode 2 encoding on-chain (K/M, slot mapping, overlays)
- **Status:** PARTIAL (DEVNET)
- **Spec:** `spec.md` §6.2, §8.1.3; Appendix B (2); `rfcs/rfc-mode2-onchain-state.md`
- **Notes:** Today, RS profile is encoded in `service_hint` and slots are represented via `providers[]`. Mainnet needs explicit typed state + upgrade strategy (now specified in RFC).

#### CHAIN-102 — Rotation policy + governance-gated bootstrap mode
- **Status:** MISSING
- **Spec:** `spec.md` §4.3, §5.1, §5.3; Appendix B (1, 4)

#### CHAIN-103 — HealthState / self-healing placement + eviction curve
- **Status:** PARTIAL (DEVNET)
- **Spec:** `spec.md` §7.9; Appendix B (1, 4)

#### CHAIN-104 — Deletion semantics (deal cancel, expiry enforcement, crypto-erasure UX hooks)
- **Status:** PARTIAL (DEVNET)
- **Spec:** `spec.md` §6.3, §8.4.4; Appendix B (6, 8)
- **Notes:** “Crypto-erasure” is a client contract; chain still needs consistent cancellation semantics and post-expiry invariants.

#### CHAIN-105 — Third-party sponsorship / funding flows (viral debt mitigation)
- **Status:** PARTIAL (DEVNET)
- **Spec:** `spec.md` §5.2
- **Notes:** confirm whether `MsgAddCredit` is sufficient for sponsorship (non-owner funding) and whether UI exposes it.

#### CHAIN-106 — EVM module production posture (simulation vs runtime)
- **Status:** PARTIAL (DEVNET)
- **Spec/Notes:** `AGENTS.md` Phase 5 notes; `nilchain/app/app.go` simulation exclusions
- **Notes:** EVM/FeeMarket are excluded from simulation to avoid signer panics; ensure production builds are safe and tested.

### Gateway / Provider (`nil_gateway/`)

#### GW-201 — Strict session enforcement on data-plane fetches
- **Status:** PARTIAL (DEVNET)
- **Spec:** `spec.md` Appendix A “Gateway/API note”, §7.2
- **DoD:** gateway/SP enforce `X‑Nil‑Session‑Id` when sessions required; out-of-session range fetches are rejected; consistent error JSON.

#### GW-202 — Repair tooling + deterministic reconstruction for Mode 2 slots
- **Status:** PARTIAL (DEVNET)
- **Spec:** `spec.md` §8.4, §8.2

#### GW-203 — Upload delegation (third-party uploader pattern)
- **Status:** MISSING
- **Notes:** `notes/launch_todos.md`

#### GW-204 — S3 adapter polish + bidirectional sync scripts (nilstore ↔ S3)
- **Status:** PARTIAL (DEVNET)
- **Spec/Notes:** roadmap milestone 5, `notes/launch_todos.md`

### Web / UX (`nil-website/`)

#### WEB-301 — Provider onboarding wizard (“Become a Provider”)
- **Status:** MISSING
- **Notes:** `notes/roadmap_milestones_strategic.md` (Milestone 1)

#### WEB-302 — Hybrid client “unified namespace” + sync manager (OPFS ↔ Gateway ↔ Network)
- **Status:** PARTIAL (DEVNET)
- **Spec/Notes:** `notes/roadmap_milestones_strategic.md` (Milestone 1)

#### WEB-303 — Educational content remediation (Mode 2, Triple Proof, Deputy)
- **Status:** MISSING
- **Source:** `nil-website/AGENTS.md` §8

### Core crypto / WASM (`nil_core/`)

#### CORE-401 — WebGPU KZG commitments/proofs (client-side velocity)
- **Status:** MISSING
- **Notes:** `notes/kzg_gpu_design.md`

#### CORE-402 — Determinism harness (cross-runtime, cross-platform)
- **Status:** PARTIAL (DEVNET)
- **DoD:** stable outputs for commitments across Mac/Linux and browser/gateway; fuzzers for edge-cases.

### CLI / Automation (`nil_cli/`, `scripts/`)

#### CLI-501 — Enterprise upload job runner (delegated key, scoped funding, teardown)
- **Status:** MISSING
- **Notes:** `notes/launch_todos.md`

#### CLI-502 — Fast download / mirror scripts (provider → local, nilstore → S3)
- **Status:** PARTIAL (DEVNET)
- **Notes:** `notes/launch_todos.md`

### P2P (`nil_p2p/`)

#### P2P-601 — Production transport + discovery (beyond stubs)
- **Status:** PARTIAL (DEVNET)
- **Spec:** `spec.md` Appendix B (9)

## Spec ↔ Implementation Divergences To Track Explicitly

- **Deal sizing naming (resolved):** `spec.md` uses `Deal.size`/`size_bytes` (logical bytes) plus `Deal.total_mdus` + `Deal.witness_mdus` (slab bounds). Gateway may still return legacy `allocated_length` as an alias for `total_mdus` (count).
- **Mode 2 on-chain representation (specified):** explicit typed `(K,M)`, slot mapping, generations, and repair state frozen in `rfcs/rfc-mode2-onchain-state.md`.
- **EVM simulation posture:** EVM/FeeMarket excluded from simulation to avoid signer panics; ensure this doesn’t mask mainnet correctness issues.

## Suggested Sequencing (Pragmatic)

1. **CORE-001 One-core migration** (reduce drift risk first).
2. **ECON-001 Lock-in + escrow accounting** (mainnet business logic).
3. **PERF-001 GPU + ingest parallelism** (make the product usable at scale).
4. **CHAIN-001/002/003/103** (repair, challenges, fraud proofs, health).
5. **P2P-001 deputy + audit debt** (adversarial resilience).
6. **OPS-001 audits + hardening** (gate before mainnet).

---

## Sprint Roadmap (Proposed)

Assumption: **2-week engineering sprints**, with a strict “test gate” on every sprint exit. Adjust duration as needed; keep the **scope** bounded.

### Sprint 0 — RFC closure + interfaces freeze (Protocol planning sprint)
- **Goal:** turn Appendix B “unspecified” items into implementable, testable contracts.
- **Delivers (Docs + reference code stubs):**
  - Finalize the target on-chain representation for Mode 2: explicit `(K,M)`, slot mapping, overlay state, slot status, and generation fields (Appendix B #2, #6).
  - Finalize challenge derivation + proof quota policy (Appendix B #3, #4).
  - Finalize pricing/escrow accounting policy (Appendix B #5).
  - Decide and document the `allocated_length` vs `size` vs `total_mdus` naming convergence (see “Divergences” section).
- **Outputs (Sprint 0):**
  - `rfcs/rfc-mode2-onchain-state.md`
  - `rfcs/rfc-challenge-derivation-and-quotas.md`
  - `rfcs/rfc-pricing-and-escrow-accounting.md`
  - `spec.md` naming + Appendix B references aligned to the RFCs
- **Exit criteria:** updated RFCs/spec deltas + a checklist of exact protobuf/state transitions to implement in the next sprints.

### Sprint 1 — “One core” foundation (NilFS + commitments unified)
- **Targets:** **P0-CORE-001**, **CORE-402** (partial), plus the “Divergences” naming decision groundwork.
- **Goal:** eliminate browser/gateway drift risk by centralizing NilFS layout + commitment computation in `nil_core`.
- **Delivers:**
  - Port NilFS layout/builder primitives from `nil_gateway/pkg/*` into `nil_core` (Rust) with a stable API surface.
  - WASM bindings used by `nil-website` AND CGO/FFI bindings used by `nil_gateway` point to the same implementation.
  - Parity tests: same file set → identical manifest root + per-MDU roots across browser(WASM) and gateway(native).
- **Test gate:** new parity test suite + existing `./scripts/e2e_browser_smoke.sh`.

### Sprint 2 — Economic model v1 (lock-in, caps, top-ups)
- **Targets:** **P0-ECON-001**, **CHAIN-105**.
- **Goal:** make “user-funded elasticity + storage rent” real and enforceable (not a narrative).
- **Delivers:**
  - Implement pay-at-ingest debit schedule (or equivalent lock-in) for `UpdateDealContent*` and retrieval session fees accounting.
  - Enforce `max_monthly_spend` in code paths that can increase cost (uploads/elasticity triggers).
  - Clarify and implement third-party sponsorship semantics (whether `MsgAddCredit` supports it safely, and how UI exposes it).
- **Test gate:** chain econ e2e (deal → upload → retrieve → verify balances/burns/caps) across multiple parameter sets.

### Sprint 3 — Mode 2 on-chain encoding (explicit state, not service_hint encoding)
- **Targets:** **CHAIN-101**, plus prerequisites for **P0-CHAIN-001**.
- **Goal:** move Mode 2 out of “devnet convenience encoding” into explicit typed state.
- **Delivers:**
  - Deal stores explicit `(K,M)` (or equivalent) and a canonical ordered `slot → provider` mapping.
  - Upgrade strategy from legacy `service_hint` encoding (devnet) to typed fields without breaking existing deals.
- **Test gate:** migration tests + multi-provider e2e that creates Mode 2 deals and verifies slot ordering invariants.

### Sprint 4 — Mode 2 generations + repair mode + make-before-break replacement
- **Targets:** **P0-CHAIN-001**, **GW-202** (partial).
- **Goal:** the chain can coordinate repairs safely while allowing append-only writes.
- **Delivers:**
  - `current_gen` + slot status (ACTIVE/REPAIRING) + append-only commit enforcement.
  - Replacement workflow: add new provider in REPAIRING, require catch-up proof/readiness, then promote to ACTIVE (make-before-break).
  - Gateway repair tooling for deterministic reconstruction and catch-up tasks.
- **Test gate:** multi-SP e2e that simulates slot failure → repair catch-up → slot rejoin; reads succeed throughout.

### Sprint 5 — Unified liveness v1 (quota + synthetic fill + health)
- **Targets:** **P0-CHAIN-002**, **CHAIN-103**, **GW-201** (tighten enforcement).
- **Goal:** make “Retrieval IS Storage” enforceable with deterministic fallback challenges and health accounting.
- **Delivers:**
  - Deterministic challenge derivation for synthetic fill + quota accounting.
  - Session credits reduce synthetic demand; synthetic challenges target only ACTIVE slots.
  - HealthState per (Deal, Provider/Slot) and eviction/jail integration hooks (policy from Sprint 0).
  - Enforce session-bound fetch requirements on the data plane (when enabled).
- **Test gate:** keeper unit tests + adversarial simulation + e2e showing quota enforcement and health impact.

### Sprint 6 — Fraud proofs + evidence pipeline (bad data, non-response)
- **Targets:** **P0-CHAIN-003**, **CHAIN-102** (policy hooks), **P0-OPS-001** (partial hardening).
- **Goal:** “wrong bytes” becomes slashable with a clean evidence path.
- **Delivers:**
  - On-chain evidence types + verification for wrong data and bounded non-response challenges (per spec shape).
  - Slashing/jailing/eviction curve wired to evidence outcomes (parameters from Sprint 0).
  - Clear replay/expiry protections and audit-friendly event emission.
- **Test gate:** unit tests for each evidence type + e2e that produces a slash on proven bad data.

### Sprint 7 — Deputy system (proxy retrieval) + audit debt v1
- **Targets:** **P0-P2P-001**, **P2P-601** (incremental), **spec.md** Appendix B #7.
- **Goal:** handle “ghosting SPs” and scale coverage even when users are idle.
- **Delivers:**
  - Deputy discovery + proxy retrieval path (end-to-end) with anti-griefing controls.
  - Evidence collection for repeated failures, plus the first “audit debt” scheduler shape (even if conservatively parameterized).
- **Test gate:** e2e ghosting scenario: user retrieves via deputy; evidence recorded; no false slashes from a single deputy.

### Sprint 8 — Throughput (GPU) + production ingest defaults
- **Targets:** **P0-PERF-001**, **CORE-401** (optional client track), plus perf regression gates.
- **Goal:** remove the CPU KZG bottleneck for large data ingest; ensure the fast path is default (not behind env flags).
- **Delivers:**
  - GPU KZG acceleration in the gateway/CLI ingest path (CUDA/Icicle or equivalent), plus parallel pipeline scheduling.
  - Benchmark harness + perf regression thresholds (CI “alerts on regression”, local “meets target MB/s”).
  - Decide whether WebGPU KZG is a mainnet requirement or a post-mainnet UX upgrade; if required, implement minimal viable path.
- **Test gate:** perf suite + large-file ingest e2e on a reference machine (documented).

### Sprint 9 — Enterprise surface area (S3 polish + delegation tooling)
- **Targets:** **GW-204**, **GW-203**, **CLI-501**, **CLI-502**.
- **Goal:** “looks like S3” and supports delegated upload jobs safely.
- **Delivers:**
  - S3 adapter correctness + compatibility testing (aws-cli/rclone).
  - Third-party uploader pattern: scoped key funding + teardown + audit workflow.
  - Fast download / mirroring scripts (nilstore ↔ S3) with documented performance expectations.
- **Test gate:** integration tests + scripted “upload from S3 → verify on-chain → retrieve to S3” pipeline.

### Sprint 10 — Mainnet hardening + audits + launch readiness
- **Targets:** **P0-OPS-001**, plus closure of remaining P0s.
- **Goal:** turn “working devnet” into “auditable, operable mainnet”.
- **Delivers:**
  - Audit scopes (crypto/chain/gateway), fixes, and a “must-fix before mainnet” checklist.
  - Incident response runbooks, monitoring/alerting, safe defaults, and security posture docs.
  - Final “Mainnet readiness” e2e suite and release checklist.
- **Test gate:** security test suite + external audit signoff + final e2e battery green.

## Sprint Coverage Matrix (IDs → Sprint)

- **Sprint 1:** P0-CORE-001, CORE-402 (partial)
- **Sprint 2:** P0-ECON-001, CHAIN-105
- **Sprint 3:** CHAIN-101 (and prerequisites for P0-CHAIN-001)
- **Sprint 4:** P0-CHAIN-001, GW-202 (partial)
- **Sprint 5:** P0-CHAIN-002, CHAIN-103, GW-201
- **Sprint 6:** P0-CHAIN-003, CHAIN-102 (hooks), OPS (partial)
- **Sprint 7:** P0-P2P-001, P2P-601 (incremental)
- **Sprint 8:** P0-PERF-001, CORE-401 (optional/if required)
- **Sprint 9:** GW-203, GW-204, CLI-501, CLI-502
- **Sprint 10:** P0-OPS-001 (+ remaining closure)

---

## Execution Status (Repo)

As of `main` (Jan 2026), the repo has executed and merged the following sprint branches (used as **shipping increments** toward devnet/beta stability; not all Mainnet DoDs are fully satisfied yet):

- `sprint0-rfc-freeze`: RFC freezes for Mode 2 state, challenge derivation/quotas, and pricing/escrow.
- `sprint1-one-core-foundation`: Mode 2 ingest/upload hardening (one-core migration still PARTIAL).
- `sprint2-economic-model-v1`: enforce elasticity spend caps (full lock-in accounting still PARTIAL).
- `sprint3-mode2-onchain-encoding`: typed Mode 2 slot state scaffolding on-chain.
- `sprint4-mode2-repair-workflows`: generation + slot repair state tracking (replacement policy still PARTIAL).
- `sprint5-unified-liveness-v1`: liveness constraints during repair (quota/challenge derivation still SPECIFIED (RFC)).
- `sprint6-fraud-proofs-evidence`: non-response evidence recording (full evidence taxonomy still PARTIAL).
- `sprint7-deputy-system`: router-side fetch failover (full deputy market still PARTIAL).
- `sprint8-throughput-gpu-defaults`: faster Mode 2 artifact pipeline + WASM UX hardening (GPU KZG still MISSING).
- `sprint9-enterprise-s3-delegation`: deal-backed S3 adapter + docs/sync scripts (polish still PARTIAL).
- `sprint10-mainnet-hardening`: Mode 2 idempotency + CI-aligned E2E stability fixes.
- `sprint11-gap-tracker-refresh`: record repo execution status and tighten P0 status notes.
- `sprint12-mode2-routing-order`: prefer ACTIVE Mode 2 slots for routing/provider ordering.
- `sprint13-e2e-health-readiness`: standardize E2E readiness checks on `/health`.
- `sprint14-mode2-upload-reliability`: avoid Go client-side ContentLength mismatch errors via `Expect: 100-continue`.
- `sprint15-gap-tracker-status`: expand Mainnet gap tracker with per-sprint execution status and DoD mapping.
- `sprint16-e2e-mode2-stripe-stability`: stabilize Mode 2 StripeReplica E2E flows (upload/commit/retrieve) against UI regressions.
- `sprint17-proof-context-cleanup`: replace legacy proofs wiring with LCD retrieval sessions + `useProofs` polling where needed.
- `sprint18-remove-legacy-proofs`: remove stale dashboard UI paths that relied on the old `/proofs` store.
- `sprint19-upload-benchmark`: print upload wall-time and MiB/s in `scripts/e2e_lifecycle.sh` to prevent silent perf regressions.
- `sprint20-mode2-finalize-race`: harden Mode 2 slab finalize against rename races and make finalize idempotent under retries.
- `sprint21-dashboard-cleanup`: restore CI/E2E compatibility by removing redundant dashboard controls and keeping a single transport preference selector.
- `sprint22-wallet-unlock-detection`: detect MetaMask authorization (`eth_accounts`) early so “Create deal” prompts unlock before submit.
- `sprint23-gap-tracker-status`: record sprint22 execution status in the tracker (doc hygiene).
- `sprint24-one-core-payload-ffi`: move NilFS payload encode/decode into `nil_core` FFI to reduce cross-runtime drift.
```

```MAINNET_ECON_PARITY_CHECKLIST.md
# Mainnet Parity + Devnet/Testnet Launch Checklist

Companion docs:
- `notes/mainnet_policy_resolution_jan2026.md` (concrete proposal for “B” + staged plan)
- `MAINNET_GAP_TRACKER.md` (canonical gap tracking + DoDs + test gates)

## Stage 0 — Policy freeze → params + interfaces (unblocks engineering)
- [ ] Extend `nilchain/proto/nilchain/nilchain/v1/params.proto` to encode B1/B2/B4/B5/B6 (with validation + genesis defaults).
- [ ] Encode audit budget sizing/caps (Option A): `audit_budget_bps`, `audit_budget_cap_bps`, and bounded carryover (≤2 epochs) for unused budget.
- [ ] Document chosen defaults + rationale in `notes/mainnet_policy_resolution_jan2026.md` and reference from `MAINNET_GAP_TRACKER.md`.

## Stage 1 — Storage lock-in pricing + escrow accounting (A1)
- [ ] Implement pay-at-ingest lock-in pricing on `UpdateDealContent*` per `rfcs/rfc-pricing-and-escrow-accounting.md` (`nilchain/`).
- [ ] Implement deterministic spend window reset + deterministic elasticity debits (`nilchain/`).
- [ ] Add econ e2e: create deal → upload/commit → verify escrow and module account flows (`scripts/`, `tests/`).

## Stage 2 — Retrieval session economics (A2)
- [ ] Enforce session open burns base fee + locks variable fee; rejects insufficient escrow (`nilchain/`).
- [ ] Enforce completion settlement: burn cut + provider payout; cancel/expiry refunds locked fee only (`nilchain/`).
- [ ] Extend econ e2e: open → complete; open → cancel/expire; verify burns/payouts/refunds (`scripts/`, `tests/`).

## Stage 3 — Deterministic challenge derivation + quotas + synthetic fill (A3)
- [ ] Implement deterministic challenge derivation + quota accounting (SPECIFIED in `rfcs/rfc-challenge-derivation-and-quotas.md`) (`nilchain/`).
- [ ] Implement enforcement outcomes: invalid proof → hard fault; quota shortfall → HealthState decay (no slash by default) (`nilchain/`).
- [ ] Add keeper unit tests for determinism + exclusions (REPAIRING slots excluded).
- [ ] Add adversarial sim test gate for anti-grind properties (`scripts/`, `performance/`).

## Stage 4 — HealthState + eviction curve (A6)
- [ ] Implement per-(deal, provider/slot) HealthState updates from hard/soft failures (`nilchain/`).
- [ ] Implement eviction triggers (`evict_after_missed_epochs_hot/cold`) and hook into Mode 2 repair start (`nilchain/`).
- [ ] Add queries/events for observability; add unit tests.

## Stage 5 — Mode 2 repair + make-before-break replacement (A5)
- [ ] Implement make-before-break replacement per `rfcs/rfc-mode2-onchain-state.md` (`nilchain/`).
- [ ] Implement deterministic candidate selection + churn controls (B4) (`nilchain/`).
- [ ] Ensure reads avoid `REPAIRING` slots; synthetic challenges ignore `REPAIRING`; repairing slots do not earn rewards (`nilchain/`, `nil_gateway/`).
- [ ] Add multi-SP repair e2e: slot failure → candidate catch-up → promotion; reads succeed throughout (`scripts/`, `tests/`).

## Stage 6 — Evidence / fraud proofs pipeline (A4)
- [ ] Implement evidence taxonomy + verification + replay protection (`nilchain/`).
- [ ] Wire penalties (slash/jail/evict) to B1 params; integrate with repair start (`nilchain/`).
- [ ] Add unit tests per evidence type + e2e demonstrating slash on proven bad data (`scripts/`, `tests/`).

## Stage 7 — Deputy market + proxy retrieval + audit debt (P0-P2P-001)
- [ ] Implement deputy/proxy retrieval end-to-end: selection, routing, and settlement (B5) (`nil_p2p/`, `nilchain/`, `nil_gateway/`).
- [ ] Implement proof-of-failure aggregation with threshold/window (B1) and anti-griefing (B5) (`nilchain/`).
- [ ] Add ghosting-provider e2e: still retrieve via deputy and record evidence (`scripts/`).

## B) Policy decisions to encode (proposal summary)

See `notes/mainnet_policy_resolution_jan2026.md` for full details.

- [ ] **B1 Slashing/jailing ladder:** hard faults slash immediately; non-response uses threshold/window; quota shortfall decays HealthState.
- [ ] **B2 Bonding:** base provider bond + assignment collateral scaled by slot bytes and `storage_price`.
- [ ] **B3 Pricing defaults:** derive `storage_price` from GiB-month target; define retrieval base + per-blob + burn bps; define halving interval policy.
- [ ] **B4 Replacement selection:** deterministic candidate ranking seeded by epoch randomness; cooldown + attempt caps.
- [ ] **B5 Deputy incentives:** proxy premium payout + evidence bond/bounty + audit debt funding choice (Option A vs B).
- [ ] **B6 Credits phase-in:** implement accounting first; enable quota reduction caps later (devnet→testnet→mainnet).

## Test gates (launch blockers)
- [ ] Chain econ e2e with multiple parameter sets (`scripts/`, `tests/`).
- [ ] Challenge determinism + anti-grind sim (`scripts/`, `performance/`).
- [ ] Ghosting-provider deputy e2e (`scripts/`).
- [ ] Health/repair e2e (replacement without read outage) (`scripts/`).
```

```notes/mainnet_policy_resolution_jan2026.md
# Mainnet Policy Resolution (Jan 2026, Final Defaults + Implementation Notes)

This document captures **final baseline defaults** (devnet/testnet/mainnet where applicable) for the remaining underspecified Mainnet economics + reliability policies, plus implementation notes and calibration signals.

It is intended to turn the “B) underspecified items” in `MAINNET_ECON_PARITY_CHECKLIST.md` into **explicit parameters and keeper state transitions**.

## Scope

- **Economics:** escrow accounting, lock-in pricing, retrieval fee settlement, inflation/reward schedule hooks
- **Security/evidence:** slashing/jailing/ejection policy ladder, replay protections
- **Reliability:** deterministic repair/replacement selection, health tracking, deputy/proxy market incentives

## Final Defaults (Devnet / Testnet / Mainnet)

These are the baseline parameter defaults to implement and calibrate.

| Topic | Decision | Devnet | Testnet | Mainnet |
|---|---|---:|---:|---:|
| Slashing/jailing | Quota shortfall | no slash (HealthState-only) | same | same |
| Slashing/jailing | `slash_invalid_proof_bps` | 50 (0.5%) | 50 (0.5%) | 50 (0.5%) |
| Slashing/jailing | `slash_wrong_data_bps` | 500 (5%) | 500 (5%) | 500 (5%) |
| Slashing/jailing | `slash_nonresponse_bps` | 100 (1%) | 100 (1%) | 100 (1%) |
| Slashing/jailing | jail params | `3/30/10` epochs | same | same |
| Slashing/jailing | non-response conviction | `threshold=3` in `window=6` epochs | same | same |
| Slashing/jailing | hot/cold eviction | `2` / `6` missed epochs | same | same |
| Bonding | model | base bond + assignment collateral | same | same |
| Bonding | `min_provider_bond` | 100 `stake` | 100 `stake` | 10,000 `NIL` |
| Bonding | `bond_months` | 2 | 2 | 2 |
| Bonding | unbonding | `provider_unbonding_blocks = MONTH_LEN_BLOCKS` | same | same |
| Pricing | `target_GiBMonth_price` | 0.10 | 0.10 | 1.00 |
| Pricing | `target_GiBRetrieval_price` | 0.05 | 0.05 | 0.10 |
| Pricing | `base_retrieval_fee` | 0.0001 NIL | 0.0001 NIL | 0.0002 NIL |
| Pricing | `retrieval_burn_bps` | 500 (5%) | 500 (5%) | 1000 (10%) |
| Replacement | cooldown | per-slot, 7 days | same | same |
| Replacement | attempt cap | 3 / window | same | same |
| Deputy | audit debt funding | Option A (protocol-funded audit budget) | same | same |
| Deputy | audit budget sizing | `audit_budget_bps=200`, cap `500`, carryover≤2 epochs | same | `audit_budget_bps=100`, cap `200`, carryover≤2 epochs |
| Deputy | proxy premium (`premium_bps`) | 2000 (20%) | 2000 (20%) | 1000 (10%) |
| Deputy | evidence incentives | `evidence_bond=0.01`, `failure_bounty=0.02` | same | same |
| Deputy | evidence bond burn on no conviction | burn 50% on TTL expiry | same | same |
| Credits | phase-in | accounting only; caps=0 | enabled w/ caps | disabled at launch; caps=0 |
| Credits | caps (hot/cold) | `0/0` | `2500/1000` | launch `0/0` → later `5000/2500` |

## Implementation Note: Params That Exist Today vs Proposed Additions

The current on-chain params are defined in `nilchain/proto/nilchain/nilchain/v1/params.proto` and already include (non-exhaustive):
- `storage_price`, `base_retrieval_fee`, `retrieval_price_per_blob`, `retrieval_burn_bps`
- `month_len_blocks`, `epoch_len_blocks`
- `quota_bps_per_epoch_hot/cold`, `quota_min_blobs`, `quota_max_blobs`
- `credit_cap_bps`
- `evict_after_missed_epochs` (single value; proposal suggests a hot/cold split)

This proposal introduces additional parameters (slashing/jailing, bonding, replacement cooldown/attempt caps, deputy premiums, evidence incentives, and credit cap splits). These require **adding new fields** to `Params` (and wiring validation/defaults) before keeper logic can rely on them.

## B) Underspecified Items — Proposed Resolutions

### B1) Slashing + jailing policy (hard vs soft failures)

**Intent:**
- **Hard faults** (cryptographically verifiable) are slashable immediately.
- **Soft faults** (statistical / threshold-verifiable) should not slash on a single report; use a threshold within a window; otherwise decay HealthState and eventually repair/evict.
- **Quota shortfall** is a *soft* failure: default is **no slash**, only HealthState decay + repair trigger.

**Evidence classes:**
1) **Hard-fault (chain-verifiable):**
   - Invalid synthetic proof (verification fails)
   - Wrong data fraud proof (bytes/proof mismatch)
   - **Action:** immediate slash + jail + trigger slot repair
2) **Soft-fault (threshold-verifiable):**
   - Non-response proof-of-failure (deputy transcript hash + attestation)
   - **Action:** convict only after distinct failures exceed threshold within window; otherwise HealthState decay
3) **Protocol non-compliance (no evidence):**
   - Quota shortfall at epoch end
   - **Action:** HealthState decay; repair trigger after `evict_after_missed_epochs_*`

**Proposed params (defaults):**
| Param | Default | Meaning |
|---|---:|---|
| `slash_invalid_proof_bps` | 50 | 0.5% slash on invalid proof (hard-fault) |
| `slash_wrong_data_bps` | 500 | 5% slash on wrong data proof (hard-fault) |
| `slash_nonresponse_bps` | 100 | 1% slash once non-response conviction triggers |
| `jail_invalid_proof_epochs` | 3 | jail duration after invalid proof |
| `jail_wrong_data_epochs` | 30 | jail duration after wrong-data fraud proof |
| `jail_nonresponse_epochs` | 10 | jail duration after confirmed non-response |
| `nonresponse_threshold` | 3 | ≥3 distinct failures needed to convict |
| `nonresponse_window_epochs` | 6 | failures must occur within this window |
| `evict_after_missed_epochs_hot` | 2 | hot deals: start repair after 2 missed epochs |
| `evict_after_missed_epochs_cold` | 6 | cold deals: start repair after 6 missed epochs |
| `max_strikes_before_global_jail` | 10 | global jail after repeated repair triggers |
| `strike_window_epochs` | 100 | rolling window for “strikes” |

Notes:
- Splitting `evict_after_missed_epochs` by service class (“hot/cold”) is recommended so sensitivity matches quota rates.
- Values are **starting defaults**; expect calibration during testnet.
- Jail params are expressed in **epochs**, but should be enforced using **block height** (e.g., `jail_end_height = now + jail_epochs*epoch_len_blocks`) to avoid ambiguity if epoch params change later.

### B2) Provider staking / bond requirements

**Goal:** slashing must be economically material and scale with responsibility.

**Proposed model (two-layer bond):**
1) **Base provider bond** (anti-sybil, minimum skin-in-the-game)
   - `min_provider_bond` default: 10,000 NIL (mainnet), 100 stake (devnet/testnet)
2) **Assignment collateral requirement** (scales with slot-responsible bytes)
   - Define:
     - `slot_bytes(deal, slot)` from Mode 2 profile (or Mode 1 full replica bytes)
     - `MONTH_LEN_BLOCKS` protocol param
   - Require:
     - `required_bond = ceil(bond_months * storage_price * MONTH_LEN_BLOCKS * slot_bytes)`
   - `bond_months` default: 2
3) **Unbonding / lock**
   - `provider_unbonding_blocks` default: `MONTH_LEN_BLOCKS`
   - provider cannot drop below requirement while assigned to active slots (or while a pending repair candidate)
4) **Failure handling**
   - if provider bond < required: ineligible for new assignments; can trigger eviction on affected deals

Fallback (simpler, weaker): flat bond only (no assignment collateral).

### B3) Pricing parameters + equilibrium targets

**Accounting contract (frozen):** see `rfcs/rfc-pricing-and-escrow-accounting.md`.

**Deriving storage price from “GiB-month”:**
- `storage_price = target_GiBMonth_price / (GiB * MONTH_LEN_BLOCKS)`

**Proposed defaults:**
- Devnet/testnet: `target_GiBMonth_price = 0.10 NIL / GiB-month`
- Mainnet: `target_GiBMonth_price = 1.00 NIL / GiB-month`

**Retrieval fees:**
- Base fee (burned): `base_retrieval_fee`
  - Dev/test: 0.0001 NIL
  - Mainnet: 0.0002 NIL
  - Rationale: keep “base fee share” under ~20% for typical 1–10 MiB reads; monitor spam metrics closely.
- Variable fee (locked at open, settled at completion): `retrieval_price_per_blob` per 128 KiB blob
  - derive from GiB retrieval target:
    - `retrieval_price_per_blob ≈ target_GiBRetrieval_price / 8192`
  - Dev/test: `target_GiBRetrieval_price = 0.05 NIL / GiB`
  - Mainnet: `target_GiBRetrieval_price = 0.10 NIL / GiB`
- Burn cut on completion: `retrieval_burn_bps`
  - Dev/test: 500 (5%)
  - Mainnet: 1000 (10%)

**Inflation decay / halving schedule:**
- Keep `HalvingIntervalBlocks` roughly “1 year in blocks” as a sticky parameter; allow governance to adjust base reward but avoid frequent halving-interval changes.

### B4) Repair/replacement selection policy (deterministic, anti-grind)

**Trigger repair when:**
- hard-fault evidence occurs (immediate), or
- `missed_epochs > evict_after_missed_epochs_{hot,cold}` (from HealthState)

**Deterministic candidate selection:**
- seed:
  - `seed = SHA256("nilstore/replace/v1" || R_e || deal_id || slot || current_gen || replace_nonce)`
- rank provider registry by `SHA256(seed || provider_addr)` and choose first eligible.

**Eligibility filter:**
- not jailed
- sufficient capacity (if tracked)
- sufficient bond (B2)
- not already in deal (including pending provider)
- meets protocol version constraints

**Anti-churn controls (proposed params):**
| Param | Default | Meaning |
|---|---:|---|
| `replacement_cooldown_blocks` | 7 days in blocks | limit replacement churn per slot |
| `max_repair_attempts_per_slot_per_window` | 3 | cap candidate attempts |
| `repair_attempt_window_blocks` | `MONTH_LEN_BLOCKS` | rolling window for attempts |

**Repeated failure fallback (behavioral rule):**
- After a slot hits `max_repair_attempts_per_slot_per_window`, enter a **repair backoff** until the attempt window resets (avoid thrash), and emit an operator-visible alert/event.
- Optional testnet ops escape hatch: a “trusted/top-bonded allowlist” override. On mainnet this must be governance-controlled (or omitted).

### B5) Deputy market compensation + evidence incentives + audit debt funding

**Proxy retrieval payment (premium):**
- Open proxy session locks `base_fee + variable_fee + premium_fee` from deal escrow.
- `premium_fee = ceil(variable_fee * premium_bps / 10_000)`
- Proposed `premium_bps`:
  - Dev/test: 2000 (20%)
  - Mainnet: 1000 (10%)
- On success: provider paid as normal; deputy receives `premium_fee`.

**Evidence incentives (non-response):**
- require deputy to lock `evidence_bond` when submitting proof-of-failure
- if conviction triggers within window: refund bond + pay `failure_bounty`
- if not convicted within window: partially burn bond (anti-grief)
- baseline default: burn **50%** of `evidence_bond` on TTL expiry and refund 50% (discourages spam without chilling reporting).

Suggested param for implementation:
- `evidence_bond_burn_bps_on_expiry = 5000` (burn 50% when a proof-of-failure does not result in conviction within TTL).

Proposed defaults:
| Param | Default |
|---|---:|
| `evidence_bond` | 0.01 NIL |
| `failure_bounty` | 0.02 NIL |
| `proof_of_failure_ttl_epochs` | `nonresponse_window_epochs` |

**Audit debt funding options:**
- Option A (recommended): protocol-funded audit budget (minted per epoch) pays audit retrieval traffic.
- Option B: SP-funded audits, reimbursed via storage rewards (simpler, more liquidity pressure).

**Option A implementation (closed): audit budget sizing + caps**

Define an “epoch slot rent” baseline:
- `epoch_slot_rent = storage_price * total_active_slot_bytes * epoch_len_blocks`

Mint audit budget as a bounded fraction of `epoch_slot_rent`:
- `audit_budget_mint = ceil(audit_budget_bps / 10_000 * epoch_slot_rent)`
- hard cap: `audit_budget_mint <= ceil(audit_budget_cap_bps / 10_000 * epoch_slot_rent)`
- carryover: allow unused budget to roll forward up to `audit_budget_carryover_epochs = 2` epochs (avoid unbounded accumulation).

Proposed params:
- Devnet/testnet: `audit_budget_bps=200` (2%), `audit_budget_cap_bps=500` (5%), `audit_budget_carryover_epochs=2`
- Mainnet: `audit_budget_bps=100` (1%), `audit_budget_cap_bps=200` (2%), `audit_budget_carryover_epochs=2`

Implementation note:
- `total_active_slot_bytes` should be computed deterministically from chain state (Mode 2 slots in `ACTIVE`, plus Mode 1 assignments), and must exclude `REPAIRING` slots.

### B6) Organic retrieval credits (quota reduction) — accrual + caps + phase-in

Adopt credit accrual rules per `rfcs/rfc-challenge-derivation-and-quotas.md`.

**Proposed caps:**
- `credit_cap_bps_hot = 5000` (up to 50% quota via credits)
- `credit_cap_bps_cold = 2500` (up to 25% quota via credits)

**Phase-in plan:**
- Devnet: implement accounting, set credit caps to 0 (no quota reduction yet)
- Testnet: enable conservative caps (hot 25%, cold 10%)
- Mainnet: **launch with caps = 0**; enable after determinism + evidence gates are green; then increase to target caps (hot 50%, cold 25%)

## Calibration Signals (Testnet Monitoring)

These are recommended dashboards/alerts before changing defaults.

### Slashing + jailing
- Invalid proof rate: target <0.1%, alert >0.5%.
- Wrong-data convictions: target ~0; any non-zero is severity-1 triage.
- Non-response conviction rate: target <1% of sessions, alert >3%.
- Jailed provider share: target <5%, alert >10% sustained.
- Repair triggers/day from soft failures: hot target <0.5%/day, cold <0.2%/day.

### Provider bonding
- Participation: active providers with bond ≥ min and meeting collateral requirement (expect growth; alert on plateau).
- Candidate rejected for insufficient bond: target ~0 after initial week; alert >1% of selections.
- Bond headroom distribution: target median >25%; alert if many near ~0%.
- Assignment concentration: top-10 providers’ share of slot bytes (target <60% early; alert if increasing).

### Pricing
- Affordability: median escrow duration at creation ≥ requested duration; alert on systematic underfunding.
- Retrieval spam: sessions opened per block per address; alert if one address dominates (>5%/hour).
- Base fee share for 1–10 MiB reads: target <20%; alert if base fee dominates typical reads.
- Burn/mint ratio: track; alert if burn ≈0 (no sink) or >30% (may starve incentives).

### Replacement + churn
- Repair completion latency (start→promotion): track median/P95 by service class.
- First-candidate success rate: target >70%; alert <40%.
- Replacements per slot per month: target <0.2; alert >1.0.
- Slots hitting attempt cap: target ~0; alert on repeated caps (tooling/eligibility issues).

### Deputy + audit debt
- Proxy success rate: target >99%; track time-to-first-byte P95 vs SLA.
- Deputy-served fraction of retrievals: target <1%; alert >5%.
- Evidence quality: convictions/submissions target 30–70%; alert <10% (spam) or >90% (systemic outage).
- Audit debt backlog: target clears in <2 epochs; alert if sustained growth.
- Audit budget utilization: `spent/minted` per epoch; alert if >95% (cap binding) or <10% sustained (overmint or not used).
- Audit budget fairness: distribution of audit spend across providers; alert if top-10 consume >60% without matching slot-byte share.

### Credits
- Credit usage vs cap: monitor `credits_blobs/quota_blobs` by hot/cold; alert if many hit cap immediately.
- Synthetic coverage floor: hot ≥50%, cold ≥75% (given caps).
- Duplicate attempts rate: repeated credit ids rejected (wash indicators).
- State growth: per-epoch credit uniqueness set size; alert if pruning lags.

## A) Delivery Plan — Staged Roadmap (Test-Gated)

This aligns with the “A) well-defined steps” in `MAINNET_ECON_PARITY_CHECKLIST.md`.

0) Policy freeze → encode params + interfaces (unblocks engineering)
1) Storage lock-in pricing + escrow accounting + spend windows
2) Retrieval session fee lifecycle (burn/lock/settle/refund)
3) Deterministic challenge derivation + quotas + synthetic fill scheduling
4) HealthState + eviction curve (soft failures → repair triggers)
5) Mode 2 make-before-break repair + promotion + read routing around REPAIRING
6) Evidence / fraud proofs pipeline (verify + replay-protect + penalty wiring)
7) Deputy market + audit debt end-to-end (proxy retrieval + evidence aggregation + compensation)

Each stage should ship with its own test gate (keeper unit tests and/or e2e scripts), as specified in `MAINNET_GAP_TRACKER.md`.

## Risks if policy is deferred (top 5)

1) Slashing not economically material → “honesty is optional.”
2) Undercollateralized providers → slashing does not deter large deal cheating.
3) Replacement grinding/churn → capture or instability via repeated replacements.
4) Deputy market never clears → ghosting providers become unrecoverable outages.
5) Quota/credit instability → either no coverage (too many credits) or too strict (provider churn).

## Open items (explicitly contentious)

These are “agree on targets” items rather than “can’t implement” items:
- the exact **bps** values and jail durations (B1) vs observed fault rates
- bond sizes (B2) vs operator constraints on testnet
- pricing targets (B3) vs target UX and provider costs
- base retrieval fee level (B3): baseline is low; if spam emerges, increase carefully to preserve small-read UX
- evidence-bond burn fraction (B5): baseline is 50% but can be tuned if it chills reporting or invites spam
- credit cap phase-in schedule (B6) vs measurable determinism confidence
- “trusted allowlist override” for repeated repair failures: whether to allow on testnet, and how it is governance-gated (or omitted) on mainnet
```

```ECONOMY.md
# NilStore Economy & Tokenomics

## Overview

The NilStore economy is designed to align incentives between Storage Providers (SPs), Data Owners (Users), and the Protocol itself using a single utility token: **$NIL** ($STOR). The model enforces physical infrastructure commitment while enabling elastic, user-funded scaling.

## 1. The Performance Market (Proof-of-Useful-Data)

Unlike "Space Race" models that reward random data filling, NilStore rewards **latency**.

### 1.1 Unified Liveness
Storage proofs (`MsgProveLiveness`) serve two functions:
1.  **Storage Audit:** Proves the SP holds the data (PoUD via KZG).
2.  **Performance Check:** The block height of proof inclusion determines the reward tier.

### 1.2 Tiered Rewards
Rewards are calculated based on the delay between the **Challenge Block** and the **Proof Inclusion Block**.

**Note:** The tier windows and multipliers below are illustrative examples; the canonical tier cutoffs are protocol parameters (see `spec.md`).

| Tier | Latency (Blocks) | Reward Multiplier | Requirement |
| :--- | :--- | :--- | :--- |
| **Platinum** | 0 - 1 | 100% | NVMe / RAM |
| **Gold** | 2 - 5 | 80% | SSD |
| **Silver** | 6 - 10 | 50% | HDD |
| **Fail** | > 10 | 0% (Slash) | Offline / Glacier |

### 1.3 Inflationary Decay
The base reward per proof follows a halving schedule to cap total supply.
`Reward = BaseReward * (1 / 2 ^ (BlockHeight / HalvingInterval))`

## 2. Elasticity & Scaling

NilStore allows data to scale automatically to meet demand without manual intervention.

### 2.1 Virtual Stripes
A file is stored on a "Stripe" (12 providers). If these providers become saturated (high latency or load), they can signal saturation (`MsgSignalSaturation`).

### 2.2 The Budget Check
The protocol checks the Data Owner's `MaxMonthlySpend` limit.
*   **If Budget Allows:** The protocol spawns a new "Virtual Stripe" (12 new providers) and replicates the data "Hot".
*   **If Budget Exceeded:** The scaling request is denied to protect the user's wallet.

## 3. Token Flow

### 3.1 Inflow (Users)
Users fund deals by depositing $NIL into **Escrow**.
*   `MsgCreateDeal`: Initial deposit.
*   `MsgAddCredit`: Top-up escrow.

### 3.2 Outflow (Providers)
Providers earn tokens via:
1.  **Inflation:** Minted $NIL for valid proofs (Base Capacity Reward).
2.  **Bandwidth Fees:** Paid from User Escrow for retrieval receipts.

### 3.3 Sinks (Burning)
*   **Slashing:** Example policy: missed proofs / non-response violations trigger a slash and potential jailing. Exact windows and amounts are protocol parameters.
*   **Burner:** The `nilchain` module has burn permissions to remove slashed assets from circulation.

## 5. Protocol Parameters (Proposal Defaults)

This section records **baseline defaults** intended to unblock implementation and testnet calibration.

Canonical accounting rules are frozen in `rfcs/rfc-pricing-and-escrow-accounting.md`. Policy defaults and open questions are tracked in `notes/mainnet_policy_resolution_jan2026.md`.

### 5.1 Storage Price (Lock-in at Ingest)

Derive `storage_price` (Dec per byte per block) from a human target “GiB-month price”:

`storage_price = target_GiBMonth_price / (GiB * MONTH_LEN_BLOCKS)`

Proposed targets:
- Devnet/testnet: `0.10 NIL / GiB-month`
- Mainnet: `1.00 NIL / GiB-month`

### 5.2 Retrieval Fees (Session Settlement)

- `base_retrieval_fee`: burned at session open (anti-spam).
  - Devnet/testnet: `0.0001 NIL`
  - Mainnet: `0.0002 NIL`
- `retrieval_price_per_blob`: locked at session open; settled at completion; per `128 KiB` blob.
  - derive from a GiB target: `retrieval_price_per_blob ≈ target_GiBRetrieval_price / 8192`
  - Devnet/testnet: `0.05 NIL / GiB`
  - Mainnet: `0.10 NIL / GiB`
- `retrieval_burn_bps`: burn cut on completion.
  - Devnet/testnet: `500` (5%)
  - Mainnet: `1000` (10%)

### 5.3 Slashing/Jailing Ladder (Hard vs Soft Failures)

Proposed intent:
- Invalid proofs / wrong-data proofs are **hard faults** (slash immediately).
- Non-response is **thresholded** (convict only after N failures within a window).
- Quota shortfall is **soft** (HealthState decay → repair/evict; no slash by default).

See `notes/mainnet_policy_resolution_jan2026.md` for the proposed parameter table.

### 5.4 Provider Bonding

Proposed model:
- a base provider bond (anti-sybil), plus
- assignment collateral scaled by slot bytes and `storage_price`.

See `notes/mainnet_policy_resolution_jan2026.md`.

### 5.5 Deputy Market + Audit Debt (Defaults)

Baseline decisions:
- Audit debt funding: Option A (protocol-funded audit budget).
- Proxy retrieval premium: 20% (devnet/testnet), 10% (mainnet).
- Non-response evidence incentives: `evidence_bond=0.01 NIL`, `failure_bounty=0.02 NIL`, burn 50% of evidence bond on TTL expiry.

Audit budget sizing (Option A):
- Define: `epoch_slot_rent = storage_price * total_active_slot_bytes * epoch_len_blocks`
- Mint: `audit_budget_mint = ceil(audit_budget_bps/10_000 * epoch_slot_rent)`, capped by `audit_budget_cap_bps`.
- Carryover: unused budget may roll forward up to 2 epochs (bounded).
- Defaults:
  - Devnet/testnet: `audit_budget_bps=200`, `audit_budget_cap_bps=500`, carryover≤2 epochs
  - Mainnet: `audit_budget_bps=100`, `audit_budget_cap_bps=200`, carryover≤2 epochs

See `notes/mainnet_policy_resolution_jan2026.md`.

### 5.6 Credits (Organic Retrieval → Quota Reduction)

Baseline phase-in:
- Devnet: accounting only; credits do not reduce quota (caps=0).
- Testnet: credits enabled with conservative caps (hot 25%, cold 10%).
- Mainnet: launch with caps=0; enable later after determinism + evidence gates are green.

See `notes/mainnet_policy_resolution_jan2026.md`.

## 4. S3 Adapter (Web2 Gateway)

The `nil_gateway` adapter allows Web2 applications to write to NilStore using standard S3 APIs.
*   **PUT:** Shards file -> Computes KZG -> Creates Deal on Chain.
*   **GET:** Retrieves shards -> Verifies KZG -> Reconstructs File.
```

```spec.md
# NilStore Core v 2.4

### Cryptographic Primitives & Proof System Specification

---

## Abstract

NilStore is a decentralized storage network that unifies **Storage** and **Retrieval** into a single **Demand-Driven Performance Market**. Instead of treating storage audits and user retrievals as separate events, NilStore implements a **Unified Liveness Protocol**: user retrievals *are* storage proofs.

It specifies:
1.  **Unified Liveness:** Organic user retrieval sessions act as valid storage proofs.
2.  **Synthetic Challenges:** The system acts as the "User of Last Resort" for cold data.
3.  **Tiered Rewards:** Storage rewards are tiered by latency.
4.  **System-Defined Placement:** Deterministic assignment to ensure diversity, optimized by **Service Hints**.
5.  **Traffic Management:** User-funded **Elastic Scaling** triggered by **Saturation Signals** from Providers.

---

## § 1 Overview (Meta-Specification)

NilStore’s protocol design is guided by a small set of architectural tenets:

1.  **Retrieval IS Storage:** completed retrieval sessions count as valid storage proofs.
2.  **The System is the User of Last Resort:** cold data is maintained via synthetic challenges when organic demand is low.
3.  **Optimization via Hints:** clients express intent (`Hot`/`Cold`) while the chain enforces system-defined placement and diversity.
4.  **Elasticity is User-Funded:** bandwidth and replication are increased only when the user’s escrow/budget can pay for it.

---

## § 2 The Deal Object (Conceptual)

The `Deal` is the central on-chain state object. This spec describes its semantics without requiring an exact protobuf layout.

Key fields:
*   **Identity:** `deal_id` (uint64), `owner` (address).
*   **Commitment Root:** `manifest_root` (48‑byte KZG commitment, BLS12‑381 G1 compressed). This is the protocol’s anchor for all proofs (§7.3).
*   **Provisioning:** thin-provisioned container expanded only via content commits (§6.0.3).
    *   **Logical size:** `Deal.size` / `size_bytes` (sum of non-tombstone NilFS file lengths).
    *   **Slab bounds:** `Deal.total_mdus` (count of committed MDU roots in the Manifest commitment; includes MDU #0 + witness + user MDUs).
    *   **Metadata size:** `Deal.witness_mdus` (count of witness MDUs after MDU #0; required to derive the user‑MDU range).
    *   **Gateway compat:** some REST responses may include legacy `allocated_length` as an alias for `total_mdus` (count), not bytes.
*   **Placement:** `providers[]` is the assigned provider set.
    *   **Mode 1:** unordered replica set; any single provider can satisfy retrievals.
    *   **Mode 2:** ordered slot list `slot → provider` of length `N = K+M` (§7.1.1, §8.1.3).
*   **Service Hint:** `Hot | Cold` informs placement/elasticity policy (§6.0.2).
*   **Economics:** `escrow` (combined storage + bandwidth), plus `max_monthly_spend` for user-funded elasticity (§6.1.2).
*   **Redundancy Mode:** Mode 1 (FullReplica) or Mode 2 (StripeReplica / RS(K,K+M)) (§6.2, §8).

Constants:
*   `MDU_SIZE = 8,388,608` bytes (8 MiB) is an immutable protocol constant.
*   `BLOB_SIZE = 128 KiB` is the cryptographic atom for KZG verification (§8.1.1).

---

## § 3 System-Defined Placement (Conceptual)

At a high level, provider selection is deterministic and anti-sybil:

**Function (conceptual):** `AssignProviders(deal_id, epoch_seed, active_set, hint)`

1.  **Filter:** select candidates consistent with the Deal’s `ServiceHint` and provider capabilities (§6.0.1–6.0.2).
2.  **Seed:** derive a deterministic seed from `(deal_id, chain randomness)`.
3.  **Select:** sample providers deterministically from the candidate set.
4.  **Diversity:** enforce distinct failure domains (e.g., ASN/subnet) subject to bootstrap constraints (§5.1).

This section is intentionally conceptual; concrete placement optimization is an RFC target (Appendix B).

---

## § 4 Economics & Flow Control (Conceptual)

NilStore’s economics combine a performance market with user-funded scaling:

### 4.1 Tiered Rewards (Parameters)
Providers are rewarded by observed inclusion/latency tiers (e.g., Platinum/Gold/Silver/Fail). Exact tier windows and multipliers are protocol parameters (Appendix B).

### 4.2 Saturation & Elasticity (Parameters)
Providers may signal saturation to trigger user-funded replica/overlay expansion, subject to damping and a minimum TTL to respect data gravity (§6.1–6.2).

### 4.3 Rotation (Planned)
The protocol anticipates rotation/rebalancing flows where an old provider is only released after a new provider proves readiness (“make-before-break”) (§5.3).

---

## § 5 System Constraints & Meta-Risks (Planned Safeguards)

This section documents accepted architectural risks and required safeguards.

### 5.1 Cold Start Fragility (Bootstrap Mode)
*   **Risk:** system-defined placement assumes a large, diverse active set. When the active set is small (early testnet), strict diversity constraints may be impossible to satisfy.
*   **Safeguard:** the chain SHOULD support a governance-gated **Bootstrap Mode** that relaxes diversity constraints until `ActiveSetSize > Threshold`.

### 5.2 Viral Debt Risk (Third-Party Sponsorship)
*   **Risk:** user-funded elasticity creates a hard stop; if escrow is depleted during a viral event, content throttles.
*   **Assessment:** this is an acceptable economic state (“you get what you pay for”), but the protocol SHOULD support **third-party sponsorship** (e.g., `MsgFundEscrow`) to let communities fund important content.

### 5.3 Data Gravity & Non-Atomic Migration (Make-Before-Break)
*   **Risk:** moving data takes time; when a provider is rotated or replaced, there is a gap before the new provider is ready.
*   **Safeguard:** migration MUST be overlapping: the old provider is not removed until the new provider submits an initial valid proof at the current generation (§8.4).

### 5.4 Economic Sybil Assumption (Wash-Traffic)
*   **Risk:** unified liveness could be exploited via fake traffic.
*   **Safeguard:** (1) retrieval sessions require on-chain Data Owner authorization; (2) data is stored as ciphertext; (3) protocol burn/debit ensures wash-trading has a real cost.

## § 6 Product-Aligned Economics
*(This section’s economic rationale is expanded in [RFC: Data Granularity & Economic Model](rfcs/rfc-data-granularity-and-economics.md). Legacy “capacity tiers / DealSize” language is deprecated; the normative semantics are **thin provisioning** with a per‑deal hard cap.)*

### 6.0 System-Defined Placement (Anti-Sybil & Hints)

To prevent "Self-Dealing," clients cannot choose their SPs. However, to optimize performance, the selection algorithm respects **Service Hints**.

#### 6.0.1 Provider Capabilities
When registering, SPs declare their intended service mode via `MsgRegisterProvider(Capabilities)`:
*   **Archive:** High capacity, standard latency.
*   **General (Default):** Balanced.
*   **Edge:** Low capacity, ultra-low latency.

#### 6.0.2 Deal Hints
`MsgCreateDeal` includes a `ServiceHint`:
*   **Cold:** Biased towards `Archive` / `General`.
*   **Hot:** Biased towards `General` / `Edge`.

#### 6.0.3 Deal Sizing (Dynamic)
NilStore utilizes **Dynamic Thin Provisioning** for all storage deals.

*   **No Tiers:** Users do not pre-select a capacity tier.
*   **Dynamic Expansion:** Deals start with minimal state and automatically expand as content is added via `MsgUpdateDealContent`.
*   **Thin-Provision Semantics:** `MsgCreateDeal*` creates a deal with `manifest_root = empty`, `size = 0`, and `total_mdus = 0` until the first `MsgUpdateDealContent*` commits content.
*   **Hard Cap:** The protocol enforces a maximum capacity of **512 GiB** per Deal ID to prevent state bloat and ensure manageable failure domains. Large datasets should be split across multiple Deals.

The `MDU_SIZE` (Mega-Data Unit) remains an immutable protocol constant of **8,388,608 bytes (8 MiB)**.

### 6.1 The Unified Market & Elasticity

#### 6.1.1 Traffic Management (Saturation)
To prevent punishment of high-performing nodes during viral events, the protocol supports **Pre-emptive Scaling**.

1.  **Saturation Signal:** An SP submits `MsgSignalSaturation(DealID)`.
    *   *Condition:* SP must be currently **Platinum/Gold** and have high retrieval session volume.
2.  **Action:** The Chain increases `Deal.CurrentReplication` (e.g., 12 -> 15) and triggers `SystemPlacement` to recruit **Edge** nodes.
3.  **Incentive:** The signaling SP is NOT penalized. They maintain their tier on manageable traffic, while overflow is routed to new replicas.

#### 6.1.2 User-Funded Elasticity
Scaling is not free. It is strictly constrained by the User's budget.

*   **Funding Source:** `Deal.Escrow`.
*   **Budget Cap:** `Deal.MaxMonthlySpend`.
*   **Logic:**
    *   If `Escrow > Cost(NewReplica)` AND `Spend < Cap`: **Spawn Replica.**
    *   Else: **Reject Scaling.** The file becomes rate-limited naturally.

### 6.2 Auto-Scaling (Stripe-Aligned Elasticity)

NilStore supports two redundancy modes at the policy level:

*   **Mode 1 – FullReplica (Alpha):** Each `Deal` is replicated in full across `CurrentReplication` providers. Scaling simply adds or removes full replicas. Retrieval is satisfied by any single provider in `Deal.providers[]`.
*   **Mode 2 – StripeReplica (Implemented):** Each `Deal` is encoded per SP‑MDU under **RS(K, K+M)** (K data slots, M parity slots; default `K=8`, `M=4`, with `K | 64`). Providers store per‑slot shard Blobs for each SP‑MDU, and scaling operates at the stripe layer. This mode uses the **Blob‑Aligned Striping** model defined in **§ 8**.

**Profile selection (current implementation):** the RS profile is encoded in `service_hint` as `rs=K+M` (for example, `General:replicas=12,rs=8+4`). If `rs=` is present, the chain treats the deal as **Mode 2** and assigns `N = K+M` ordered providers as slots.

To ensure effective throughput scaling, the protocol avoids "bottlenecking" by scaling the entire dataset uniformly.

#### 6.2.1 The Stripe Unit
*   **Principle:** Increasing the capacity of Shard #1 does not help if Shards #2-12 are saturated.
*   **Mechanism (Mode 2):** Scaling operations occur in **Stripe Units**. When triggered, the protocol recruits `n` new Overlay Providers, creating one new replica for *each* shard index. In Mode 1, this is approximated by adding `n` full replicas (additional providers in `Deal.providers[]`) without per-stripe awareness.

#### 6.2.2 Damping & Hysteresis (Intelligent Triggers)
To prevent oscillation (rapidly spinning nodes up and down) and account for the cost of data transfer:
1.  **Trigger:** The protocol tracks the **Exponential Moving Average (EMA)** of retrieval session volume.
    *   **Scale Up:** If `Load > 80%` of current capacity.
    *   **Scale Down:** If `Load < 30%` of current capacity.
2.  **Minimum TTL (Data Gravity):** New Overlay Replicas have a mandatory **Minimum TTL** (e.g., 24 hours).
    *   *Rationale:* Moving data consumes network resources. Spawning a replica is an "investment" that must be amortized over a minimum service period.
    *   *Cost:* The User's escrow is debited for this minimum period upon spawn.

### 6.3 Deletion (Crypto-Erasure)
*   **Mechanism:** True physical deletion cannot be proven. NilStore relies on **Crypto-Erasure**.
*   **Process:** To "delete" a file, the Data Owner destroys their copy of the `FMK`. Without this key, the stored ciphertext is statistically indistinguishable from random noise.
*   **Garbage Collection:** When a Deal is cancelled (`MsgCancelDeal`) or expires, SPs act economically: they delete the data to free up space for paying content.

## § 8 Mode 2: StripeReplica & Erasure Coding (Normative Extension)

This section norms the **Blob-Aligned Striping** model required for Mode 2 operation, resolving the conflict between cryptographic verification (KZG) and network distribution (Erasure Coding).

### 8.0 Mode 2 Ingestion (Gateway-Optional)
Mode 2 deals require **RS(K, K+M) encoding** of each SP‑MDU before upload. In devnet, Mode 2 ingestion MAY be performed by:
* **Local Gateway (preferred when present):** performs packing, witness generation, RS encoding, and uploads bytes to providers.
* **Browser/WASM (fallback default):** performs the same work in a worker and persists artifacts to OPFS.
* **CLI** tooling for debugging and automation.

The **local gateway is optional** and MUST NOT sign on the user’s behalf. All chain transactions remain user‑signed (MetaMask / wallet).

**Provider role (normative):** providers are a **dumb pipe** for bytes addressed by `(deal_id, mdu_index, slot, manifest_root)` and do not need to understand Mode 1 vs Mode 2 beyond storing and serving the requested objects.

**Determinism & repair confidence:** implementations SHOULD aim for byte‑identical artifact bytes across Gateway and Browser for the same input and RS profile. If strict byte‑identity is not feasible across runtimes, they MUST still agree on all cryptographic commitments (manifest root, MDU roots, witness commitments) so repairs and verification remain correct.

**Devnet UX default (policy):** if a reachable local gateway reports Mode 2 support, clients SHOULD use the gateway path; otherwise clients SHOULD use the browser Mode 2 path (not Mode 1) as the default suggested flow.

**Devnet artifact layout (recommended):** clients SHOULD follow the canonical `mode2-artifacts-v1` contract (`notes/mode2-artifacts-v1.md`) for local persistence and repairs:
* replicated metadata (`mdu_0.bin`, `mdu_1.bin .. mdu_W.bin`, `manifest.bin`), and
* per‑slot user shards (`mdu_<slab_index>_slot_<slot>.bin` where `slab_index = 1 + W + user_ordinal`).

### 8.1 The "Aligned" Striping Model

To enable **Shared-Nothing Verification** (where a provider can verify their own shard without network communication), the atomic unit of striping must match the atomic unit of KZG verification: the **Blob**.

#### 8.1.1 Constants
*   **Blob (Atom):** 128 KiB ($2^{12}$ field elements).
*   **MDU (Retrieval Unit):** 8 MiB (64 Blobs).
*   **Erasure Configuration (Mode 2):** RS(K, K+M) with default `K=8`, `M=4`, and constraint `K | 64`.

#### 8.1.2 The "Card Dealing" Algorithm
An 8 MiB SP‑MDU consists of 64 **data Blobs**. Conceptually, these are a deck of cards (`data_blob_id ∈ [0..63]`) and Mode 2 “deals” them into `K` data slots in *rows* so striping aligns with the Blob‑level KZG atom.

Let:
* `K` = data slots, `M` = parity slots, `N = K+M`
* `rows = 64 / K` (requires `K | 64`)

Define a conceptual matrix of data Blobs `D[row][col]` with:
* `row ∈ [0..rows-1]`, `col ∈ [0..K-1]`
* `data_blob_id = row*K + col`

For each `row`, apply RS(K, K+M) across slots to produce `N` shard Blobs `S[slot][row]`:
* Data slots: `slot ∈ [0..K-1]` correspond to the original `D[row][col]` blobs.
* Parity slots: `slot ∈ [K..N-1]` are parity Blobs derived from the row.

**Benefit:** Each provider stores complete 128 KiB Blobs (its `rows` shards per SP‑MDU), so it can verify and prove each Blob individually using standard KZG.

#### 8.1.3 Locked: Slot-major `leaf_index` ordering

To prioritize the hot-path (serving/proving), Mode 2 uses a **slot-major** canonical leaf ordering for the per-SP‑MDU Merkle tree.

Index spaces:
* `data_blob_id ∈ [0..63]` refers to the 64 logical data Blobs inside the unencoded SP‑MDU (conceptual packing only).
* `leaf_index ∈ [0..L-1]` refers to the Merkle leaf index for the encoded per‑slot shard Blobs.
* In **Mode 2**, `ChainedProof.blob_index` MUST be interpreted as `leaf_index`.

Definitions:
*   `K` = data slots
*   `M` = parity slots
*   `N = K+M` = total slots/providers
*   Constraint: `K | 64` (so `rows` are integral)
*   `rows = 64 / K`
*   `L = N * rows` (Merkle leaves per SP‑MDU in Mode 2)

Leaf mapping (canonical):
*   `leaf_index = slot * rows + row`
*   `slot = leaf_index / rows`
*   `row  = leaf_index % rows`

In this ordering, each provider slot owns a contiguous range of leaf indices for each SP‑MDU, which simplifies witness lookup and on-chain enforcement.

### 8.2 Parity & Homomorphism
To generate the `M` parity Blobs for each `row`:
*   Parity is calculated across the row’s `K` data Blobs (`D[row][0..K-1]`).
*   Due to the homomorphic property of KZG, the Parity Shards are also composed of valid 128 KiB KZG polynomials.
*   Parity Nodes are indistinguishable from Data Nodes in terms of verification logic.

**Determinism (Normative):** For a fixed `(K, M)` profile and the canonical leaf ordering (§8.1.3), RS encoding/decoding MUST be deterministic, so that repairing a missing slot reconstructs a bit‑identical shard Blob to what the evicted provider stored for the same `(mdu_index, leaf_index)`.

### 8.3 Replicated Metadata Policy
To support this model, the "Map" must be fully replicated:
*   **User Data MDUs:** **Striped** (1 slot shard per Provider).
*   **Metadata MDUs (MDU #0 + Witness):** **Fully Replicated** (Copy on All `N = K+M` Providers).

**Witness Expansion:** For each data‑bearing SP‑MDU, the Witness MDUs MUST contain KZG commitments for **ALL `L = (K+M) * (64/K)` shard Blobs** (data + parity). This allows any provider (data or parity) to prove its holding against the global root. (Default `K=8`, `M=4` gives `L=96`.)

**MDU index convention (Mode 2):** NilFS metadata occupies the lowest `mdu_index` values (`MDU #0` first, followed by the Witness MDUs). Synthetic challenges MUST be derived only over striped user‑data MDUs; metadata MDUs are replicated and are not used for per‑slot accountability.

### 8.4 Deal Generations & Repair Mode (Planned, Forward-Compatible)

Mode 2 requires the chain to represent “where the deal is in time” so repairs, reads, and writes can safely overlap.

#### 8.4.1 Deal generation fields (conceptual)
A Mode 2 Deal is associated with a monotonic **generation**:
* `Deal.current_gen` (monotonic counter)
* `Deal.manifest_root` and `Deal.total_mdus` are interpreted as the **current generation**’s committed state.

Any on-chain update that changes `Deal.manifest_root` MUST increment `Deal.current_gen`.

#### 8.4.2 Repair mode (maintenance)
The chain MAY mark one or more provider slots as being in repair:
* `slot_status[slot] ∈ { ACTIVE, REPAIRING }`

While `slot_status[slot] = REPAIRING`:
* **Reads** remain valid and SHOULD route around the repairing slot (fetch any `K` healthy slots per SP‑MDU).
* **Synthetic challenges** MUST NOT target repairing slots; per-slot accountability applies only to ACTIVE slots.
* A liveness proof submitted by a REPAIRING slot MUST be rejected for reward/health accounting (but the underlying proof format remains valid against `Deal.manifest_root`).

When the replacement provider has reconstructed and stored its shard Blobs up to the current generation, the chain transitions the slot back to ACTIVE.

#### 8.4.3 Append-only writes during repair (near-term rule)
To avoid write/repair races while keeping the system usable, Mode 2 supports **append-only** deal updates even while one or more slots are REPAIRING.

An update is append-only iff:
* `new_total_mdus >= old_total_mdus`, and
* for all `mdu_index < old_total_mdus`, the committed MDU roots for those indices are unchanged (only new MDU indices are added).

Append-only updates advance `Deal.current_gen` and `Deal.manifest_root`. Repairing slots simply catch up by reconstructing the newly appended shard Blobs before rejoining ACTIVE.

#### 8.4.4 Future: full versioned writes
In future versions, non-append mutations (rewrite, delete/GC, compaction) SHOULD be represented as a new “pending generation” promoted to current only once placement conditions are met. This generalizes the append-only rule without changing the read/repair model.

## Appendix A: Core Cryptographic Primitives

### A.3 File Manifest & Crypto Policy (Normative)

NilStore MAY use a content‑addressed *file* manifest at the application layer (encryption metadata, UX-level references). This is distinct from the protocol-level Deal commitment (`Deal.manifest_root`, the 48‑byte KZG root used by the Triple Proof) and NilFS path addressing.

**Gateway/API note:** Some app codepaths may still label the deal commitment as a `cid`. In all protocol-facing APIs:

*   `cid` is a legacy alias for the *deal-level* `Deal.manifest_root` (not the Root/DU CIDs below).
*   For REST/path params, `manifest_root` parsing is strict: 48‑byte compressed BLS12‑381 G1 (96 hex chars, optional `0x` prefix), rejecting invalid encodings and invalid subgroup points (return `400`).
*   Retrieval/proof flows are keyed by NilFS `file_path` and validated against `Deal.manifest_root` (no `uploads/index.json` or “single-file deal” fallbacks).
*   `file_path` is **mandatory** and MUST be unique within a deal; uploads to an existing path overwrite deterministically and `GET /gateway/list-files/{manifest_root}` returns a deduplicated view (latest non-tombstone record per path).
*   `file_path` decoding is strict: decode at most once, reject traversal/absolute paths, and beware `+` vs `%20` (clients should use JS `encodeURIComponent`).
*   For devnet convenience endpoints (e.g., `/gateway/fetch/{manifest_root}`, `/gateway/list-files/{manifest_root}`, `/gateway/prove-retrieval`), the gateway MUST (a) require `deal_id` + `owner` for access control and (b) reject stale `manifest_root` values that do not match on-chain deal state (prefer `409`).
*   Retrieval session enforcement (Gamma‑4): when sessions are enabled, data-plane fetches MUST include `X‑Nil‑Session‑Id`, and the server MUST reject out‑of‑session ranges. Proof submission MUST be session‑bound and submitted via `/gateway/session-proof` (forwarded to a provider) or `/sp/session-proof` directly. The gateway is a relay/compute helper only; user authorization lives on‑chain (EVM precompile).
*   Non-200 responses MUST be JSON `{ "error": "...", "hint": "..." }` (even if the success path is a byte stream). Missing/invalid `file_path` returns `400` with a remediation hint (call `/gateway/list-files/{manifest_root}` to discover valid paths).

  * **Root CID** = `Blake2s-256("FILE-MANIFEST-V1" || CanonicalCBOR(manifest))`.
  * **DU CID** = `Blake2s-256("DU-CID-V1" || ciphertext||tag)`.
  * **Encryption:** All data is encrypted client-side before ingress. Deal commitments (and KZG proofs) bind to the **ciphertext bytes**; decryption is purely a client concern.
  * **Metadata confidentiality (optional):** NilFS metadata (MDU #0 and higher-level manifests) MAY be encrypted the same way as file data. If metadata is encrypted, SPs remain oblivious (they store bytes), while clients decrypt after verifying against `Deal.manifest_root`.
  * **Deletion:** Achieved via key destruction (Crypto-Erasure).

## § 7 Retrieval Semantics (Mode 1 Implementation)

This section norms the retrieval path for **Mode 1 – FullReplica** in the current devnet implementation and defines the evidence model used for retrievability and accountability. Several subsections are explicitly marked as planned, forward-compatible extensions.

### 7.0 Core Invariants (Planned, North-Star)

NilStore’s retrieval system is designed to satisfy two invariants:

1.  **Retrievability / Accountability**
    *   For every `(Deal, Provider)` assignment, either:
        *   the encrypted data is reliably retrievable under protocol rules, **or**
        *   there exists high‑probability, verifiable evidence of failure that can be used to penalize and eventually evict the provider.
2.  **Self‑Healing Placement**
    *   Persistently underperforming or malicious providers SHOULD be detected via evidence/health metrics and replaced without manual intervention.

### 7.0.1 Challenge Families (Planned)

To support the invariants, the protocol uses three challenge families, all binding back to the Deal’s on‑chain commitments (§7.3):

1.  **Synthetic Storage Challenges (System‑Driven)**
    *   For each epoch `e` and `(Deal, Provider)`, the chain derives a finite set `S_e(D,P)` of `(mdu_index, blob_index)` pairs from epoch randomness `R_e`.
    *   Providers earn storage rewards by satisfying sufficient synthetic coverage over time (direct synthetic proofs or credited retrieval sessions).
2.  **Retrieval Liveness Challenges (Client / Auditor‑Driven)**
    *   Normal user reads, provider-initiated audits, and third‑party watchers all issue retrieval challenges.
    *   Each retrieval SHOULD map deterministically to a verifiable checkpoint so retrievals can satisfy synthetic demand when aligned with `S_e(D,P)`.
3.  **Escalated On‑Chain Challenges (Panic Mode)**
    *   Watchers MAY post explicit on‑chain challenges that force a provider to respond within a fixed block window; non‑response is hard evidence of unavailability (§7.5).

### 7.1 Data Plane: Fetching From Providers

1.  **Lookup (Deal):** Given a `deal_id`, the client queries chain state for the corresponding `Deal` and reads `Deal.providers[]`.
2.  **Resolve (NilFS):** The requested file within the Deal is identified by `file_path` (NilFS). The client mounts the Deal’s NilFS File Table (MDU #0) to map `file_path` → byte offsets / MDU ranges.
3.  **Selection:** The client selects a single Provider from `Deal.providers[]` (e.g., the nearest or least loaded). In Mode 1, each Provider holds a full replica, so any assigned Provider is sufficient.
4.  **Delivery:** The client fetches the file (or an 8 MiB MDU) from that Provider using an application‑level protocol (HTTP/S3 adapter, gRPC, or a custom P2P layer). The data is served as encrypted MDUs with accompanying KZG proof material. A local gateway may proxy these calls, but it is optional; direct‑to‑provider fetches are first‑class.

In Mode 1, bandwidth aggregation across multiple Providers is **not** required. The protocol only assumes that at least one assigned Provider can serve a valid chunk per retrieval. Mode 2 uses stripe‑aware fetching across any `K` slots per MDU.

#### 7.1.1 Mode 2: Stripe-aware retrieval & challenges

For Mode 2, `Deal.providers[]` is interpreted as an ordered slot list `slot → provider` of length `N = K+M`.

* **Retrieval (hot path):** for each required SP‑MDU, the client fetches shard Blobs for any `K` slots (simple routing: take the first `K` slots by index), verifies each received shard against `Deal.manifest_root` using a `ChainedProof` (with `Proof.blob_index = leaf_index` per §8.1.3), then RS‑decodes to reconstruct the SP‑MDU bytes.
* **Synthetic challenges (accountability):** the protocol derives challenges keyed by `(deal_id, slot)` so every slot is independently accountable. In a Mode 2 proof, the chain enforces that the submitting provider matches the challenged `slot` (see §7.4).

#### 7.1.2 Client bootstrap & caching (Non-normative guidance)

Clients (Gateways, CLIs, browsers) SHOULD treat NilStore as a content-addressed system at the deal layer and cache aggressively:
* **Bootstrap:** given `(deal_id, owner)` and the on-chain `Deal.manifest_root`, a client MUST be able to fetch and verify NilFS metadata (MDU #0 + Witness MDUs) and enumerate valid `file_path` entries without any out-of-band index.
* **Metadata caching:** cache verified metadata by `(deal_id, Deal.current_gen, mdu_index)`; in Mode 2 this is not per-provider because metadata MDUs are replicated and bit-identical across all slots.
* **Browser caching:** when running in-browser, clients SHOULD persist slabs in OPFS to enable gateway‑absent reads and multi‑tab continuity.
* **Data caching:** cache reconstructed plaintext files (or reconstructed SP‑MDUs) behind an LRU keyed by `(deal_id, Deal.current_gen, file_path, byte_range)` to avoid repeated network fetches; revalidation can be performed by re-checking on-chain `Deal.manifest_root` and (optionally) re-verifying proofs on cache fill.

### 7.2 Control Plane: Retrieval Sessions, Proof-of-Retrieval, and Completion (Planned → Mandated)

NilStore’s devnet is converging on a **Retrieval Session** control-plane that makes retrievals accountable and grief-resistant while staying aligned to NilFS + Triple Proof and the protocol’s atomic units:

* **Atomic unit:** 128 KiB **Blob** (`BLOB_SIZE`). All on-chain accounting is in blob counts / blob-aligned bytes.
* **Session unit:** a contiguous sequence of blobs that may span MDUs (8 MiB = 64 blobs).

The intended end state is: a provider only gets credit for a retrieval once the chain has evidence of **both** (a) a user-authorized request and (b) a user-confirmed successful completion, plus the provider’s cryptographic proof-of-retrieval.

1.  **Open a Retrieval Session (User, on-chain tx):**
    *   The user opens a session bound to a specific `(deal_id, provider, manifest_root, blob-range)`:
        *   `{deal_id, owner, provider, manifest_root, start_mdu_index, start_blob_index, blob_count, nonce, expires_at}`.
    *   Invariants:
        *   Provider MUST be assigned in `Deal.providers[]`.
        *   `manifest_root` MUST match the current on-chain `Deal.manifest_root` (pin content).
        *   **Mode 1:** `start_blob_index < BLOBS_PER_MDU`.
        *   **Mode 2:** `start_blob_index < leaf_count` where `leaf_count = (K+M) * rows`, `rows = 64 / K`.
        *   `blob_count > 0`.
        *   `total_bytes = blob_count * 131072` and MUST be a multiple of 128 KiB (by construction).
    *   **Session identity:** `session_id = keccak256(canonical_encode(fields...))` (canonical encoding MUST be specified and test-vectored; EVM precompile uses `abi.encode(...)`).

2.  **Serve bytes (Provider, off-chain):**
    *   Providers SHOULD refuse remote fetches that are not bound to an `OPEN` session (`X-Nil-Session-Id`).
    *   Each HTTP `Range` response MUST map to exactly one blob (bounded by blob boundaries); a session is satisfied by fetching the declared contiguous blob range (chunking is a client/gateway concern).

3.  **Submit proof-of-retrieval (Provider, on-chain tx):**
    *   The provider submits `ChainedProof` objects for the served blobs referencing `session_id`.
    *   v1 (devnet): one `ChainedProof` per blob in the session; later iterations may replace this with sampling and/or aggregated multi-openings.

4.  **Confirm completion (User, on-chain tx):**
    *   After a successful download, the user submits a session confirmation transaction bound to `session_id`.
    *   This is the protocol’s “proof-of-validation” that the user considers the retrieval complete.

5.  **Completion + accounting (Chain):**
    *   A session becomes `COMPLETED` once the chain has:
        *   provider proof-of-retrieval for the declared blob range, and
        *   user confirmation,
        *   all before `expires_at`.
    *   Only `COMPLETED` sessions increment `DealHeatState.successful_retrievals_total` and contribute to rewards/health.

#### 7.2.1 Gamma-4 Retrieval Fees (Devnet, Normative)

For Gamma-4, retrieval pricing is **fee-based** (no credits). Fees are charged at session open and settled only on completion:

* **Base fee (anti-spam):** On `MsgOpenRetrievalSession`, the chain MUST charge `base_retrieval_fee` and burn it (non-refundable).
* **Variable fee (per blob):** On `MsgOpenRetrievalSession`, the chain MUST lock `variable = retrieval_price_per_blob * blob_count` against `Deal.escrow_balance`.
* **Completion payout:** When a session reaches `COMPLETED`, the chain MUST:
  * burn `ceil(variable * retrieval_burn_bps / 10000)`, and
  * transfer the remaining `variable - burn_cut` from the `nilchain` module account to the Provider.
* **Expiry/refund:** If a session expires without completion, the locked `variable` amount MAY be unlocked by an owner-initiated cancel transaction (base fee remains burned).

Retrieval credits and byte-based allowances are out of scope for Gamma-4 and may be introduced later.

#### 7.2.2 Legacy: receipt-based liveness (devnet convenience, deprecated)

Earlier devnet iterations used per-range user message signatures (`RetrievalReceipt`) and session message signatures (`DownloadSessionReceipt`) to avoid explicit on-chain session state. These remain useful as a reference and may exist as compatibility paths, but the long-lived protocol direction is the on-chain Retrieval Session model above (tx-only user actions, blob-range semantics, explicit status on-chain).

### 7.3 Data Commitment Binding (Normative: The Triple Proof)

To prevent proofs over arbitrary data while enabling scalability to Petabyte datasets, all Mode 1 retrieval and storage proofs MUST use the **Triple Proof (Chained Verification)** architecture. This mechanism enables the blockchain to verify a specific byte of data while storing only a single 48-byte commitment (`ManifestRoot`) for the entire Deal.

1.  **Deal Commitments:** For each `Deal`, the chain stores only the **Manifest Root** (48-byte KZG Commitment). This root commits to a Manifest Polynomial $P(x)$ where each evaluation $y = P(i)$ corresponds to the scalar field representation of the Merkle Root of MDU $i$.
    *   `Deal.manifest_root` is the anchor of trust for the entire file.
2.  **Chained Proof Binding:** Any proof used in `MsgProveLiveness` (specifically `ChainedProof`) MUST bridge the gap from `Deal.manifest_root` to the specific data byte in three hops:
    *   **Hop 1 (Identity - KZG):** Prove that the MDU Merkle Root (as a scalar `mdu_root_fr`) is committed in the Manifest Polynomial at the correct `mdu_index`.
        *   `VerifyKZG(Deal.manifest_root, mdu_index, mdu_root_fr, manifest_opening)`
    *   **Hop 2 (Structure - Merkle):** Prove that the 128KB Blob Commitment is a leaf in the MDU's Merkle Tree.
        *   `VerifyMerkle(mdu_root_fr, blob_commitment, merkle_path)`
    *   **Hop 3 (Data - KZG):** Prove that the Data Byte is the evaluation of the Blob Polynomial at the challenge point.
        *   `VerifyKZG(blob_commitment, z_value, y_value, kzg_opening_proof)`

### 7.4 The Verification Algorithm

The verifier (Chain Node) executes the following logic inside the `MsgProveLiveness` handler to validate a `ChainedProof`.

**Algorithm: `VerifyChainedProof(Deal, Challenge, Proof)`**

1.  **Input Sanity Check:**
      * Ensure `Proof.mdu_index` matches the MDU index derived from `Challenge`.
      * Ensure `Proof.mdu_index < Deal.total_mdus`.
      * Ensure `Proof.blob_index` is in range for the Deal’s redundancy mode:
          * **Mode 1:** require `Proof.blob_index < 64`.
          * **Mode 2:** compute `rows = 64 / K`, `L = (K+M) * rows`, require `Proof.blob_index < L`, and for striped user‑data MDUs require `slot(Proof.blob_index) == slot(msg.creator)` using `slot(i) = i / rows`.

2.  **Hop 1: Verify Identity (The Map) [KZG]**
      * *Goal:* Prove that the SP isn't lying about the Merkle Root of the target MDU.
      * *Check:* `VerifyKZG(Deal.manifest_root, Proof.mdu_index, Proof.mdu_root_fr, Proof.manifest_opening)` MUST return TRUE.

3.  **Hop 2: Verify Structure (The MDU) [Merkle]**
      * *Goal:* Prove that the specific 128KB Blob is actually part of that MDU.
      * *Check:* `VerifyMerkle(Proof.mdu_root_fr, Proof.blob_commitment, Proof.merkle_path)` MUST return TRUE.
      * *Note:* `Proof.mdu_root_fr` is a scalar; it must be converted or hashed to match the Merkle root format.

4.  **Hop 3: Verify Data (The Blob) [KZG]**
      * *Goal:* Prove that the SP possesses the data inside that Blob.
      * *Check:* `VerifyKZG(Proof.blob_commitment, Proof.z_value, Proof.y_value, Proof.kzg_opening_proof)` MUST return TRUE.

5.  **Result:**
      * If all 3 hops pass, the proof is valid. The SP has proven possession of the specific byte requested by the protocol.

### 7.5 Evidence Types & Fraud Proofs

NilStore recognizes several classes of evidence derived from retrievals and synthetic checks. All evidence MUST ultimately be verifiable against the Deal’s on‑chain commitments (Section 7.3) and attributable to a specific `(deal_id, provider_id, epoch_e, mdu_index, blob_index)` (Mode 2: `blob_index = leaf_index`, §8.1.3).

1.  **Synthetic Storage Proofs (System‑Initiated):**
    *   For each epoch `e` and assignment `(deal_id, provider_id)`, the protocol derives a finite challenge set `S_e(D,P)` of `(mdu_index, blob_index)` pairs from `R_e`.
    *   A `SyntheticStorageProof` message carries:
        *   `(deal_id, provider_id, epoch_e, mdu_index, blob_index, eval_x, eval_y, kzg_commitment, kzg_proof, merkle_paths…)`.
    *   On‑chain verification MUST check:
        *   `(mdu_index, blob_index) ∈ S_e(D,P)`,
        *   Merkle paths reconstruct the Deal’s commitment(s),
        *   KZG opening is valid at `(eval_x, eval_y)`.
    *   A satisfied synthetic challenge contributes to storage rewards and positive health for `(D,P)`.
2.  **Retrieval‑Based Proofs (Client‑Initiated):**
    *   A provider-submitted proof-of-retrieval tied to an on-chain **Retrieval Session** (§ 7.2) MAY be submitted on‑chain as a `RetrievalProof`.
    *   Verification MUST:
        *   Recompute `DeriveCheckPoint` and match `(mdu_index, blob_index, eval_x)`,
        *   Verify Merkle + KZG against the Deal’s commitments,
        *   Verify the session binding (deal, provider, pinned `manifest_root`, blob-range) and anti‑replay checks (nonce, expiry),
        *   Ensure `(mdu_index, blob_index) ∈ S_e(D,P)` if the retrieval proof is to satisfy a synthetic challenge.
    *   Successful retrieval proofs count equivalently to synthetic proofs for storage reward and health.
3.  **Fraud Proofs (Wrong Data):**
    *   If a client or auditor receives a response whose KZG/Merkle proof fails against `root_cid(deal_id)`, they MAY construct a `FraudProof` that includes:
        *   The offending `session_id` plus the invalid proof material (and, optionally, the Provider’s signed response).
    *   On‑chain verification MUST:
        *   Re‑run Merkle/KZG checks and confirm failure relative to the stored commitments.
    *   A confirmed fraud proof MUST trigger slashing for the implicated `(deal_id, provider_id)` and degrade the Provider’s global health.
4.  **On‑Chain Challenge Non‑Response (Liveness Panic Path):**
    *   In extreme cases, a watcher MAY post an explicit on‑chain challenge (referencing a specific `(deal_id, provider_id, epoch_e, mdu_index, blob_index, eval_x)`).
    *   The chain MUST enforce a bounded response window; failure by the Provider to submit a corresponding `SyntheticStorageProof` within that window is treated as hard evidence of unavailability and MUST be slashable.

These evidence types collectively support the retrievability invariant: for each `(Deal, Provider)`, data is either retrievable under protocol rules or there exists high‑probability, verifiable evidence of failure that can be used to punish and eventually evict the Provider.

### 7.6 Proof Demand Policy (Planned, Parameters TBD)

The protocol requires an explicit policy for **how often** providers must prove possession and **how retrieval sessions reduce synthetic proof demand**.

This spec intentionally does not lock constants yet, but the target shape is:
* For each epoch `e` and assignment `(deal_id, provider_id)`, compute a required proof quota `required_e(D,P)` as a function of (at minimum) deal size (`Deal.size` / `Deal.total_mdus`), `ServiceHint` (Hot/Cold), and recent retrieval session volume.
* **Session credits:** Completed retrieval sessions (and any legacy receipt paths) contribute credits toward `required_e(D,P)`, potentially weighted by `bytes_served` with caps to prevent one large transfer from satisfying an entire epoch indefinitely.
* **Synthetic fill:** If `credits < required_e(D,P)`, the chain derives and enforces `required_e(D,P) - credits` synthetic challenges for that epoch.
* **Penalties:** Invalid proofs are slashable immediately; failure to meet quota SHOULD degrade reputation and eventually lead to eviction (a slower penalty path than invalid proof slashing).

The normative requirement is that `required_e(D,P)` and the synthetic challenge derivation are deterministic and computable from on-chain state plus epoch randomness `R_e`.

### 7.7 Deputy / Proxy Retrieval (Planned, Anti-griefing Semantics)

NilStore anticipates a “Deputy” (proxy) pattern where a provider may delegate *data-plane* serving (bandwidth, caching, egress) to an untrusted helper, while keeping *control-plane* accountability on the assigned Provider slot.

Normative intent:
* **Accountability remains with the assigned Provider:** rewards, liveness, and slashing attach to the on-chain provider assignment, not to deputies.
* **Client verification is mandatory:** clients MUST verify Merkle/KZG proof material before confirming completion of a Retrieval Session, preventing deputies from serving arbitrary bytes.
* **Anti-griefing:** retrieval session opens and completion confirmations MUST be replay-protected (nonce/expiry) and SHOULD be rate-limited / optionally funded, so a third party cannot force unbounded work on providers or deputies.

Detailed deputy selection, advertisement, and any explicit on-chain delegation/compensation mechanism is out of scope for v2.4 and should be specified in a dedicated RFC.

### 7.8 SP Audit Debt & Coverage Scaling (Planned)

To ensure coverage scales with total stored data—even when clients are dormant—NilStore MAY introduce **audit debt** as a source of retrieval-style challenges.

Conceptual shape:
1.  **Audit Debt Definition**
    *   For each epoch `e` and Provider `P`, compute an obligation proportional to stored bytes:
        * `audit_debt_bytes(P,e) = α * stored_bytes(P,e)` where `α` is a protocol parameter.
2.  **Task Assignment**
    *   Using `R_e`, the chain deterministically assigns `P` a set of retrieval tasks targeting other `(Deal, Provider')` pairs, aggregating to ≈ `audit_debt_bytes(P,e)`.
3.  **Execution & Incentives**
    *   `P` executes these audits as an ordinary client.
    *   Misbehavior (bad proofs, non‑response) discovered can be converted into fraud proofs or escalated challenges (with potential bounties).
4.  **Enforcement**
    *   Failure to satisfy audit debt SHOULD reduce placement priority and/or rewards until `P` catches up (distinct from invalid-proof slashing).

### 7.9 Health Metrics & Self‑Healing Placement (Planned)

Self‑healing can be expressed via per‑assignment and per‑provider health metrics:

1.  **Per‑Assignment Health**
    *   For each `(Deal, Provider)`, track a rolling `HealthState` (e.g., synthetic success ratio, retrieval success ratio, bad data rate, and non‑slashable QoS latency metrics).
2.  **Eviction & Re‑Replication**
    *   If `(Deal, Provider)` remains unhealthy long enough, the placement engine recruits replacements, adds them in a pending state, and only removes the old provider after the new provider proves readiness (make‑before‑break, §5.3).
3.  **Global Provider Health**
    *   Providers with consistently poor health lose eligibility for new placements and may be jailed/removed by governance.

---

## Appendix B: Intentionally Underspecified (v2.4) / RFC Targets

This specification defines normative *interfaces* and verification rules but intentionally leaves several “policy” and “parameterization” areas underspecified for v2.4. The following items SHOULD be captured as dedicated RFCs before mainnet hardening:

1. **System Placement Algorithm:** deterministic provider selection/weighting, hint scoring, anti-correlation rules, and upgrade strategy without reshuffling failure domains unexpectedly.
2. **Mode 2 On-Chain Encoding:** explicit representation of `(K, M)`, ordered `slot → provider` mapping, overlay scaling state, and replacement triggers/authorization. *(See `rfcs/rfc-mode2-onchain-state.md`.)*
3. **Challenge Derivation Function:** exact mapping from `(deal_id, epoch_e, provider/slot)` to a finite challenge set with anti-grind properties and coverage guarantees. *(See `rfcs/rfc-challenge-derivation-and-quotas.md`.)*
4. **Penalty & Eviction Curve:** concrete slashing parameters, reputation decay, jail/unjail, and eviction thresholds; distinguish invalid-proof slashing vs quota non-compliance. *(See `rfcs/rfc-challenge-derivation-and-quotas.md`.)*
5. **Pricing & Escrow Accounting:** bandwidth pricing model, debit schedule, tier reward curves, and how user-funded elasticity is bounded/enforced. *(See `rfcs/rfc-pricing-and-escrow-accounting.md`.)*
6. **Write Semantics Beyond Append-Only:** pending-generation promotion rules, rewrite/compaction/delete behavior, and any on-chain finalization criteria. *(Near-term repair/write constraints: see `rfcs/rfc-mode2-onchain-state.md`.)*
7. **Deputy/Proxy Mechanics:** discovery, routing, compensation/delegation (if any), and additional griefing defenses beyond nonce/expiry and rate limits.
8. **Encryption & Key Management Details:** exact encryption constructions, key derivation/rotation, metadata leakage model, padding strategy, and client recovery UX.
9. **Transport/Wire Protocol:** concrete fetch/prove message formats, range/chunking rules, retry/backoff, and gateway/SP interoperability requirements.

---

## Appendix C: Devnet Alpha Target Matrix (Non-normative Profile)

This appendix defines a pragmatic “Devnet Alpha” scope meant to get a **multi-provider network** running with **low expectations** and minimal protocol surface.

### C.1 Guiding constraints

* **Mode 2 available (devnet):** RS(K, K+M) striping is supported when `service_hint` includes `rs=K+M`. Repair/rebalancing remain deferred.
* **Mode 1 replication remains minimal:** Mode 1 is still treated as a single-provider deal unless `replicas=` is specified.
* **Serving provider is the prover:** bytes and proof material MUST come from the provider that will be named in the session proof (or from an explicit deputy, once specified).
* **Endpoint discovery is on-chain:** providers advertise transport endpoints as Multiaddrs; HTTP is used initially, libp2p is future-compatible.

### C.2 Target matrix

| Capability | Devnet Alpha Target | Notes |
|---|---:|---|
| Multiple providers registered | MUST | ≥ 3 providers on the devnet |
| On-chain provider endpoint discovery | MUST | `Provider.endpoints[]` as Multiaddr strings |
| HTTP transport | MUST | e.g. `/dns4/sp1.example.com/tcp/8080/http` |
| libp2p transport | DEFER | Multiaddr format reserved (`/p2p/<peerid>`) |
| Mode 1 replication (`providers[]` length > 1) | NO | Devnet Alpha uses `replicas=1` in `ServiceHint` |
| Mode 2 RS deals | YES (devnet) | `service_hint` includes `rs=K+M` |
| Gateway role | OPTIONAL | routing + cache helper; direct‑to‑provider is first‑class and preferred for Mode 2 |
| Provider role | MUST | stores deal slab; serves bytes+proof headers; owns fetch/download session state |
| Upload/ingest | MUST | per‑slot upload to assigned providers; Mode 2 encoding is client‑side (WASM/CLI); gateway mirroring optional |
| Retrieval by `file_path` + `Range` | MUST | chunked retrievals; max chunk ≤ one blob (`BLOB_SIZE`) |
| Session proof submission | MUST | session-bound proofs; provider submits to chain |
| Bundled session proofs | SHOULD | reduce wallet prompts / tx count |
| Synthetic challenges | DEFER | no hard quotas; sessions are still accepted evidence |
| Deputy / proxy routing | DEFER | tracked as an RFC / later sprint |
| Repair / rotation / rebalancing | NO | deferred to Mode 2 + deputy + policy |
| Docker/devnet orchestration | SHOULD | compose scripts to run 1 gateway + N providers |

### C.3 Definition of Done (Devnet Alpha)

Given 3–5 providers with advertised HTTP Multiaddrs:
1. Create a deal with `replicas=1`.
2. Upload content to the assigned provider and commit `Deal.manifest_root`.
3. Fetch a multi-chunk range through the gateway/router from that provider.
4. Submit a bundled session proof (or batched proofs) and observe `MsgSubmitRetrievalSessionProof` succeed on-chain.
```

```rfcs/rfc-pricing-and-escrow-accounting.md
# RFC: Pricing & Escrow Accounting (Lock-in + Retrieval Fees + Elasticity Caps)

**Status:** Sprint‑0 Frozen (Ready for implementation)
**Scope:** Chain economics (`nilchain/`) + gateway/UI intent fields
**Motivation:** `spec.md` §6.1–§6.2, §7.2.1; Appendix B #5
**Depends on:** `rfcs/rfc-data-granularity-and-economics.md`

---

## 0. Executive Summary

This RFC freezes the **economic accounting contracts** required for mainnet hardening:
- **Storage lock-in pricing** at ingest (`UpdateDealContent*`) using `storage_price` (Dec per byte per block)
- **Retrieval fees** via session-based settlement (base fee burn + per-blob variable fee lock, then burn cut + provider payout)
- **User-funded elasticity caps** enforced via `Deal.max_monthly_spend` and a deterministic spend window

This RFC intentionally does **not** introduce retrieval “credits” for Gamma‑4. Credits may be introduced later once quota enforcement exists (see `rfcs/rfc-challenge-derivation-and-quotas.md`).

---

## 1. Canonical Denoms & Accounts (Frozen)

### 1.1 Denom
- All fees/deposits are in `sdk.DefaultBondDenom` (devnet: `stake`).

### 1.2 Module accounts
- `authtypes.FeeCollectorName`: receives `deal_creation_fee`.
- `types.ModuleName` (`nilchain` module account): holds escrow and performs burns/transfers for retrieval settlement.

---

## 2. Parameters (Frozen)

From `nilchain/nilchain/v1/params.proto`:
- `deal_creation_fee: Coin`
- `min_duration_blocks: uint64`
- `storage_price: Dec` (per byte per block)
- `base_retrieval_fee: Coin` (burned at session open)
- `retrieval_price_per_blob: Coin` (locked at session open)
- `retrieval_burn_bps: uint64` (basis points of variable fee burned on completion)
- `base_stripe_cost: uint64` (unit cost used for elasticity budgeting; denom = bond denom)

From Deal state:
- `max_monthly_spend: Int` (cap for user-funded elasticity)
- `escrow_balance: Int` (remaining funds available to pay protocol-defined charges)

---

## 3. Deal Lifecycle Charges (Frozen)

### 3.1 CreateDeal (`MsgCreateDeal*`)
**Inputs:** `duration_blocks`, `initial_escrow_amount`, `max_monthly_spend`, `service_hint`

**Validation:**
- `duration_blocks >= min_duration_blocks`
- `initial_escrow_amount >= 0`
- `max_monthly_spend >= 0`

**Accounting:**
1. If `deal_creation_fee > 0`, transfer `deal_creation_fee` from creator → fee collector.
2. If `initial_escrow_amount > 0`, transfer `initial_escrow_amount` from creator → module account.
3. Initialize deal with:
   - `manifest_root = empty`
   - `size_bytes = 0`
   - `total_mdus = 0` (until first commit; see `rfcs/rfc-mode2-onchain-state.md`)
   - `escrow_balance = initial_escrow_amount`

### 3.2 AddCredit (`MsgAddCredit`)
Transfers `amount` from sender → module account and increments `Deal.escrow_balance += amount`.

---

## 4. Storage Lock-in Pricing (Frozen)

### 4.1 UpdateDealContent (`MsgUpdateDealContent*`)
When content is committed and `size_bytes` increases, the protocol charges a **term deposit** at the current `storage_price`.

Let:
- `old_size = Deal.size_bytes`
- `new_size = msg.size_bytes`
- `delta = max(0, new_size - old_size)`
- `duration = Deal.end_block - Deal.start_block` (fixed at deal creation for v1)

**Cost function:**
```
storage_cost = ceil(storage_price * delta * duration)
```

**Accounting:**
- If `storage_cost > 0`, transfer `storage_cost` from owner → module account.
- Update `Deal.escrow_balance += storage_cost`.

**Normative properties:**
- Only incremental bytes are charged at the new spot price.
- Previously committed bytes are not repriced.

### 4.2 Future extension (out of scope)
Extending lifetime past `end_block` requires a `MsgExtendDeal` (or equivalent) and a lock-in charge using the spot `storage_price` at extension time.

---

## 5. Retrieval Fees (Gamma‑4, Frozen)

This section is normative and matches `spec.md` §7.2.1.

### 5.1 Session open (`MsgOpenRetrievalSession`)
Let:
- `blob_count` be the requested contiguous blob-range length (128 KiB units)
- `base_fee = Params.base_retrieval_fee`
- `variable_fee = Params.retrieval_price_per_blob * blob_count`
- `total = base_fee + variable_fee`

**Must-fail conditions:**
- `Deal.escrow_balance < total` → reject
- `manifest_root` must match `Deal.manifest_root` (pin)

**Accounting at open:**
1. Burn `base_fee` from module account (non-refundable).
2. Lock `variable_fee` against the session and decrement deal escrow:
   - `Deal.escrow_balance -= (base_fee + variable_fee)`
   - `session.locked_fee = variable_fee` (store on session object)

### 5.2 Completion (`MsgConfirmRetrievalSession` + proof present)
On transition to `COMPLETED`, settle the locked variable fee:

```
burn_cut = ceil(variable_fee * retrieval_burn_bps / 10_000)
payout   = variable_fee - burn_cut
```

**Accounting:**
- Burn `burn_cut` from module account.
- Transfer `payout` from module account → provider account.

### 5.3 Expiry/cancel (refund path)
If a session expires without completion, the owner may cancel:
- `MsgCancelRetrievalSession` unlocks the remaining `session.locked_fee` and refunds it to `Deal.escrow_balance`.
- Base fee is never refunded.

---

## 6. Elasticity Spend Caps (Freeze)

Elasticity is user-funded and must be bounded by `Deal.max_monthly_spend` (a cap) and `Deal.escrow_balance` (available funds).

### 6.1 Spend window
Define:
- `MONTH_LEN_BLOCKS` (param; e.g. 30 days worth of blocks)

Add per-deal accounting fields:
- `spend_window_start_height: uint64`
- `spend_window_spent: Int`

Window logic (deterministic):
- If `height >= spend_window_start_height + MONTH_LEN_BLOCKS`, reset:
  - `spend_window_start_height = height`
  - `spend_window_spent = 0`

### 6.2 Scaling event cost
For any elasticity action that increases replication/overlays by `delta_replication`:

```
elasticity_cost = base_stripe_cost * delta_replication
```

**Must-fail:**
- `spend_window_spent + elasticity_cost > max_monthly_spend`
- `Deal.escrow_balance < elasticity_cost`

**Accounting:**
- `Deal.escrow_balance -= elasticity_cost`
- `spend_window_spent += elasticity_cost`

**Implementation note:** current devnet `MsgSignalSaturation` enforces the cap but does not debit; mainnet requires the debit.

---

## 7. Required Interface/State Changes (for implementation sprints)

1. `Deal` fields (if not already present):
   - `spend_window_start_height`
   - `spend_window_spent`
2. Ensure `UpdateDealContent*` continues to carry `size_bytes` (and, per Sprint‑0 naming freeze, also carries `total_mdus` + `witness_mdus`; see `rfcs/rfc-mode2-onchain-state.md`).
3. Ensure retrieval session settlement burns/transfers use module account funds and update `Deal.escrow_balance` deterministically.

---

## 8. Test Gates (for later sprints)

- Storage lock-in: update content with increasing size charges `delta*duration*price` and rejects if insufficient funds.
- Retrieval fees: open burns base fee, locks variable, completion burns cut + pays provider, cancel refunds variable.
- Elasticity: scaling denied when exceeding `max_monthly_spend` or `escrow_balance`.

```

```rfcs/rfc-challenge-derivation-and-quotas.md
# RFC: Challenge Derivation & Proof Quota Policy (Unified Liveness v1)

**Status:** Sprint‑0 Frozen (Ready for implementation)
**Scope:** Chain protocol policy (`nilchain/`)
**Motivation:** `spec.md` §7.6; Appendix B #3 (challenge derivation), #4 (quota + penalty curve)
**Depends on:** `spec.md`, `rfcs/rfc-mode2-onchain-state.md`, `rfcs/rfc-blob-alignment-and-striping.md`

---

## 0. Executive Summary

NilStore’s “Unified Liveness” requires the chain to deterministically answer:
1. **What** positions a provider must prove for a given epoch (synthetic challenges)
2. **How many** proofs are required (quota)
3. **How organic retrieval** reduces synthetic demand (credits)
4. **What happens** when a provider is invalid vs merely non-compliant (penalty curve)

This RFC freezes:
- a deterministic, anti-grind challenge derivation function
- a quota computation function with explicit parameters
- an accounting model for credits and synthetic fills
- enforcement + penalty outcomes (invalid proof slashing vs quota failure health decay)

---

## 1. Definitions

### 1.1 Epoch
NilStore defines a **liveness epoch** with fixed length:
- `EPOCH_LEN_BLOCKS` (param; e.g. 100 blocks)
- `epoch_id = floor(block_height / EPOCH_LEN_BLOCKS)`

### 1.2 Assignment
An **assignment** is:
- Mode 1: `(deal_id, provider)` where `provider ∈ Deal.providers[]`
- Mode 2: `(deal_id, slot)` where `slot ∈ [0..K+M-1]` and `slot.provider` is the accountable provider

### 1.3 Challenge position
A synthetic challenge position is a pair:
- `(mdu_index, blob_index)`
  - Mode 1: `blob_index ∈ [0..63]` (Blob within MDU)
  - Mode 2: `blob_index` MUST be interpreted as `leaf_index` per slot-major ordering (§8.1.3); `blob_index ∈ [0..leafCount-1]`

### 1.4 Credit
A **credit** is a unit of evidence earned via organic retrieval that reduces synthetic demand.
This RFC accounts credits in **blob-proofs** (not bytes) to avoid ambiguity across Mode 1 vs Mode 2.

---

## 2. Required Chain Inputs (Frozen)

Challenge derivation and quota computation MUST be computable from:
- current block height (for epoch)
- `Deal`: `redundancy_mode`, `service_hint` (legacy), `providers[]`
- **Frozen additions:** `Deal.total_mdus`, `Deal.witness_mdus`, and for Mode 2 the explicit `(K,M)` and slot order (see `rfcs/rfc-mode2-onchain-state.md`)
- epoch randomness `R_e` (see §3.1)
- per-epoch counters for credits + satisfied synthetic challenges (new state; see §5)

---

## 3. Deterministic Challenge Derivation (Anti-grind)

### 3.0 Canonical encoding (must be deterministic)
Unless otherwise stated, hashes are computed over byte concatenation using:
- `U64BE(x)`: 8-byte big-endian unsigned integer
- `U32BE(x)`: 4-byte big-endian unsigned integer
- `ADDR20(provider)`: 20-byte account address obtained by bech32-decoding the provider string (reject invalid)

`SHA256(tag || …)` means SHA-256 over the concatenated byte slices, where `tag` is ASCII bytes.

### 3.1 Epoch randomness
Define the epoch seed as:

```
epoch_start_height = epoch_id * EPOCH_LEN_BLOCKS
R_e = SHA256("nilstore/epoch/v1" || chain_id || epoch_id || block_hash(epoch_start_height))
```

Rationale:
- deterministic and locally computable by all nodes
- unpredictable prior to the epoch boundary (assuming honest majority of validators)
- does not rely on any off-chain RNG or trusted beacon

### 3.2 Challenge set size
For each assignment, the chain derives a target challenge count:

```
quota_blobs = required_blobs(deal, assignment, epoch_id)        // §4
credits_blobs = credits_applied(deal, assignment, epoch_id)      // §5
synthetic_needed = max(0, quota_blobs - credits_blobs)
```

The synthetic challenge set for the assignment is:
- `S_e(deal, assignment) = { C_i | i ∈ [0..synthetic_needed-1] }`

### 3.3 Mode 2: slot-major derivation

Let:
- `K,M` be the deal’s Mode 2 profile
- `N = K+M`
- `rows = 64 / K`
- `leafCount = N * rows`
- `meta_mdus = 1 + witness_mdus`
- `user_mdus = total_mdus - meta_mdus` (must be > 0 for challenges)

For slot `s ∈ [0..N-1]` and challenge ordinal `i`:

```
seed = SHA256("nilstore/chal/v1" || R_e || U64BE(deal_id) || U64BE(current_gen) || U64BE(slot) || U64BE(i))
mdu_ordinal = U64BE(seed[0..8]) % user_mdus
row        = U64BE(seed[8..16]) % rows

mdu_index  = meta_mdus + mdu_ordinal
leaf_index = slot*rows + row
```

The challenge position is `(mdu_index, blob_index=leaf_index)`.

**Exclusions (frozen):**
- Synthetic challenges MUST NOT target metadata MDUs (`mdu_index < meta_mdus`).
- Synthetic challenges MUST NOT target Mode 2 slots with `status != ACTIVE` (repairing slots are excluded).

### 3.4 Mode 1: replica derivation

Let:
- `meta_mdus = 1 + witness_mdus`
- `user_mdus = total_mdus - meta_mdus`

For provider `P` and challenge ordinal `i`:

```
seed = SHA256("nilstore/chal/v1" || R_e || U64BE(deal_id) || U64BE(current_gen) || ADDR20(provider) || U64BE(i))
mdu_ordinal = U64BE(seed[0..8]) % user_mdus
blob_index  = U64BE(seed[8..16]) % 64
mdu_index   = meta_mdus + mdu_ordinal
```

The challenge position is `(mdu_index, blob_index)`.

---

## 4. Required Proof Quota (Policy Freeze)

### 4.1 Parameters
All of the following are chain params:
- `quota_bps_per_epoch_hot` (basis points of stored bytes proved per epoch)
- `quota_bps_per_epoch_cold`
- `quota_min_blobs` (floor)
- `quota_max_blobs` (cap)
- `credit_cap_bps` (max fraction of quota satisfiable via credits)

### 4.2 Normalized “slot bytes”
Quota targets are computed over **slot-responsible bytes** (not entire deal bytes):
- Mode 2: each slot stores `rows * BLOB_SIZE` per user MDU.
  - `slot_bytes = user_mdus * rows * BLOB_SIZE`
- Mode 1: each provider stores full MDUs.
  - `slot_bytes = user_mdus * MDU_SIZE`

### 4.3 Required blobs function

```
quota_bps = (service_hint_base == Hot) ? quota_bps_per_epoch_hot : quota_bps_per_epoch_cold
target_bytes = ceil(slot_bytes * quota_bps / 10_000)
target_blobs = ceil(target_bytes / BLOB_SIZE)
quota_blobs  = clamp(quota_min_blobs, target_blobs, quota_max_blobs)
```

Notes:
- using `BLOB_SIZE` as the unit makes Mode 1 and Mode 2 comparable
- caps ensure quotas remain operationally feasible on low-end nodes

---

## 5. Credit Accounting (Organic Retrieval → Quota Reduction)

### 5.1 What counts as credit
Credits accrue from **completed user retrieval** evidence paths that include valid blob proofs:
- `MsgSubmitRetrievalSessionProof` (preferred)
- `MsgProveLiveness` receipt paths (`user_receipt`, `user_receipt_batch`) while in transition

### 5.2 Credit unit
Each *unique proved blob* counts as **1 credit blob**.
- A session proof covering `blob_count` blobs yields `blob_count` credits, subject to caps.

### 5.3 Credit caps (anti-wash + determinism)
To prevent a single large download from satisfying all synthetic demand indefinitely:
- credits applied per `(deal, assignment, epoch)` are capped:

```
credit_cap = ceil(quota_blobs * credit_cap_bps / 10_000)
credits_blobs = min(credit_cap, unique_proved_blobs_in_epoch)
```

Uniqueness is enforced by storing a per-epoch set keyed by:
`credit_id = SHA256("nilstore/credit/v1" || epoch_id || deal_id || assignment || mdu_index || blob_index)`.

---

## 6. Enforcement & Penalty Curve (Freeze)

### 6.0 Proof acceptance rules (must-fail)
- `system_proof` MUST match one derived synthetic challenge for that assignment and epoch.
  - The chain checks membership by recomputing `C_i` for `i ∈ [0..synthetic_needed-1]` and comparing `(mdu_index, blob_index)`.
  - Duplicate synthetic proofs for the same `(epoch, assignment, mdu_index, blob_index)` MUST NOT be double-counted.
- `session_proof` and receipt paths MAY be outside the synthetic challenge set; they still accrue credits (§5).

### 6.1 Invalid proofs (hard failures)
- A proof that fails verification MUST be slashable immediately (existing devnet behavior).
- Invalid proofs also increment an assignment health failure counter (see `CHAIN-103`).

### 6.2 Quota shortfall (soft failures)
- If, at epoch end, `credits_blobs + satisfied_synthetic_blobs < quota_blobs`, the assignment is **non-compliant**.
- Non-compliance is NOT immediately slashable by default; it:
  - decays the assignment’s `HealthState`
  - reduces placement priority
  - increments a rolling `missed_epochs` counter

### 6.3 Eviction trigger (policy hook)
When `missed_epochs` exceeds `evict_after_missed_epochs` (param), the chain SHOULD:
- mark the slot as `REPAIRING`
- select and attach a `pending_provider` candidate (see `rfcs/rfc-mode2-onchain-state.md`)

---

## 7. Required State Additions (for implementation sprints)

To implement the above without storing per-proof raw history, add collections:

- `QuotaState(deal_id, assignment, epoch_id)`:
  - `quota_blobs`
  - `credits_blobs`
  - `synthetic_satisfied_blobs`
  - `missed_epochs` (rolling)

- `CreditSeen(credit_id)` with TTL to prevent replay/double-counting.
- `SyntheticSeen(challenge_id)` to prevent counting the same synthetic proof twice.

All keys are deterministic hashes to keep store keys bounded.

---

## 8. Test Gates (for later sprints)

- Determinism tests: same chain state + epoch → identical challenge set across nodes.
- Anti-grind tests: challenge set changes with epoch; cannot be precomputed far in advance.
- E2E: no organic traffic → synthetic proofs required; with organic traffic → synthetic needed drops.
```

```rfcs/rfc-retrieval-validation.md
# RFC: Retrieval Validation & The Deputy System

**Status:** Draft / Normative Candidate
**Scope:** Retrieval Markets, Proof of Delivery, Dispute Resolution
**Key Concepts:** Proxy Relay, Audit Debt, Ephemeral Identity

---

## 1. The Core Problem: "He Said, She Said"

In a trustless retrieval market, we must distinguish between:
1.  **Service Failure:** The SP is offline or malicious.
2.  **Griefing:** The User claims the SP is offline, but the SP is actually fine.

We solve this not by "Judging" the dispute, but by **Routing Around It**.

---

## 2. The Solution: The Deputy (Proxy) System

Instead of a complex "Court System," we implement a **"CDN of Last Resort."**

### 2.1 The "Proxy" Workflow (UX-First)
When a Data User (DU) fails to retrieve a file from their assigned Storage Provider (SP):

1.  **Escalation:** The DU broadcasts a P2P request: *"I need Chunk X from SP Y. I will pay MarketRate + Premium."*
2.  **The Deputy:** A random third-party Node (The Deputy) accepts the job.
3.  **The Relay:**
    *   The Deputy connects to the SP using a fresh, **Ephemeral Keypair** (acting as a new customer).
    *   The Deputy retrieves the chunk and pays the SP.
    *   The Deputy forwards the chunk to the DU and collects the `MarketRate + Premium`.
4.  **Outcome:**
    *   **Success:** The DU gets their file. The SP gets paid (unknowingly serving a proxy). The Deputy earns a fee.
    *   **Failure:** If the SP refuses/fails to serve the Deputy, the Deputy signs a `ProofOfFailure`.

### 2.2 Why This Works (Indistinguishability)
We do **not** need complex privacy mixers or ZK-Vouchers.
*   **Rationality Assumption:** A Rational SP wants to earn money.
*   **The Trap:** When the Deputy connects with an ephemeral key, the SP sees a **New Paying Customer**.
    *   If SP serves: They avoid slashing, but the DU gets the data (Goal achieved).
    *   If SP refuses: They lose revenue AND generate a `ProofOfFailure` (Slashing Risk).

---

## 3. "Audit Debt": The Engine of Honesty

How do we ensure there are enough Deputies? We **Conscript** them.

### 3.1 The Rule
**"To earn Storage Rewards, you must prove you are checking your neighbors."**

### 3.2 The Mechanism
1.  **Assignment:** The Protocol deterministically assigns `AuditTargets` to every SP based on the Random Beacon (DRB).
2.  **The Job:** The SP must act as a Deputy/Mystery Shopper for these targets.
3.  **The Reward Gate:**
    *   `ClaimableReward = min(BaseInflationReward, AuditWorkDone * Multiplier)`
    *   If an SP stores 1PB of data but performs 0 audits, their **Effective Reward** is 0.
4.  **Proof of Audit:** The SP submits the `RetrievalReceipt` they obtained from the Target SP.
    *   *Side Effect:* This generates a constant hum of "Organic Traffic" that proves the network is live, even when real users are asleep.

---

## 4. The Sad Path: Verified Failure

If a Deputy attempts to retrieve a chunk (for a User or for Audit Debt) and fails:

1.  **Evidence:** Deputy creates a `ProofOfFailure` (signed attestation + transcript hash).
2.  **Accumulation:** The Chain tracks `FailureCount(SP)`.
3.  **Slashing:**
    *   If `FailureCount > Threshold` within `Window`, the SP is jailed/slashed.
    *   *Safety:* A single malicious Deputy cannot slash an SP. It requires a consensus of failures from distinct, randomly selected Deputies.

---

## 5. Implementation Strategy (MVP)

**Phase 1: The Proxy (Client-Side Only)**
*   Implement the P2P `AskForProxy` message.
*   No consensus changes. Just networking logic.

**Phase 2: Audit Debt (Consensus)**
*   Add `AuditDebt` tracking to the `StorageProvider` struct.
*   Update `BeginBlocker` to check Audit compliance before minting rewards.

**Phase 3: Slashing**
*   Implement `MsgSubmitFailureEvidence`.

---

## 6. Summary

This RFC moves the protocol from a "Legal System" (Disputes) to a "Logistics System" (Relays).
*   **User Problem:** "I can't get my file." -> **Solution:** "A Deputy gets it for you."
*   **Network Problem:** "Are nodes online?" -> **Solution:** "Nodes must audit each other to get paid."```

```rfcs/rfc-mode2-onchain-state.md
# RFC: Mode 2 On-Chain State (Slots, Generations, Repairs)

**Status:** Sprint‑0 Frozen (Ready for implementation)
**Scope:** Chain protocol state (`nilchain/`)
**Depends on:** `spec.md` §6.2, §8.3–§8.4; `rfcs/rfc-blob-alignment-and-striping.md`
**Motivation:** Appendix B #2 (Mode 2 encoding), #6 (write semantics beyond append-only; near-term constraints)

---

## 0. Executive Summary

Devnet Mode 2 currently relies on **implicit encoding**:
- `(K,M)` is derived by parsing `Deal.service_hint` (`rs=K+M`)
- `Deal.providers[]` is treated as the slot order (by convention)

Mainnet requires **explicit typed state** so the chain can:
- enforce invariants (slot ordering, RS profile consistency)
- coordinate **repairs and make‑before‑break replacement**
- derive deterministic per-slot policy (synthetic challenges, quotas, health)

This RFC freezes a **concrete on-chain representation** for Mode 2 and a minimal lifecycle state machine that is forward-compatible with “pending generation” writes later.

---

## 1. Definitions / Invariants

### 1.1 Slot / Profile
- **Profile:** RS(`K`, `K+M`) with `N = K+M`
- **Slot:** integer `slot ∈ [0..N-1]`
- **Base slots:** the canonical `N` providers currently responsible for the deal’s stripe shards
- **Overlay slots:** additional providers per slot (elasticity or replacement candidates); not required for Sprint 3/4, but state is reserved here

### 1.2 Generations
- **Generation:** a monotonically increasing counter `current_gen`
- Every on-chain content commit that changes `Deal.manifest_root` MUST increment `current_gen`.
- Reads are always defined against the **current generation**.

### 1.3 Slab accounting fields (naming freeze)
For chain policy and bounds checks we freeze:
- `size_bytes`: total logical bytes of file contents in NilFS (sum of non-tombstone file lengths)
- `total_mdus`: total number of committed MDU roots in the Manifest commitment (includes metadata + witness + user MDUs)
- `witness_mdus`: number of witness MDUs committed after MDU #0 (metadata region size)
- `user_mdus = total_mdus - 1 - witness_mdus` (derived; must be non-negative)

Notes:
- This RFC intentionally avoids `allocated_length` in protocol state. Gateway/UI MAY keep `allocated_length` as a legacy alias for `total_mdus` (count), per `nil_gateway/nil-gateway-spec.md`.

---

## 2. Proposed On-Chain Schema (Protobuf Freeze)

### 2.1 New messages

```proto
// StripeReplica profile parameters for Mode 2.
message StripeReplicaProfile {
  uint32 k = 1; // data slots
  uint32 m = 2; // parity slots
}

enum SlotStatus {
  SLOT_STATUS_UNSPECIFIED = 0;
  SLOT_STATUS_ACTIVE = 1;
  SLOT_STATUS_REPAIRING = 2; // slot is being replaced/catching up; excluded from quota + rewards
}

// Slot state for Mode 2 (base slot + optional replacement candidate).
message DealSlot {
  uint32 slot = 1; // 0..N-1
  string provider = 2; // current accountable provider (bech32)
  SlotStatus status = 3;

  // Make-before-break: replacement candidate for this slot (optional).
  // While set, the old provider remains accountable; the candidate proves readiness, then is promoted.
  string pending_provider = 4; // bech32 or empty

  int64 status_since_height = 5;
  uint64 repair_target_gen = 6; // == Deal.current_gen when repair starts
}
```

### 2.2 Deal additions (non-breaking)

We keep existing fields for devnet compatibility (notably `providers[]` and `service_hint`), but freeze the new canonical fields:

```proto
message Deal {
  // existing fields...

  // --- Mode 2 explicit encoding (new canonical state) ---
  StripeReplicaProfile mode2_profile = 15; // set iff redundancy_mode == 2
  repeated DealSlot mode2_slots = 16;      // length N, slot-ordered

  // --- Generation / write coordination ---
  uint64 current_gen = 17; // increments on every manifest_root change

  // --- Slab accounting (bounds + policy) ---
  uint64 total_mdus = 14;     // already exists; MUST be set on first content commit
  uint64 witness_mdus = 18;   // NEW; set on first content commit
}
```

**Canonical source of truth:**
- If `redundancy_mode != 2`, `mode2_profile` and `mode2_slots` MUST be unset/empty.
- If `redundancy_mode == 2`, `mode2_profile.k+m == len(mode2_slots)` MUST hold and `mode2_slots[i].slot == i`.

**Legacy fields during migration window:**
- `providers[]` remains populated for LCD/UI convenience and backwards compatibility.
- For Mode 2, `providers[]` MUST equal `[slot.provider for slot in mode2_slots]` until `providers[]` can be deprecated.
- `service_hint` may still include `rs=K+M`, but once `mode2_profile` exists, it is treated as **intent only**, not canonical state.

---

## 3. Lifecycle State Machine (Freeze)

### 3.1 CreateDeal (Mode 2)
At `MsgCreateDeal*` time:
- `mode2_profile` and `mode2_slots` are derived from the request (legacy: parsed from `service_hint`)
- `current_gen = 0`
- `manifest_root = empty`, `size_bytes = 0`, `total_mdus = 0`, `witness_mdus = 0`

### 3.2 UpdateDealContent (commit new manifest)
At `MsgUpdateDealContent*` time:
- Validate `manifest_root` format (already implemented)
- Require `size_bytes > 0`
- Require `total_mdus > 0` and `witness_mdus >= 0` (new fields in message; see §4)
- Set:
  - `Deal.manifest_root = new`
  - `Deal.size_bytes = new`
  - `Deal.total_mdus = new_total_mdus`
  - `Deal.witness_mdus = new_witness_mdus`
  - `Deal.current_gen += 1`

### 3.3 Repair / replacement (make-before-break)

**Start repair:** mark a slot as repairing and set a candidate.
- `slot.status = REPAIRING`
- `slot.pending_provider = candidate`
- `slot.repair_target_gen = Deal.current_gen`

**Candidate catch-up:** performed off-chain (gateway/SP tooling) by reconstructing and storing the required shards up to `repair_target_gen` (or `current_gen` if it advanced).

**Complete repair:** promote candidate and return slot to active.
- `slot.provider = slot.pending_provider`
- `slot.pending_provider = ""`
- `slot.status = ACTIVE`
- `slot.repair_target_gen = 0`

**Policy note:** While a slot is `REPAIRING`:
- clients SHOULD route around that slot for Mode 2 reads (fetch any `K` ACTIVE slots per MDU)
- synthetic challenges and quota accounting MUST ignore repairing slots
- repairing slots MUST NOT earn rewards for liveness proofs (they may still submit a “readiness proof” message; not defined here)

---

## 4. Required Message / Interface Changes (Freeze for Sprint 3+)

### 4.1 UpdateDealContent must carry slab accounting

To make `Deal.total_mdus` and `Deal.witness_mdus` enforceable, the update intent must include them:

```proto
message MsgUpdateDealContent {
  // existing fields...
  uint64 size = 4;         // logical bytes
  uint64 total_mdus = 5;   // NEW: manifest root count
  uint64 witness_mdus = 6; // NEW: metadata witness count
}

message EvmUpdateContentIntent {
  // existing fields...
  uint64 size_bytes = 4;
  uint64 total_mdus = 7;   // NEW
  uint64 witness_mdus = 8; // NEW
}
```

**Gateway/UI contract:** the upload/ingest pipeline already knows these values by inspecting `mdu_0.bin` / slab layout. The gateway response SHOULD include `total_mdus` and `witness_mdus` explicitly; `allocated_length` MAY remain as a legacy alias for `total_mdus`.

---

## 5. Upgrade / Migration Strategy (Devnet → Typed State)

### 5.1 Store migration
Add a one-time migration that:
- For each Deal with `redundancy_mode == 2`:
  - parse `(K,M)` from `service_hint` (legacy)
  - set `mode2_profile`
  - set `mode2_slots` from existing `providers[]` (slot order = list order)
  - initialize `slot.status = ACTIVE`, `pending_provider = ""`, `current_gen = 0` if unset
- Ensure `providers[]` and `mode2_slots[].provider` remain identical.

### 5.2 Post-migration behavior
- New deals write both legacy (`service_hint`, `providers[]`) and canonical (`mode2_*`) fields.
- Chain logic MUST prefer canonical typed fields when present.

---

## 6. Test Gates (for later sprints)

- **Migration test:** legacy Mode 2 deals survive upgrade with identical slot ordering and `(K,M)` values.
- **Invariants tests:** reject inconsistent `(K,M)` vs slot length; reject invalid slot indices.
- **Repair e2e:** multi-SP: mark slot repairing → candidate catch-up → promote → reads stay available (fetch any `K`).

---

## 7. Implementation Checklist (Sprint 3/4)

1. Protobuf + codegen:
   - `nilchain/proto/nilchain/nilchain/v1/types.proto`: add `StripeReplicaProfile`, `DealSlot`, `SlotStatus`, `Deal.current_gen`, `Deal.witness_mdus`, `Deal.mode2_*`.
   - `nilchain/proto/nilchain/nilchain/v1/tx.proto`: extend `MsgUpdateDealContent` + `EvmUpdateContentIntent`.
2. Keeper logic:
   - Populate typed fields at `CreateDeal`.
   - Persist `total_mdus/witness_mdus/current_gen` at `UpdateDealContent*`.
3. Read path constraints:
   - Update `stripeParamsForDeal()` and `providerSlotIndex()` to use typed fields when present.
4. Gateway/UI:
   - Ensure `/gateway/upload` returns `total_mdus` and `witness_mdus` (keep legacy alias fields for transition).
5. Store migration:
   - Add an upgrade handler to backfill typed Mode 2 state for existing deals.

```

```nilchain/proto/nilchain/nilchain/v1/params.proto
syntax = "proto3";
package nilchain.nilchain.v1;

import "amino/amino.proto";
import "gogoproto/gogo.proto";
import "cosmos/base/v1beta1/coin.proto";

option go_package = "nilchain/x/nilchain/types";

// Params defines the parameters for the module.
message Params {
  option (amino.name) = "nilchain/x/nilchain/Params";
  option (gogoproto.equal) = true;

  uint64 base_stripe_cost = 1; // NIL per epoch
  uint64 halving_interval = 2; // Blocks
  uint64 eip712_chain_id = 3; // Numeric EIP-712 domain chainId (default 31337 for localhost devnet)

  string storage_price = 4 [
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ]; // Price per byte per block in base denom (devnet: stake)

  cosmos.base.v1beta1.Coin deal_creation_fee = 5 [
    (gogoproto.nullable) = false
  ];

  uint64 min_duration_blocks = 6;

  cosmos.base.v1beta1.Coin base_retrieval_fee = 7 [
    (gogoproto.nullable) = false
  ]; // Fixed fee charged on retrieval session open (burned).

  cosmos.base.v1beta1.Coin retrieval_price_per_blob = 8 [
    (gogoproto.nullable) = false
  ]; // Variable retrieval price per 128KiB blob (locked on session open).

  uint64 retrieval_burn_bps = 9; // Burn cut in basis points (e.g., 500 = 5%).

  // Length of the elasticity spend window ("month") in blocks.
  // When the chain height exceeds spend_window_start_height + month_len_blocks,
  // the window resets and spend_window_spent returns to 0.
  uint64 month_len_blocks = 10;

  // --- Unified Liveness / Quotas (Mode 1 + Mode 2) ---
  // Length of a liveness epoch in blocks. Used for deterministic challenge derivation.
  uint64 epoch_len_blocks = 11;

  // Proof quota per epoch, in basis points of slot-responsible bytes.
  // See `rfcs/rfc-challenge-derivation-and-quotas.md`.
  uint64 quota_bps_per_epoch_hot = 12;
  uint64 quota_bps_per_epoch_cold = 13;

  // Minimum/maximum number of blobs required per epoch per assignment.
  uint64 quota_min_blobs = 14;
  uint64 quota_max_blobs = 15;

  // Cap on how much of the quota can be satisfied via organic retrieval credits.
  uint64 credit_cap_bps = 16;

  // Evict (trigger repair) after this many consecutive missed epochs.
  uint64 evict_after_missed_epochs = 17;

  // --- Slashing + jailing policy (B1) ---
  uint64 slash_invalid_proof_bps = 18;
  uint64 slash_wrong_data_bps = 19;
  uint64 slash_nonresponse_bps = 20;

  uint64 jail_invalid_proof_epochs = 21;
  uint64 jail_wrong_data_epochs = 22;
  uint64 jail_nonresponse_epochs = 23;

  uint64 nonresponse_threshold = 24;
  uint64 nonresponse_window_epochs = 25;

  uint64 max_strikes_before_global_jail = 26;
  uint64 strike_window_epochs = 27;

  uint64 evict_after_missed_epochs_hot = 28;
  uint64 evict_after_missed_epochs_cold = 29;

  // --- Bonding (B2) ---
  cosmos.base.v1beta1.Coin min_provider_bond = 30 [
    (gogoproto.nullable) = false
  ];
  uint64 bond_months = 31;
  uint64 provider_unbonding_blocks = 32;

  // --- Replacement (B4) ---
  uint64 replacement_cooldown_blocks = 33;
  uint64 repair_attempts_cap = 34;
  uint64 repair_attempt_window_blocks = 35;

  // --- Deputy + audit budget (B5) ---
  uint64 premium_bps = 36;
  cosmos.base.v1beta1.Coin evidence_bond = 37 [
    (gogoproto.nullable) = false
  ];
  cosmos.base.v1beta1.Coin failure_bounty = 38 [
    (gogoproto.nullable) = false
  ];
  uint64 evidence_bond_burn_bps_on_expiry = 39;
  uint64 proof_of_failure_ttl_epochs = 40;
  uint64 audit_budget_bps = 41;
  uint64 audit_budget_cap_bps = 42;
  uint64 audit_budget_carryover_epochs = 43;

  // --- Credits phase-in (B6) ---
  uint64 credit_cap_bps_hot = 44;
  uint64 credit_cap_bps_cold = 45;

  // --- Trusted repair override (dev/test only) ---
  bool repair_override_enabled = 46;
}
```

