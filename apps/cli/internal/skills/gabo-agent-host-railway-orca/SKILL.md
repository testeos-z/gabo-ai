---
name: gabo-agent-host-railway-orca
description: "Trigger: agent host, agent_host adapter, cloud agents, Railway server, remote Orca, start agent session, stop agent session, where agents run. Railway plus Orca adapter for the gabo agent_host slot."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
---
## Activation Contract

Use when an agent starts, stops or reports agent sessions that run away from the owner's machine. This adapter fills the optional cloud slot contract `agent_host` (`gabo-contract-agent-host`, TRD 9.1; R8, R11, R21, D2) with one Railway service running a headless Orca server, reached from the owner's machine through a saved Orca environment. Railway is read through the Railway MCP or the `railway` CLI; sessions are Orca terminals driven through the `orca` CLI. Point `slots.agent_host.adapter` at this skill's `SKILL.md`; configure the slot in a separate assistant session, never during first setup. Orca commands verified against Orca CLI 1.4.227 (`orca <command> --help`).

## Hard Rules

- Every remote operation needs the owner's explicit authorization naming the host (the Railway service and the Orca environment below), the operation and the credential. Never discover, inspect or reuse ambient Railway logins, SSH agents, Orca pairings or tokens; their presence is not permission.
- Secrets are named only by environment variable, never by value (R21): never print, echo, log or pass a secret as an argument. Never put a GitHub or model token in Railway service variables: every process in the container can read them.
- Railway is read-only here: status and logs only. Never redeploy, restart, scale or delete the service, and never run `railway ssh`, unless the owner authorized that exact operation.
- Every `orca` command takes `--environment "$GABO_ORCA_ENVIRONMENT"` and `--json`; read handles and results from the JSON, never from screen text. Never run `orca worktree rm`, `orca terminal close --all` or `orca environment stop|rm|forget`.
- Open a session only in an existing task worktree on the server, selected with `--worktree path:<worktree>`. Name it `gabo-<role>-<task id>` with `--title`.
- A session is a resource: record its handle, role and worktree in the wave ledger when it starts, and close it or report it as still open.
- If Railway does not report the service as running, or `orca status` fails, start nothing and report the host unavailable with the error.

## Fields

The former cloud keys of the core config (DB_SCHEMA 2.3, TRD 9.3) live here. Values a team fills are read from environment variables named below, set in the owner's local shell, never committed.

