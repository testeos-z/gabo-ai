# APP_FLOW template

Scaffold for `docs/product/APP_FLOW.md`: every surface or screen, the journeys of each story and one flow diagram. A product without a GUI lists its surfaces (conversation, CLI, tracker) instead of screens. Answer every fixed question before filling its section; an answer you do not have goes to "Open questions", never an invented decision. Keep the table columns exactly as written.

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | Which screens or surfaces does the user see, and how do they reach each one? | Surfaces |
| Q2 | What is the primary action on each surface, and what do its empty and error states show? | Surfaces |
| Q3 | Which journey does each user story follow, and what triggers it? | Journeys |
| Q4 | Who are the actors of each journey? | Journeys: actors |
| Q5 | Which steps and decision points does each journey have? | Journeys: steps, decision points |
| Q6 | What is the outcome, and what happens on failure or escalation? | Journeys: outcome, failure and escalation |
| Q7 | How do the journeys connect into one flow? | Flow |
| Q8 | What is still undecided? | Open questions |

## Template

```markdown
# App Flow

| field | value |
|---|---|
| status | Draft |
| owner | gabo-frontend |
| date | <YYYY-MM-DD> |
| linked docs | [PRD](PRD.md), [TRD](TRD.md), [UX_UI](UX_UI.md), [stories](stories/README.md) |

## Surfaces

| surface | entry points | primary action | empty and error states |
|---|---|---|---|
| <screen or surface> | <how the user gets here> | <main thing the user does> | <what the user sees when there is nothing, or when it fails> |

## Journeys

Every journey uses the same shape. One journey per story; link it from the story's notes.

### J1 <Journey> — [US-0NN](stories/US-0NN-<kebab>.md) (<R..>)

- **Trigger.** <What starts the journey.>
- **Actors.** <Who takes part.>
- **Steps.**
  1. <step on a surface>
- **Decision points.** <Where the flow branches and on what.>
- **Outcome.** <Observable end state.>
- **Failure and escalation.** <What the user sees and who is asked when a step fails.>

## Flow

<One Mermaid flowchart connecting the surfaces and journeys.>

## Open questions

1. <question> (<owner or role that answers it>)
```
