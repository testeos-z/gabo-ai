---
name: gabo-contract-direct-sessions
description: "Trigger: direct sessions contract, direct_sessions slot, talk to a coder, intervene in a task, session in a worktree, agent session handle, adapter coverage. Operations every gabo direct sessions adapter must cover."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.1"
---
## Activation Contract

Load when writing, checking or using the adapter of the `direct_sessions` slot (`slots.direct_sessions.adapter`, TRD 9.1). This contract states WHAT the owner or the assistant needs to talk to an agent session that `agent_host` reports, on this machine or on a remote host, and to intervene in one task through a live agent session in that task's worktree (R14, R18, US-005, intent 0019 S1); the adapter states HOW with the team's own session tool (D2, D9).

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: `gabo check` compares them with the adapter named in `slots.<name>.adapter` and lists every operation it misses; a heading inside a fenced code block does not count.
- A session opens only inside an existing task worktree; it never creates, moves or removes worktrees and never touches another task's worktree.
- The brief names the task id, the worktree path and the branch, and repeats the coder's hard rules: never merge, never force-push, never post the reviewer marker.
- A session is not a second coder: while the task's coder is running, messages go to the open session only when the owner asked for it.
- Every session the adapter opens is closed or reported as still open; a session is a resource and goes into the wave ledger.
- Any remote use needs the owner's explicit authorization, given in this session, that names the host, the operation (`send message` or `read state`) and the credential or session to use. A missing authorization sends and reads nothing, and the adapter asks the caller for it. An authorization for host A never reaches host B. The adapter never discovers or reuses ambient credentials, SSH agents, ControlMaster sockets, Orca pairings or authenticated sessions.
- A direct session is live conversation only: a question, an answer or a decision that must persist goes to the tracker as an issue comment through the `tasks` adapter, and a decision is also saved through the `memory` slot before the session closes. Nothing the team needs is left only in a session.

## Handle

- A handle has two named fields, `host` and `handle`, never one joined string.
- `host` is the host name that `agent_host` reports for where the session runs; the literal `local` names the local host.
- `handle` is the session handle that host returned.
- A handle that names no known host is refused, and nothing is sent or read.
- The ledger records both fields of the handle.

## Operations

### open session

Start an agent session in one task worktree with a brief. Return a session handle.

### send message

Send one message to an open session by its handle: the handle of any session that `agent_host` reports (`start agent session` or `report where agents run`), on this machine or on a remote host, under the handle defined under `## Handle`.

### read state

Read whether the session is running, waiting for input or finished, and its latest output, for the handle of any session that `agent_host` reports (`start agent session` or `report where agents run`), on this machine or on a remote host, under the handle defined under `## Handle`. An adapter that cannot return the reply says so and names where the answer is read: for a cloud Claude session reached through `claude -p ... --cloud`, the output is `reply not returned by this command; read it in the session at claude.ai/code` with the send result. It never returns an empty output as if the agent were silent.

### close session

Close a session this slot opened, by its handle, and confirm it is gone. This operation closes only a session this slot opened. For any other handle, refuse and name `stop agent session` of the host that reported it; `agent_host` owns the lifecycle of the sessions it started.

## Output Contract

An adapter answers every operation with the exact command, the handle it returns or takes, and how to confirm the result. An agent using it returns the handle, the worktree and the final state (`open` or `closed`).

Example adapter: `examples/sessions-orca.md` (Orca CLI terminals).
