# User stories

User stories for the gbai framework. Each story lives in its own file, `US-0NN-<kebab-title>.md`, and is indexed in the [PRD](../PRD.md) section 7.

## Rules

- INVEST applies: every story is Independent, Negotiable, Valuable, Estimable, Small and Testable.
- Acceptance criteria use Given-When-Then, are written from the user's point of view, and stay consistent with the linked requirements' acceptance criteria; a story adds no product behavior.
- Every story links at least one requirement; the dependency graph between stories stays acyclic.

## Fixed questions

The template answers these seven questions:

1. **Who** is the user? (`user`)
2. **What** do they want? (Story: goal)
3. **Why** do they want it? (Story: benefit)
4. **How do we know it is done?** (Acceptance criteria)
5. **What is out of scope?** (Out of scope)
6. **What are the dependencies?** (`depends on`)
7. **What is the priority?** (`priority`, the highest priority among its requirements)

## Template

Approved (owner 2026-10-26): the canonical STORY template and its naming rule live in the pm directiva skill (TRD D19); this copy mirrors it.

```
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
