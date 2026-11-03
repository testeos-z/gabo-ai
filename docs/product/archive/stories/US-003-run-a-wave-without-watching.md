# US-003 Run a wave without watching

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | must |
| requirements | R4, R5, R15 |
| depends on | US-001, US-002 |

## Story

As a solo developer, I want the assistant to start each wave and receive its report, so that work advances without me launching, watching or relaying anything.

## Acceptance criteria

- Given the plan has tasks whose dependencies are done, When a wave runs, Then the assistant starts it and the wave report reaches the assistant.
- Given a full wave runs, When it completes, Then I have performed no launch, relay or copy step.
- Given a coder raises a question during the wave, When the wave report is produced, Then the question appears in it and is not closed without a recorded answer and who gave it.
- Given an auto-mode run is in progress, When the run proceeds, Then I receive no prompts, only the final answer, and I can query the recorded decisions.

## Out of scope

- Being an IDE or a coding agent (non-goal).
- Providing its own TODO or dashboard tool (non-goal).

## Notes

- App flow journey: [J3 in APP_FLOW](../APP_FLOW-0.7.md).
