# SP Discipline and Reliability Enforcement PR Stack (2026-04)

## Goal
Implement a deterministic, testable discipline system so unreliable SPs are progressively restricted and removed from active service, with clear on-chain state transitions and operator UX.

## Testing Depth Contract (Mandatory)
1. No PR in this stack may be marked ready for review without local green results for all listed PR test gates.
2. Every PR must include:
- Exact command transcript (or summarized output with exit codes) for required test gates.
- Added/updated tests that fail before the change and pass after it.
- Negative-path tests for adversarial inputs (invalid proofs, replayed evidence, stale sessions, wrong provider identity).
3. Any flaky test must be fixed or quarantined with owner + issue link before merge approval can be requested.
4. “Thorough testing” for this stack means all of:
- Keeper unit coverage for changed behavior.
- Determinism/replay checks for epoch logic.
- At least one end-to-end repair path execution where applicable.
- Manual operator UX verification for any onboarding/dashboard text/state changes.

## Merge Safety Contract (Mandatory)
1. All work lands as a stacked PR series, one PR per phase below.
2. No PR in this stack may be merged to `main` without explicit human approval containing the exact phrase `YES MERGE`.
3. The phrase must appear as a human-authored PR comment on that PR.
4. For each PR, the merge actor must copy the comment URL into the merge commit message or PR description before merge.
5. If `YES MERGE` is missing, the PR stays open even if CI is green.

## Branch and PR Stack Shape
1. `stack/sp-discipline-00-merge-gate`
2. `stack/sp-discipline-01-taxonomy-and-signals`
3. `stack/sp-discipline-02-conviction-state`
4. `stack/sp-discipline-03-status-transitions`
5. `stack/sp-discipline-04-economic-penalties`
6. `stack/sp-discipline-05-repair-integration`
7. `stack/sp-discipline-06-ui-and-runbooks`
8. `stack/sp-discipline-07-test-hardening`
9. `stack/sp-discipline-08-gate-runner`
10. `stack/sp-discipline-09-evidence-capture`
11. `stack/sp-discipline-10-stack-integrity-guard`
12. `stack/sp-discipline-11-doc-consistency-guard`
13. `stack/sp-discipline-12-ci-policy-handoff`
14. `stack/sp-discipline-13-policy-install-state`
15. `stack/sp-discipline-14-policy-install-tests`
16. `stack/sp-discipline-15-doc-consistency-scenarios`
17. `stack/sp-discipline-16-stack-integrity-scenarios`
18. `stack/sp-discipline-17-policy-template-scenarios`
19. `stack/sp-discipline-18-yes-merge-scenarios`
20. `stack/sp-discipline-19-evidence-capture-scenarios`

Each branch is cut from the previous stack branch, not from `main`.

## PR 00: Merge Gate and Process Enforcement
Scope:
- Add a CI workflow that fails PR validation unless a comment contains exactly `YES MERGE`.
- Add a small script to validate the phrase from GitHub API payload for local reproducibility.
- Document required branch protection setting to make this check required on `main`.

Files (expected):
- `.github/workflows/require-yes-merge.yml`
- `scripts/ci/check_yes_merge.sh`
- `docs/AGENTS_RUNBOOK_REPO_ANCHORED.md`

Test Gate:
1. `bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/no_approval.json` must fail.
2. `bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/yes_merge.json` must pass.
3. `go test ./...` is not required for this PR if no Go files changed.

Exit Criteria:
- PR check is red without `YES MERGE`, green with it.

## PR 01: Offense Taxonomy and Signal Hygiene
Scope:
- Normalize offense/event naming for invalid proofs, non-response, quota miss, deputy miss.
- Remove misleading wording that implies stake slashing where only repair is performed.
- Keep behavior unchanged; this PR is semantics and observability cleanup.

Files (expected):
- `nilchain/x/nilchain/keeper/slashing.go`
- `nilchain/x/nilchain/keeper/msg_server.go`
- `docs/manual-devnet-runbook.md`

