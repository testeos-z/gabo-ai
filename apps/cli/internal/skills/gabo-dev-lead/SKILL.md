---
name: gabo-dev-lead
description: "Trigger: TRD, technical design, implementation plan, task breakdown, dependencies, task issues, dev lead. Writes TRD, the plan and the task issues with dependencies."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.1"
---
## Activation Contract

Load in the `gabo-dev-lead` subagent to write `docs/product/TRD.md` and `docs/product/IMPLEMENTATION_PLAN.md` and to turn plan tasks into tracker issues through the `tasks` slot adapter.

## Hard Rules

- TRD on arc42 with C4 views and MADR-style decisions (D14), scaffolded from `templates/TRD.md`: decisions table `| id | decision | options considered | choice | consequences | status | requirement ids |` and requirement coverage table `| req | decisions | sections | stories | verification → test layer |`; every decision traces to PRD requirement ids.
- Plan scaffolded from `templates/IMPLEMENTATION_PLAN.md`: tasks in `T###` notation with the table `| id | title | stories | requirements | decisions | depends_on | unblocks | acceptance | size |`, `[P]` for parallel. `depends_on` and `unblocks` must mirror each other and be acyclic.
- Templates are canonical (D19): answer each template's `## Fixed questions` before writing its sections, and never rename or reorder its table columns; the traceability check reads them.
- Each task is issue-ready: one reviewable PR (about 400 changed lines), test-first, acceptance a reviewer can check.
- Traceability check (D19) before returning: no requirement without decision, task or verification; no task without story; no dependency cycle, dangling id or `depends_on`/`unblocks` mismatch. A gap blocks the step.
- Issues: one per task, sub-issues under their milestone parent, blocked-by links matching `depends_on`, plan labels; every write verified by re-read (R24).
- Plan changes after the plan exists: own branch, issue and PR touching only `docs/product` and tasks (D7); never inside a code PR.

## Execution Steps

1. Read PRD, DB_SCHEMA, UX_UI and APP_FLOW if present; when TRD or the plan is missing, start it from its template.
2. TRD: answer the fixed questions, then fill quality goals, constraints, context, solution strategy, building blocks and interfaces, runtime and deployment views, crosscutting concepts, decisions and requirement coverage, quality requirements and testing strategy, risks and technical debt, glossary and open questions.
3. Plan: answer the fixed questions, then fill prerequisites, tasks, waves (a wave = tasks whose dependencies are all done), coverage, milestones and risks.
4. Run the traceability check; then create or update the issues.

## Output Contract

Return the waves (task ids per wave), issue numbers per task, unverified writes, and tasks blocked on an open product question.
