# SP Discipline Execution Evidence (2026-04)

## Merge Safety Contract
- Stacked branches are used; no direct merge to `main`.
- Human approval phrase `YES MERGE` is mandatory before merge.
- Local check helper: `scripts/ci/check_yes_merge.sh`

## Active Stack Branches
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

## Mandatory Gate Results
All commands below were executed from repo root unless noted.

1. `bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/no_approval.json`
- Result: fail (expected)
- Output includes: `FAIL: missing required human-authored approval phrase: YES MERGE`

2. `bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/yes_merge.json`
- Result: pass
- Output includes: `PASS: found human-authored YES MERGE approval`

3. `GOFLAGS=-mod=mod go test ./x/nilchain/keeper -run 'TestCheckMissedProofs_.*|Test.*Discipline.*|Test.*Status.*' -count=2`
- Workdir: `nilchain/`
- Result: pass

4. `GOFLAGS=-mod=mod go test ./x/nilchain/keeper`
- Workdir: `nilchain/`
- Result: pass

5. `GOFLAGS=-mod=mod go test ./x/nilchain/...`
- Workdir: `nilchain/`
- Result: pass

6. `bash ./scripts/e2e_deputy_ghost_repair_multi_sp.sh`
- Result: pass
- Output includes: `Deputy ghost repair E2E passed.`

7. `npm --prefix nil-website run test:unit`
- Result: pass

8. `npm --prefix nil-website run build:app`
- Result: pass

9. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Executes checks 1-8 in one command and reports a final all-green summary.

10. `bash scripts/ci/run_sp_discipline_stack_gates.sh` (second run)
- Result: pass
- Notes: Deterministic re-run also returned all-green.

11. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Produced timestamped log + markdown summary under `_artifacts/ci/` and preserved the `YES MERGE` merge block reminder.

12. `bash scripts/ci/check_sp_discipline_stack_integrity.sh`
- Result: pass
- Notes: Verified remote stack branch presence, ancestry order, and “not merged into origin/main” condition.

13. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Includes integrity check as first step, then all existing gate checks.

14. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated a fresh timestamped evidence bundle after enabling integrity checks in the gate runner.

15. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Verified stack branch list parity and explicit `YES MERGE` wording in both plan and evidence docs.

16. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Includes stack integrity + docs consistency checks before test matrix execution.

17. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated a fresh evidence bundle with docs consistency guard enabled.

18. `ruby -e "require 'yaml'; YAML.load_file('ci/workflow_templates/sp_discipline_stack_policy.yml')"`
- Result: pass
- Notes: Validated CI policy workflow template syntax.

19. `bash scripts/ci/check_sp_discipline_policy_template.sh`
- Result: pass
- Notes: Confirmed template YAML parses and installer dry-run succeeds.

20. `bash scripts/ci/check_sp_discipline_stack_integrity.sh`
- Result: pass
- Notes: Stack chain and unmerged-main guard remain green with PR12 branch.

21. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Plan/evidence parity and `YES MERGE` reminder checks pass.

22. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass including policy-template precheck.

23. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated fresh timestamped evidence bundle after PR12 handoff additions.

24. `bash scripts/ci/check_sp_discipline_policy_install_state.sh`
- Result: pass
- Notes: Passes in template-only mode when workflow file is not yet installed.

25. `bash scripts/ci/check_sp_discipline_policy_install_state.sh --require-installed`
- Result: fail (expected)
- Notes: Strict mode correctly reports missing installed workflow and install command.

26. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass including policy install-state check.

27. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated fresh timestamped evidence bundle after PR13 install-state additions.

## Test Harness Hardening Included
- `scripts/run_devnet_alpha_multi_sp.sh`
  - nil_core symbol detection now normalizes platform-specific symbol naming (e.g., `_symbol` on macOS), preventing false stale-build failures.
- `scripts/e2e_deputy_ghost_repair_multi_sp.sh`
  - create-deal gas is now configurable with safer default: `CREATE_DEAL_GAS` (default `450000`)
  - deal-id extraction is retry-based to tolerate tx indexing lag
  - retrieval session expiry is bounded by deal `end_block`, preventing invalid-request failures on long setup paths

## Final Merge Rule (Explicit)
No branch in this stack may be merged into `main` until a human posts an approval comment containing the exact phrase:

`YES MERGE`
