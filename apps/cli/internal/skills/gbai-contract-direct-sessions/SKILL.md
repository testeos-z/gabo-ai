---
name: gbai-contract-direct-sessions
description: "Trigger: direct sessions contract, direct_sessions slot, talk to a coder, intervene in a task, session in a worktree, adapter coverage. Operations every gbai direct sessions adapter must cover."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Load when writing, checking or using the adapter of the `direct_sessions` slot (`slots.direct_sessions.adapter`, TRD 9.1). This contract states WHAT the owner or the assistant needs to intervene in one task through a live agent session in that task's worktree (R14, R18, US-005); the adapter states HOW with the team's own session tool (D2, D9).

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: the planned coverage check in `gbai generate` (T011) compares them; today only the skills test enforces them, for the bundled example.
- A session opens only inside an existing task worktree; it never creates, moves or removes worktrees and never touches another task's worktree.
- The brief names the task id, the worktree path and the branch, and repeats the coder's hard rules: never merge, never force-push, never post the reviewer marker.
- A session is not a second coder: while the task's coder is running, messages go to the open session only when the owner asked for it.
- Every session the adapter opens is closed or reported as still open; a session is a resource and goes into the wave ledger.

## Operations

### open session

Start an agent session in one task worktree with a brief. Return a session handle.

### send message

Send one message to an open session by its handle.

### read state

Read whether the session is running, waiting for input or finished, and its latest output.

### close session

Close the session by its handle and confirm it is gone.

## Output Contract

An adapter answers every operation with the exact command, the handle it returns or takes, and how to confirm the result. An agent using it returns the handle, the worktree and the final state (`open` or `closed`).

Example adapter: `examples/sessions-orca.md` (Orca CLI terminals).
