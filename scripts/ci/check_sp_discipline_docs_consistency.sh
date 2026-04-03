#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

PLAN_DOC="${SP_DISCIPLINE_PLAN_DOC_PATH:-docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md}"
EVIDENCE_DOC="${SP_DISCIPLINE_EVIDENCE_DOC_PATH:-docs/planning/SP_DISCIPLINE_EXECUTION_EVIDENCE_2026-04.md}"

if [ ! -f "$PLAN_DOC" ]; then
  echo "ERROR: missing plan doc: $PLAN_DOC" >&2
  exit 2
fi
if [ ! -f "$EVIDENCE_DOC" ]; then
  echo "ERROR: missing evidence doc: $EVIDENCE_DOC" >&2
  exit 2
fi

extract_branches() {
  local file="$1"
  sed -n 's/^[0-9][0-9]*\. `\(stack\/sp-discipline-[^`]*\)`/\1/p' "$file"
}

check_no_duplicates() {
  local label="$1"
  local branches="$2"
  local dupes
  dupes="$(printf '%s\n' "$branches" | sort | uniq -d)"
  if [ -n "$dupes" ]; then
    echo "ERROR: duplicate stack branches found in $label doc list:" >&2
    echo "$dupes" >&2
    exit 1
  fi
}

PLAN_BRANCHES="$(extract_branches "$PLAN_DOC")"
EVIDENCE_BRANCHES="$(extract_branches "$EVIDENCE_DOC")"

if [ -z "$PLAN_BRANCHES" ] || [ -z "$EVIDENCE_BRANCHES" ]; then
  echo "ERROR: failed to parse stack branch lists from docs" >&2
  exit 1
fi

check_no_duplicates "plan" "$PLAN_BRANCHES"
check_no_duplicates "evidence" "$EVIDENCE_BRANCHES"

if [ "$PLAN_BRANCHES" != "$EVIDENCE_BRANCHES" ]; then
  echo "ERROR: stack branch lists differ between plan and evidence docs" >&2
  echo "--- plan branches ---" >&2
  echo "$PLAN_BRANCHES" >&2
  echo "--- evidence branches ---" >&2
  echo "$EVIDENCE_BRANCHES" >&2
  exit 1
fi

for file in "$PLAN_DOC" "$EVIDENCE_DOC"; do
  if ! grep -Fq "YES MERGE" "$file"; then
    echo "ERROR: merge approval phrase missing in $file" >&2
    exit 1
  fi
done

echo "SP discipline docs consistency check passed."
