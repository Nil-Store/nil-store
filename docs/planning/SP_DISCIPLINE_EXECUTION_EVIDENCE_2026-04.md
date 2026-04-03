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
