---
name: git-commit-plan
description: Inspect, split, and optionally stage working-tree changes as a sequence of atomic commits. Always use this skill when the user asks to stage files, split or organize commits, prepare changes for committing, clean up commit boundaries before a pull request, or when the staged index contains mixed work. This skill plans what belongs in each commit; use git-commit afterward to write and execute each message.
---

# Git Commit Planning Workflow

Turn staged and unstaged changes into a complete, reviewable sequence of atomic commit candidates. This skill decides what belongs together; it does not write final commit messages or run `git commit`.

## Required Workflow

1. Inspect context:
   - `git status --short`
   - `git branch --show-current`
   - `git log --oneline -10`
   - `GIT_PAGER=cat git --no-pager diff --staged --no-ext-diff --no-textconv --unified=5`
   - `GIT_PAGER=cat git --no-pager diff --no-ext-diff --no-textconv --unified=5`
2. Classify every changed file by its single intent and change category.
3. Produce a Commit Plan covering every changed file before changing the index.
4. Validate every candidate with the Atomicity Gate and File Count Gate.
5. If the user asked only for a plan, stop after presenting the validated plan.
6. If staging was requested and the index already matches the first candidate, leave it unchanged.
7. If the index is empty, stage only the first candidate with explicit paths: `git add -- <path>...`.
8. If the index is mixed, show the exact correction and ask for approval before changing it. After approval, unstage the mixed index and stage only the first candidate with explicit paths.
9. Re-read the staged diff and confirm it exactly matches the first candidate.
10. Hand off to `git-commit` for message drafting and commit execution.

## Commit Plan Gate

Use this exact structure:

```text
Commit 1
Intent: one observable outcome
Files:
- path/to/file-a
- path/to/file-b
Reason: why these files are inseparable for this intent
```

Rules:

- Cover every changed file exactly once or mark it explicitly as excluded.
- Give each candidate one observable intent and one narrow responsibility.
- Explain why every listed file must land with the others.
- Order candidates so each commit is independently reviewable and leaves the branch coherent.
- Do not finalize commit message wording here; `git-commit` owns message composition.
- If an intent or reason needs `and`, `/`, multiple scopes, or multiple outcomes, split again.

## Atomicity Gate

A candidate is atomic only when every included file is required for the same single intent.

- Could file A land without file B? If yes -> split.
- Could file A be reverted while keeping file B? If yes -> split.
- Are dependency/build changes mixed with business code? Split.
- Are tests mixed with business code? Split.
- Are docs/examples mixed with code? Split.
- Are formatting/lint-only changes mixed with behavior? Split.
- Keep generated files, lockfiles, snapshots, or fixtures only when they are mandatory results of the same change; otherwise split.

Default to smaller candidates. One or two files is ideal when possible.

## File Count Gate

- Reject a candidate containing more than three files by default.
- For an exception, explain why every file is inseparable from the same single intent.
- If any file can be independently reviewed, landed, or reverted, split it out.
- Do not waive this gate silently. A candidate over three files requires explicit user approval before staging.

## Index Safety

- Never run `git add .`, `git add -A`, or another command that stages the whole working tree.
- Never change the index before presenting a validated Commit Plan.
- Never unstage user-staged work without explicit approval.
- Stage only explicit paths belonging to one approved candidate.
- If one file contains changes for multiple candidates, stop and identify the hunks that must be staged separately.
- Do not run `git commit` or `git push`.

## Output

Return the validated Commit Plan. When staging was requested, also report the exact candidate staged and direct the next step to `git-commit`.