Test Gate:
1. `go test ./nilchain/x/nilchain/keeper -run 'TestCheckMissedProofs|TestCancelRetrievalSession_RecordsNonResponseEvidence|TestProveLiveness_HealthFailures_StartMode2Repair'`
2. `go test ./nilchain/x/nilchain/keeper`

Exit Criteria:
- Terminology accurately reflects implemented behavior.
- Existing keeper tests remain green.

## PR 02: Conviction State (ProviderDisciplineState)
Scope:
- Add keeper state for rolling offense counters by provider and offense class.
- Add deterministic epoch-window accumulation and decay.
- Track only session-bound or deterministic chain-derived offenses.

Files (expected):
- `nilchain/proto/nilchain/nilchain/v1/types.proto`
- `nilchain/x/nilchain/keeper/keeper.go`
- `nilchain/x/nilchain/keeper/msg_server.go`
- `nilchain/x/nilchain/keeper/slashing.go`
- `nilchain/x/nilchain/types/keys.go`

Test Gate:
1. New unit tests for counter increment, window decay, and deterministic replay.
2. `go test ./nilchain/x/nilchain/keeper -run 'Test.*Discipline.*|TestCancelRetrievalSession_RecordsNonResponseEvidence|TestCheckMissedProofs_.*'`
3. `go test ./nilchain/x/nilchain/keeper`

Exit Criteria:
- Conviction state updates are deterministic and reproducible from chain events.

## PR 03: Status Transitions (Active -> Offline -> Jailed)
Scope:
- Introduce param-driven thresholds and transitions.
- Enforce assignment exclusion for `Offline` and `Jailed`.
- Add unjail/probation path with explicit cooldown parameters.

Files (expected):
- `nilchain/proto/nilchain/nilchain/v1/params.proto`
- `nilchain/proto/nilchain/nilchain/v1/tx.proto`
- `nilchain/x/nilchain/types/params.go`
- `nilchain/x/nilchain/keeper/keeper.go`
- `nilchain/x/nilchain/keeper/msg_server.go`

Test Gate:
1. New tests for transitions and cooldown path.
2. `go test ./nilchain/x/nilchain/keeper -run 'Test.*Status.*|TestAssignProviders_SkipsDrainingProviders|TestCheckMissedProofs_.*'`
3. `go test ./nilchain/x/nilchain/keeper`

Exit Criteria:
- Providers in `Offline` or `Jailed` cannot receive new assignments.
- Transition and recovery logic is fully test-covered.

## PR 04: Economic Penalties
Scope:
- Implement punitive path for invalid proofs.
- Implement progressive penalties for repeated non-response/quota offenses.
- Keep base reward gating behavior and make punitive accounting explicit.

Files (expected):
- `nilchain/x/nilchain/keeper/msg_server.go`
- `nilchain/x/nilchain/keeper/base_rewards.go`
- `nilchain/x/nilchain/keeper/economics_gamma4_test.go`

Test Gate:
1. New tests for penalty math and accounting invariants.
2. `go test ./nilchain/x/nilchain/keeper -run 'TestBaseRewardPool_.*|Test.*Penalty.*|Test.*Evidence.*'`
3. `go test ./nilchain/x/nilchain/keeper`

Exit Criteria:
- Penalty events are deterministic and reflected in rewards/escrow accounting.

## PR 05: Repair Integration and Mode1 Gap Closure
Scope:
- Trigger repair escalation on status degradation.
- Add Mode1 replacement mechanism parity where feasible.
- Preserve churn and repairing budget guardrails.

Files (expected):
- `nilchain/x/nilchain/keeper/slashing.go`
- `nilchain/x/nilchain/keeper/draining.go`
- `nilchain/x/nilchain/keeper/rotation.go`

Test Gate:
1. `go test ./nilchain/x/nilchain/keeper -run 'TestCheckMissedProofs_.*|TestCheckMissedProofs_CompletesMode2SlotRepairWhenQuotaMet|TestCheckMissedProofs_DeputyServedTriggersRepairEvenIfQuotaMet|TestCheckMissedProofs_SchedulesDrainRepairs'`
2. `go test ./nilchain/x/nilchain/keeper`
3. `./scripts/e2e_deputy_ghost_repair_multi_sp.sh`

