---
name: create-issue
description: Draft and create one GitHub issue from a bug, feature, documentation request, or experiment using the repository's issue templates. Use when the user asks to open, create, file, or refine a GitHub issue.
---

# Create Issue

Create one focused GitHub issue. Draft first, then publish only after explicit user approval.

## Scope

- Handle one issue at a time.
- Support the repository templates for bug, feature, documentation, and experiment work.
- Keep implementation, branch, commit, and pull request work out of this workflow; use `split-issues` for multi-issue plans.

## Workflow

### 1. Gather context

Read the conversation and the repository's `.github/ISSUE_TEMPLATE/*.yml` files. Identify:

- The problem, question, or desired outcome.
- Affected users, assets, modules, platforms, or versions.
- Scope and explicit exclusions.
- Observable acceptance criteria and, when relevant, reproduction steps, evidence, dependencies, and risks.

Ask targeted questions when required facts are missing; do not fill gaps with guesses.

### 2. Select the issue type

Use the narrowest matching template:

- `01-bug-report.yml`: defect, regression, or unexpected behavior.
- `02-feature-request.yml`: new capability or behavior improvement.
- `03-document.yml`: documentation addition, correction, or rewrite.
- `04-experiment.yml`: bounded validation of a hypothesis or technical approach.
If the request contains multiple independent outcomes, recommend splitting it instead of hiding them in one issue.

### 3. Check related work

Before drafting, inspect the current repository and search open issues when GitHub access is available:

```bash
gh issue list --state open --search "<distinctive keywords>" --limit 20
```
Treat similar issues as possible duplicates or references. Do not edit, close, or comment on existing issues unless explicitly asked.

### 4. Draft the issue

Produce a concise title using the template prefix, such as `[Bug]:` or `[Feature]:`. Mirror the selected template's required fields as markdown headings and include optional fields only when they contain useful information.

The body must:

- Describe the problem or uncertainty before prescribing an implementation.
- Use project terminology, clear boundaries, and observable acceptance criteria.
- Separate facts, assumptions, and proposed approach.
- Avoid stale file paths, unnecessary code snippets, and invented metrics, versions, tests, issue numbers, or outcomes.

Default to the repository's language convention. These templates are English, so use English unless the user requests another language.

### 5. Review the draft

Show the user the title, selected template, complete body, and proposed labels, assignee, milestone, or related links, if any.

Check completeness, duplicate risk, scope, and acceptance criteria. Do not publish until the user explicitly approves the draft.

### 6. Publish after approval

Verify authentication and discover available labels before mutating GitHub:

```bash
gh auth status
gh label list --limit 200
```

Use only existing labels. Create the issue with a body file to avoid shell quoting errors:

```bash
gh issue create --title "[Type]: concise title" --body-file "<body-file>"
```

Add `--label`, `--assignee`, or `--milestone` only when requested and valid. Do not use `--web` unless requested, and do not claim success until the command returns the issue URL.

## Output

Before approval, return the draft marked as awaiting approval. After creation, return the issue number, title, URL, and requested metadata. On failure, preserve the draft and report the error without claiming publication.

## Safety Rules

- Never create an issue without explicit approval in the current conversation.
- Never modify a parent or related issue as a side effect or invent labels, assignees, milestones, links, or verification results.
- Do not create a blank issue when repository templates are available.
- Keep the issue focused on one deliverable or one bounded investigation.

## Examples

- “Open an issue for the asset importer crashing” -> bug template.
- “Propose WebP export support” -> feature template; “document asset naming rules” -> documentation template.
- “Break this roadmap into tickets” -> use `split-issues`, not this skill.
