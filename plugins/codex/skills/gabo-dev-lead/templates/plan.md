# plan template

Scaffold for `docs/product/intents/<NNNN-slug>/plan.md`: the ordered steps and the tasks that deliver one accepted `spec.md`. The intent is the parent of every task; there is no global plan, roadmap or global wave, and the tracker is the global view. Start the plan only after a human accepted the spec. Answer every fixed question before filling its section; an answer you do not have goes to "Open questions", never an invented decision. Keep the table columns exactly as written: the traceability check reads them.

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | In which order must the change be built? | Steps |
| Q2 | Which tasks, each one reviewable PR of about 400 changed lines, deliver the spec? | Tasks |
| Q3 | Which spec requirements and decisions does each task cover? | Tasks: spec refs |
| Q4 | Which tasks does each task depend on and unblock, in this or another intent, with no cycle? | Tasks: depends_on, unblocks |
| Q5 | What test-first acceptance can a reviewer check for each task, and what size is it? | Tasks: acceptance, size |
| Q6 | Does every spec requirement have at least one task? | Tasks: spec refs |
| Q7 | What could derail the plan, and how is it mitigated? | Risks to the plan |
| Q8 | What is still undecided? | Open questions |
| Q9 | Does the change need a `gabo check` update, and which task covers it? | Steps |

## Template

```markdown
# Plan <NNNN> <Title>

| field | value |
|---|---|
| intent | [<NNNN-slug>](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | <YYYY-MM-DD> |

## Steps

1. <Ordered step and the spec requirements it delivers.>
2. <The task that updates `gabo check`, or why the change needs no update (TRD 8.5).>

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | <title> | <S1, D1> | — | <T2> | Test first: <failing test or check that proves the outcome>; <acceptance a reviewer can check>. | <S / M / L> |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| <risk> | <tasks affected> | <action> |

## Open questions

1. <question> (<owner or role that answers it>)
```
