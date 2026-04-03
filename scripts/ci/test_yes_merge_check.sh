#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

CHECK_SCRIPT="scripts/ci/check_yes_merge.sh"
if [ ! -x "$CHECK_SCRIPT" ]; then
  echo "ERROR: missing executable script: $CHECK_SCRIPT" >&2
  exit 2
fi

TMPDIR="$(mktemp -d)"
cleanup() {
  if [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup EXIT

expect_pass() {
  local label="$1"
  local fixture="$2"
  if ! bash "$CHECK_SCRIPT" --fixture "$fixture" >/dev/null; then
    echo "ERROR: expected pass for case: $label" >&2
    exit 1
  fi
  echo "OK: expected pass for case: $label"
}

expect_fail() {
  local label="$1"
  local fixture="$2"
  if bash "$CHECK_SCRIPT" --fixture "$fixture" >/dev/null; then
    echo "ERROR: expected failure for case: $label" >&2
    exit 1
  fi
  echo "OK: expected failure for case: $label"
}

cat >"$TMPDIR/human_exact.json" <<'EOF'
[
  {
    "user": { "type": "User", "login": "maintainer" },
    "body": "YES MERGE"
  }
]
EOF

cat >"$TMPDIR/human_punctuated.json" <<'EOF'
[
  {
    "user": { "type": "User", "login": "maintainer" },
    "body": "Final checks passed: YES MERGE."
  }
]
EOF

cat >"$TMPDIR/bot_only.json" <<'EOF'
[
  {
    "user": { "type": "Bot", "login": "ci-bot" },
    "body": "YES MERGE"
  }
]
EOF

cat >"$TMPDIR/human_near_miss.json" <<'EOF'
[
  {
    "user": { "type": "User", "login": "maintainer" },
    "body": "YES MERGED"
  }
]
EOF

cat >"$TMPDIR/human_wrong_case.json" <<'EOF'
[
  {
    "user": { "type": "User", "login": "maintainer" },
    "body": "yes merge"
  }
]
EOF

cat >"$TMPDIR/human_no_body.json" <<'EOF'
[
  {
    "user": { "type": "User", "login": "maintainer" }
  }
]
EOF

echo "==> Case 1: human exact phrase passes"
expect_pass "human-exact" "$TMPDIR/human_exact.json"

echo "==> Case 2: human phrase with punctuation passes"
expect_pass "human-punctuated" "$TMPDIR/human_punctuated.json"

echo "==> Case 3: bot-only phrase fails"
expect_fail "bot-only" "$TMPDIR/bot_only.json"

echo "==> Case 4: near-miss phrase fails"
expect_fail "human-near-miss" "$TMPDIR/human_near_miss.json"

echo "==> Case 5: wrong case fails"
expect_fail "human-wrong-case" "$TMPDIR/human_wrong_case.json"

echo "==> Case 6: missing body fails"
expect_fail "human-missing-body" "$TMPDIR/human_no_body.json"

echo "YES MERGE checker scenario tests passed."