Exit Criteria:
- Bad providers are rotated out under deterministic caps without violating liveness.

## PR 06: UI, Operator Guidance, and Runbooks
Scope:
- Expose discipline state and exact remediation in onboarding and dashboard.
- Add explicit copy-paste remediation commands for wrong-provider and jailed/offline conditions.
- Align web text with runtime persona terminology.

Files (expected):
- `nil-website/src/...` onboarding and dashboard components
- `docs/manual-devnet-runbook.md`
- `docs/runtime-personas.md` if terminology updates are required

Test Gate:
1. Frontend unit tests for state rendering and recommendation cards.
2. Frontend e2e flow for provider onboarding to dashboard transition.
3. Manual smoke against local devnet with one intentionally degraded provider.

Exit Criteria:
- Operators understand why a provider is blocked and exactly how to recover.

## PR 07: Test Harness Hardening and Execution Evidence
Scope:
- Harden long-running e2e scripts so CI/manual runs are deterministic across macOS/Linux.
- Remove false negatives from nil_core symbol checks on platforms where C symbols are prefixed with `_`.
- Stabilize retrieval-session setup in deputy repair e2e (bounded expiry and tx query retries).
- Record executed command evidence for the full stack test matrix.

Files (expected):
- `scripts/run_devnet_alpha_multi_sp.sh`
- `scripts/e2e_deputy_ghost_repair_multi_sp.sh`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/no_approval.json` (must fail)
2. `bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/yes_merge.json` (must pass)
3. `GOFLAGS=-mod=mod go test ./x/nilchain/keeper -run 'TestCheckMissedProofs_.*|Test.*Discipline.*|Test.*Status.*' -count=2` (run from `nilchain/`)
4. `GOFLAGS=-mod=mod go test ./x/nilchain/keeper` (run from `nilchain/`)
5. `GOFLAGS=-mod=mod go test ./x/nilchain/...` (run from `nilchain/`)
6. `bash ./scripts/e2e_deputy_ghost_repair_multi_sp.sh` (must pass)
7. `npm --prefix nil-website run test:unit`
8. `npm --prefix nil-website run build:app`

Exit Criteria:
- E2E repair script passes without local one-off edits.
- Evidence log includes command list + pass/fail outcomes for all mandatory gates.

## PR 08: Stack Gate Runner and Repeatable Validation
Scope:
- Add a one-command runner for the full SP discipline gate matrix.
- Keep `YES MERGE` negative/positive fixture checks as explicit first-class gates.
- Make local stack validation repeatable before opening or updating stacked PRs.

Files (expected):
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass)
2. Re-run `bash scripts/ci/run_sp_discipline_stack_gates.sh` to confirm deterministic pass/fail behavior.

Exit Criteria:
- A single CI/local command executes all mandatory stack gates.
- The runbook/evidence docs list that command as canonical pre-merge validation.

## PR 09: Evidence Capture Automation
Scope:
- Add a wrapper command that runs the full stack gates and writes timestamped evidence artifacts.
- Emit both raw logs and a concise markdown summary with exit code and merge-safety reminder.
- Keep artifacts outside tracked docs by default to avoid manual transcript drift.

Files (expected):
- `scripts/ci/capture_sp_discipline_gate_evidence.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass)
2. Verify generated files exist under `_artifacts/ci/`:
   - `sp_discipline_stack_gates_<timestamp>.log`
   - `sp_discipline_stack_gates_<timestamp>.md`

Exit Criteria:
- Operators can generate a full, timestamped evidence bundle with one command.
- Evidence summary explicitly includes the `YES MERGE` merge block requirement.

## PR 10: Stack Integrity Guardrail
Scope:
- Add a stack-integrity checker that validates branch presence and ancestry order from the planning doc.
- Fail fast if any stack branch appears already merged into `origin/main` before explicit approval flow.
- Integrate this check into the one-command gate runner so process safety is enforced every run.

