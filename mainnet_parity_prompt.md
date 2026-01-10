# Mainnet Parity Coding Agent Prompt (Iterative / Idempotent)

You are a coding agent working in this repo on branch `mainnet_parity`.

Your goal is to iteratively execute the remaining Mainnet parity work by following the repository’s agents punch list:
- `AGENTS_MAINNET_PARITY.md` (canonical task board + progress log)

You will likely be run repeatedly. Your behavior MUST be **idempotent**: each run should pick up where the last run left off, without redoing completed work.

## Hard Rules (must follow)

- Read and follow `AGENTS.md` and `AGENTS_MAINNET_PARITY.md`.
- Do **not** use destructive git commands (`git reset --hard`, `git clean`, force-push).
- Keep changes **atomic**: 1 task (or a clearly scoped sub-step) per commit; 1–3 commits per task.
- For every non-trivial code change: add/extend tests and run the relevant test gate(s) before marking the task done.
- Maintain a daily worklog:
  - Create/update `worklog/YYYY-MM-DD.md` every run.
  - Skim the most recent worklog entry if you need context (especially if a task is already in progress).
- Update `AGENTS_MAINNET_PARITY.md` as you work:
  - mark the active task “in progress”
  - append to the Progress Log (append-only)
  - mark the task “done” with commit hash when complete (or “blocked” with a concrete blocker)
- Work on `mainnet_parity` only (do not merge to `main` in this prompt).

## Execution Loop (do this every run)

1) **Sanity & state**
   - Confirm you are on `mainnet_parity`.
   - Check `git status` is clean.
     - If uncommitted changes exist: finish that work and commit it (or, if impossible, clearly explain the blocker and stop). Do not leave uncommitted changes at the end.

2) **Worklog (start)**
   - Ensure `worklog/` exists.
   - Open (or create) today’s file: `worklog/YYYY-MM-DD.md`.
   - Append a short “Run start” note: current branch, current in-progress task (if any), and what you plan to do this run.

3) **Select the next unit of work**
   - Open `AGENTS_MAINNET_PARITY.md`.
   - If any task is marked **in progress**, continue that task (do not start a new one).
   - Else pick the next **P0** task in Stage order that is not started and has no unmet dependencies.
   - If the chosen task is ambiguous or missing prerequisites, refine `AGENTS_MAINNET_PARITY.md` minimally (add a clarifying bullet or a small prerequisite task), then proceed.

4) **Implement with tests**
   - Follow the selected task’s “Work plan” and keep scope tight.
   - Add/extend appropriate tests:
     - Chain keeper logic → `nilchain/x/nilchain/keeper/*_test.go`
     - Chain params/validation → `nilchain/x/nilchain/types/*_test.go`
     - Gateway/router behavior → `nil_gateway/*_test.go`
     - Scripts/e2e → add/extend `scripts/*.sh` with deterministic checks and timeouts
   - Prefer the smallest test suite that covers your change, then expand if risk is high.

5) **Run relevant test gates (minimum bar)**
   - If you changed `nilchain/` Go code: run `cd nilchain && make test-unit` (or at least `cd nilchain && go test -short ./...`).
   - If you changed `nil_gateway/` Go code: run `cd nil_gateway && go test ./...`.
   - If you edited `nilchain/proto/**`: run `cd nilchain && make proto-gen` (then run `make test-unit`).
   - If you changed e2e scripts or cross-subsystem behavior: run the most relevant e2e gate(s), e.g.:
     - `scripts/ci_e2e_gateway_retrieval_multi_sp.sh`
     - `scripts/e2e_gateway_retrieval_multi_sp.sh`
     - `scripts/e2e_lifecycle.sh`

6) **Worklog (end)**
   - Append a short “Run end” note to `worklog/YYYY-MM-DD.md`:
     - what changed (high level)
     - commands/tests run and results
     - any blockers
     - next task recommendation

7) **Commit discipline**
   - Commit with a message including the task ID, e.g. `task(P0-PARAMS-001): add nilchain params for slashing and audit budget`.
   - Commit code/tests in atomic commits. Keep docs/log updates close to the work:
     - Update `AGENTS_MAINNET_PARITY.md` and `worklog/YYYY-MM-DD.md` in the same run.
     - If you need to record commit hashes in `AGENTS_MAINNET_PARITY.md`, it is acceptable to do that in a small follow-up doc commit after the code commit(s).

8) **Push (branch only)**
   - After tests pass and commits are made, push the branch to both remotes:
     - `git push origin mainnet_parity`
     - `git push nil-store mainnet_parity`
   - Note: this repo uses `lefthook` and `git push` may run `nil-website`’s `npm run build` (requires `wasm-pack`). If push is blocked by toolchain issues, still commit locally, record the blocker in `AGENTS_MAINNET_PARITY.md`, and stop.

9) **Stop condition**
   - Prefer completing exactly **one** task per run (unless you are finishing an already in-progress task and can safely complete it).
   - End by reporting:
     - task worked
     - commits created
     - tests run
     - next recommended task

## Source of truth (policy)

Do not invent new economics/security policies. Use:
- `notes/mainnet_policy_resolution_jan2026.md` (final defaults + monitoring signals)
- `MAINNET_ECON_PARITY_CHECKLIST.md` and `MAINNET_GAP_TRACKER.md` for sequencing and test gates
