---
name: gabo-direct-sessions-orca
description: "Trigger: direct session, talk to the coder, intervene in a task, open a session in a worktree, Orca terminal, direct_sessions adapter, remote session handle. Orca CLI adapter for the gabo direct_sessions slot."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.2"
---
## Activation Contract

Use when the owner or the assistant opens, drives, reads or closes a direct session. This adapter fills the `direct_sessions` slot contract (`gabo-contract-direct-sessions`, TRD 9.1; R14, R18, US-005) with the Orca CLI: one named Orca terminal per session running the agent CLI of the resolved coder engine (`claude` or `pi`, see Engine Resolution), driven through the terminal handle Orca returns. `send message` and `read state` also reach an Orca terminal session that `agent_host` started on a remote Orca host, through its `host` and `handle` fields (see Handle). Point `slots.direct_sessions.adapter` at this skill's `SKILL.md`. Verified against Orca CLI 1.4.227 (`orca <command> --help`); on that CLI `orca terminal show --help`, `orca terminal read --help` and `orca terminal send --help` list `--environment`.

## Hard Rules

- Open a session only in an existing task worktree, selected with `--worktree path:<worktree>`; never run `orca worktree create`, `orca worktree rm` or `orca terminal close --all`, and never touch another task's worktree.
- `open session` runs on the local Orca only; a session on a remote host is started by `agent_host`.
- Name every session `gabo-<task id>` with `--title`, so the owner can find it in Orca.
- The brief names the task id, the worktree path and the branch, and repeats the coder's hard rules: never merge, never force-push, never post the reviewer marker.
- While the task's coder is running, send messages only when the owner asked for it.
- Every command takes `--json`; read the handle and results from its JSON output, never from screen text.
- A session is a resource: record its `host` and `handle` and its worktree in the wave ledger when it opens, and close it or report it as still open.
- If `orca status --json` fails or the worktree is unknown to Orca, open nothing and report Orca as unavailable with the error; the caller keeps mediating.

## Handle

- A handle has two named fields, `host` and `handle`, never one joined string.
- `host` is `local` for the local host, the literal that `gabo-contract-direct-sessions` names; it adds no `--environment`.
- A remote `host` is the Orca environment name that `agent_host` reports for where the session runs: `gabo-agent-host-railway-orca` returns it with the session handle. The adapter reads that name from the env var `GABO_ORCA_ENVIRONMENT` and never a secret value. The caller records the `host` with the handle when it starts the session, since `start agent session` prints the handle; the name is the one its Output Contract returns.
- `handle` is the terminal handle that Orca returned for the session.
- This adapter drives Orca terminals only: a session that `gabo-agent-host-local` started is a background process with a pid and a session id, not an Orca terminal, so its handle is not reachable through this adapter and is refused before any `orca` command. Its `report where agents run` names the machine's hostname, which is an unknown host here.
- A `host` that is neither `local` nor the Orca environment in `GABO_ORCA_ENVIRONMENT` is an unknown host: it is refused before any `orca` command, and nothing is sent or read.
- A remote `host` must equal the value of `GABO_ORCA_ENVIRONMENT`; a mismatched `host` is refused before any `orca` command, so a handle of host B never runs against host A.
- If `GABO_ORCA_ENVIRONMENT` is unset or empty, refuse before any `orca` command, name the missing variable, and send and read nothing.

## Remote Use

- Before any command to a remote `host`, the owner's explicit authorization, given in this session, must name the host, the operation (`send message` or `read state`) and the credential or session to use.
- With no authorization that names all three, send and read nothing and ask the caller for it.
- An authorization for host A never reaches host B.
- Never discover, reuse or create an Orca pairing, a credential, an SSH agent, a ControlMaster socket or an authenticated session; their presence is not permission.
- The `local` host needs no authorization.

## Engine Resolution

Resolve the coder engine from `gabo.yaml` before `open session` (intent 0013 S10), the same way as the `agent_host` adapters; a refusal opens nothing and is reported instead of a handle:

