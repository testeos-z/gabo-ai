# DB Schema (this project has no database: configuration schema)

Owner: gbai-db-master.
Updated: 2026-10-26 (owner answers to the open questions applied; see [Open questions](#8-open-questions); Go CLI as agent compiler only and installed reviewer scripts applied, TRD D18, D20; semantic coherence pass 2026-10-26, open questions 10–11 added; second owner pass 2026-10-26: questions 10–11 resolved, reviewer skill at user level, direct YAML writes by agents, per-runtime installer table, personalization layers, TRD D18, D20–D22).

## 1. Database and configuration

Persistence type: file-based (YAML configuration files, validated with JSON Schema); no database engine.

**gbai has no database.** It runs no server-side store, keeps no tables and needs no pooling, extensions or backups of its own. Its only own persistence is configuration files plus the files it generates or installs from them (approved owner 2026-10-26: agent files, skills, adapters and reviewer scripts, TRD D18). Everything else lives in the team's tools, reached through slot adapters (TRD [Dynamic configuration](TRD.md#81-dynamic-configuration), D2):

- **Tasks and dependencies** live in the tracker (tasks slot, D4, D12).
- **Decisions and auto-mode records** live in the memory provider (memory slot, D5).
- **Review verdicts** live as PR comments on the repository host (D6).

This document therefore replaces "schema" with the file formats gbai owns, "migrations" with config version migrations, and "indexes" with the lookup keys gbai uses (git email in the developer catalog, slot name in `slots`).

**Format.** YAML, validated with JSON Schema (editor autocomplete and validation). TOON is not used for config files.

**No silent defaults (R12).** The assistant asks before writing any key. Documented defaults are suggestions the assistant offers, not values gbai applies behind the user's back. The loader fills no key (migration step 5, T005): a missing required key fails with `✗ <key> missing` and `next: run setup`, and the former code defaults are exposed only as a read-only suggestions list (`config.Suggestions`).

### Persistence inventory

| file | location | owner / scope | written by | committed? |
|---|---|---|---|---|
| Global user config | `$XDG_CONFIG_HOME/gbai/config.yaml`, else `~/.config/gbai/config.yaml` | One person, all repositories | Agents by editing the YAML (setup and work), the user, or `gbai config set` (Approved (owner 2026-10-26), TRD D18) | No (lives outside any repository) |
| Project config | `gbai.yaml` at the git top-level | Team and repository | Agents by editing the YAML (setup and work), the user, or `gbai config set` (Approved (owner 2026-10-26), TRD D18) | Yes |
| Slot adapters | Paths referenced from `slots.*.adapter`, for example `.gbai/adapters/tasks-gh.md` | Team and repository | The user, or co-written with the assistant | Yes |
| Generated agent files | Runtime-native project locations: `.claude/agents/*.md` for Claude Code, `.pi/agents/*.md` for Pi (Approved (owner 2026-10-26): Pi core has no subagents, so Pi targets need a subagent package; with the recommended gentle-pi, `.pi/agents/` is a verified project discovery path in plain Pi and in `gentle-shell`; TRD 9.6, question 11 resolved) | Team and repository | gbai generator only (D3) | Yes, so PRs show agent changes |
| Installed skills and tool registrations | Per the installer table (TRD 9.6, D21): project skills in `.claude/skills/` and `.pi/skills/`; MCP entries in `.mcp.json` (Claude Code) and `.pi/mcp.json` (Pi); Pi packages recorded in `.pi/settings.json`; Claude Code plugins under `~/.claude/plugins` | Team and repository (plugins: user) | `gbai generate` / `gbai init` only (approved owner 2026-10-26, TRD D18, D21) | Project files yes, so PRs show skill and registration changes; user-level files no |
| `gbai-reviewer` skill and its scripts | User level only, never in the repository: `~/.claude/skills/gbai-reviewer/` and `<pi-agent-home>/skills/gbai-reviewer/` (the active Pi agent home: plain Pi `~/.pi/agent`, `gentle-shell` `~/.gentle-shell/agent`; detected per TRD 9.6), each with a `scripts/` folder (Approved (owner 2026-10-26), [open question 10](#8-open-questions), TRD D20) | One person, all repositories | `gbai generate` / `gbai init` only, from the embedded skill | No; PR code cannot modify the scripts that review it |
| System config (current) | `/etc/gbai/gbai.yaml` | One server | Server provisioning | No; moves into the `slots.agent_host` adapter under the approved split |
| Reviewer state | `review.state_dir`, else `$XDG_STATE_HOME/gbai/reviewer`, else `~/.local/state/gbai/reviewer`; never inside the coder's checkout | One reviewer host | The scripts of the user-level `gbai-reviewer` skill (TRD D20; the Go review slice is retired) | No; disposable clone, worktrees and check workspaces, not configuration |

## 2. Layering and precedence

### Current behavior (implemented)

The loader (`apps/cli/internal/config/layers.go`) resolves three layers, lowest to highest precedence:

1. **system**: `/etc/gbai/gbai.yaml`
2. **user**: `$XDG_CONFIG_HOME/gbai/config.yaml` (only when absolute), else `~/.config/gbai/config.yaml`
3. **project**: `gbai.yaml` at `git rev-parse --show-toplevel`, or the current directory outside a git repository

Rules:

- Missing files are skipped. If no file exists at all, loading fails and lists every path it looked in.
- **Strict keys per layer.** Each file is decoded against the full `Config` type with unknown keys rejected, so a typo fails in the file that contains it.
- **Deep merge.** Mappings present in both layers merge key by key; any other value (scalar, list, null) in the higher layer replaces the lower one; a key absent in the higher layer keeps the lower value. Lists are replaced, never concatenated.
- **Version per file.** Each file's `version` is checked on its own (0 to the newest supported version; missing reads as 0) and an unsupported value fails naming that file. `version` is not merged.
- Defaults and validation run once, on the merged result.
- `--config <file>` loads exactly that one file, with no layering and no key-class filtering: personal keys in it are used as written, because the flag means "exactly this file".

Personal keys set in the project file are dropped with a warning (see the split below); every other key is accepted in every layer.

### Approved split (loader implemented in T003)

| key class | keys | valid in | precedence |
|---|---|---|---|
| Personal | `alias`, `interaction_level`, `tone`, `language` | Global user config | **Global always wins.** If a project file sets one, it is ignored and the global value is used. Every command that loads the layered config prints one warning per ignored key on stderr, once per process. |
| Project | Everything in [Project `gbai.yaml`](#32-project-gbaiyaml) | Project file (global file may hold a personal fallback) | **Local wins**, using the existing deep merge. |

This reverses the merge order for personal keys only: project keys keep user → project (the system layer moves into the `slots.agent_host` adapter, see below), personal keys resolve user first. The developer catalog stays in the project file because it is shared by the team (D10); a person's own `alias` in the global file wins over their catalog entry (D11).

### Core keys versus the cloud slot

The current config is cloud-shaped. Under the approved split, cloud fields leave the core config and move into the "where agents run" slot adapter, `slots.agent_host`, configured in a separate assistant session (TRD [Deployment view](TRD.md#6-deployment-view)).

| current key | destination | reason |
|---|---|---|
| `server.railway.{project_id, environment_id, service_id}` | `slots.agent_host` adapter | Railway server ids |
| `users.orca.{name, home, uid, extra_path}` | `slots.agent_host` adapter | OS user on the server |
| `bot.login` | `slots.agent_host` adapter | Bot identity used by the server reviewer |
| `sessions.{coder, reviewer, assistant}.prompt_file` | `slots.agent_host` adapter | Orca sessions on the server |
| `sessions.local_orca` | `slots.agent_host` adapter | Orca CLI name belongs with where agents run |
| System layer `/etc/gbai/gbai.yaml` | `slots.agent_host` adapter | Exists only for the server; core config no longer reads it |
| `runtimes.{coder, reviewer, assistant}` | Core | Runtime per agent; `runtimes.reviewer` is the only reviewer runtime key |
| `repo.{name, integration_branch}` | Core | Target repository and auto-mode merge branch (D13) |
| `coder.login` | Core | Coder-merge audit rule |
| `review.{checks, auto_merge, human_only_paths, pi_credential_env, state_dir}` | Core | Read by the installed reviewer scripts (Approved (owner 2026-10-26), TRD D20) |
| `parallelism` | Core | Coder sessions per wave |
| `worktree.keep` | Core | Read by the owning coder at task close and by the cycle skill at the wave-close sweep; `gbai worktree close` is removed (Approved (owner 2026-10-26), TRD D15, D18; amended (owner 2026-10-27), TRD D23) |

The system layer `/etc/gbai/gbai.yaml` exists today only for the server, so it moves into the `slots.agent_host` adapter together with `sessions.local_orca`; the core loader keeps only the user and project layers.

## 3. Schema per file

Every key not implemented today is marked **proposed (not implemented)**. Implemented keys show their suggestion, the former code default, which gbai no longer applies (R12, T005).

### 3.1 Global user config

| key | type | required | description | example |
|---|---|---|---|---|
| `version` | integer | no (missing = 0) | Schema version of this file, validated per file. Loaded and validated (T003). | `1` |
| `alias` | string | no | How the assistant addresses this person (R16, D11). Loaded and validated (T003). | `Leo` |
| `interaction_level` | enum `low`, `balanced`, `high` | no | Preferred number of interactions (R17). The assistant adjusts it from what it learns about the user, for example the wave-close comfort question. Loaded and validated (T003). | `balanced` |
| `tone` | string | no | Tone and speaking style, free text (R17). Loaded and validated (T003). | `direct, brief` |
| `language` | string | no | Conversation language as a BCP 47 tag. Generated artifacts stay English. Loaded and validated (T003). | `es` |

Today the user layer accepts the same keys as the project file; under the split it holds only personal keys (plus any project fallback the owner keeps there deliberately).

**Personalization layers.** Approved (owner 2026-10-26), TRD D22. This file is the **stable layer**: every compiled agent reads `alias`, `language`, `tone` and `interaction_level` from it. The **learning layer** is not a gbai file: observations (for example "prefers short answers") are saved through the `memory` slot (Engram or claude-mem, per adapter) under the fixed topic `user/preferences`, personal ones in the global memory scope and project ones in the project scope, and the assistant reads them at session start. An observation repeated about 3 times is proposed for this file: in manual mode the assistant asks the owner; in auto mode it edits the file and records the decision in memory. The wave-close comfort answer edits `interaction_level` and `tone` here. Neither layer ever holds secrets, credentials or sensitive personal data (R21).

### 3.2 Project `gbai.yaml`

**Implemented keys that stay core:**

| key | type | required | description | example |
|---|---|---|---|---|
| `repo.name` | string `owner/name` | yes | Target GitHub repository | `testeos-z/gbai-team` |
| `repo.integration_branch` | string | yes (suggestion `dev`) | Only branch auto mode may merge into, never main (D13) | `dev` |
| `runtimes.coder` | enum `claude`, `pi` | yes (suggestion `claude`) | Coder runtime | `claude` |
| `runtimes.reviewer` | enum `claude`, `pi` | yes (suggestion `pi`) | Reviewer runtime, the only key for it (no separate engine key under `reviewer`); TRD requires the assistant to ask (no default) | `pi` |
| `runtimes.assistant` | enum `claude`, `pi` | yes (suggestion `claude`) | Assistant runtime | `claude` |
| `coder.login` | string | no | Coder GitHub login; audit flags merges by it; must differ from `bot.login` | `gbai-coder` |
| `review.checks` | list of string | yes (suggestion: bun install, build, check-types, lint, test) | Commands run in order in a credential-free copy of the merged tree; split on whitespace, no shell | `["go test ./..."]` |
| `review.auto_merge` | boolean | no (`false`) | Reviewer merges on approve | `false` |
| `review.human_only_paths` | list of relative path | yes (suggestion: the former default list) | Paths only a human may change | `[".github/", "AGENTS.md"]` |
| `review.pi_credential_env` | env var name | when `runtimes.reviewer` is `pi` (suggestion `ANTHROPIC_API_KEY`) | Env var that receives the pi model credential; the value never enters config (R21) | `ANTHROPIC_API_KEY` |
| `review.state_dir` | absolute path | no | Reviewer clone and workspaces; see inventory | `/data/gbai/reviewer` |
| `parallelism` | positive integer | yes (suggestion `3`) | Most coder sessions at once | `3` |
| `worktree.keep` | list of string | no | Worktree names or absolute paths never closed | `["main"]` |

**New keys:**

| key | type | required | description | example |
|---|---|---|---|---|
| `version` | integer | yes | Schema version of this file. Loaded and validated (T003); failing when missing is T005 | `1` |
| `mode` | enum `manual`, `auto` | yes | Who resolves questions (R6). Loaded and validated when set (T004); failing when missing is T005 | `manual` |
| `slots.<name>.adapter` | relative path | per slot | Adapter Markdown file filling that slot's contract (D2). Slot names follow the TRD contracts: `tasks`, `memory`, `notifications`, `direct_sessions`, `methodology`, `repository_host`, `reviewer_placement`, plus the cloud slot `agent_host`. Loaded and validated when set (T004); failing when missing is T005 | `.gbai/adapters/tasks-gh.md` |
| `slots.methodology.adapter` | skill name or relative path | yes | Methodology slot, linked as `gbai-methodology` (R8). Coder-side review (for example GGA or RDD) is configured inside this adapter, not as its own key. Recommended, not required (owner 2026-10-26): a gentle-ai ODD adapter with RDD as coder-side validation, offered by setup and used only if the user accepts. Loaded and validated when set (T004); failing when missing is T005 | `.gbai/adapters/methodology-tdd.md` |
| `slots.agent_host.adapter` | relative path | no | Cloud slot: where agents run. Holds the server ids, OS user, bot identity, session prompt files, the Orca CLI name and the former system layer. Configured in a separate assistant session. Loaded and validated when set (T004); failing when missing is T005 | `.gbai/adapters/agent-host-railway.md` |
| `reviewer.placement` | enum `local`, `server`, `github-actions`, `gitlab-ci` | yes | Where the reviewer runs; first release `local` only (D6). Loaded and validated when set (T004); failing when missing is T005 | `local` |
| `reviewer.signing` | mapping | no | How the reviewer signs or marks its verdict comment (PRD 10.1 reviewer signing or marker); fields follow the reviewer comment format. Loaded and validated when set (T004); failing when missing is T005 | `{marker: gbai-review}` |
| `developers` | map: git email → alias | no | Developer catalog (D10, R16); the assistant adds a missing entry once. Loaded and validated when set (T004); failing when missing is T005 | `{"ana@example.com": "Ana"}` |
| `plan_change.scope` | enum `task`, `group`, `global` | yes | Widest plan scope that may change (R19, D7). Loaded and validated when set (T004); failing when missing is T005 | `group` |
| `plan_change.global_without_owner` | boolean | yes | Whether the global plan may change without the owner, still after the plan-change reviewer agrees (D7). Loaded and validated when set (T004); failing when missing is T005 | `false` |
| `generation.targets` | removed | — | Removed (owner 2026-10-27, TRD D24; T006): the runtimes to build for come from `runtimes.*`; a file carrying `generation.*` fails as an unknown key | — |

`plan_change.*`, `developers` and `repo.integration_branch` follow the names in TRD [Configuration shapes](TRD.md#93-configuration-shapes).

### 3.3 Slot adapter files

Free Markdown, one per slot, in the repository (suggested folder `.gbai/adapters/`). No YAML schema: the contract coverage check (TRD [Testing strategy](TRD.md#11-testing-strategy-proposed-auto-awaiting-owner)) verifies each adapter explains every operation of its contract.

### 3.4 Generated agent files

Output only. Never hand-edited; regenerated on every config change, and the generator's no-diff check guards drift. Their format is the runtime's own (Claude Code agent frontmatter, Pi agent Markdown), not a gbai schema. Locations: `.claude/agents/` for Claude Code and `.pi/agents/` for Pi, both in the project. Approved (owner 2026-10-26): Pi core has no subagents, so Pi agent discovery comes from a subagent package. With the recommended gentle-pi, agents are discovered in `<cwd>/.pi/agents/`, `<cwd>/.pi/subagents/`, `~/.pi/agent/agents/` and `~/.pi/agent/subagents/` (project beats global, `subagents/` beats `agents/`), plus `subagents.json` at global and project level (installed gentle-pi 4.3.0 docs, `docs/gentle-shell.md` lines 215–219 and `README.md` lines 480–522). The project path works in both plain Pi and `gentle-shell`. gbai writes only its agent files and never edits the package's own configuration such as `subagents.json`. With another subagent package, T044 confirms its project path; `gbai generate` refuses Pi targets when no subagent package is present.

## 4. Examples

### 4.1 Global user config

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/testeos-z/gbai-team/main/schemas/gbai.user.schema.json
version: 1
alias: Leo
interaction_level: balanced
tone: direct, brief
language: es
```

### 4.2 Project `gbai.yaml`

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/testeos-z/gbai-team/main/schemas/gbai.project.schema.json
version: 1
mode: manual

repo:
  name: testeos-z/gbai-team
  integration_branch: dev

runtimes:
  coder: claude
  assistant: claude
  reviewer: pi

slots:
  tasks:
    adapter: .gbai/adapters/tasks-gh.md
  memory:
    adapter: .gbai/adapters/memory-engram.md
  notifications:
    adapter: .gbai/adapters/notify-task-comment.md
  direct_sessions:
    adapter: .gbai/adapters/sessions-orca.md
  methodology:
    adapter: .gbai/adapters/methodology-tdd.md
  repository_host:
    adapter: .gbai/adapters/host-github.md
  reviewer_placement:
    adapter: .gbai/adapters/reviewer-local.md

reviewer:
  placement: local
  signing:
    marker: gbai-review

review:
  checks:
    - go test ./...
    - go vet ./...
  auto_merge: false
  human_only_paths: [.github/, AGENTS.md, CLAUDE.md]
  pi_credential_env: ANTHROPIC_API_KEY

developers:
  "leo@example.com": Leo

plan_change:
  scope: group
  global_without_owner: false

parallelism: 3
```

No cloud keys appear here: a team that runs agents on a server adds `slots.agent_host.adapter` in a separate session.

## 5. JSON Schema approach

All of this section is proposed (not implemented).

- **Location.** `schemas/gbai.user.schema.json` and `schemas/gbai.project.schema.json` at the repository root, JSON Schema draft 2020-12. The CLI embeds the same files so validation works offline; one source, no copies by hand.
- **Strictness.** `additionalProperties: false` at every level, matching today's strict known-fields decoding. Patterns mirror the Go validators (repo, branch, login, path, env var name).
- **Versioning.** Each file carries `version`. A breaking change bumps it and ships a new schema file (`...v2.schema.json`); the loader rejects an unknown version with a message naming the migration. Additive, optional keys do not bump it.
- **Validation points.**
  - **Load:** every gbai command (all are agent-compiler commands, TRD D18) validates each layer against its schema, then the merged result.
  - **`gbai generate`:** validates both files and adapter coverage before writing anything; this replaces the removed `gbai doctor` command (approved owner 2026-10-26, TRD D18).
  - **Read-only `gbai config`:** removed (Approved (owner 2026-10-26), open question 11); `gbai generate` reports each schema error with its key path and file.
  - **Setup and work-time edits:** agents write the YAML directly; validation runs at the next `gbai generate` (Approved (owner 2026-10-26), TRD D18). `gbai config set`, when the owner uses it, validates the one key before writing.
  - **CI:** the reviewer validates `gbai.yaml` as part of the generator no-diff check.
- **Editor integration.** gbai writes a first-line `# yaml-language-server: $schema=<url>` comment in both files, so YAML language servers give autocomplete and inline errors without extra setup. The URL points at the published schema for the file's `version`.

The Go structs stay the source of runtime truth; a unit test asserts the schema and the Go validation accept and reject the same fixtures.

## 6. Migration

Today there is one cloud-shaped `gbai.yaml` whose validation requires the Railway ids. Steps, forward-only:

| order | migration | purpose |
|---|---|---|
| 1 | Add `version` and the personal / project key classes to the loader | Enable personal-key precedence without breaking existing files |
| 2 | Make `server.*`, `users.*`, `bot.*`, `sessions.*.prompt_file` optional and stop core validation from requiring them | Local use works with no cloud keys |
| 3 | Add the new project keys (`mode`, `slots` including `slots.methodology` and `slots.agent_host`, `reviewer.{placement, signing}`, `developers`, `plan_change.*`; `generation.targets`, added here, was removed by T006 per TRD D24) | Record setup answers |
| 4 | Move cloud keys, `sessions.local_orca` and the system layer `/etc/gbai/gbai.yaml` into the `slots.agent_host` adapter; drop the system layer from the core loader; the assistant runs that session for existing server users | Clean core config |
| 5 | Replace silent code defaults with suggestions the assistant asks about (R12); missing required keys fail with "run setup" instead of being filled | No silent defaults |
| 6 | Publish the JSON Schemas and write the `$schema` comment | Editor validation |

**Backward compatibility.** A file without `version` is read as version 0, the current shape. During the transition the loader accepts the legacy cloud keys and warns that they belong in the `slots.agent_host` adapter; `gbai generate` warns with the suggested split. Once that adapter exists, version 1 rejects them like any unknown key. The assistant performs the rewrite with the owner's approval; gbai never rewrites a config file silently. Previous versions remain in git history, so a migration is reversible by checkout.

## 7. Data

- **Seed and reference data.** None. Setup starts from the owner's answers; documented values are suggestions.
- **PII.** The developer catalog stores git emails and aliases in a committed file, so it is visible to everyone with repository read access. The global user file holds an alias and preferences, readable only by that OS user. Preference observations (`user/preferences`, TRD D22) live in the memory provider, follow its access rules and never hold secrets or sensitive personal data.
- **Secrets.** Never stored in any gbai file or generated agent file (R21). Config holds only the name of an env var (`review.pi_credential_env`), never its value.
- **Retention.** Config and adapters live as long as the repository; history is git history. Reviewer state is disposable. Decisions and tasks follow the retention of the external memory provider and tracker.

## 8. Open questions

Questions 1–11 are resolved (owner, 2026-10-26).

1. **Key spelling for plan changes and the catalog.** Resolved (owner, 2026-10-26): `plan_change.{scope, global_without_owner}` and `developers`, as in TRD 9.3.
2. **Methodology: key or slot.** Resolved (owner, 2026-10-26): methodology is a slot. `slots.methodology.adapter` replaces the former methodology key under `coder` everywhere and is linked as `gbai-methodology` (PRD R8 updated).
3. **Engine key under `reviewer` versus `runtimes.reviewer`.** Resolved (owner, 2026-10-26): the engine key under `reviewer` is removed; the reviewer runtime is `runtimes.reviewer` only, alongside the other roles' runtimes.
4. **`interaction_level` values.** Resolved (owner, 2026-10-26): one of `low`, `balanced`, `high`; the assistant adjusts it from what it learns about the user.
5. **`generation.targets` "current only".** Resolved (owner, 2026-10-26): store concrete runtimes, for example `[claude]` or `[claude, pi]`, so changes show in diffs.
6. **Cloud slot name and the system layer.** Resolved (owner, 2026-10-26): the cloud slot key is `slots.agent_host`. The system layer `/etc/gbai/gbai.yaml` and `sessions.local_orca` move into that adapter; core config no longer includes them.
7. **`integration_branch` placement.** Resolved (owner, 2026-10-26): it stays at `repo.integration_branch` (TRD 9.3 updated).
8. **Other slots from PRD 10.1.** Resolved (owner, 2026-10-26): reviewer signing or marker lives under `reviewer.signing`; coder-side review (GGA, RDD) is configured inside the methodology slot's adapter, not as its own key.
9. **Pi agent file location.** Resolved (owner, 2026-10-26): generated Pi agents live in the project at `.pi/agents/`, and Claude Code agents in `.claude/agents/`. Verified (Approved (owner 2026-10-26)) against the installed gentle-pi 4.3.0 docs, `docs/gentle-shell.md` lines 215–219 and `README.md` lines 480–522: gentle-pi discovers `<cwd>/.pi/agents/`, `<cwd>/.pi/subagents/`, `~/.pi/agent/agents/` and `~/.pi/agent/subagents/`; Pi core has no subagents, so a subagent package is a prerequisite for Pi targets (TRD 9.6, D21, question 11 resolved).
10. **Reviewer scripts location.** Resolved (owner, 2026-10-26): not in the repository; the scripts live in the `scripts/` folder of the `gbai-reviewer` skill, installed at user level so PR code cannot modify them (TRD D20). The proposed `.gbai/reviewer/` is dropped.
11. **Read-only `gbai config`.** Resolved (owner, 2026-10-26): removed; validation feedback comes only from `gbai generate`, and the assistant reads the files (TRD D18).
