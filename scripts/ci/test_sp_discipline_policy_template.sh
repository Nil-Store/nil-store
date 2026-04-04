#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

CHECK_SCRIPT="scripts/ci/check_sp_discipline_policy_template.sh"
if [ ! -x "$CHECK_SCRIPT" ]; then
  echo "ERROR: missing executable check script: $CHECK_SCRIPT" >&2
  exit 2
fi

TMPDIR="$(mktemp -d)"
cleanup() {
  if [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup EXIT

VALID_TEMPLATE="$TMPDIR/template-valid.yml"
INVALID_TEMPLATE="$TMPDIR/template-invalid.yml"
MISSING_TEMPLATE="$TMPDIR/template-missing.yml"
INSTALLER_OK="$TMPDIR/installer-ok.sh"
INSTALLER_FAIL="$TMPDIR/installer-fail.sh"
MISSING_INSTALLER="$TMPDIR/installer-missing.sh"

cat >"$VALID_TEMPLATE" <<'EOF'
name: SP Discipline Policy
on:
  workflow_dispatch:
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok
EOF

cat >"$INVALID_TEMPLATE" <<'EOF'
name: SP Discipline Policy
on:
  workflow_dispatch:
jobs:
  - [broken
EOF

cat >"$INSTALLER_OK" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
MODE="${1:-dry-run}"
if [ "$MODE" = "--dry-run" ] || [ "$MODE" = "dry-run" ]; then
  exit 0
fi
if [ "$MODE" = "--apply" ]; then
  exit 0
fi
exit 2
EOF

cat >"$INSTALLER_FAIL" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "simulated installer dry-run failure" >&2
exit 1
EOF

chmod +x "$INSTALLER_OK" "$INSTALLER_FAIL"

run_case() {
  local template_path="$1"
  local installer_path="$2"
  SP_POLICY_TEMPLATE_PATH="$template_path" \
  SP_POLICY_INSTALLER_PATH="$installer_path" \
  bash "$CHECK_SCRIPT"
}

expect_fail() {
  local label="$1"
  local template_path="$2"
  local installer_path="$3"
  if run_case "$template_path" "$installer_path"; then
    echo "ERROR: expected failure for case: $label" >&2
    exit 1
  fi
  echo "OK: expected failure for case: $label"
}

echo "==> Case 1: valid template and installer pass"
run_case "$VALID_TEMPLATE" "$INSTALLER_OK"

echo "==> Case 2: missing template fails"
expect_fail "missing-template" "$MISSING_TEMPLATE" "$INSTALLER_OK"

echo "==> Case 3: invalid YAML fails"
expect_fail "invalid-yaml" "$INVALID_TEMPLATE" "$INSTALLER_OK"

echo "==> Case 4: missing installer fails"
expect_fail "missing-installer" "$VALID_TEMPLATE" "$MISSING_INSTALLER"

echo "==> Case 5: installer dry-run failure propagates"
expect_fail "installer-dry-run-failure" "$VALID_TEMPLATE" "$INSTALLER_FAIL"

echo "SP discipline policy template scenario tests passed."