Files (expected):
- `scripts/ci/check_sp_discipline_stack_integrity.sh`
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/check_sp_discipline_stack_integrity.sh` (must pass)
2. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass with integrity check included)
3. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass and write artifacts)

Exit Criteria:
- Stack branch order is machine-checked before every full gate run.
- The gate runner blocks when stack branches are missing, out of order, or already merged into `main`.

## PR 11: Plan/Evidence Doc Consistency Guard
Scope:
- Add a docs consistency check that verifies stack branch lists match between plan and evidence docs.
- Verify `YES MERGE` reminder text exists in both docs to preserve explicit merge safety messaging.
- Integrate this check into the one-command gate runner.

Files (expected):
- `scripts/ci/check_sp_discipline_docs_consistency.sh`
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/check_sp_discipline_docs_consistency.sh` (must pass)
2. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass with docs check included)
3. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass and write artifacts)

Exit Criteria:
- Plan and evidence docs cannot silently drift on stack branch ordering.
- Gate runner fails immediately if docs diverge on branch list or missing `YES MERGE` language.

## PR 12: CI Policy Workflow Handoff (No Workflow-Scope Token)
Scope:
- Add a checked-in workflow template for SP discipline policy enforcement outside `.github/workflows/`.
- Add installer/check scripts so maintainers with workflow-scoped credentials can apply the template safely.
- Integrate template validation into the one-command gate runner.

Files (expected):
- `ci/workflow_templates/sp_discipline_stack_policy.yml`
- `scripts/ci/install_sp_discipline_policy_workflow.sh`
- `scripts/ci/check_sp_discipline_policy_template.sh`
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `ruby -e "require 'yaml'; YAML.load_file('ci/workflow_templates/sp_discipline_stack_policy.yml')"` (must pass)
2. `bash scripts/ci/check_sp_discipline_policy_template.sh` (must pass)
3. `bash scripts/ci/check_sp_discipline_stack_integrity.sh` (must pass)
4. `bash scripts/ci/check_sp_discipline_docs_consistency.sh` (must pass)
5. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass)
6. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass and write artifacts)

Exit Criteria:
- Policy workflow content is versioned and validated without requiring direct workflow-file push permissions.
- Maintainers can install the workflow template with a single audited command when workflow-scope credentials are available.

## PR 13: Policy Install-State Guard
Scope:
- Add an install-state checker that reports whether the workflow template is installed and synchronized.
- Support strict mode (`--require-installed`) for environments that require active workflow installation.
- Integrate non-strict install-state checks into the one-command gate runner.

Files (expected):
- `scripts/ci/check_sp_discipline_policy_install_state.sh`
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/check_sp_discipline_policy_install_state.sh` (must pass in template-only mode)
2. `bash scripts/ci/check_sp_discipline_policy_install_state.sh --require-installed` (must fail when workflow is not installed)
3. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass with install-state check included)
4. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass and write artifacts)

Exit Criteria:
- Operators can see whether policy enforcement is template-only or fully installed.
- Strict mode fails with actionable guidance when the workflow file is missing or drifted.

## PR 14: Policy Install-State Scenario Tests
Scope:
- Add deterministic scenario tests for policy install-state behavior across template-only, strict-missing, strict-synced, and drifted cases.
- Allow install-state checker path overrides via env vars so scenarios run in isolated temp directories.
- Integrate scenario tests into the one-command gate runner.

Files (expected):
- `scripts/ci/check_sp_discipline_policy_install_state.sh`
- `scripts/ci/test_sp_discipline_policy_install_state.sh`
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/test_sp_discipline_policy_install_state.sh` (must pass)
2. `bash scripts/ci/check_sp_discipline_policy_install_state.sh --require-installed` (must fail in template-only repo state)
3. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass with scenario tests included)
4. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass and write artifacts)

