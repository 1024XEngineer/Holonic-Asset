#!/usr/bin/env bash

set -euo pipefail

main_branch="main"
current_branch="$(git branch --show-current)"

if ! git show-ref --verify --quiet "refs/heads/${main_branch}"; then
  printf 'Local branch "%s" does not exist.\n' "$main_branch" >&2
  exit 1
fi

if [ "$current_branch" != "$main_branch" ]; then
  printf 'Switch to "%s" before deleting merged branches (currently on "%s").\n' "$main_branch" "$current_branch" >&2
  exit 1
fi

deleted=0

while IFS= read -r branch; do
  [ "$branch" = "$main_branch" ] && continue

  if git merge-base --is-ancestor "$branch" "$main_branch"; then
    git branch -d "$branch"
    deleted=$((deleted + 1))
  fi
done < <(git for-each-ref --format='%(refname:lstrip=2)' refs/heads)

printf 'Deleted %d branch(es) merged into "%s".\n' "$deleted" "$main_branch"
