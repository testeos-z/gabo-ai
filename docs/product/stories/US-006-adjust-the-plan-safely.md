# US-006 Adjust the plan safely

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | must |
| requirements | R7, R19 |
| depends on | US-003 |

## Story

As a solo developer, I want the plan adjusted between or during waves without re-planning and without silent critical changes, so that the plan stays alive and under my control.

## Acceptance criteria

- Given a wave has finished or is in progress, When directiva adjusts tasks, Then the global plan is not recreated and existing tasks keep their identity.
- Given a development question needs research, When directiva is asked, Then it returns a researched answer.
- Given a critical question arises in any mode, When it is handled, Then it becomes a plan-change proposal with its own issue and PR that change only plan documents and tasks, and it is never resolved silently.
- Given a plan-change proposal exists, When it is debated, Then it is debated with me when present, otherwise with a plan-change reviewer that must agree or propose a different improvement, and the change stays within the configured scope.

## Out of scope

- Redoing the global plan as a routine adjustment (goal 4).
- Replacing the human in product decisions or final validation (non-goal).

## Notes

- App flow journey: [J6 in APP_FLOW](../APP_FLOW.md).
