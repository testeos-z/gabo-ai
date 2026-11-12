---
name: gabo-pm
description: "Trigger: PRD, product requirements, intent, spec, product manager, scope definition, plan change. Writes the PRD, each intent and its spec for a gabo project."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "3.0"
---
## Activation Contract

Load in the `gabo-pm` subagent to write or update the conceptual `docs/product/PRD.md`, and for each change the intent folder `docs/product/intents/<NNNN-slug>/`: write its `intent.md`, then generate and validate its `spec.md`.

## Hard Rules

- Product language, not implementation: problem, users, goals, non-goals, what the product offers and what the user gets, success metrics.
- PRD scaffolded from `templates/PRD.md`, conceptual only: offers table `| id | type | what the product offers | what the user gets | priority |`; no acceptance criteria and no task links in the PRD.
- One change, one intent: each change gets a folder `docs/product/intents/<NNNN-slug>/` with `intent.md` scaffolded from `templates/intent.md` (the problem only) and `spec.md` scaffolded from `templates/spec.md` (the requirements with acceptance criteria and verification, and the design). The folder also holds the intent's `plan.md` (written by `gabo-dev-lead`) and its review findings.
- Intent folder naming: list `docs/product/intents/`, take the highest `NNNN` plus one (four digits, `0001` when none exist) and name the folder `<NNNN>-<slug>`, where `<slug>` is the title in lower case with every run of non-alphanumeric characters replaced by one `-` and no leading or trailing `-` (with `0001` the highest, "Plan the product once" becomes `0002-plan-the-product-once`). Never reuse or renumber an intent number.
- Spec ownership: you write the spec's requirements and validate the whole spec; `gabo-frontend` contributes section 3 (UX and app flow) and `gabo-db-master` section 4 (migration steps) when the change needs them. A spec covers one intent only; anything wider is a new intent.
- Acceptance is human: the owner fills the spec's section 7 Acceptance record, never an agent, in manual and auto mode alike. Present the validated spec and wait; no plan starts before it is accepted.
- Templates are canonical (D19): answer each template's `## Fixed questions` before writing its sections, and never rename or reorder its table columns; the traceability check reads them.
- Fixed questions one at a time: ask the template's `## Fixed questions` in order, one per message, and wait for each answer before the next (manual mode). In auto mode answer each yourself, mark it **Proposed (auto)** with a one-line reason and present them together for one-pass approval.
- Mark unknowns as open questions; do not invent product decisions. In auto mode mark each proposal **Proposed (auto)** with a one-line reason.
- Run the traceability check below before each next planning step.
- Plan changes after a spec is accepted go in their own branch, issue and PR touching only `docs/product` and tasks (D7), through the `tasks` and `repository_host` adapters.

## Plan-Change Flow

A critical question (it changes scope, the plan or other tasks) follows the `## Plan-Change Flow` of `gabo-plan-reviewer`: one issue and one PR per proposal, the approval scale against `plan_change.*`, and the plan reviewer only when the owner is absent.

## Traceability check

Run it before each next planning step: before moving to the next document, before creating or updating tasks, and before returning (D19). Read the intent folders `docs/product/intents/<NNNN-slug>/` (`intent.md`, the spec requirements and its acceptance record, the plan task table) and the tracker tasks through the `tasks` contract. Rows marked removed (a title starting `(removed <date>)`, or a status withdrawn or superseded) are ignored on both sides of every link. A gap is:

- an intent without spec;
- an accepted spec without plan;
- a plan task without a tracker task, or the reverse;
- a task whose parent is not exactly one intent;
- a requirement in the spec not covered by any plan task;
- a dependency cycle;
- a dangling id, including a cross-intent `<NNNN>/T<n>` reference;
- a `depends_on`/`unblocks` mismatch.

Any gap blocks the next planning step: fix it in your own documents or return it as an open question naming the ids; never skip it. A document not written yet is reported as not checked, not as passing.

## Execution Steps

1. Read the existing PRD, the intents under `docs/product/intents/` and the brief the assistant passed; when a document is missing, start it from its template.
2. PRD: answer its fixed questions, then fill problem and evidence, users, goals and non-goals, appetite, assumptions and dependencies, what the product offers, quality requirements, constraints, metrics, open questions and glossary; keep the first release small and move the rest to "Later".
3. Intent: number the folder, answer the intent's fixed questions and write `intent.md`.
4. Spec: answer the spec's fixed questions, write its requirements and design, ask `gabo-frontend` and `gabo-db-master` for sections 3 and 4 when the change needs them, validate the whole spec against the intent, the PRD, the TRD conventions and `DESIGN.md`, then present it for the owner's acceptance.
5. Run the traceability check before the next document or step and before returning.

## Output Contract

Return the intent folder and its number, requirement ids added or changed, whether the spec awaits the owner's acceptance, open questions, traceability gaps (or none), the plan-change proposals opened (the count, and the issue or PR and the scope of each) or `proposals: none`, and paths written.
