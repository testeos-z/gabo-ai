# US-002 Plan the product once

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | must |
| requirements | R2, R3, R20 |
| depends on | US-001 |

## Story

As a solo developer, I want directiva to produce the global plan once from my goal, so that every task exists in the tracker with its dependencies before work starts.

## Acceptance criteria

- Given I state a goal, When directiva plans, Then it produces `docs/product` and creates the tracker tasks with blocked-by / unblocks dependencies, and no task is created outside the dependency map.
- Given a planning document is being created, When directiva fills it, Then it uses the template from its skill (gabo-ai's own) and asks the template's fixed questions one at a time.
- Given a story has no requirement, a requirement has no decision, task or verification, a task has no story, or the dependency graph has a cycle, When the next planning step is attempted, Then the traceability check that directiva runs from its skill instruction blocks it.
- Given a task was modified after a wave, When the next wave starts, Then the cycle reads the task again with its changes.

## Out of scope

- Replacing the tracker or being one (non-goal).
- Replacing the human in product decisions or final validation (non-goal).
- Tracker providers other than GitHub issues (Later).

## Notes

- App flow journey: [J2 in APP_FLOW](../APP_FLOW-0.7.md).
- Approved (owner 2026-10-26): acceptance criteria aligned with TRD D18–D20 (the Go CLI only compiles the initial configuration; work is done by agents guided by skills).
