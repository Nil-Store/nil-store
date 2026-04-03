#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

STACK_DOC="${SP_STACK_DOC:-docs/planning/SP_DISCIPLINE_PR_STACK_2026-04.md}"
REMOTE="${SP_STACK_REMOTE:-origin}"
BASE_REF="${SP_STACK_BASE_REF:-refs/remotes/$REMOTE/main}"

if [ ! -f "$STACK_DOC" ]; then
  echo "ERROR: stack doc not found: $STACK_DOC" >&2
  exit 2
fi

if ! git remote get-url "$REMOTE" >/dev/null 2>&1; then
  echo "ERROR: git remote not found: $REMOTE" >&2
  exit 2
fi

extract_numbered_branches() {
  sed -n 's/^\([0-9][0-9]*\)\. `\(stack\/sp-discipline-[^`]*\)`/\1 \2/p' "$STACK_DOC"
}

check_list_and_branch_alignment() {
  local expected_item=1
  local expected_branch=0
  local saw_any=0
  local row

  while IFS= read -r row; do
    [ -n "$row" ] || continue
    saw_any=1
    local item_raw="${row%% *}"
    local branch="${row#* }"
    local item_num=$((10#$item_raw))
    local idx_raw
    idx_raw="$(printf '%s' "$branch" | sed -nE 's#^stack/sp-discipline-([0-9]+)-.*#\1#p')"
    if [ -z "$idx_raw" ]; then
      echo "ERROR: could not parse stack index from branch entry: $branch" >&2
      exit 1
    fi
    local branch_num=$((10#$idx_raw))

    if [ "$item_num" -ne "$expected_item" ]; then
      printf 'ERROR: non-contiguous markdown list numbering in %s (expected item %d, got %d at %s)\n' \
        "$STACK_DOC" "$expected_item" "$item_num" "$branch" >&2
      exit 1
    fi
    if [ "$branch_num" -ne "$expected_branch" ]; then
      printf 'ERROR: list/branch index mismatch in %s (item %d expects branch %02d, got %02d at %s)\n' \
        "$STACK_DOC" "$item_num" "$expected_branch" "$branch_num" "$branch" >&2
      exit 1
    fi

    expected_item=$((expected_item + 1))
    expected_branch=$((expected_branch + 1))
  done < <(extract_numbered_branches)

  if [ "$saw_any" -ne 1 ]; then
    echo "ERROR: expected numbered stack branch rows in $STACK_DOC" >&2
    exit 2
  fi
}

echo "==> Fetching remote refs from $REMOTE..."
git fetch "$REMOTE" --prune >/dev/null

check_list_and_branch_alignment

STACK_BRANCHES=()
while IFS= read -r branch; do
  [ -n "$branch" ] || continue
  STACK_BRANCHES+=("$branch")
done < <(
  sed -n 's/^[0-9][0-9]*\. `\(stack\/sp-discipline-[^`]*\)`/\1/p' "$STACK_DOC"
)

if [ "${#STACK_BRANCHES[@]}" -lt 2 ]; then
  echo "ERROR: expected at least 2 stack branches in $STACK_DOC" >&2
  exit 2
fi

DUP_BRANCHES="$(
  printf '%s\n' "${STACK_BRANCHES[@]}" | sort | uniq -d
)"
if [ -n "$DUP_BRANCHES" ]; then
  echo "ERROR: duplicate stack branches found in $STACK_DOC:" >&2
  echo "$DUP_BRANCHES" >&2
  exit 1
fi

HEAD_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
STACK_REFS=()

echo "==> Resolving stack branch refs..."
for branch in "${STACK_BRANCHES[@]}"; do
  remote_ref="refs/remotes/$REMOTE/$branch"
  local_ref="refs/heads/$branch"
  if git show-ref --verify --quiet "$remote_ref"; then
    STACK_REFS+=("$remote_ref")
    echo "    OK: $REMOTE/$branch"
    continue
  fi
  if [ "$branch" = "$HEAD_BRANCH" ] && git show-ref --verify --quiet "$local_ref"; then
    STACK_REFS+=("$local_ref")
    echo "    OK: $branch (local only, not pushed yet)"
    continue
  fi
  echo "ERROR: missing required stack branch ref: $REMOTE/$branch" >&2
  exit 1
done

echo "==> Validating ancestry order across stacked branches..."
prev_ref="${STACK_REFS[0]}"
for i in "${!STACK_REFS[@]}"; do
  if [ "$i" -eq 0 ]; then
    continue
  fi
  cur_ref="${STACK_REFS[$i]}"
  if ! git merge-base --is-ancestor "$prev_ref" "$cur_ref"; then
    echo "ERROR: stack order violation: $prev_ref is not an ancestor of $cur_ref" >&2
    exit 1
  fi
  echo "    OK: $prev_ref -> $cur_ref"
  prev_ref="$cur_ref"
done

if ! git show-ref --verify --quiet "$BASE_REF"; then
  echo "ERROR: base ref not found: $BASE_REF" >&2
  exit 2
fi

echo "==> Ensuring stack branches are not already merged into $BASE_REF..."
for i in "${!STACK_BRANCHES[@]}"; do
  branch="${STACK_BRANCHES[$i]}"
  cur_ref="${STACK_REFS[$i]}"
  if git merge-base --is-ancestor "$cur_ref" "$BASE_REF"; then
    echo "ERROR: stack branch appears merged into base ref: $branch ($cur_ref) -> $BASE_REF" >&2
    echo "       Merge is blocked unless a human explicitly approves with YES MERGE." >&2
    exit 1
  fi
  echo "    OK: not merged -> $branch"
done

echo "SP discipline stack integrity check passed."
