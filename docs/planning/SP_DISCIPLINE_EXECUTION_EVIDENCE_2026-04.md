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
15. `stack/sp-discipline-14-policy-install-tests`
16. `stack/sp-discipline-15-doc-consistency-scenarios`
17. `stack/sp-discipline-16-stack-integrity-scenarios`
18. `stack/sp-discipline-17-policy-template-scenarios`
19. `stack/sp-discipline-18-yes-merge-scenarios`
20. `stack/sp-discipline-19-evidence-capture-scenarios`
21. `stack/sp-discipline-20-gate-runner-scenarios`
22. `stack/sp-discipline-21-evidence-collision-guard`
23. `stack/sp-discipline-22-doc-sequence-guard`
24. `stack/sp-discipline-23-doc-ordinal-guard`

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
- Notes: Template-only mode still reports clear install guidance.

25. `bash scripts/ci/check_sp_discipline_policy_install_state.sh --require-installed`
- Result: fail (expected)
- Notes: Strict mode fails as expected when workflow file is not installed.

26. `bash scripts/ci/test_sp_discipline_policy_install_state.sh`
- Result: pass
- Notes: Scenario matrix validated template-only, strict-missing, strict-synced, and drifted cases.

27. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass with policy install-state scenario tests included.

28. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated fresh timestamped evidence bundle after PR14 scenario-test additions.

29. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Revalidated full matrix on `stack/sp-discipline-14-policy-install-tests` before commit.

30. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated `_artifacts/ci/sp_discipline_stack_gates_20260403T095647Z.log` and `_artifacts/ci/sp_discipline_stack_gates_20260403T095647Z.md`.

