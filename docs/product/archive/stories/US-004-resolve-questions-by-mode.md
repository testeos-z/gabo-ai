# US-004 Resolve questions by mode

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | must |
| requirements | R5, R6, R17 |
| depends on | US-003 |

## Story

As a solo developer, I want questions resolved according to the mode I chose and at the interaction level I am comfortable with, so that I receive only the decisions that matter to me.

## Acceptance criteria

- Given manual mode, When a methodology or development question arises, Then it reaches me through the assistant.
- Given auto mode, When a question arises, Then the assistant decides, records the decision and its reason in persistent memory, and I can query those records later.
- Given any mode, When a coder raises a question, Then it reaches the assistant and is never closed without a recorded answer and who gave it.
- Given a wave closes, When the assistant asks how comfortable I felt with the interactions, Then my answer and my stored preferences (number of interactions, speaking style) apply in the next wave.

## Out of scope

- Replacing the human in product decisions or final validation (non-goal).
- Target values for interaction counts (Later; the first release measures without targets).

## Notes

- Approved (owner 2026-10-26): stored preferences have a stable layer in the global user file and a learning layer of observations in the memory slot (`user/preferences`); an observation repeated about 3 times is proposed for the stable layer (PRD 10.1 Personalization, TRD D22).
- App flow journey: [J4 in APP_FLOW](../APP_FLOW-0.7.md).
