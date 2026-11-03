# US-009 Single point of contact

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | must |
| requirements | R1 |
| depends on | — |

## Story

As a solo developer, I want the assistant as my default point of contact while still being able to talk to any agent directly, so that I stop being the glue between agents.

## Acceptance criteria

- Given I open a new session, When it starts, Then the assistant is my point of contact.
- Given I want to talk to a specific directiva or development agent, When I address it directly, Then the assistant does not block it.
- Given I talk to the assistant, When my request concerns the flow, Then I do not need to relay it to other agents myself.

## Out of scope

- Making the assistant part of directiva (roles).
- Being an IDE or a coding agent (non-goal).

## Notes

- App flow journey: [J9 in APP_FLOW](../APP_FLOW-0.7.md).
