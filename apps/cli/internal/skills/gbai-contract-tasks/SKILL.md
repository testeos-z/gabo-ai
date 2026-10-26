---
name: gbai-contract-tasks
description: "Trigger: tasks contract, tasks slot, tracker adapter, adapter coverage, write a tasks adapter. Operations every gbai tasks adapter must cover."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Load when writing, checking or using the adapter of the `tasks` slot (`slots.tasks.adapter`, TRD 9.1). This contract states WHAT agents must be able to do with the team's tracker; the adapter states HOW with the user's own tool (D2, D4). gbai ships no tracker code.

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: the coverage check in `gbai generate` compares them.
- Every write is verified by re-reading it (R24). A write that does not show up after one retry is reported as an unverified write, never as a success.
- A task is identified by a stable id the tracker returns (for example an issue number); plan task ids (`T###`) appear in the title.
- Never close, relabel or reassign a task the agent was not asked to work on.

## Operations

### create

Create one task with title, body, labels and its dependencies: the tasks it is blocked by and the tasks it unblocks. Return the new task id.

### read unblocked

List the open tasks whose blockers are all closed and that carry no in-progress, blocked or needs-user status. Return ids in plan order.

### read one

Read one task: id, title, state, labels or status, assignee, body and comments.

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
