# US-011 Resume without losing work

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | should |
| requirements | R25 |
| depends on | US-003 |

## Story

As a solo developer, I want an interrupted wave to resume from where it stopped, so that finished tasks are not redone.

## Acceptance criteria

- Given a wave was interrupted, When it resumes, Then it continues from the tracker state and memory traces.
- Given some tasks were already DONE before the interruption, When the wave resumes, Then those tasks are not reworked.
- Given the wave resumes, When it completes, Then I did not need to reconstruct its state by hand.

## Out of scope

- Being an IDE or a coding agent (non-goal).
- Replacing the tracker or being one (non-goal).

## Notes

- App flow journey: [J11 in APP_FLOW](../APP_FLOW.md).
