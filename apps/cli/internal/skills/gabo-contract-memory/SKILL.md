---
name: gabo-contract-memory
description: "Trigger: memory contract, memory slot, decision records, resource ledger, wave trace, preferences, adapter coverage. Operations every gabo memory adapter must cover."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.1"
---
## Activation Contract

Load when writing, checking or using the adapter of the `memory` slot (`slots.memory.adapter`, TRD 9.1). This contract states WHAT agents must be able to record and read back across sessions; the adapter states HOW with the user's own memory tool (D2, D4). Recommended, not required: Engram.

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: `gabo check` compares them with the adapter named in `slots.<name>.adapter` and lists every operation it misses; a heading inside a fenced code block does not count.
- Never store secrets, tokens, credentials or sensitive personal data (R21).
- Every save is read back; a record that does not show up after one retry is reported as unverified, never as saved.
- Project scope for everything about the repository; global (personal) scope only for preference observations.

## Decision Record

Every decision, whether the owner answered or an agent decided in auto mode, carries these fields (TRD 9.2, R6, R30):

| field | meaning |
|---|---|
| `what` | what was decided, one line |
| `question` | the question that needed the decision |
| `choice` | the option taken |
| `reason` | why that option (the "why") |
| `mode` | `manual` or `auto` when it was decided |
| `wave` | wave id, or `none` outside a wave |
| `task` | plan task id (`T###`), or `global` |
| `resolver` | who decided (the "who"): `owner`, `assistant`, a directiva role or the plan-change reviewer |

## Operations

### save decision

Save one decision record with all fields above. Return the record id or key.

### list decisions

List decision records, filter by `mode`, `wave` and `task` (any combination, none means all). Return each record's id, `what`, `mode`, `wave`, `task` and `resolver`.

### search

Full-text search over the project's records. Return ids and one-line titles.

### read one

Read one record in full by its id or key.

### save ledger

Append lines to the per-wave resource ledger (D15): one line per resource the workflow created, with kind (worktree, branch, container, test database, direct session), name or path, creator agent, task and state (`live`, `removed`, `leftover`).

### read ledger

Read the whole resource ledger of one wave.

### save wave trace

Save or update the trace of one wave (D17): its tasks with status, PR, verdict and open questions, enough to resume the wave.

### read wave trace

Read the trace of one wave, or of the latest unfinished wave.

### save preference

Save or update one preference observation (observation, evidence, count) under the fixed topic `user/preferences`, in the global or the project scope (D22).

### read preferences

Read the observations under `user/preferences` from both scopes; project overrides global.

## Output Contract

An adapter answers every operation with the exact tool calls, the keys or ids they use, and how to read the result back. An agent using it returns the keys written and each save as `verified` or `unverified`.

Example adapter: `examples/memory-engram.md` (Engram MCP tools).
