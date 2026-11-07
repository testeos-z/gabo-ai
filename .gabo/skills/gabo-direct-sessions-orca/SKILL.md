---
name: gabo-direct-sessions-orca
description: "Trigger: direct session, talk to the coder, intervene in a task, open a session in a worktree, Orca terminal, direct_sessions adapter. Orca CLI adapter for the gabo direct_sessions slot."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.1"
---
## Activation Contract

Use when the owner or the assistant opens, drives or closes a direct session in one task worktree. This adapter fills the `direct_sessions` slot contract (`gabo-contract-direct-sessions`, TRD 9.1; R14, R18, US-005) with the Orca CLI: one named Orca terminal per session running the agent CLI of the resolved coder engine (`claude` or `pi`, see Engine Resolution), driven through the terminal handle Orca returns. Point `slots.direct_sessions.adapter` at this skill's `SKILL.md`. Verified against Orca CLI 1.4.227 (`orca <command> --help`).

## Hard Rules

- Open a session only in an existing task worktree, selected with `--worktree path:<worktree>`; never run `orca worktree create`, `orca worktree rm` or `orca terminal close --all`, and never touch another task's worktree.
- Name every session `gabo-<task id>` with `--title`, so the owner can find it in Orca.
- The brief names the task id, the worktree path and the branch, and repeats the coder's hard rules: never merge, never force-push, never post the reviewer marker.
- While the task's coder is running, send messages only when the owner asked for it.
- Every command takes `--json`; read the handle and results from its JSON output, never from screen text. A remote Orca host adds `--environment <name>` to every command; that host belongs to the `agent_host` adapter.
- A session is a resource: record its handle and worktree in the wave ledger when it opens, and close it or report it as still open.
- If `orca status --json` fails or the worktree is unknown to Orca, open nothing and report Orca as unavailable with the error; the caller keeps mediating.

## Engine Resolution

Resolve the coder engine from `gabo.yaml` before `open session` (intent 0013 S10), the same way as the `agent_host` adapters; a refusal opens nothing and is reported instead of a handle:

- `session` is the engine of the calling session: the `gabo-engine:` value of the caller's built agent (`{{engine}}`, intent 0013 S6), the agent that loaded this adapter (for example `gabo-assistant`). A caller whose built agent has no engine line (built before the slot) cannot resolve `session`: open nothing and report `refused: session engine unknown (rebuild the agents)`.
- `runtimes.coder` `session` resolves to the calling session's engine, and a fixed `runtimes.coder` is that engine. Example: `coder: session` from a Pi session ⇒ `pi`.
- This adapter opens `claude` and `pi` sessions only: a resolved `codex` or `opencode` is refused with `refused: <engine> has no Orca session command in gabo-direct-sessions-orca`.

## Operations

### open session

Write the brief to a file in a `mktemp -d` dir, outside the worktree. Then:

```bash
orca status --json
orca terminal create --worktree path:<worktree> --title "gabo-<task id>" --command "<agent cli>" --json
orca terminal wait --terminal <handle> --for tui-idle --timeout-ms 60000 --json
orca terminal send --terminal <handle> --text "$(cat <brief file>)" --enter --wait-submit 10 --json
```

`<agent cli>` is the resolved coder engine (`claude` or `pi`, Engine Resolution). Keep the terminal handle returned by `terminal create`; it is the session handle. Confirm with `orca terminal list --worktree path:<worktree> --json`: the handle is listed.

### send message

```bash
orca terminal send --terminal <handle> --text "<message>" --enter --wait-submit 10 --json
```

The result reports whether the input was accepted and submitted. After an ambiguous failure, reissue the exact command with the reported `--retry-request <id>`; never resend blindly.

### read state

```bash
orca terminal show --terminal <handle> --json
orca terminal read --terminal <handle> --screen --json
```

`terminal show` gives the terminal metadata (an unknown handle means the session is gone: `finished`); `terminal read --screen` gives the rendered screen, which shows whether the agent is working or waiting for input. For new output since a previous read use `orca terminal read --terminal <handle> --cursor <nextCursor> --json`.

### close session

```bash
orca terminal close --terminal <handle> --tab --json
orca terminal list --worktree path:<worktree> --json
```

The handle must no longer appear in the list; if it does, report the session as still open.

## Output Contract

Return the session handle, its title, the resolved coder engine, the worktree, the commands run with their JSON result, and the final state (`open` or `closed`). Report Orca unavailable with its error, or the refusal line, instead of a handle.
