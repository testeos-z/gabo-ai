# IMPLEMENTATION_PLAN template

Scaffold for `docs/product/IMPLEMENTATION_PLAN.md`: tasks in `T###` notation with `[P]` for parallel tasks and story links (D14). Answer every fixed question before filling its section; an answer you do not have goes to "Open questions", never an invented decision. Keep the table columns exactly as written: the traceability check reads them (D19).

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | What must exist or be merged before the first task starts? | Prerequisites |
| Q2 | Which tasks, each one reviewable PR of about 400 changed lines, deliver the TRD? | Tasks |
| Q3 | Which stories, requirements and decisions does each task trace to? | Tasks: stories, requirements, decisions |
| Q4 | Which tasks does each task depend on and unblock, with no cycle? | Tasks: depends_on, unblocks |
| Q5 | What test-first acceptance can a reviewer check for each task, and what size is it? | Tasks: acceptance, size |
| Q6 | Which tasks share no files with the rest of their wave (`[P]`)? | Tasks: title |
| Q7 | Which waves follow from the dependencies, and what is each wave's goal? | Waves |
| Q8 | Does every story and every requirement have at least one task? | Coverage |
| Q9 | Which milestones group the waves, and what exit check closes each? | Roadmap and milestones |
| Q10 | What could derail the plan, and how is it mitigated? | Risks to the plan |
| Q11 | What is still undecided? | Open questions |

## Template

```markdown
# Implementation Plan

| field | value |
|---|---|
| status | Draft |
| owner | gabo-dev-lead |
| date | <YYYY-MM-DD> |
| linked docs | [PRD](PRD.md), [stories](stories/README.md), [TRD](TRD.md), [DB_SCHEMA](DB_SCHEMA.md), [APP_FLOW](APP_FLOW.md), [UX_UI](UX_UI.md) |

## Prerequisites

- <What must exist or be merged before T001; "none" when nothing.>

## Tasks

`[P]` marks a task that shares no files with the other tasks of its wave. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

| id | title | stories | requirements | decisions | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|---|---|
| T001 | <[P]> <title> | <US-0NN> | <R..> | <D..> | — | <T002> | Test first: <failing test or check that proves the outcome>; <acceptance a reviewer can check>. | <S / M / L> |

## Waves

A wave is the set of tasks whose dependencies are all done. Waves larger than `parallelism` run in batches.

| wave | tasks | goal |
|---|---|---|
| 1 | <T001> | <outcome of the wave> |

## Coverage

Every story and every requirement has at least one live task; every live task links at least one story and one requirement.

| story | tasks |
|---|---|
| US-001 | <T..> |

| req | tasks |
|---|---|
| R1 | <T..> |

## Roadmap and milestones

| milestone | goal | tasks | waves | exit check |
|---|---|---|---|---|
| M1 <name> | <goal> | <T..> | <1–n> | <observable check that closes the milestone> |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| <risk> | <tasks or waves affected> | <action> |

## Open questions

1. <question> (<owner or role that answers it>)
```
