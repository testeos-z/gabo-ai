---
name: gabo-contract-tasks
description: "Trigger: tasks contract, tasks slot, tracker adapter, adapter coverage, write a tasks adapter. Operations every gabo tasks adapter must cover."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
---
## Activation Contract

Load when writing, checking or using the adapter of the `tasks` slot (`slots.tasks.adapter`, TRD 9.1). This contract states WHAT agents must be able to do with the team's tracker; the adapter states HOW with the user's own tool (D2, D4). gabo ships no tracker code.

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: `gabo check` compares them with the adapter named in `slots.<name>.adapter` and lists every operation it misses; a heading inside a fenced code block does not count.
- Every write is verified by re-reading it (R24). A write that does not show up after one retry is reported as an unverified write, never as a success.
- A task is identified by a stable id the tracker returns (for example an issue number); plan task ids (`T###`) appear in the title.
- One task has exactly one parent intent; `blocked-by` links may cross intents. An intent is itself a tracker item (a sub-issue or a checklist item under a product card) that links to its repo folder `docs/product/intents/<NNNN-slug>/`. A product card may hold several intents.
- Intent ids are `I` plus the folder's four digits (`I0001` for `0001-slug`) and appear in the intent item's title (assumption: the id format is a gabo convention, not a tracker feature).
- Never close, relabel or reassign a task the agent was not asked to work on.

## Operations

### create

Create one task with title, body, labels and its dependencies: the tasks it is blocked by and the tasks it unblocks. Optional input `parent`: the intent item the task belongs to; when given, attach the task under it and re-read the link. Return the new task id.

### read unblocked

List the open tasks whose blockers are all closed and that carry no in-progress, blocked or needs-user status. Optional input `intent`: return only the unblocked tasks whose parent is that intent (a wave reads one intent). A blocker in another intent still counts. Return ids in plan order.

### read one

Read one task: id, title, state, labels or status, assignee, body, comments and its parent intent (none when unset).

### claim

Mark a task in progress for one agent and wave, so no other coder takes it.

### log progress

Append one short progress note to the task.

### change status

Move a task to DONE (only after its PR merged) or BLOCKED (with the reason).

### link PR

Link the task and its pull request in both directions.

### verify by re-read

Read back the field a write changed and compare it with the intended value; report `verified` or `unverified`.

## Output Contract

An adapter answers every operation with the exact commands or tool calls, the identifiers they need, and how to read the result back. An agent using it returns the task ids touched and each write as `verified` or `unverified`.

Example adapter: `examples/tasks-gh.md` (GitHub Issues through `gh`).

## Tracker mapping

The parent is an input of `create`, an output of `read one` and a filter of `read unblocked`, not an operation of its own, so existing adapters keep their headings. How trackers carry it (only GitHub has an example adapter):

- GitHub Issues: product card issue → intent sub-issue → task sub-issues. A GitHub issue has at most one parent, which enforces one intent per task.
- Plane: parent and child work items (product card → intent work item → task work items), read with `childOf("<intent identifier>")`.
- Trello is flat: the intent is a checklist item on the product card holding `I####` and the folder link; each task is a card linked to the product card and carrying an `intent:I####` label, which the filter reads.
