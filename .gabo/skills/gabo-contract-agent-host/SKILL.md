---
name: gabo-contract-agent-host
description: "Trigger: agent host contract, agent_host slot, local agents, cloud, server, where agents run, Railway, adapter coverage. Operations every gabo agent host adapter must cover."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
---
## Activation Contract

Load when writing, checking or using the adapter of the optional slot `agent_host` (`slots.agent_host.adapter`, TRD 9.1). This contract states WHAT gabo needs from a host that runs agents as their own sessions, on the owner's machine or on a server; the adapter states WHERE and HOW, and holds every host setting. A local-only team needs no `agent_host` adapter, or points it at the shipped `gabo-agent-host-local` when a role must run as its own local session process. The slot is configured in a separate assistant session, never during first setup.

## Hard Rules

- An adapter covers an operation only with a `### <operation>` heading, spelled exactly as below, under its own `## Operations` section. The headings are stable: `gabo check` compares them with the adapter named in `slots.<name>.adapter` and lists every operation it misses; a heading inside a fenced code block does not count.
- Host settings (cloud or local) live in the adapter, never in the core config: `gabo.yaml` holds only `slots.agent_host.adapter`.
- The adapter names secrets only by environment variable name or reference, never by value (R21).
- Starting or stopping a remote session needs the owner's explicit authorization of the host, the operation and the credential to use; an adapter never discovers or reuses ambient credentials.

## Fields

The former cloud keys of the core config move into the adapter (DB_SCHEMA 2.3, TRD 9.3). An adapter states a value or `not used` for each:

| former key | holds |
|---|---|
| `server.railway.{project_id, environment_id, service_id}` | server ids on the hosting provider |
| `users.orca.{name, home, uid, extra_path}` | OS user that runs agents on the server |
| `bot.login` | bot identity used by an isolated reviewer |
| `sessions.{coder, reviewer, assistant}.prompt_file` | session prompt file per agent role |
| `sessions.local_orca` | Orca CLI name used to reach the host |
| `/etc/gabo/gabo.yaml` | the former system config layer, now only in this adapter |

## Operations

### start agent session

Start one agent session for a role (coder, reviewer, assistant) on the host, in a given worktree, with its prompt. Return a session handle.

### stop agent session

Stop a session by its handle and confirm it stopped.

### report where agents run

Report the host, the OS user, the reviewer mode (`single-user` or `isolated`) and the sessions running there.

## Output Contract

An adapter answers every operation with the exact command, the fields it reads and how to confirm the result. An agent using it returns the host, the handle and the final state of each session.

Example adapter: `examples/agent-host-railway.md` (a Railway service running Orca).