| former key | value in this adapter |
|---|---|
| `server.railway.project_id` | env var `GABO_RAILWAY_PROJECT_ID` |
| `server.railway.environment_id` | env var `GABO_RAILWAY_ENVIRONMENT_ID` |
| `server.railway.service_id` | env var `GABO_RAILWAY_SERVICE_ID` |
| `users.orca.{name, home, uid, extra_path}` | name `orca`, home `/data/home`, uid `1001`, extra_path `/data/home/.local/bin`; no sudo, no other users |
| `bot.login` | env var `GABO_BOT_LOGIN` for the isolated reviewer; unset means `not used` (single-user mode, the owner's own `gh` login) |
| `sessions.coder.prompt_file` | `not used`: the coder prompt is in Session prompts below |
| `sessions.reviewer.prompt_file` | `not used`: the reviewer prompt is in Session prompts below |
| `sessions.assistant.prompt_file` | `not used`: the assistant prompt is in Session prompts below |
| `sessions.local_orca` | `orca` (local CLI), saved environment named by env var `GABO_ORCA_ENVIRONMENT` |
| `/etc/gabo/gabo.yaml` | `not used`: its values are the fields above |

Credentials, by name only:

| credential | env var | where it lives |
|---|---|---|
| Railway project token (CLI, read-only use) | `RAILWAY_TOKEN` (or an account token in `RAILWAY_API_TOKEN`) | owner's local shell; the Railway MCP uses its own login |
| Model login for the reviewer engine | `CLAUDE_CODE_OAUTH_TOKEN` | on the server under `/data/home`, piped over stdin; never a Railway variable |
| GitHub login | none: `gh` logged in as `orca` on the server | `orca`'s `gh` config |

The Orca environment is saved once by the owner with `orca environment add` and the server's pairing code; this adapter never pairs.

## Session prompts

Write the role's prompt to a file in a `mktemp -d` dir on the owner's machine, never inside a worktree, filling `<task id>`, `<worktree>`, `<branch>` and `<pr>`:

| role | agent cli | prompt |
|---|---|---|
| coder | `claude` (from `runtimes.coder`) | `You are gabo-coder for task <task id>. Worktree <worktree>, branch <branch>, PR base dev. Follow the gabo-coder skill. Never merge, never force-push, never post the reviewer marker.` |
| reviewer | `pi` or `claude` (from `runtimes.reviewer`) | `You are gabo-reviewer. Review PR #<pr> head <sha> with the user-level gabo-reviewer scripts and return their VERDICT line. Do not edit, comment or merge yourself.` |
| assistant | `claude` (from `runtimes.assistant`) | `You are gabo-assistant for this project. Follow the gabo-assistant skill.` |

## Operations

### start agent session

Check the host, then open one terminal for the role and send its prompt:

```bash
railway status --json
orca status --environment "$GABO_ORCA_ENVIRONMENT" --json
orca terminal create --environment "$GABO_ORCA_ENVIRONMENT" --worktree path:<worktree> --title "gabo-<role>-<task id>" --command "<agent cli>" --json
orca terminal wait --environment "$GABO_ORCA_ENVIRONMENT" --terminal <handle> --for tui-idle --timeout-ms 60000 --json
orca terminal send --environment "$GABO_ORCA_ENVIRONMENT" --terminal <handle> --text "$(cat <prompt file>)" --enter --wait-submit 10 --json
```

`railway status --json` runs with `RAILWAY_TOKEN` set and the directory linked to `GABO_RAILWAY_PROJECT_ID`, `GABO_RAILWAY_ENVIRONMENT_ID` and `GABO_RAILWAY_SERVICE_ID`; with the Railway MCP, read the latest deployment status of the same service instead. The deployment must be running. Fields read: the Railway ids, `GABO_ORCA_ENVIRONMENT`, the role's agent cli and prompt. The handle returned by `terminal create` is the session handle. Confirm with `orca terminal list --environment "$GABO_ORCA_ENVIRONMENT" --worktree path:<worktree> --json`: the handle is listed.

### stop agent session

```bash
orca terminal close --environment "$GABO_ORCA_ENVIRONMENT" --terminal <handle> --tab --json
orca terminal list --environment "$GABO_ORCA_ENVIRONMENT" --worktree path:<worktree> --json
```

Fields read: `GABO_ORCA_ENVIRONMENT`, the handle and worktree from the ledger. Confirm: the handle no longer appears in the list; if it does, report the session as still open. An unknown handle means the session already stopped.

### report where agents run

```bash
railway status --json
orca environment show --environment "$GABO_ORCA_ENVIRONMENT" --json
orca terminal list --environment "$GABO_ORCA_ENVIRONMENT" --json
```

Report: host Railway service `GABO_RAILWAY_SERVICE_ID` in project `GABO_RAILWAY_PROJECT_ID`, environment `GABO_RAILWAY_ENVIRONMENT_ID` (ids only, never tokens), reached through Orca environment `GABO_ORCA_ENVIRONMENT`; OS user `orca` (home `/data/home`); reviewer mode `isolated` when `GABO_BOT_LOGIN` is set, otherwise `single-user`; and every `gabo-*` terminal from the list with its handle, title and worktree. Fields read: all fields above except credentials.

## Output Contract

Return the host (Railway service id and Orca environment name), each session's handle, role, title and worktree, the commands run with their JSON result, and the final state (`open`, `closed` or `still open`). Report the host unavailable with its error instead of a handle. Never include a secret value.
