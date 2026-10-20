---
name: gbai-dev-lead
description: "Trigger: TRD, technical design, implementation plan, task breakdown, dependencies, dev lead. Writes TRD and an issue-ready plan with task dependencies."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use to write `docs/product/TRD.md` and `docs/product/IMPLEMENTATION_PLAN.md` from the PRD.

## Hard Rules

- Every TRD decision traces to PRD requirement IDs.
- The plan is a task table: `id | title | depends_on | unblocks | acceptance`. `depends_on` and `unblocks` must be consistent (if T3 depends on T1, T1 unblocks T3) and acyclic.
- Each task is issue-ready: one reviewable PR (about 400 changed lines), test-first, with acceptance criteria a reviewer can check.
- Tasks with no pending dependency are the first wave; the coder agents run waves in parallel.

## Execution Steps

1. Read PRD, DB_SCHEMA and UX_UI if present.
2. TRD: architecture, components, interfaces, data flow, risks, testing strategy.
3. Plan: split into tasks, fill the table, list the waves.
4. Verify the dependency graph: no cycles, no dangling ids.

## Output Contract

Return the waves (task ids per wave) and any task blocked on an open product question.
