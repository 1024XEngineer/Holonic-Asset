---
name: split-issues
description: Decompose a plan, specification, PRD, roadmap, or oversized issue into ordered, independently implementable issue drafts using vertical slices and explicit dependencies. Use when the user asks to break work down, split a feature into issues, or prepare multiple implementation tickets.
---

# Split Issues

Turn one source of work into a small set of focused, independently reviewable issue drafts. This skill designs the breakdown; `create-issue` validates and publishes each approved draft.

## Scope

- Split plans, specs, PRDs, roadmaps, or oversized issues into multiple issue drafts.
- Use `create-issue` when the user needs only one issue.
- Do not implement code, create branches, publish issues, or modify source issues.

## Workflow

### 1. Gather the source

Read all supplied documents, conversation context, referenced issue bodies, and relevant comments. Identify the goal, users, scope, exclusions, acceptance criteria, constraints, and unresolved decisions.

If required context is missing or conflicting, ask targeted questions before drafting. Do not invent product or architectural decisions.

### 2. Inspect the project

Inspect only the relevant code, documentation, ADRs, issue templates, and open issues needed to understand current behavior and terminology. Search for duplicate or already-completed work when GitHub access is available.

### 3. Draft vertical slices

Prefer tracer-bullet slices that deliver a narrow end-to-end behavior across every required layer. Each slice should:

- Produce one observable outcome and fit one reviewable pull request.
- Be independently testable, demoable, or otherwise verifiable.
- Include all layers required for that outcome instead of creating separate frontend, backend, or test tickets.
- State explicit exclusions and avoid speculative follow-up work.
- Depend on as few other slices as possible.

Create a separate enabling issue only when shared infrastructure cannot reasonably ship inside the first vertical slice.

### 4. Classify and order

For every proposed issue, provide:

- `Draft ID`: stable local identifier such as `S1`, `S2`, or `S3`.
- `Title`: concise, outcome-focused, and aligned with project terminology.
- `Template`: bug, feature, documentation, or experiment.
- `Mode`: `AFK` when an agent can complete it independently; `HITL` when a human decision or review is required.
- `Outcome`: the user-visible or verifiable result.
- `Acceptance criteria`: observable completion conditions.
- `Blocked by`: draft IDs or `None - can start immediately`.

Order blockers before dependents. Prefer AFK slices and isolate genuine HITL decisions into the smallest possible issue.

### 5. Review with the user

Present the complete numbered breakdown before handing anything to `create-issue`. Ask the user to confirm granularity, dependencies, scope, and HITL/AFK classification. Revise until explicitly approved.

If approval covers only selected slices, hand off only those slices and clearly leave the others as drafts.

### 6. Prepare the handoff

For every approved slice, prepare a self-contained input package for `create-issue`. Preserve the selected repository template and use this body structure:

```markdown
## Parent
<source issue reference; omit when there is no parent>

## Outcome
<one complete end-to-end result>

## Scope and exclusions
<included and excluded behavior>

## Acceptance criteria
- [ ] Observable criterion

## Blocked by
<draft IDs, or None - can start immediately>
```

Hand off blockers before dependents. When the user asks to publish, load `create-issue` for each approved slice; it owns duplicate checks, final template fields, labels, authentication, approval, and `gh issue create`. Replace draft IDs with real issue links as blockers are created.

## Output

Before approval, return the ordered draft map. After approval, return the handoff packages in dependency order and identify which package should be passed to `create-issue` first. Never claim that an issue was published from this skill alone.

## Safety Rules

- Do not run `gh issue create` under this skill.
- Do not split by technical layer unless that layer delivers an independently valuable outcome.
- Do not create placeholder, cleanup, testing-only, or documentation-only slices unless they are independently necessary and verifiable.
- Do not invent product decisions, dependencies, issue numbers, or parent relationships.

## Examples

- “Split this asset import feature into issues” -> draft vertical feature slices.
- “Turn this PRD into implementation tickets” -> review the breakdown, then prepare the `create-issue` handoff.
- “Open one bug for this crash” -> use `create-issue`, not this skill.
