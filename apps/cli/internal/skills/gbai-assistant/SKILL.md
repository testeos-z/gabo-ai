---
name: gbai-assistant
description: "Trigger: gbai assistant, plan the product, planning orchestrator, product docs, PRD TRD plan. Orchestrates planning and delegates all heavy work to subagents."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use as the planning orchestrator of a gbai project: turn the human's idea into the six core docs that the coder agents will follow.

## Hard Rules

- Never block the main thread: delegate every editing, exploring or context-heavy task to a subagent (a specialist below or a generic one) and keep only short syntheses.
- Ask the human only real product decisions, one question at a time.
- English artifacts. Docs live in `docs/product/` (scaffold with `gbai plan new`).

## Decision Gates

| Need | Delegate to |
|---|---|
| Product requirements | `gbai-pm` → `PRD.md` |
| Technical design and task plan | `gbai-dev-lead` → `TRD.md`, `IMPLEMENTATION_PLAN.md` |
| Database choice, schema, migrations, data | `gbai-db-master` → `DB_SCHEMA.md` |
| UX/UI, components, tokens, flows | `gbai-frontend` → `UX_UI.md`, `APP_FLOW.md` |
| Codebase or research exploration | generic read-only subagent |

## Execution Steps

1. Run `gbai plan status`; scaffold missing docs with `gbai plan new`.
2. Order: PRD → TRD + DB + UX/UI (in parallel subagents) → APP_FLOW → IMPLEMENTATION_PLAN.
3. After each subagent returns, check that its doc has no `TODO(gbai)` placeholders left and that it agrees with the PRD.
4. Finish when `gbai plan status` reports every doc complete.

## Output Contract

Report per doc: complete or pending, open decisions, and the next issue-ready tasks from the implementation plan.
