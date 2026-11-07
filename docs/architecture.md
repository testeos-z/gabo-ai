# Architecture

`gabo` is one Go binary (`apps/cli`): an agent compiler only (TRD D18). It installs the embedded gabo skills (`gabo init`) and validates a project read-only (`gabo check`: config, including `runtimes.assistant` as one engine or a list without duplicates such as `[claude, pi]` and `session` only in `runtimes.coder` and `runtimes.reviewer`, spec agents in the declared agent directory of each engine of the engine set (every engine listed in `runtimes.assistant` plus `runtimes.coder` and `runtimes.reviewer` when each names an engine; `session` adds none), agent version lines (never skipped: a missing line fails, and without a valid stamp the versions are reported as not compared), each agent file's `gabo-engine:` line against the engine of its agent directory (`✗ <file>: built for <value>, not <engine>`; a file with no engine line passes), per-agent skill lists and declared agent names read where each runtime skill declares them (`metadata.agent_skills`, `metadata.agent_name`), skill links in each runtime's skill directory (Codex, OpenCode and Pi share `.agents/skills`; Claude Code keeps `.claude/skills`), no `gabo-reviewer` in the project, no gabo link left in a runtime's former skill directory (`metadata.former_skill_dir`), the main-session entry of every engine listed in `runtimes.assistant` and no stale entry of a full-tier engine that is not listed (below), slot coverage (every `slots.<name>.adapter` exists and has a `### <operation>` heading under `## Operations` for each operation of its contract skill in `.gabo/skills/`) and the catalog version stamp; a project file or global config dir still under the name from before the gabo-ai rename fails with the rename to make, and is never read); the work itself runs in the agents through those skills. The work-time commands (`status`, `audit`, `sessions`, `provision`, `secrets`, `repo`, `worktree`, `mcp`, `doctor`, `config`) and the gabo MCP server were retired by T042; `review` and `plan` remain until T043 retires them. `docs/cloud-agent-flow.md` describes the retired cloud flow and is kept for history only.

The owner's main session runs as the gabo assistant (intent 0009), on any engine listed in `runtimes.assistant` (intent 0013): each listed engine gets its own entry and none removes another. `gabo init` writes the config key of each listed engine's entry, `agent` or `default_agent`, removes it from every engine that is not listed, and writes OpenCode's `experimental.subagent_depth`; the runtime skill of each listed engine builds the agent file, the skill link and the Codex and Pi block on every agent build, from the table in `gabo-agent-templates` (for a flexible-tier engine the key stays with its builder-made runtime skill, because `gabo init` sets nothing engine-specific there). Claude Code gets the agent file `.claude/agents/gabo-assistant.md` and `agent: "gabo-assistant"` in `.claude/settings.json`; OpenCode gets `.opencode/agents/gabo-assistant.md` (`mode: primary`) and `default_agent: "gabo-assistant"` in its project config; Codex gets a scoped `<!-- GABO_ASSISTANT_START -->` … `<!-- GABO_ASSISTANT_END -->` block in `AGENTS.override.md`, else `AGENTS.md`; Pi gets the same block in the first existing of `AGENTS.override.md`, `AGENTS.md`, `CLAUDE.md`; Codex and Pi listed together share one block. A main session knows its engine from the `gabo-engine:` line of its agent file (Claude Code, OpenCode) or from the fixed line of the block's scope paragraph (``Session engine: loaded by Codex, this session's engine is `codex`; loaded by Pi, it is `pi`.``); without either it refuses with `session engine unknown (rebuild the agents)`. Each also links `gabo-assistant` into its skill directory (`.claude/skills` or `.agents/skills`). Role and headless sessions keep their role. `gabo check` reads what each entry is from the runtime skill's `metadata.main_session`, so the check slice names no engine; an entry of a full-tier engine that is not listed fails with `✗ <file>: gabo assistant entry for <engine>, but runtimes.assistant is <engines joined with ", ">`; a missing, stale or low key gets a `next: run gabo init (…)` hint, every other piece `next: ask your agent to rebuild the agents` (TRD 8.1, 9.4).

Several main sessions can run waves in one project at the same time, one per intent (intent 0013). `runtimes.coder` and `runtimes.reviewer` set to `session` run on the engine of the session that starts the wave. The coordination lives in the skills, not in Go: `gabo-cycle` claims its intent on the intent issue through the `tasks` slot before any task, and a session that finds it held claims nothing and returns `intent <I####> is claimed by session <session id> (engine <engine>) since <UTC time>; it is released when that wave closes`; each wave keeps its own ledger `gabo/ledger/<wave>` and worktrees under `<repo-parent>/<repo-name>-worktrees/<wave>/<task>`, and cleans only its own resources.

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
