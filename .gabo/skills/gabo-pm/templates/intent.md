# intent template

Scaffold for `docs/product/intents/<NNNN-slug>/intent.md`, the first document of one change. The folder holds `intent.md`, `spec.md`, `plan.md` and the review findings of that change. `<NNNN>` is the next free four-digit number under `docs/product/intents/` and `<slug>` is the title in lower case with every run of non-alphanumeric characters replaced by one `-`. Never reuse or renumber an intent id.

An intent states the problem only. It holds no solution: designs, technologies, tables and screens belong in `spec.md`. Answer every fixed question before filling its section; an answer you do not have goes to "Open questions", never an invented decision. Keep the table columns exactly as written.

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | Which product card does this change belong to? | field: product card |
| Q2 | What problem does the user or the team have today? | Problem |
| Q3 | Who is affected, and how often? | Who is affected |
| Q4 | What observable outcome tells us the problem is solved? | Desired outcome |
| Q5 | Which limits apply (time, budget, policy, compatibility, PRD or TRD conventions)? | Constraints |
| Q6 | What is still unknown before the spec can be written? | Open questions |
| Q7 | Which documents, cards or other intents does this change relate to? | Links |

## Template

```markdown
# Intent <NNNN> <Title>

| field | value |
|---|---|
| intent id | <NNNN-slug> |
| status | <Draft / Ready for spec / Specified / Done / Dropped> |
| owner | gabo-pm |
| product card | <tracker card reference> |
| date | <YYYY-MM-DD> |

## Problem

<What is wrong or missing today, in the user's words. No solution.>

## Who is affected

<User or role, the situation they are in and how often it happens.>

## Desired outcome

- <Observable outcome that tells the problem is solved.>

## Constraints

- <Limit the change must respect, with its source (PRD, TRD convention, policy, deadline).>

## Open questions

| # | question | status |
|---|---|---|
| 1 | <question> | <open / answered: <answer> / carried to spec> |

## Links

| kind | reference |
|---|---|
| product card | <tracker card reference> |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| related intent | <NNNN-slug or —> |
```
