---
name: git-commit
description: Draft and execute an English Angular-style commit message for one already-staged atomic change. Always use this skill when the user asks to write, revise, preview, or execute a commit message. If changes still need staging or splitting, use git-commit-plan first.
---

# Git Commit Message Workflow

Describe one staged atomic change accurately, then commit it only after explicit approval.

## Preconditions

- The index must contain exactly one atomic change.
- Planning and staging belong to `git-commit-plan`.
- If nothing is staged, the staged files are mixed, or more than three files are staged without an approved exception, stop and use `git-commit-plan`.
- Do not inspect or describe unrelated unstaged changes.

## Required Workflow

1. Inspect context:
   - `git status --short`
   - `git branch --show-current`
   - `git log --oneline -10`
   - `GIT_PAGER=cat git --no-pager diff --staged --no-ext-diff --no-textconv --unified=5`
2. Confirm the staged diff represents one intent. Do not repair or reorganize the index in this skill.
3. Draft one English Angular-style commit message from the staged diff only.
4. Wait for explicit user approval.
5. Run the approved commit with direct `git commit -m ...` arguments only.

## Message Gate

Reject the draft and return to `git-commit-plan` when:

- The title needs `and`, `/`, multiple scopes, or multiple outcomes.
- A staged file can be independently reviewed, landed, or reverted.
- Dependency/build, test, docs/examples, formatting-only, and business changes are mixed.
- Generated files, lockfiles, snapshots, or fixtures are not mandatory results of the same change.

Do not hide mixed work behind `update`, `cleanup`, `misc`, `various`, or `apply changes`.

## Commit Format

```text
type(scope): subject

Context or motivation.

Main change.

Result or impact.

Optional footer.
```

Rules:

- Title: `type(scope): subject`.
- Subject: imperative, lowercase first word, no trailing period.
- Title <= 80 characters.
- Full message < 600 characters.
- Body: exactly three short natural paragraphs without labels such as `Problem:` or `Change:`.
- Footer only for breaking changes or special notes.
- Drafts, previews, and actual commit messages must be English only.

## Types

- `feat`: new user-facing capability
- `fix`: bug/regression fix
- `docs`: docs only
- `style`: formatting/lint-only
- `refactor`: internal structure, no behavior change
- `perf`: performance/resource improvement
- `test`: tests only
- `build`: dependencies/package/build config
- `ci`: CI/CD
- `chore`: maintenance with no narrower type
- `revert`: rollback

Use a narrow module, feature, or component scope. Avoid `app`, `misc`, and `core` unless truly accurate.

## Hard Rules

- Do not stage or unstage files in this skill.
- Do not commit without explicit user approval.
- Do not run `git push`.
- Do not create temporary commit message files.
- Do not include translations or localized alternatives in commit output.

## Output

```text
type(scope): subject

Context or motivation.

Main change.

Result or impact.
```

After approval, execute:

```sh
git commit -m 'type(scope): subject' -m 'Context or motivation.' -m 'Main change.' -m 'Result or impact.'
```
