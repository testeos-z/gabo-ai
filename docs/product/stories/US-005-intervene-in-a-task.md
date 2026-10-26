# US-005 Intervene in a task

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | must |
| requirements | R14, R18 |
| depends on | US-003 |

## Story

As a solo developer, I want to intervene in a task through the assistant or a direct session, so that I can steer the work without redoing it or managing many terminals.

## Acceptance criteria

- Given I want to intervene in a task, When I talk to the assistant, Then it mediates with the working agent in one session by default.
- Given mediation drags on, When the assistant offers a direct session and I accept, Then it opens a named Orca session in the task's worktree, briefed with prior context, and I switch to it.
- Given I finished intervening by either path, When the assistant resumes, Then it continues from the traces (commits, task, memory), coordinates with the coder and closes the task without redoing work.
- Given a session starts, When terminals are opened, Then only the assistant's Orca terminal opens, and additional terminals open only when I request them.

## Out of scope

- Direct sessions with other terminal managers, for example herdr (Later).
- Being an IDE or a coding agent (non-goal).

## Notes

- App flow journey: [J5 in APP_FLOW](../APP_FLOW.md).
