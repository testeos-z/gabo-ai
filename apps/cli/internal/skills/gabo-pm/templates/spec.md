# spec template

Scaffold for `docs/product/intents/<NNNN-slug>/spec.md`: the requirements and the design of this one change, answering its `intent.md`. `gabo-pm` writes the requirements and validates the whole spec; `gabo-frontend` contributes section 3 and `gabo-db-master` contributes section 4 when the change needs them. A spec covers one intent only; anything wider is a new intent.

Sections 3 and 4 are optional: when the change has no user-facing flow or no data change, write "not applicable" with the reason. Design norms stay in the repository `DESIGN.md`; this spec cites their rule ids and never repeats a rule. Answer every fixed question before filling its section; an answer you do not have goes to section 6, never an invented decision. Keep the table columns exactly as written: the plan's `spec refs` cite the requirement and decision ids.

Acceptance is a human gate. An agent never fills section 7 and never marks the spec accepted; it presents the spec and waits for a human to accept it.

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | Which requirements, each with acceptance criteria a reviewer can check, solve the intent's problem? | 1. Requirements |
| Q2 | How is each requirement verified (test layer or check)? | 1. Requirements: verification |
| Q3 | Which design decisions does the change need, which options were considered and why the choice? | 2. Design |
| Q4 | Does the change touch a user-facing surface; if so, which surfaces and journeys change? | 3. UX and app flow (optional) |
| Q5 | Does the change alter stored data; if so, which ordered migrations or scripts, and how is each rolled back? | 4. DB migration steps (optional) |
| Q6 | Does the change conflict with a PRD rule, a TRD convention, `DESIGN.md` or another policy, and how is each conflict resolved? | 5. Policy conflicts |
| Q7 | Which of the intent's open questions are answered here, and which are carried, to where? | 6. Open questions |
| Q8 | Who accepted the spec, and when? | 7. Acceptance |

## Template

```markdown
# Spec <NNNN> <Title>

| field | value |
|---|---|
| intent | [<NNNN-slug>](intent.md) |
| status | <Draft / Accepted / Superseded> |
| owner | gabo-pm |
| contributors | <gabo-frontend / gabo-db-master / —> |
| date | <YYYY-MM-DD> |

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | <When <trigger>, the <system> shall <response>.> | <Given <context> When <action> Then <observable outcome>> | <test layer or check> |

## 2. Design

<How the change works: affected components, interfaces and data flow. Follow the TRD conventions; a departure goes to section 5.>

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | <decision> | <option A / option B> | <choice and reason> | <what follows> |

## 3. UX and app flow (optional)

<"Not applicable: <reason>" when the change has no user-facing surface. Tokens and color follow [`DESIGN.md`](../../DESIGN.md) sections 2-3 (tokens, color), accessibility follows its section 9 (accessibility) and message wording its section 10 (wording); cite their rule ids here, never a tokens table or generic accessibility or wording rules.>

| surface | entry points | primary action | empty and error states |
|---|---|---|---|
| <screen or surface> | <how the user gets here> | <main thing the user does> | <what the user sees when there is nothing, or when it fails> |

- **Journey.** Trigger, actors, steps, decision points, outcome, failure and escalation. Product-specific keyboard paths stay in the journey.
- **Flow.** <One Mermaid flowchart of the changed journey.>

Messages: one row per concrete case of this change; wording follows `DESIGN.md` section 10.

| case | what happened | why | what to do next |
|---|---|---|---|
| <concrete case, e.g. "save fails offline"> | <what the user is told happened> | <why it happened> | <the action the user can take> |

## 4. DB migration steps (optional)

<"Not applicable: <reason>" when the change alters no stored data. Forward-only: never edit an applied migration. Each row is also appended to `docs/product/MIGRATIONS.md`.>

| order | file | purpose | rollback |
|---|---|---|---|
| 1 | <path of the migration or script> | <why> | <how to revert, or "none: forward-only"> |

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| <PRD rule / TRD convention / DESIGN.md / other, or "none"> | <what clashes> | <resolution, or "plan change proposed"> |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | <question from the intent or raised here> | <answered: <answer> / carried to <intent, plan or owner>> |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
| <name> | <YYYY-MM-DD> | <conditions or "—"> |
```
