# STORY template

Scaffold for one user story in `docs/product/stories/` (INVEST, Given-When-Then, D14). The template body mirrors `docs/product/stories/README.md`. Name the file with the rule in the gbai-pm skill: `US-0NN-<kebab>.md`.

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | Who is the user? | user |
| Q2 | What do they want? | Story: goal |
| Q3 | Why do they want it? | Story: benefit |
| Q4 | How do we know it is done? | Acceptance criteria |
| Q5 | What is out of scope? | Out of scope |
| Q6 | What are the dependencies? | depends on |
| Q7 | What is the priority (the highest among its requirements)? | priority |

## Template

```markdown
# US-0NN <Title>

| field | value |
|---|---|
| status | <Proposed | Approved ...> |
| user | <solo developer | tech lead> |
| priority | <must | should> |
| requirements | <R..> |
| depends on | <US-.. or —> |

## Story
As a <user>, I want <goal>, so that <benefit>.

## Acceptance criteria
- Given <context> When <action> Then <observable outcome>

## Out of scope
- <item from the PRD non-goals or Later>

## Notes
- App flow journey: to be linked when APP_FLOW exists.
```
