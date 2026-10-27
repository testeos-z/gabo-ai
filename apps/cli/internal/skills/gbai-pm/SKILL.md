---
name: gbai-pm
description: "Trigger: PRD, product requirements, user stories, product manager, scope definition, plan change. Writes the PRD and stories for a gbai project."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.1"
---
## Activation Contract

Load in the `gbai-pm` subagent to write or update `docs/product/PRD.md` and `docs/product/stories/`.

## Hard Rules

- Product language, not implementation: problem, users, goals, non-goals, requirements, success metrics.
- PRD on ISO/IEC/IEEE 29148 (D14), scaffolded from `templates/PRD.md`: requirements table `| id | type | requirement | acceptance criteria | priority | stories | verification |` with EARS wording; stories on INVEST with Given-When-Then, scaffolded from `templates/STORY.md`.
- Templates are canonical (D19): answer each template's `## Fixed questions` before writing its sections, and never rename or reorder its table columns; the traceability check reads them.
- Story file naming: list `docs/product/stories/US-*.md`, take the highest `NNN` plus one (three digits, `US-001` when none exist) and name the file `US-0NN-<kebab>.md`, where `<kebab>` is the title in lower case with every run of non-alphanumeric characters replaced by one `-` and no leading or trailing `-` (with `US-001` the highest, "Plan the product once" becomes `US-002-plan-the-product-once.md`). Never reuse or renumber an id; add the story's row to PRD section 7.
- Mark unknowns as open questions; do not invent product decisions. In auto mode mark each proposal **Proposed (auto)** with a one-line reason.
- Traceability check (D19) before returning: every story has a requirement; every requirement has a story or an explicit "no story yet". A gap blocks the step: fix it or return it as an open question.
- Plan changes after the plan exists go in their own branch, issue and PR touching only `docs/product` and tasks (D7), through the `tasks` and `repository_host` adapters.

## Execution Steps

1. Read the existing PRD, stories and the brief the assistant passed; when a document is missing, start it from its template.
2. Answer the template's fixed questions, then fill problem and evidence, users, goals and non-goals, appetite, assumptions and dependencies, requirements, quality requirements, constraints, metrics, open questions and glossary; keep the first release small and move the rest to "Later".
3. Run the traceability check.

## Output Contract

Return requirement ids added or changed, open questions, traceability gaps (or none), and paths written.
