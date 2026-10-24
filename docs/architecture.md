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
| review | cobra, MCP (`review_request`) | `GitHub` → `gh` with the owner's login; `Git` → the reviewer's own clone in its state dir; `Checks`/`Engine` → in-process `LocalCI` with an environment built from scratch; `Workspace` → `ci/` in the state dir; `Processes` → `/proc` + pidfds; `Lock` → flock; `Orca` → orca CLI |
| sessions | cobra, MCP | `Terminals` |
| status | cobra, MCP | `RemoteShell`, `Terminals`, GitHub → `gh` |
| audit | cobra, MCP | GitHub → `gh` |
| worktree | cobra (`worktree close`) | `Git` → `git` (never `--force`, `branch -d` only); `GitHub` → `gh pr list` (read-only); `Paths` → `os` |
| doctor | cobra, MCP | `Runner`, `RemoteShell`, local `PATH` |
| init | cobra | `FileSystem`, `Runner` (`claude mcp`) |
| plan | cobra | `FileSystem`; templates embedded |
| mcp | MCP over stdio | function ports onto the other slices' use cases |
| configcmd | cobra (`config`, read-only) | the composition root's config resolver (layers + merged config); credential-looking values redacted |

## Composition root

`cmd/gbai` is the only place that builds adapters and wires them into use cases. `factories.go` builds each use case lazily from `gbai.yaml` (so `--help` never needs configuration); the cobra commands and the MCP tools (`mcp.go`) share those factories. The config comes from `internal/config`: without `--config`, `LoadLayers` merges the system (`/etc/gbai/gbai.yaml`), user (`$XDG_CONFIG_HOME/gbai/config.yaml`, else `~/.config/gbai/config.yaml`) and project (`gbai.yaml` at the git top-level, found with `git rev-parse --show-toplevel` through the `Runner`, else the working directory) layers; `Merge` is pure (mappings merge key by key, scalars and lists replace), unknown keys fail per file, and defaults and validation run once on the result. `--config <file>` loads exactly that file (`LoadFile`), which is how the server's `review run` is invoked.

## Users and boundaries

| Where | Identity | May |
|---|---|---|
| Local machine | the owner | run every command; `gh` as the repository owner; type secrets into `gbai secrets set` |
| Local agent | the owner's agent | call MCP tools; apply tools only with `confirm: true`; never sees a secret |
| Server, `orca` | coder and session agents **and the reviewer** (single-user mode) | push branches, open PRs, `gbai review request`; the reviewer posts with the owner's `gh` login, sets `reviewer/tests` and `reviewer/review`, never approves, and merges to the integration branch pinned to the reviewed SHA |

There is no `reviewer` or `reviewer-ci` OS user and no sudo rule. **This is weaker than the former three-user model** and trust-based, like vulcam: the coder sessions run as the same OS user, so they can read the owner's `gh` login and the model credential files, and PR code run by the checks can write anywhere the owner can (including the owner's home and the reviewer state). What remains: checks and the engine get an environment built from scratch (no `GH_TOKEN`/`GITHUB_TOKEN`, `GH_CONFIG_DIR` at a fresh empty directory), their `PATH` drops entries inside the reviewer's `ci/` dir, and leftover processes are killed only when their cwd is inside `ci/`. See `docs/cloud-agent-flow.md`. Secrets travel only over stdin, never in argv, environment variables, logs or MCP tool inputs.
