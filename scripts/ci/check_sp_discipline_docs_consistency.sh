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

extract_numbered_branches() {
  local file="$1"
  sed -n 's/^\([0-9][0-9]*\)\. `\(stack\/sp-discipline-[^`]*\)`/\1 \2/p' "$file"
}

check_branch_sequence() {
  local label="$1"
  local branches="$2"
  local expected=0
  local saw_any=0

  while IFS= read -r branch; do
    [ -n "$branch" ] || continue
    saw_any=1
    local idx_raw
    idx_raw="$(printf '%s' "$branch" | sed -nE 's#^stack/sp-discipline-([0-9]+)-.*#\1#p')"
    if [ -z "$idx_raw" ]; then
      echo "ERROR: could not parse stack index from $label branch entry: $branch" >&2
      exit 1
    fi
    local idx_num=$((10#$idx_raw))
    if [ "$idx_num" -ne "$expected" ]; then
      printf 'ERROR: non-contiguous stack numbering in %s doc list (expected %02d, got %02d at %s)\n' \
        "$label" "$expected" "$idx_num" "$branch" >&2
      exit 1
    fi
    expected=$((expected + 1))
  done <<< "$branches"

  if [ "$saw_any" -ne 1 ]; then
    echo "ERROR: no stack branches parsed from $label doc list" >&2
    exit 1
  fi
}

check_list_and_branch_alignment() {
  local label="$1"
  local numbered="$2"
  local expected_item=1
  local expected_branch=0
  local saw_any=0

  while IFS= read -r row; do
    [ -n "$row" ] || continue
    saw_any=1
    local item_raw="${row%% *}"
    local branch="${row#* }"
    local item_num=$((10#$item_raw))
    local idx_raw
    idx_raw="$(printf '%s' "$branch" | sed -nE 's#^stack/sp-discipline-([0-9]+)-.*#\1#p')"
    if [ -z "$idx_raw" ]; then
      echo "ERROR: could not parse stack index from $label branch entry: $branch" >&2
      exit 1
    fi
    local branch_num=$((10#$idx_raw))

    if [ "$item_num" -ne "$expected_item" ]; then
      printf 'ERROR: non-contiguous markdown list numbering in %s doc list (expected item %d, got %d at %s)\n' \
        "$label" "$expected_item" "$item_num" "$branch" >&2
      exit 1
    fi
    if [ "$branch_num" -ne "$expected_branch" ]; then
      printf 'ERROR: list/branch index mismatch in %s doc list (item %d expects branch %02d, got %02d at %s)\n' \
        "$label" "$item_num" "$expected_branch" "$branch_num" "$branch" >&2
      exit 1
    fi

    expected_item=$((expected_item + 1))
    expected_branch=$((expected_branch + 1))
  done <<< "$numbered"

  if [ "$saw_any" -ne 1 ]; then
    echo "ERROR: no numbered stack branches parsed from $label doc list" >&2
    exit 1
  fi
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
PLAN_NUMBERED="$(extract_numbered_branches "$PLAN_DOC")"
EVIDENCE_NUMBERED="$(extract_numbered_branches "$EVIDENCE_DOC")"

if [ -z "$PLAN_BRANCHES" ] || [ -z "$EVIDENCE_BRANCHES" ] || [ -z "$PLAN_NUMBERED" ] || [ -z "$EVIDENCE_NUMBERED" ]; then
  echo "ERROR: failed to parse stack branch lists from docs" >&2
  exit 1
fi

check_no_duplicates "plan" "$PLAN_BRANCHES"
check_no_duplicates "evidence" "$EVIDENCE_BRANCHES"
check_list_and_branch_alignment "plan" "$PLAN_NUMBERED"
check_list_and_branch_alignment "evidence" "$EVIDENCE_NUMBERED"
check_branch_sequence "plan" "$PLAN_BRANCHES"
check_branch_sequence "evidence" "$EVIDENCE_BRANCHES"

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
