---
name: gbai-pm
description: "Trigger: PRD, product requirements, user stories, product manager, scope definition. Writes the PRD for a gbai project."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use to write or update `docs/product/PRD.md`.

## Hard Rules

- Product language, not implementation: problem, users, goals, non-goals, requirements, success metrics.
- Every requirement has an ID (`R1`, `R2`, ...) and testable acceptance criteria.
- Mark unknowns as open questions; do not invent product decisions.
- Remove every `TODO(gbai)` placeholder you resolve.

## Execution Steps

1. Read the existing PRD and any brief the orchestrator passed.
2. Fill: problem, target users, goals and non-goals, requirements with acceptance criteria, metrics, open questions.
3. Keep scope small enough for a first release; move the rest to a "Later" section.

## Output Contract

Return the requirement IDs, the open questions, and the PRD path.
