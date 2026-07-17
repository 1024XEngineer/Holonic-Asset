---
name: git-branch
description: Generate concise Git branch names from issue context or task descriptions. Never include assistant branding such as codex.
---

# Git Branch Naming

Create one clear branch name for current task. Keep output short and executable.

## Hard Rules

- Never include `codex`, `openai`, `chatgpt`, `claude`, `gemini`, agent names, model names, or author prefixes unless user explicitly requests them.
- Default shape: `<type>/<short-description>`.
- With issue ID: `<type>/<issue-id>-<short-description>`.
- Use English, lowercase branch text, kebab-case words.
- Allowed chars: `a-z`, `0-9`, `/`, `-`. Preserve issue IDs only if tracker needs uppercase, e.g. `PROJ-123`.
- Description: 2-5 useful words, no filler, no trailing separators.
- Do not use spaces, underscores, camelCase, quotes, emoji, or punctuation.

## Types

- `feat`: new capability
- `fix`: bug/regression fix
- `refactor`: internal structure change, no behavior change
- `perf`: performance/resource improvement
- `docs`: documentation only
- `test`: tests only
- `build`: dependencies, package manager, build config
- `ci`: CI/CD workflow
- `chore`: maintenance that fits no narrower type
- `revert`: revert previous change

## Workflow

1. Read task/issue context.
2. Pick narrowest type.
3. Generate one best branch name.
4. Provide exact command: `git checkout -b <branch-name>`.
5. Create branch only after explicit user approval or direct user request.

## Examples

Good:

```text
feat/social-account-linking
fix/post-schedule-timezone
refactor/provider-auth-flow
build/update-playwright
test/post-validation-rules
```

Bad:

```text
codex/feat/social-account-linking
kuroda/feat/social-account-linking
feat/add social login
fix_login
Feature/AddLogin
```

## Output

For Chinese users, explain in Chinese. Branch name stays English.