- `session` is the engine of the calling session: the `gabo-engine:` value of the caller's built agent (`{{engine}}`, intent 0013 S6), the agent that loaded this adapter (for example `gabo-assistant`). A caller whose built agent has no engine line (built before the slot) cannot resolve `session`: open nothing and report `refused: session engine unknown (rebuild the agents)`.
- `runtimes.coder` `session` resolves to the calling session's engine, and a fixed `runtimes.coder` is that engine. Example: `coder: session` from a Pi session ⇒ `pi`.
- This adapter opens `claude` and `pi` sessions only: a resolved `codex` or `opencode` is refused with `refused: <engine> has no Orca session command in gabo-direct-sessions-orca`.

## Operations

### open session

Opens on the local Orca only; a session on a remote host is started by `agent_host`.

Write the brief to a file in a `mktemp -d` dir, outside the worktree. Then:

```bash
orca status --json
orca terminal create --worktree path:<worktree> --title "gabo-<task id>" --command "<agent cli>" --json
orca terminal wait --terminal <handle> --for tui-idle --timeout-ms 60000 --json
orca terminal send --terminal <handle> --text "$(cat <brief file>)" --enter --wait-submit 10 --json
```

`<agent cli>` is the resolved coder engine (`claude` or `pi`, Engine Resolution). Keep the terminal handle returned by `terminal create` as the `handle` field, with `host` `local`. Confirm with `orca terminal list --worktree path:<worktree> --json`: the handle is listed.

### send message

Check the handle under `## Handle` and, for a remote `host`, the authorization under `## Remote Use` before any command. For a remote `host`, check that it equals `GABO_ORCA_ENVIRONMENT` and that the variable is set and non-empty; otherwise refuse and run no command. A `local` handle:

```bash
orca terminal send --terminal <handle> --text "<message>" --enter --wait-submit 10 --json
```

A remote handle runs the same command with the environment flag after `orca terminal send`:

```bash
orca terminal send --environment "$GABO_ORCA_ENVIRONMENT" --terminal <handle> --text "<message>" --enter --wait-submit 10 --json
```

It drives an Orca terminal, so it works for an Orca terminal session of any engine, including a `codex` session that `agent_host` started on a remote Orca host. The result reports whether the input was accepted and submitted. After an ambiguous failure, reissue the exact command with the reported `--retry-request <id>`; never resend blindly.

### read state

Check the handle under `## Handle` and, for a remote `host`, the authorization under `## Remote Use` before any command. For a remote `host`, check that it equals `GABO_ORCA_ENVIRONMENT` and that the variable is set and non-empty; otherwise refuse and run no command. A `local` handle:

```bash
orca terminal show --terminal <handle> --json
orca terminal read --terminal <handle> --screen --json
```

A remote handle runs the same two commands with the environment flag:

```bash
orca terminal show --environment "$GABO_ORCA_ENVIRONMENT" --terminal <handle> --json
orca terminal read --environment "$GABO_ORCA_ENVIRONMENT" --terminal <handle> --screen --json
```

`terminal show` gives the terminal metadata (an unknown handle means the session is gone: `finished`); `terminal read --screen` gives the rendered screen, which shows whether the agent is working or waiting for input. For new output since a previous read use `orca terminal read --terminal <handle> --cursor <nextCursor> --json`, with the environment flag as above for a remote handle. It drives an Orca terminal, so it works for an Orca terminal session of any engine, including a `codex` session that `agent_host` started on a remote Orca host.

### close session

Closes only a handle this slot opened on the local host and recorded in the wave ledger. For any other handle, including every remote `host`, refuse and name `stop agent session` of the host that reported it; `agent_host` owns the lifecycle of the sessions it started.

```bash
orca terminal close --terminal <handle> --tab --json
orca terminal list --worktree path:<worktree> --json
```

The handle must no longer appear in the list; if it does, report the session as still open.

## Output Contract

Return the session handle as its `host` and `handle` fields, its title, the resolved coder engine, the worktree, the commands run with their JSON result, and the final state (`open` or `closed`). Report Orca unavailable with its error, or the refusal line, instead of a handle.
