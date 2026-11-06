# Architecture

`gabo` is one Go binary (`apps/cli`): an agent compiler only (TRD D18). It installs the embedded gabo skills (`gabo init`) and validates a project read-only (`gabo check`: config, spec agents in each runtime's declared agent directory, agent version lines (never skipped: a missing line fails, and without a valid stamp the versions are reported as not compared), per-agent skill lists and declared agent names read where each runtime skill declares them (`metadata.agent_skills`, `metadata.agent_name`), skill links in each runtime's skill directory (Codex, OpenCode and Pi share `.agents/skills`; Claude Code keeps `.claude/skills`), no `gabo-reviewer` in the project, no gabo link left in a runtime's former skill directory (`metadata.former_skill_dir`), slot coverage (every `slots.<name>.adapter` exists and has a `### <operation>` heading under `## Operations` for each operation of its contract skill in `.gabo/skills/`) and the catalog version stamp; a project file or global config dir still under the name from before the gabo-ai rename fails with the rename to make, and is never read); the work itself runs in the agents through those skills. The work-time commands (`status`, `audit`, `sessions`, `provision`, `secrets`, `repo`, `worktree`, `mcp`, `doctor`, `config`) and the gabo MCP server were retired by T042; `review` and `plan` remain until T043 retires them. `docs/cloud-agent-flow.md` describes the retired cloud flow and is kept for history only.

## Hexagonal vertical slices

Each command is a slice under `internal/slices/<name>/`:

- `usecase.go` — the core: use case and domain types. It never imports cobra, `os/exec` or adapters.
- `ports.go` — interfaces the core owns, when the slice needs its own.
- `command.go` — the driving adapter (cobra command).
- `adapters/` — slice-specific driven adapters.

Use-case tests use hand-written in-memory fakes of the ports.

Ports shared by several slices live in `internal/kernel` (`Runner`, `FileSystem`), with driven adapters in its subpackages and fakes in `internal/kernel/kerneltest`.

| Kernel port | Adapter |
|---|---|
| `Runner` (local processes, secrets only on stdin) | `kernel/execrunner` (os/exec) |
| `FileSystem` (local files) | `kernel/osfs` |

## Ports and adapters per slice

| Slice | Driving | Driven ports → adapters |
|---|---|---|
| version | cobra | — |
| init | cobra | `FileSystem`; skills embedded from `internal/skills` |
| review (until T043) | cobra | `GitHub` → `gh` with the owner's login; `Git` → the reviewer's own clone in its state dir; `Checks`/`Engine` → in-process `LocalCI` with an environment built from scratch; `Workspace` → `ci/` in the state dir; `Processes` → `/proc` + pidfds; `Lock` → flock; `Orca` → orca CLI |
| plan (until T043) | cobra | `FileSystem`; templates embedded |

## Composition root

`cmd/gabo` is the only place that builds adapters and wires them into use cases. Use cases are built lazily, so `--help` never needs configuration. The config comes from `internal/config`: without `--config`, `LoadLayers` merges the user (`$XDG_CONFIG_HOME/gabo/config.yaml`, else `~/.config/gabo/config.yaml`) and project (`gabo.yaml` at the git top-level, found with `git rev-parse --show-toplevel` through the `Runner`, else the working directory) layers (there is no system layer: `/etc/gabo/gabo.yaml` is never read); `Merge` is pure (mappings merge key by key, scalars and lists replace), unknown keys fail per file, and validation runs once on the result, filling no default. `--config <file>` loads exactly that file (`LoadFile`). The former cloud keys (`server`, `users`, `bot`, `sessions`) are no longer config types, so a file carrying them fails as unknown keys naming the key path and `slots.agent_host`; the `generation` key is removed too (TRD D24). gabo never rewrites a config file.

## Users and boundaries

Single-user mode: the owner, the owner's agents and the reviewer run as the same OS user with the owner's `gh` login. The reviewer posts with that login, never approves, and merges only into the integration branch pinned to the reviewed SHA. This is trust-based, not an OS-user boundary: PR code run by the checks can write anywhere the owner can. What remains: checks and the engine get an environment built from scratch (no `GH_TOKEN`/`GITHUB_TOKEN`, `GH_CONFIG_DIR` at a fresh empty directory), their `PATH` drops entries inside the reviewer's `ci/` dir, and leftover processes are killed only when their cwd is inside `ci/`. The reviewer reads its model credential from `~/secrets` of the user running it. Secrets travel only over stdin, never in argv, environment variables or logs.
