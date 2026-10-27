---
name: gabo-doc-guide
description: "Trigger: which doc to read, where is it documented, doc map, find the right skill, product docs, setup prompt, gbai documents. Map of the gbai documents: which one answers which question."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Load when an agent or the owner needs to know which gbai document answers a question. This guide is only a map: it adds no rule, and every rule stays in the document it points to. When a document and this map disagree, the document wins.

## Product docs

Each project keeps its product docs in `docs/product/`, written from the templates of the directiva skills.

| question | read | written by |
|---|---|---|
| What are we building, for whom, which requirements (R#), what is out of scope? | `docs/product/PRD.md` | `gabo-pm` |
| What does one user story ask for, with its acceptance criteria? | `docs/product/stories/` (`US-NNN-*.md`) | `gabo-pm` |
| How is it built: constraints, building blocks, decisions (D#), slot contracts, open technical questions? | `docs/product/TRD.md` | `gabo-dev-lead` |
| Which tasks exist, their dependencies, waves and coverage of stories and requirements? | `docs/product/IMPLEMENTATION_PLAN.md` | `gabo-dev-lead` |
| Which database or configuration schema, keys, layering and migrations? | `docs/product/DB_SCHEMA.md` | `gabo-db-master` |
| How does it look and read: principles, components, messages, accessibility? | `docs/product/UX_UI.md` | `gabo-frontend` |
| Which design tokens are the source of truth? | `DESIGN.md` (repository root) | `gabo-frontend` |
| How does a user move through the product, and how are plan changes approved? | `docs/product/APP_FLOW.md` | `gabo-frontend` |

## Setup prompt

| question | read |
|---|---|
| How does a project start using gbai, in which order, and which skills does setup load? | The setup prompt: `gabo init` prints it on a fresh project; the gabo-ai repository keeps it in `docs/setup-prompt.md`. |
| How are the config files and slot adapters chosen and verified? | `gabo-setup` |

## Catalog skills

`gabo init` copies the skill catalog into `.gbai/skills/`. Each skill's frontmatter `description` starts with `Trigger:` and lists the questions it answers; for a skill not listed below, read that line.

| question | skill |
|---|---|
| Who talks to the owner, and how are questions resolved by mode? | `gabo-assistant` |
| How does a wave run, and how are worktrees, the ledger and cleanup handled? | `gabo-cycle` |
| How does a coder work one task, open its PR and close the task? | `gabo-coder` |
| How does the review gate work, and what does a `VERDICT` line mean? | `gabo-reviewer-notes` |
| How are product requirements and stories written? | `gabo-pm` |
| How are the TRD, the plan and the task issues written? | `gabo-dev-lead` |
| How is the database or configuration schema designed? | `gabo-db-master` |
| How are UX, UI, tokens and the app flow designed? | `gabo-frontend` |
| Which agents exist and what is each one's body template? | `gabo-agent-templates` |
| Which operations must an adapter for a slot cover? | `gabo-contract-<slot>` (`tasks`, `repository-host`, `memory`, `methodology`, `reviewer-placement`, `notifications`, `direct-sessions`, `agent-host`), each with an example adapter under `examples/` |
| How does the reviewer run on the owner's machine (local placement)? | `gabo-reviewer-placement-local` |
| How do agents run as their own session on the owner's machine? | `gabo-agent-host-local` |
| How do agents run on a Railway server driven through Orca? | `gabo-agent-host-railway-orca` |
| How does the owner open a direct session with a coder through Orca? | `gabo-direct-sessions-orca` |

## Output Contract

Answer with the document or skill to read and the question it answers; quote the rule from that document, never from this guide.
