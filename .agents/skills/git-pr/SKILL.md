---
name: git-pr
description: Draft concise Git PR titles and descriptions from committed branch diff, verify branch-triggered CI locally, and submit PRs with GitHub CLI after approval. Keep metadata focused on actual changes, motivation, implementation, and verification.
---

# Git PR Writing

Draft PR metadata from committed branch diff. Do not invent context or verification.

## Required Workflow

1. Inspect context:
   - `git status --short`
   - `git branch --show-current`
   - `git log --oneline -20`
2. Find base branch:
   - Use user-provided base first.
   - Else use upstream PR base if tooling exposes it.
   - Else use `origin/HEAD`, `main`, or `master`.
   - Ask only if base ambiguity changes diff.
3. Inspect branch diff:
   - `git diff --stat <base>...HEAD`
   - `git diff --name-status <base>...HEAD`
   - `git diff --find-renames --find-copies <base>...HEAD`
4. Read relevant commits/files until behavior, implementation, and verification are clear.
5. Identify PR-triggered CI and run local equivalents where available.
6. Draft title/body.
7. Create/edit PR only after explicit approval.

## Hard Rules

- Branch diff is source of truth.
- Ignore unrelated unstaged/uncommitted work unless user asks for working-tree draft.
- Do not invent tests, screenshots, issue links, metrics, or outcomes.
- If verification was not run, write `Not run (reason).`
- Keep reviewer-focused. Skip implementation trivia.
- Do not create, edit, merge, close, or push PR without explicit approval.
- Use English headings by default unless user requests localization.

## Local CI Before PR

- Inspect changed files, workflow triggers, path filters, package scripts, and local tool config to identify CI jobs the branch will trigger.
- Derive and run the closest local equivalents before submitting. Prefer exact commands already defined by workflows or package manager scripts.
- Do not hardcode repo-specific CI command lists in this skill; CI changes should be discovered from the repository at PR time.
- Put exact commands and pass/fail results in PR `Testing`. If a local tool is unavailable, write `Not run (reason)`.

## Submit With GitHub CLI

After approval to create PR:

```bash
git push -u origin HEAD
gh pr create --base <base-branch> --head "$(git branch --show-current)" --title "<type>(<scope>): <subject>" --body-file <body-file>
```

- Use `--draft` for draft PRs.
- Use `--reviewer <handle>` when reviewer requested.
- Use `--web` only when user wants browser completion.

## Title

Format:

```text
<type>(<scope>): <subject>
```

Rules:

- Use narrowest type/scope matching branch diff.
- Imperative subject: `add`, `fix`, `remove`, `optimize`.
- Keep about <= 72 chars.
- Avoid vague words: `update`, `improve`, `misc`, `various`, `stuff`.

Types: `feat`, `fix`, `refactor`, `perf`, `docs`, `style`, `test`, `build`, `ci`, `chore`, `revert`.

## Body

Default structure:

```markdown
Title: <type>(<scope>): <subject>

## Feature Description
- ...

## Implementation Approach
- ...

## Testing
- ...
```

Use `## Change Description` instead of `## Feature Description` for fixes, refactors, docs, tests, build, CI, chores, or removals.

Section rules:

- Description: 1-4 bullets. State purpose, behavior, observable effect.
- Implementation: 1-3 bullets. Explain core approach, important boundaries, reused libs.
- Testing: list only checks actually run and result. If none: `Not run (reason).`

Optional sections only when useful:

- `## Why`
- `## Screenshots`
- `## Risks`
- `## Follow-ups`
- `## Related`

If user asks only title or only body, return only requested piece.
