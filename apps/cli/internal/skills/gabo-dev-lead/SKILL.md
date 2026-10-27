---
name: gabo-dev-lead
description: "Trigger: TRD, technical design, implementation plan, task breakdown, dependencies, task issues, dev lead. Writes TRD, the plan and the task issues with dependencies."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.0"
---
## Activation Contract

Load in the `gabo-dev-lead` subagent to write `docs/product/TRD.md` and `docs/product/IMPLEMENTATION_PLAN.md` and to turn plan tasks into tracker issues through the `tasks` slot adapter.

## Hard Rules

- TRD on arc42 with C4 views and MADR-style decisions (consequences, status); every decision traces to PRD requirement ids (D14).
- Plan tasks in `T###` notation: `id | title | depends_on | unblocks | stories | acceptance`, `[P]` for parallel. `depends_on` and `unblocks` must mirror each other and be acyclic.
- Each task is issue-ready: one reviewable PR (about 400 changed lines), test-first, acceptance a reviewer can check.
- Traceability check (D19) before returning: no requirement without decision, task or verification; no task without story; no dependency cycle, dangling id or `depends_on`/`unblocks` mismatch. A gap blocks the step.
- Issues: one per task, sub-issues under their milestone parent, blocked-by links matching `depends_on`, plan labels; every write verified by re-read (R24).
- Plan changes after the plan exists: own branch, issue and PR touching only `docs/product` and tasks (D7); never inside a code PR.

## Execution Steps

1. Read PRD, DB_SCHEMA, UX_UI and APP_FLOW if present.
2. TRD: context, building blocks, runtime view, interfaces, decisions, risks, testing strategy.
3. Plan: tasks, dependencies, waves (a wave = tasks whose dependencies are all done).
4. Run the traceability check; then create or update the issues.

## Output Contract

Return the waves (task ids per wave), issue numbers per task, unverified writes, and tasks blocked on an open product question.
