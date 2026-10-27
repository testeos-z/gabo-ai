# PRD template

Scaffold for `docs/product/PRD.md` (ISO/IEC/IEEE 29148, D14). Answer every fixed question before filling its section; an answer you do not have goes to section 13 as an open question, never an invented decision.

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
| Q8 | Which user stories group the requirements? | 7. User stories |
| Q9 | What must the product do, how is each requirement accepted and verified? | 8. Requirements |
| Q10 | What are the quality requirements (security, reliability, cost, performance)? | 9. Quality requirements |
| Q11 | Which constraints are fixed (platforms, standards, budget, legal)? | 10. Constraints |
| Q12 | Which measurable outcomes tell us it worked? | 11. Success metrics |
| Q13 | What is still undecided? | 13. Open questions |
| Q14 | Which terms need a shared meaning? | 14. Glossary |

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
| linked docs | [TRD](TRD.md), stories `docs/product/stories/` |

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
- **Later:** see section 12.

## 5. Appetite

<Time or effort the first release is worth, and what scope bounds it.>

## 6. Assumptions and dependencies

- **<Assumption or dependency>.** <What we rely on and what happens if it fails.>

## 7. User stories

Stories follow `templates/STORY.md` of the gabo-pm skill and live in `docs/product/stories/`.

| id | title | user | requirement ids | status | file |
|---|---|---|---|---|---|
| US-001 | <Title> | <user> | R1 | proposed | [US-001-<kebab>.md](stories/US-001-<kebab>.md) |

## 8. Requirements

Type: F functional, Q quality, C constraint. Requirements use EARS wording (When/While/If ... the system shall ...).

| id | type | requirement | acceptance criteria | priority | stories | verification |
|---|---|---|---|---|---|---|
| R1 | F | <EARS requirement> | <observable acceptance> | must | US-001 | <test, inspection, demo or analysis> |

## 9. Quality requirements

<Which rows of section 8 are type Q, grouped by security, reliability, cost, performance, observability.>

## 10. Constraints

<Fixed platforms, standards, budget or legal limits; the C rows of section 8.>

## 11. Success metrics

| metric | how it is measured |
|---|---|
| <metric> | <source and method> |

## 12. Later

Out of the first release.

- <Deferred item.>

## 13. Open questions

| # | question | status |
|---|---|---|
| 1 | <question> | open |

## 14. Glossary

| term | meaning |
|---|---|
| <term> | <meaning> |
```
