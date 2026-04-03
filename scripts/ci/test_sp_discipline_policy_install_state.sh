#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

CHECK_SCRIPT="scripts/ci/check_sp_discipline_policy_install_state.sh"
SOURCE_TEMPLATE="ci/workflow_templates/sp_discipline_stack_policy.yml"

if [ ! -x "$CHECK_SCRIPT" ]; then
  echo "ERROR: missing executable check script: $CHECK_SCRIPT" >&2
  exit 2
fi
if [ ! -f "$SOURCE_TEMPLATE" ]; then
  echo "ERROR: missing source template: $SOURCE_TEMPLATE" >&2
  exit 2
fi

TMPDIR="$(mktemp -d)"
cleanup() {
  if [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup EXIT

TEMPLATE="$TMPDIR/template.yml"
INSTALLED="$TMPDIR/installed.yml"
cp "$SOURCE_TEMPLATE" "$TEMPLATE"

run_case() {
  SP_POLICY_TEMPLATE_PATH="$TEMPLATE" \
  SP_POLICY_INSTALLED_PATH="$INSTALLED" \
  bash "$CHECK_SCRIPT" "$@"
}

expect_fail() {
  local label="$1"
  shift
  if run_case "$@"; then
    echo "ERROR: expected failure for case: $label" >&2
    exit 1
  fi
  echo "OK: expected failure for case: $label"
}

echo "==> Case 1: template-only mode passes"
run_case

echo "==> Case 2: strict mode fails when installed workflow is missing"
expect_fail "strict-missing" --require-installed

echo "==> Case 3: strict mode passes when installed workflow is synchronized"
cp "$TEMPLATE" "$INSTALLED"
run_case --require-installed

echo "==> Case 4: strict mode fails when installed workflow drifts from template"
printf '\n# drift-marker\n' >> "$INSTALLED"
expect_fail "strict-drift" --require-installed

echo "SP discipline policy install-state scenario tests passed."
