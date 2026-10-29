---
name: gabo-pm
description: "Trigger: PRD, product requirements, user stories, product manager, scope definition, plan change. Writes the PRD and stories for a gabo project."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "2.3"
---
## Activation Contract

Load in the `gabo-pm` subagent to write or update `docs/product/PRD.md` and `docs/product/stories/`.

## Hard Rules

- Product language, not implementation: problem, users, goals, non-goals, requirements, success metrics.
- PRD on ISO/IEC/IEEE 29148 (D14), scaffolded from `templates/PRD.md`: requirements table `| id | type | requirement | acceptance criteria | priority | stories | verification |` with EARS wording; stories on INVEST with Given-When-Then, scaffolded from `templates/STORY.md`.
- Templates are canonical (D19): answer each template's `## Fixed questions` before writing its sections, and never rename or reorder its table columns; the traceability check reads them.
- Fixed questions one at a time: ask the template's `## Fixed questions` in order, one per message, and wait for each answer before the next (manual mode). In auto mode answer each yourself, mark it **Proposed (auto)** with a one-line reason and present them together for one-pass approval.
- Story file naming: list `docs/product/stories/US-*.md`, take the highest `NNN` plus one (three digits, `US-001` when none exist) and name the file `US-0NN-<kebab>.md`, where `<kebab>` is the title in lower case with every run of non-alphanumeric characters replaced by one `-` and no leading or trailing `-` (with `US-001` the highest, "Plan the product once" becomes `US-002-plan-the-product-once.md`). Never reuse or renumber an id; add the story's row to PRD section 7.
- Mark unknowns as open questions; do not invent product decisions. In auto mode mark each proposal **Proposed (auto)** with a one-line reason.
- Every story has a requirement and every requirement a story or an explicit "no story yet"; run the traceability check below before each next planning step.
- Plan changes after the plan exists go in their own branch, issue and PR touching only `docs/product` and tasks (D7), through the `tasks` and `repository_host` adapters.

## Plan-Change Flow

A critical question (it changes scope, the plan or other tasks) follows the `## Plan-Change Flow` of `gabo-plan-reviewer`: one issue and one PR per proposal, the approval scale against `plan_change.*`, and the plan reviewer only when the owner is absent.

## Traceability check

Run it before each next planning step: before moving to the next document, before creating or updating tasks, and before returning (D19). Read the stable template tables: PRD stories and requirements, TRD decisions and requirement coverage, plan tasks and coverage. Rows marked removed (a title starting `(removed <date>)`, or a status withdrawn or superseded) are ignored on both sides of every link. A gap is:

- a story without requirement;
- a requirement without decision, task or verification;
- a task without story;
- a dependency cycle, a dangling id (a story, requirement, decision or task id no live row defines), or a `depends_on` / `unblocks` mismatch.

Any gap blocks the next planning step: fix it in your own documents or return it as an open question naming the ids; never skip it. A document not written yet is reported as not checked, not as passing.

## Execution Steps

1. Read the existing PRD, stories and the brief the assistant passed; when a document is missing, start it from its template.
2. Answer the template's fixed questions, then fill problem and evidence, users, goals and non-goals, appetite, assumptions and dependencies, requirements, quality requirements, constraints, metrics, open questions and glossary; keep the first release small and move the rest to "Later".
3. Run the traceability check before the next document or step and before returning.

## Output Contract

Return requirement ids added or changed, open questions, traceability gaps (or none), and paths written.
