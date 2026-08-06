#!/usr/bin/env bash

set -euo pipefail

main_branch="main"
current_branch="$(git branch --show-current)"

if ! git show-ref --verify --quiet "refs/heads/${main_branch}"; then
  printf 'Local branch "%s" does not exist.\n' "$main_branch" >&2
  exit 1
fi

if [ "$current_branch" != "$main_branch" ]; then
  printf 'Switch to "%s" before deleting other local branches (currently on "%s").\n' "$main_branch" "$current_branch" >&2
  exit 1
fi

while IFS= read -r branch; do
  [ "$branch" = "$main_branch" ] && continue
  git branch -D "$branch"
done < <(git for-each-ref --format='%(refname:lstrip=2)' refs/heads)

printf 'Deleted every local branch except "%s".\n' "$main_branch"
