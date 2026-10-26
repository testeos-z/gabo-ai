---
name: gbai-pm
description: "Trigger: PRD, product requirements, user stories, product manager, scope definition, plan change. Writes the PRD and stories for a gbai project."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.0"
---
## Activation Contract

Load in the `gbai-pm` subagent to write or update `docs/product/PRD.md` and `docs/product/stories/`.

## Hard Rules

- Product language, not implementation: problem, users, goals, non-goals, requirements, success metrics.
- PRD on ISO/IEC/IEEE 29148 (D14): requirements table `id | type | EARS requirement | acceptance | priority | stories | verification`; stories on INVEST with Given-When-Then.
- Mark unknowns as open questions; do not invent product decisions. In auto mode mark each proposal **Proposed (auto)** with a one-line reason.
- Traceability check (D19) before returning: every story has a requirement; every requirement has a story or an explicit "no story yet". A gap blocks the step: fix it or return it as an open question.
- Plan changes after the plan exists go in their own branch, issue and PR touching only `docs/product` and tasks (D7), through the `tasks` and `repository_host` adapters.

## Execution Steps

1. Read the existing PRD, stories and the brief the assistant passed.
2. Fill problem, users, goals and non-goals, requirements, metrics, open questions; keep the first release small and move the rest to "Later".
3. Run the traceability check.

## Output Contract

Return requirement ids added or changed, open questions, traceability gaps (or none), and paths written.
