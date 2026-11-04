---
name: gabo-doc-guide
description: "Trigger: which doc to read, where is it documented, doc map, find the right skill, product docs, setup prompt, gabo documents. Map of the gabo documents: which one answers which question."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "2.0"
---
## Activation Contract

Load when an agent or the owner needs to know which gabo document answers a question. This guide is only a map: it adds no rule, and every rule stays in the document it points to. When a document and this map disagree, the document wins.

## Product docs

Each project keeps its product docs in `docs/product/`, written from the templates of the directiva skills. Global docs hold what every change shares; each change is an intent with its own folder. There is no global plan: the tracker is the global view of tasks.

| question | read | written by |
|---|---|---|
| What does the product offer, for whom, and what does the user get? | `docs/product/PRD.md` | `gabo-pm` |
| Which stack did the user choose, and which architecture, code and test conventions and global decisions (D#) apply? | `docs/product/TRD.md` | `gabo-dev-lead` |
| Which global styles, tokens and components apply (optional, only when the product warrants UX/UI)? | `docs/product/DESIGN.md` | `gabo-frontend` |
| Which migrations and data scripts were applied, in which order, and how does persistence work? | `docs/product/MIGRATIONS.md` | `gabo-db-master` |
| What problem does one change solve? | `docs/product/intents/<NNNN-slug>/intent.md` | `gabo-pm` |
| What exactly does the change require and how is it designed (requirements, UX and app flow, migration steps), and who accepted it? | `docs/product/intents/<NNNN-slug>/spec.md` | `gabo-pm` (UX section `gabo-frontend`, migration section `gabo-db-master`; acceptance by the owner) |
| Which tasks deliver the change, with their dependencies? | `docs/product/intents/<NNNN-slug>/plan.md` | `gabo-dev-lead` |
| What did the reviews of the change find? | review findings in `docs/product/intents/<NNNN-slug>/` | reviewers |
| How are plan changes approved? | `gabo-plan-reviewer` | — |

## Setup prompt

| question | read |
|---|---|
| How does a project start using gabo, in which order, and which skills does setup load? | The setup prompt: `gabo init` prints it on a fresh project; the gabo-ai repository keeps it in `docs/setup-prompt.md`. |
| How are the config files and slot adapters chosen and verified? | `gabo-setup` |

## Catalog skills

`gabo init` copies the skill catalog into `.gabo/skills/`. Each skill's frontmatter `description` starts with `Trigger:` and lists the questions it answers; for a skill not listed below, read that line.

| question | skill |
|---|---|
| Who talks to the owner, and how are questions resolved by mode? | `gabo-assistant` |
| How does a wave run, and how are worktrees, the ledger and cleanup handled? | `gabo-cycle` |
| What does a wave report hold, and how is it saved and summarized? | `gabo-wave-report` |
| How is a config key written, in which file, and what happens after the write? | `gabo-config-writer` |
| How does a coder work one task, open its PR and close the task? | `gabo-coder` |
| How does the review gate work, and what does a `VERDICT` line mean? | `gabo-reviewer-notes` |
| How are the PRD, an intent and its spec written? | `gabo-pm` |
| How are the TRD, an intent's plan and the task issues written? | `gabo-dev-lead` |
| How does a plan change travel, which approval scale applies, and how does the plan reviewer decide? | `gabo-plan-reviewer` |
| How are migrations and the persistence conventions written? | `gabo-db-master` |
| How are global styles, tokens and an intent's UX and app flow designed? | `gabo-frontend` |
| Which agents exist and what is each one's body template? | `gabo-agent-templates` |
| Which operations must an adapter for a slot cover? | `gabo-contract-<slot>` (`tasks`, `repository-host`, `memory`, `methodology`, `reviewer-placement`, `notifications`, `direct-sessions`, `agent-host`), each with an example adapter under `examples/` |
| How does the reviewer run on the owner's machine (local placement)? | `gabo-reviewer-placement-local` |
| How do agents run as their own session on the owner's machine? | `gabo-agent-host-local` |
| How do agents run on a Railway server driven through Orca? | `gabo-agent-host-railway-orca` |
| How does the owner open a direct session with a coder through Orca? | `gabo-direct-sessions-orca` |

## Output Contract

Answer with the document or skill to read and the question it answers; quote the rule from that document, never from this guide.