Exit Criteria:
- Policy install-state behavior is regression-tested across both positive and negative paths.
- Gate runner includes explicit scenario-level verification, not just a single happy-path probe.

## PR 15: Docs Consistency Scenario Tests and Duplicate Detection
Scope:
- Harden docs-consistency checks so duplicate stack-branch entries fail instead of being silently deduplicated.
- Allow docs-consistency checker path overrides via env vars so scenario tests can run in isolated temp docs.
- Add deterministic scenario tests for mismatched branch lists, duplicates, and missing `YES MERGE` text.
- Integrate docs-consistency scenarios into the one-command gate runner.

Files (expected):
- `scripts/ci/check_sp_discipline_docs_consistency.sh`
- `scripts/ci/test_sp_discipline_docs_consistency.sh`
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/test_sp_discipline_docs_consistency.sh` (must pass)
2. `bash scripts/ci/check_sp_discipline_docs_consistency.sh` (must pass on repo docs)
3. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass with docs-consistency scenarios included)
4. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass and write artifacts)

Exit Criteria:
- Duplicate stack entries in either plan/evidence docs fail fast.
- Docs consistency is validated by deterministic positive and negative scenario cases.
- Gate runner executes scenario tests before the heavyweight keeper/e2e/frontend matrix.

## PR 16: Stack Integrity Scenario Tests and Duplicate Detection
Scope:
- Harden stack-integrity parsing so duplicate branch entries fail instead of being silently deduplicated.
- Add deterministic stack-integrity scenario tests covering valid order, duplicate entries, missing branches, ancestry violations, and merged-into-base blocking.
- Integrate stack-integrity scenarios into the one-command gate runner before heavy keeper/e2e/frontend checks.

Files (expected):
- `scripts/ci/check_sp_discipline_stack_integrity.sh`
- `scripts/ci/test_sp_discipline_stack_integrity.sh`
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/test_sp_discipline_stack_integrity.sh` (must pass)
2. `bash scripts/ci/check_sp_discipline_stack_integrity.sh` (must pass on repo stack)
3. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass with stack-integrity scenarios included)
4. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass and write artifacts)

Exit Criteria:
- Duplicate stack branches in plan docs fail fast.
- Stack integrity behavior is verified by deterministic positive and negative scenario cases.
- Gate runner includes explicit stack-integrity scenario coverage, not only a single happy-path check.

## PR 17: Policy Template Scenario Tests and Path Overrides
Scope:
- Add deterministic policy-template scenario tests for missing template, invalid YAML, missing installer, and installer dry-run failures.
- Allow policy-template checker and installer scripts to accept env-path overrides for isolated scenario execution.
- Integrate policy-template scenarios into the one-command gate runner ahead of heavyweight matrix steps.

Files (expected):
- `scripts/ci/check_sp_discipline_policy_template.sh`
- `scripts/ci/install_sp_discipline_policy_workflow.sh`
- `scripts/ci/test_sp_discipline_policy_template.sh`
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/test_sp_discipline_policy_template.sh` (must pass)
2. `bash scripts/ci/check_sp_discipline_policy_template.sh` (must pass on repo template)
3. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass with policy-template scenarios included)
4. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass and write artifacts)

Exit Criteria:
- Policy-template guard behavior is covered by deterministic positive and negative scenario tests.
- Template and installer scripts are testable in isolated temp environments.
- Gate runner includes explicit policy-template scenario coverage, not only a single happy-path check.

## PR 18: YES MERGE Scenario Tests and Near-Miss Guard
Scope:
- Add deterministic `YES MERGE` scenario tests covering human approvals, bot-only approvals, near-miss strings, wrong-case text, and missing bodies.
- Harden `check_yes_merge.sh` phrase matching so near-miss tokens like `YES MERGED` do not pass.
- Integrate `YES MERGE` scenario tests into the gate runner in addition to existing fixture checks.

Files (expected):
- `scripts/ci/check_yes_merge.sh`
- `scripts/ci/test_yes_merge_check.sh`
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/test_yes_merge_check.sh` (must pass)
2. `bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/no_approval.json` (must fail)
3. `bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/yes_merge.json` (must pass)
4. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass with YES MERGE scenarios included)
5. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass and write artifacts)

