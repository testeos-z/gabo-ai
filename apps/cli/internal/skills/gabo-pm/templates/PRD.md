# PRD template

Scaffold for `docs/product/PRD.md`, the conceptual product document: what the product offers and what the user gets. It holds no acceptance criteria and no task links; each change is an intent in `docs/product/intents/<NNNN-slug>/`, and its `spec.md` holds the requirements with acceptance criteria. Answer every fixed question before filling its section; an answer you do not have goes to section 12 as an open question, never an invented decision.

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | What is the product in one sentence? | 1. Product in one sentence |
| Q2 | Who has which problem today, and why does it matter now? | 2. Problem |
| Q3 | What evidence shows the problem is real (data, quotes, incidents)? | 2. Problem / Evidence |
| Q4 | Who are the users, their roles and their main journeys? | 3. Users |
| Q5 | What are the goals, the non-goals and what is deferred to later? | 4. Goals, non-goals and later |
| Q6 | How much time or effort is the first release worth (appetite)? | 5. Appetite |
| Q7 | What do we assume, and what do we depend on outside our control? | 6. Assumptions and dependencies |
| Q8 | What does the product offer, and what does the user get from each offer? | 7. What the product offers |
| Q9 | What are the quality expectations (security, reliability, cost, performance)? | 8. Quality requirements |
| Q10 | Which constraints are fixed (platforms, standards, budget, legal)? | 9. Constraints |
| Q11 | Which measurable outcomes tell us it worked? | 10. Success metrics |
| Q12 | What is still undecided? | 12. Open questions |
| Q13 | Which terms need a shared meaning? | 13. Glossary |

## Template

```markdown
# Product Requirements (PRD)

| field | value |
|---|---|
| status | Draft |
| owner | gabo-pm |
| version | 0.1 |
| date | <YYYY-MM-DD> |
| scope | <product or release> |
| linked docs | [TRD](TRD.md), intents `docs/product/intents/` |

## 1. Product in one sentence

<For whom, what it does, and the outcome it gives.>

## 2. Problem

<Who has which problem today, and why it matters now.>

### Evidence

- <Data point, user quote or incident, with its source.>

## 3. Users

### Roles

| role | needs |
|---|---|
| <role> | <main jobs to be done> |

### Journeys

1. <Journey: trigger, steps, outcome.>

## 4. Goals, non-goals and later

- **Goals:** <outcome the first release achieves>
- **Non-goals:** <what the product will not do>
- **Later:** see section 11.

## 5. Appetite

<Time or effort the first release is worth, and what scope bounds it.>

## 6. Assumptions and dependencies

- **<Assumption or dependency>.** <What we rely on and what happens if it fails.>

## 7. What the product offers

Conceptual only. Type: F functional, Q quality, C constraint. The ids are stable so an intent or a TRD decision can cite them; the acceptance criteria of each change live in its intent's `spec.md`.

| id | type | what the product offers | what the user gets | priority |
|---|---|---|---|---|
| R1 | F | <capability, in product language> | <benefit or outcome for the user> | must |

## 8. Quality requirements

<Which rows of section 7 are type Q, grouped by security, reliability, cost, performance, observability.>

## 9. Constraints

<Fixed platforms, standards, budget or legal limits; the C rows of section 7.>

## 10. Success metrics

| metric | how it is measured |
|---|---|
| <metric> | <source and method> |

## 11. Later

Out of the first release.

- <Deferred item.>

## 12. Open questions

| # | question | status |
|---|---|---|
| 1 | <question> | open |

## 13. Glossary

| term | meaning |
|---|---|
| <term> | <meaning> |
```
