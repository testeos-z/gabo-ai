# Architecture

`gbai` is one Go binary (`apps/cli`) that is both the CLI and the MCP server. The agent flow it automates is described in `docs/cloud-agent-flow.md`.

## Hexagonal vertical slices

Each command is a slice under `internal/slices/<name>/`:

- `usecase.go` — the core: use case and domain types. It never imports cobra, the MCP SDK, `os/exec` or adapters.
- `ports.go` — interfaces the core owns, when the slice needs its own.
- `command.go` — the driving adapter (cobra command; for `mcp`, the MCP server).
- `adapters/` — slice-specific driven adapters.

Use-case tests use hand-written in-memory fakes of the ports.

Ports shared by several slices live in `internal/kernel` (`Runner`, `RemoteShell`, `Terminals`, `FileSystem`), with driven adapters in its subpackages and fakes in `internal/kernel/kerneltest`.

| Kernel port | Adapter |
|---|---|
| `Runner` (local processes, secrets only on stdin) | `kernel/execrunner` (os/exec) |
| `RemoteShell` (script as an OS user on the server) | `kernel/railwayssh` (`railway ssh` + `setpriv`) |
| `Terminals` (Orca terminals) | `kernel/orcacli` (server as `orca`, or local) |
| `FileSystem` (local files) | `kernel/osfs` |

## Ports and adapters per slice

| Slice | Driving | Driven ports → adapters |
|---|---|---|
| version | cobra | — |
| secrets | cobra | `RemoteShell`; secret reader → no-echo terminal stdin |
| provision | cobra, MCP | `RemoteShell`; `BinarySource` → local file; skills tar from `internal/skills` |
| repo | cobra, MCP | GitHub `API` → `gh api` |
| review | cobra, MCP (`review_request`) | `GitHub` → `gh` as the bot; `Git` → private clone; `Checks`/`Engine` → `sudo review-ci`; `Workspace` → reviewer-ci dir; `Orca` → orca CLI |
| sessions | cobra, MCP | `Terminals` |
| status | cobra, MCP | `RemoteShell`, `Terminals`, GitHub → `gh` |
| audit | cobra, MCP | GitHub → `gh` |
| doctor | cobra, MCP | `Runner`, `RemoteShell`, local `PATH` |
| init | cobra | `FileSystem`, `Runner` (`claude mcp`) |
| plan | cobra | `FileSystem`; templates embedded |
| mcp | MCP over stdio | function ports onto the other slices' use cases |

## Composition root

`cmd/gbai` is the only place that builds adapters and wires them into use cases. `factories.go` builds each use case lazily from `gbai.yaml` (so `--help` never needs configuration); the cobra commands and the MCP tools (`mcp.go`) share those factories.

## Users and boundaries

| Where | Identity | May |
|---|---|---|
| Local machine | the owner | run every command; `gh` as the repository owner; type secrets into `gbai secrets set` |
| Local agent | the owner's agent | call MCP tools; apply tools only with `confirm: true`; never sees a secret |
| Server, `orca` | coder and session agents | push branches, open PRs, `gbai review request`; cannot read the reviewer's files |
| Server, `reviewer` | the review script | GitHub as the bot, verdict, merge to `dev` pinned to the SHA |
| Server, `reviewer-ci` | untrusted PR code and the model | checks and read-only engine in a copied tree; no GitHub credentials |

Each boundary crossing is one root-owned wrapper reached through one sudo rule (`review-run`, `review-ci`). Secrets travel only over stdin, never in argv, environment variables, logs or MCP tool inputs.
