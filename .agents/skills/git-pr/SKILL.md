---
name: git-pr
description: Draft concise Git PR titles and descriptions from a committed branch diff, follow the repository's pull request template and required issue/checklist rules, verify branch-triggered CI locally, and submit with GitHub CLI only after approval. Use whenever the user asks to prepare, review, create, or update PR metadata.
---

# Git PR Writing

Draft PR metadata from committed branch diff. Do not invent context or verification.

## Required Workflow

1. Inspect context:
   - `git status --short`
   - `git branch --show-current`
   - `git log --oneline -20`
2. Resolve base: user input > upstream PR > `origin/HEAD` > `main`/`master`; ask only if ambiguity changes diff.
3. Inspect branch diff:
   - `git diff --stat <base>...HEAD`
   - `git diff --name-status <base>...HEAD`
   - `git diff --find-renames --find-copies <base>...HEAD`
4. Read `.github/pull_request_template.md`, `.github/PULL_REQUEST_TEMPLATE/`, `CONTRIBUTING.md`, and repository-local instructions. Active template is source of truth.
5. Read relevant commits/files until behavior, implementation, and verification are clear.
6. Identify PR-triggered CI and run local equivalents where available.
7. Draft title/body against the repository rules.
8. Create/edit PR only after explicit approval.

## Hard Rules

- Branch diff is source of truth.
- Ignore unrelated unstaged/uncommitted work unless user asks for working-tree draft.
- Do not invent tests, screenshots, issue links, metrics, or outcomes.
- If verification was not run, write `Not run (reason).`
- Do not omit or rename a required repository-template section. Preserve its order and checklist items.
- Strip prompt annotations such as `(Required)` and `(Optional)` from section headings in the final PR body.
- For a required related-issue field, use only a user-provided or evidence-backed `Closes #123`, `Part of #123`, or `None`.
- Mark checklist items `[x]` only when the diff or recorded verification supports them; otherwise leave them unchecked and explain why.
- Keep reviewer-focused. Skip implementation trivia.
- Do not create, edit, merge, close, or push PR without explicit approval.
- Use English headings by default unless user requests localization.

## Local CI Before PR

- Identify triggered CI from changed files, workflow filters, scripts, and local config.
- Run closest available equivalents. Record exact commands/results in `Testing`; otherwise `Not run (reason)`.

## Submit With GitHub CLI

After approval to create PR:

```bash
git push -u origin HEAD
gh pr create --base <base-branch> --head "$(git branch --show-current)" --title "<type>(<scope>): <subject>" --body-file <body-file>
```

Add `--draft`, `--reviewer <handle>`, or `--web` only when requested.

## Title

```text
<type>(<scope>): <subject>
```

- Use narrowest type/scope matching branch diff.
- Imperative subject: `add`, `fix`, `remove`, `optimize`.
- Keep about <= 72 chars.
- Avoid vague words: `update`, `improve`, `misc`, `various`, `stuff`.

Types: `feat`, `fix`, `refactor`, `perf`, `docs`, `style`, `test`, `build`, `ci`, `chore`, `revert`.

## Repository Body Template

Current project template. Active repository template still wins.

```markdown
## Change Description (Required)

- <problem solved and observable change>

## Implementation Approach (Required)

- <important implementation details and boundaries>

## Related Issue (Required)

<Closes #123 | Part of #123 | None>

## Testing (Required)

- [x] `<exact command or check>` - <result>
- [ ] Not run (<reason>)

## Screenshots or Recordings (Optional)

<visual evidence, or Not included.>

## Risks and Follow-ups (Optional)

<known risks or deferred work, or None identified.>

## Checklist (Required)

- [x] The PR is focused and does not include unrelated changes.
- [x] Asset names, formats, dimensions, and metadata follow project conventions.
- [x] Licensing and attribution information is included or unchanged as appropriate.
- [x] Documentation or examples were updated when needed.
```

If no repository template exists, use the generic fallback below.

## Generic Body Fallback

```markdown
## Feature Description

- ...

## Implementation Approach

- ...

## Testing

- ...
```

Description: 1-4 bullets: purpose, behavior, observable effect. Implementation: 1-3 bullets: approach, boundaries, reused libs. Testing: only checks run/result; otherwise `Not run (reason)`.

Optional:

- `## Why`
- `## Screenshots`
- `## Risks`
- `## Follow-ups`
- `## Related`

If user asks only title or only body, return only requested piece.