31. `bash scripts/ci/test_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Scenario matrix validated mismatch, duplicate plan/evidence branches, and missing `YES MERGE` failures.

32. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Repo docs pass with strict duplicate-branch detection enabled.

33. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass with docs-consistency scenarios integrated into the gate runner.

34. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated `_artifacts/ci/sp_discipline_stack_gates_20260403T100655Z.log` and `_artifacts/ci/sp_discipline_stack_gates_20260403T100655Z.md` after PR15 changes.

35. `bash scripts/ci/test_sp_discipline_stack_integrity.sh`
- Result: pass
- Notes: Scenario matrix validated duplicate entries, missing branches, ancestry order violations, and merged-into-base blocking.

36. `bash scripts/ci/check_sp_discipline_stack_integrity.sh`
- Result: pass
- Notes: Repo stack passes with strict duplicate-branch detection enabled.

37. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass with stack-integrity scenarios integrated into the gate runner.

38. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated `_artifacts/ci/sp_discipline_stack_gates_20260403T101620Z.log` and `_artifacts/ci/sp_discipline_stack_gates_20260403T101620Z.md` after PR16 changes.

39. `bash scripts/ci/test_sp_discipline_policy_template.sh`
- Result: pass
- Notes: Scenario matrix validated missing template, invalid YAML, missing installer, and installer dry-run failure handling.

40. `bash scripts/ci/check_sp_discipline_policy_template.sh`
- Result: pass
- Notes: Repo policy template and installer dry-run checks pass with path-override support enabled.

41. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass with policy-template scenarios integrated into the gate runner.

42. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated `_artifacts/ci/sp_discipline_stack_gates_20260403T102612Z.log` and `_artifacts/ci/sp_discipline_stack_gates_20260403T102612Z.md` after PR17 changes.

43. `bash scripts/ci/test_yes_merge_check.sh`
- Result: pass
- Notes: Scenario matrix validated human approvals, bot-only failures, near-miss phrase rejection, wrong-case rejection, and missing-body rejection.

44. `bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/no_approval.json`
- Result: fail (expected)
- Notes: Negative fixture remains red without human-authored `YES MERGE`.

45. `bash scripts/ci/check_yes_merge.sh --fixture scripts/ci/fixtures/yes_merge.json`
- Result: pass
- Notes: Positive fixture remains green with human-authored `YES MERGE`.

46. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass with YES MERGE scenario tests integrated into the gate runner.

47. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated `_artifacts/ci/sp_discipline_stack_gates_20260403T103519Z.log` and `_artifacts/ci/sp_discipline_stack_gates_20260403T103519Z.md` after PR18 changes.

48. `bash scripts/ci/test_sp_discipline_evidence_capture.sh`
- Result: pass
- Notes: Scenario matrix validated pass/fail gate-runner outcomes, deterministic timestamp overrides, summary content assertions, and missing-runner failure handling.

49. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass with evidence-capture scenarios integrated into the gate runner.

50. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated `_artifacts/ci/sp_discipline_stack_gates_20260403T104615Z.log` and `_artifacts/ci/sp_discipline_stack_gates_20260403T104615Z.md` after PR19 changes.

51. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Revalidated plan/evidence branch-list parity and `YES MERGE` wording after PR19 doc updates.

52. `bash scripts/ci/test_sp_discipline_evidence_capture.sh`
- Result: pass
- Notes: Revalidated deterministic evidence-capture scenarios after PR19 doc updates.

53. `bash scripts/ci/test_sp_discipline_gate_runner.sh`
- Result: pass
- Notes: Scenario matrix validated fast-mode happy path, early-stop failure propagation, and expected-fail guard behavior.

54. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass with gate-runner scenarios integrated into the gate sequence.

55. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated `_artifacts/ci/sp_discipline_stack_gates_20260403T110141Z.log` and `_artifacts/ci/sp_discipline_stack_gates_20260403T110141Z.md` after PR20 changes.

56. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Revalidated branch-list parity and `YES MERGE` reminder after PR20 doc updates.

57. `bash scripts/ci/test_sp_discipline_gate_runner.sh`
- Result: pass
- Notes: Revalidated gate-runner scenario suite after PR20 doc updates.

58. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Final parity check after appending PR20 evidence entries.

59. `bash scripts/ci/test_sp_discipline_evidence_capture.sh`
- Result: pass
- Notes: Extended scenario matrix validated artifact collision failure by default and explicit overwrite replacement via `SP_DISCIPLINE_EVIDENCE_OVERWRITE=1`.

60. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass with evidence-collision scenarios integrated into evidence-capture checks.

61. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated `_artifacts/ci/sp_discipline_stack_gates_20260403T111113Z.log` and `_artifacts/ci/sp_discipline_stack_gates_20260403T111113Z.md` after PR21 changes.

62. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Revalidated branch-list parity and `YES MERGE` reminder text after PR21 doc updates.

63. `bash scripts/ci/test_sp_discipline_evidence_capture.sh`
- Result: pass
- Notes: Revalidated collision/overwrite scenario coverage after PR21 doc updates.

64. `bash scripts/ci/test_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Extended scenario matrix validated new contiguous-numbering guard, including branch-gap and out-of-order failures.

65. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Repo docs pass with contiguous stack numbering enforcement enabled for plan and evidence branch lists.

66. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass with docs-consistency sequence scenarios integrated into the gate runner.

67. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated `_artifacts/ci/sp_discipline_stack_gates_20260403T112036Z.log` and `_artifacts/ci/sp_discipline_stack_gates_20260403T112036Z.md` after PR22 changes.

68. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Final parity check after appending PR22 evidence entries.

69. `bash scripts/ci/test_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Extended scenario matrix validated markdown list ordinal continuity and ordinal-to-branch index alignment failures.

70. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Repo docs pass with combined sequence + ordinal alignment enforcement.

71. `bash scripts/ci/run_sp_discipline_stack_gates.sh`
- Result: pass
- Notes: Full matrix pass with ordinal scenario coverage integrated into docs-consistency checks.

72. `bash scripts/ci/capture_sp_discipline_gate_evidence.sh`
- Result: pass
- Notes: Generated `_artifacts/ci/sp_discipline_stack_gates_20260403T112943Z.log` and `_artifacts/ci/sp_discipline_stack_gates_20260403T112943Z.md` after PR23 changes.

73. `bash scripts/ci/check_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Final branch-list parity and ordinal alignment check after appending PR23 evidence entries.

74. `bash scripts/ci/test_sp_discipline_docs_consistency.sh`
- Result: pass
- Notes: Revalidated ordinal and sequence scenario coverage after PR23 doc updates.

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