Exit Criteria:
- `YES MERGE` guard behavior is covered by deterministic positive and negative scenario tests.
- Near-miss phrases are explicitly rejected by the checker.
- Gate runner includes explicit scenario coverage for merge-approval enforcement.

## PR 19: Evidence Capture Scenario Tests and Deterministic Overrides
Scope:
- Harden evidence capture automation with deterministic timestamp overrides for reproducible scenario assertions.
- Add gate-runner path override support so capture behavior can be tested in isolated temp environments.
- Add deterministic evidence-capture scenarios for pass/fail runner outcomes and missing-runner handling.
- Integrate evidence-capture scenarios into the gate runner before heavyweight keeper/e2e/frontend checks.

Files (expected):
- `scripts/ci/capture_sp_discipline_gate_evidence.sh`
- `scripts/ci/test_sp_discipline_evidence_capture.sh`
- `scripts/ci/run_sp_discipline_stack_gates.sh`
- `docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md`
- `docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md`

Test Gate:
1. `bash scripts/ci/test_sp_discipline_evidence_capture.sh` (must pass)
2. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (must pass with evidence-capture scenarios included)
3. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh` (must pass and write artifacts)

Exit Criteria:
- Evidence capture behavior is covered by deterministic positive and negative scenario tests.
- Evidence capture can be scenario-tested without executing the full heavyweight gate matrix.
- Gate runner includes explicit scenario coverage for evidence automation integrity.

## Cross-PR Regression Matrix
Run this matrix at minimum for PRs 02-06 (state/economics/repair/UX affecting):
1. Determinism:
- Re-run the same keeper test subset twice and confirm identical pass/fail and stable snapshots:
`go test ./nilchain/x/nilchain/keeper -run 'TestCheckMissedProofs_.*|Test.*Discipline.*|Test.*Status.*' -count=2`
2. Evidence integrity:
- `go test ./nilchain/x/nilchain/keeper -run 'TestCancelRetrievalSession_RecordsNonResponseEvidence|Test.*Evidence.*'`
3. Reward and penalty accounting:
- `go test ./nilchain/x/nilchain/keeper -run 'TestBaseRewardPool_.*|Test.*Penalty.*|Test.*RetrievalFees.*'`
4. Repair and churn safety:
- `go test ./nilchain/x/nilchain/keeper -run 'TestCheckMissedProofs_.*|TestCheckMissedProofs_SchedulesDrainRepairs|TestProveLiveness_HealthFailures_StartMode2Repair'`
5. Multi-SP end-to-end:
- `./scripts/e2e_deputy_ghost_repair_multi_sp.sh`
6. UI/operator manual checks (for PR 06 and any UX text changes):
- Verify wrong-provider, offline, jailed, and recovered-provider states are explicit and include exact remediation commands.
7. Harness reliability (for PR 07):
- Verify `scripts/e2e_deputy_ghost_repair_multi_sp.sh` passes on a clean local run without manual script edits.

## Full Stack Validation Before Any Merge to Main
Run after PR 06 rebased on latest stack head:
1. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
2. Manual runbook checks from `docs/manual-devnet-runbook.md` Section 5
3. Frontend onboarding + dashboard smoke with wrong-provider and recovered-provider scenarios

## Rollout Strategy
1. Shadow mode first: compute convictions and surface status proposals without enforcement.
2. Enforce `Offline` transitions with conservative thresholds.
3. Enable `Jailed` and punitive economics after telemetry confirms low false positives.

## Acceptance Criteria
1. Persistently bad SPs are excluded from new assignments.
2. Persistently bad SPs lose rewards and can be jailed under deterministic thresholds.
3. Slot repair and replacement remain bounded and deterministic.
4. UX clearly communicates cause, status, and exact remediation.
5. No merge to `main` occurs without a human `YES MERGE` comment.
