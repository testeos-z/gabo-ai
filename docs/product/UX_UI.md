# UX and UI

| field | value |
|---|---|
| status | Draft |
| owner | gbai-frontend |
| version | 0.6 |
| date | 2026-10-27 |
| linked docs | [PRD](PRD.md), [TRD](TRD.md), [DB_SCHEMA](DB_SCHEMA.md), [APP_FLOW](APP_FLOW.md) |

gbai is a CLI plus an agent framework with no graphical UI, so `DESIGN.md`, CSS tokens and Storybook do not apply. This document adapts the template to what gbai actually presents: the assistant's conversation, the CLI command surface (three commands: `gbai init`, `gbai check` and `gbai version`, owner 2026-10-27, PRD R31; earlier an agent compiler used only for the initial configuration, TRD D18) and its messages. Approved (owner 2026-10-26): the proposed terminal UI (TUI) is withdrawn. Superseded (owner 2026-10-27): setup starts from a prompt the user pastes into Claude Code or Pi (TRD D21); since 2026-10-27 setup has two entries (section 1.8): `gbai init` followed by the user's own agent, or the published setup prompt pasted into the agent, which installs the CLI and continues; the prompt is the same literal text embedded in the CLI (PRD R36). Every gap this document had to fill is marked **Proposed (auto)**.

## Auto-mode proposals

Written by gbai-frontend in auto mode (R6, R19). Every item is marked **Proposed (auto)** where it appears, with a one-line reason. The owner approves or rejects them in one pass.

| # | section | proposal | reason |
|---|---|---|---|
| U1 | Scope | Replace tokens / components / Storybook with three sections: conversation UX, CLI surface, messages (the TUI section is withdrawn, owner 2026-10-26). | gbai has no GUI; these are its real interfaces. |
| U2 | 1.3 | Meaning of `interaction_level`: `low` asks only what blocks a task or needs the owner by rule; `balanced` also confirms suggestions before applying them; `high` also explains options and checks in at milestones. | DB_SCHEMA fixes the three values but not what they change. |
| U3 | 1.5 | Every question uses one shape: context line, the question, suggested options (none preselected), and what stays blocked while it waits. | One question at a time (R20) and no silent defaults (R12) need a predictable format. |
| U4 | 1.5 | Wave reports reach the owner as a short summary (counts, blocked tasks, questions, decisions, leftovers) with the full report linked. | Protects the owner's energy (goal 1) while keeping R29's content one step away. |
| U5 | 1.6 | The comfort question is one question with three answers (fewer / same / more interactions) plus an optional free-text style note. | R17 asks for a brief question; fixed answers map directly onto `interaction_level`. |
| U6 | 1.7 | If `language` is unset, the assistant asks once during setup instead of guessing. | R12 forbids silent defaults; language is a personal key. |
| U7 | 2 | Amended (owner 2026-10-27, PRD R31–R33): the CLI surface is `gabo init` (copies the skill catalog, writes a version stamp, installs the user-level `gabo-reviewer` skill, prints the embedded setup prompt; updated issue #74, superseded: the neutral entry file), `gabo check` (read-only, engine-neutral validation) and `gabo version` only. `gbai generate [--check]`, `gbai config set`, the Go generation targets and the per-runtime installer table are removed; the runtime's own agent builds the config and the agents from skills. Still no wave, plan, review, status or other work-time command. Superseded (owner 2026-10-27): Approved (owner 2026-10-26), replacing the earlier proposal: the CLI surface is `gbai init` (the first generate), `gbai generate [--check]`, `gbai config set` (owner convenience at configuration time only) and `gbai version` only; the read-only `gbai config` is removed (owner 2026-10-26, section 5 question 2). No wave, plan, review, status or other work-time command: agents do that work with their own tools guided by skills and adapters. | Owner decision 2026-10-27 (issue #51): technical simplicity, a more universal tool with less configuration; earlier TRD D18, D19, D20. |
| U8 | — | Withdrawn (owner 2026-10-26): no wave command; the cycle reads tasks and their blocked-by links through the `tasks` adapter and takes the unblocked ones. | No glue CLI for work agents already do. |
| U9 | 3 | Withdrawn (owner 2026-10-26): Build the TUI with Bubble Tea, Lip Gloss and Bubbles, following gentle-ai's screen router, keybindings and style roles. | The TUI is removed; setup is done by the user's agent from a pasted prompt (section 1.8, TRD D21). |
| U10 | 3.2 | Withdrawn (owner 2026-10-26): `gbai` with no arguments on a TTY opens the TUI home; without a TTY it exits non-zero with a hint to use subcommands. | The TUI is removed; setup is done by the user's agent from a pasted prompt (section 1.8, TRD D21). |
| U11 | 3.3 | Withdrawn (owner 2026-10-26): The TUI never preselects a slot value; detected tools are shown as suggestions and every write needs an explicit confirm on a Review screen. | The TUI is removed; setup is done by the user's agent from a pasted prompt (section 1.8, TRD D21). |
| U12 | 3.3 | Withdrawn (owner 2026-10-26): The TUI cannot co-write adapters; its "write with the assistant" option hands off to the assistant conversation. | The TUI is removed; setup is done by the user's agent from a pasted prompt (section 1.8, TRD D21). |
| U13 | 3.3 | Withdrawn (owner 2026-10-26): no status view; no Go status command exists (TRD D18) and the last wave report lives in the `memory` slot. The owner asks the assistant for the state. | The TUI is removed altogether (owner 2026-10-26). |
| U14 | 3.5 | Withdrawn (owner 2026-10-26): gbai defines its own palette behind gentle-ai's style roles; honors `NO_COLOR` and adds `GBAI_NO_ANIMATION=1`. | The TUI is removed; setup is done by the user's agent from a pasted prompt (section 1.8, TRD D21). |
| U15 | 4 | Every error or refusal uses three parts: what happened, why, what to do next; one line first, details after. | Gives the owner one next action per failure, matching existing CLI messages ("run setup"). |
| U16 | 1.8 | Amended (owner 2026-10-27, PRD R32, R34): setup starts with `gabo init`, then the user asks their own agent to configure gabo-ai; the agent follows the setup prompt `gabo init` prints (updated issue #74; superseded: the entry file) and the setup, config-writer and runtime skills, then runs `gabo check`. Second entry (owner 2026-10-27, PRD R36): the published setup prompt, pasted into the agent, which installs the CLI and continues; one text, embedded in the CLI and published in the documentation. Superseded (owner 2026-10-27): Approved (owner 2026-10-26): setup starts from a prompt with a link, copied from the gbai documentation and pasted into Claude Code or Pi; that agent installs gbai, asks the setup questions, writes the config and runs `gbai generate`. | Replaces the TUI; the guided setup is agent work anyway (adapter co-writing, subagent verification). |
| U17 | 1.3 | Approved (owner 2026-10-26): preferences have a stable layer (global user file) and a learning layer (memory slot, topic `user/preferences`); an observation repeated about 3 times is proposed for the stable layer. | PRD 10.1 Personalization, TRD D22. |

## 1. Conversation UX of the assistant

### 1.1 Principles

1. **The owner's energy comes first** (goal 1). The assistant brings decisions, validations and unresolved questions, never coordination work.
2. **No silent defaults** (R12). The assistant suggests; the owner decides.
3. **One question at a time** (R20). Never a questionnaire in one message.
4. **Every question closes with an answer and a resolver** (R5).
5. **Comfort, not minimum** (R17). The goal is the number of interactions the owner needs to feel comfortable.

### 1.2 Addressing the owner

- The assistant addresses only the session owner, by the alias from the developer catalog (R16); a personal `alias` in the global file wins (D11).
- No alias: it asks once, "Should I call you by your git name X or another name?", stores it and never asks again.
- It may notify the owner through the notifications slot (chat tool or a request on the task).

### 1.3 Interaction level

Stored per user as `interaction_level` (`low`, `balanced`, `high`) with a free-text `tone` (R17). Proposed (auto) U2:

| level | the assistant asks or reports |
|---|---|
| `low` | Only what blocks a task or needs the owner by rule (manual-mode questions, plan changes waiting for the owner, BLOCKED tasks) |
| `balanced` | The above, plus confirmation before applying a suggestion |
| `high` | The above, plus the options' tradeoffs and a check-in at wave start and close |

The assistant adjusts the level from the comfort answer (1.6) and from what it learns, and says when it changed it.

Approved (owner 2026-10-26), U17: what the assistant learns is saved as observations in the memory slot under the topic `user/preferences` (for example "prefers short answers") and read at session start. An observation repeated about 3 times is proposed for the stable layer (`interaction_level`, `tone`, `language`, alias in the global file): in manual mode the assistant asks the owner; in auto mode it edits the global file and records the decision in memory. Personal observations use the global memory scope, project ones the project scope; secrets and sensitive data are never stored.

### 1.4 Manual and auto behavior

| | manual | auto |
|---|---|---|
| Methodology and development questions | Reach the owner through the assistant (R6) | The assistant decides and records decision, reason, wave and task in memory (R6, R30) |
| Prompts during a run | As the interaction level allows | None; only the final answer (R15) |
| Critical questions | Plan-change proposal debated with the owner | Plan-change proposal per the [approval scale](APP_FLOW.md#plan-change-approval-scale); never silent (R19) |
| Merges and deletions | Owner-driven | Integration branch only, never main; only resources the workflow created (10.2) |

On request, in auto mode, the assistant lists every recorded decision with its reason, wave and task (R30).

### 1.5 How things are presented

Proposed (auto) U3 and U4. Shapes, not fixed wording:

- **Question.** `Context: <one line>` · the question · `Suggestions: a) … b) …` (none preselected) · `Waiting: <task> is blocked until you answer` (or "nothing is blocked").
- **Escalation** (BLOCKED task, 4 review rounds exhausted, verify failure). `<task>: BLOCKED — <reason>` · what was tried · the one decision needed.
- **Plan-change proposal.** Scope (task / group / global) · affected tasks · proposed change · rationale · debater and outcome · link to its issue and PR · what applies automatically and what waits for the owner.
- **Wave report summary.** `Wave <n>: <done> done, <blocked> blocked` · questions (answered by whom) · decisions taken in auto mode · cleanup leftovers · interaction count · link to the full report.

### 1.6 Comfort question at wave close

Proposed (auto) U5. One brief question after the summary: "How did the number of questions feel this wave: fewer, about right, or more?" plus an optional "Anything about how I talk to you?". The answer updates `interaction_level` and `tone` for the next wave (R17); the assistant edits the global user file directly and the change is validated at the next `gbai check` (owner 2026-10-26, TRD D22; updated owner 2026-10-27, earlier `gbai generate`).

### 1.7 Language

Conversation in the owner's `language` (BCP 47, global file); all generated artifacts (docs, tasks, PRs, comments, agent files) in English. Proposed (auto) U6: if `language` is unset, the assistant asks once during setup.

### 1.8 Setup entry

Amended (owner 2026-10-27), U16, PRD R32, R34, R36. There is still no setup screen. Setup has two entries that join at step 3:

- **Entry (a), terminal.** Steps 1 and 2 below.
- **Entry (b), pasted prompt.** The user pastes the published setup prompt into their agent; the agent installs the CLI, runs `gabo init` (step 1) and continues at step 3. The published prompt is the same literal text embedded in the CLI: one source, shipped in the binary and published in the documentation.

1. The user installs the gabo CLI and runs `gabo init` in the project. It copies the skill catalog, writes the version stamp, installs the user-level `gabo-reviewer` skill and prints the embedded setup prompt as the next step (updated owner 2026-10-27, issue #74; earlier it wrote an entry file). In an existing project it never touches the config: it refreshes only the catalog subtree when the stamp is older than the CLI version or is missing, updates the CLI itself when the stamp is newer and a developer runs it interactively (PRD O12), installs the reviewer skill whenever it is missing or differs from the embedded one, and otherwise changes nothing and exits 0. A refresh replaces the copied catalog folder wholesale: skills removed in the new version are deleted and local edits to copied skills are overwritten, because customization belongs in adapters, not in the catalog.
2. The user opens their agent (Claude Code, Pi or another engine with a runtime skill) and asks it to configure gabo-ai.
3. The agent follows the setup prompt. The setup skill runs the setup question burst one question at a time (identity, slots and adapters, runtimes, `mode`, `plan_change.*`, `language`), with suggestions and no preselected value (R12).
4. The config-writer skill writes the global user file and `gabo.yaml`.
5. The runtime skill builds the agents of the neutral agent spec in the engine's format from fixed templates; for an engine without a runtime skill, the runtime builder writes one first (PRD R35).
6. The agent runs `gabo check`, fixes what fails and reruns it until it passes; failures use the section 4 shape.
7. The agent verifies each slot through a subagent (APP_FLOW J1).

PRD open question 12 (pending owner decision, not applied): the CLI may be renamed "gabo"; its scope (binary only, or also skill prefixes and the config file name) is not confirmed yet.

**Superseded (owner 2026-10-27), setup entry of 2026-10-26.** Approved (owner 2026-10-26), U16. There is no setup screen. The gbai documentation publishes one setup prompt with a link. The user pastes it into Claude Code or Pi, and that agent:

1. installs gbai;
2. runs the setup question burst one question at a time (identity, slots and adapters, runtimes and `generation.targets`, `mode`, `plan_change.*`, `language`), with suggestions and no preselected value (R12);
3. writes the global user file and `gbai.yaml` by editing them directly;
4. runs `gbai generate`, which validates the config and installs the per-runtime artifacts (TRD D21), and reports errors in the section 4 shape;
5. verifies each slot through a subagent (APP_FLOW J1).

## 2. CLI command surface

### 2.1 Commands

Names (owner 2026-10-27, PRD O9): the product is gabo-ai and the CLI is invoked as `gabo`. Skill prefix `gabo-*`, project config file `gabo.yaml` and the global user file under a `gabo` directory are approved (confirmed by the owner 2026-10-27). The existing `gbai-*` skill names, `gbai.yaml`, the `gbai` command and the `gbai` global directory stay in every other section and in the repository until the rename task lands.

Amended (owner 2026-10-27, U7, PRD R31–R33). The CLI has three commands. It writes no config and builds no agent; the user's agent does that from skills (PRD R34). Nothing in Go runs during the work itself.

| command | purpose | source |
|---|---|---|
| `gabo init` | Fresh project: copies the embedded skill catalog into a neutral project location, writes the version stamp, installs the user-level `gabo-reviewer` skill and prints the embedded setup prompt. Existing project: never touches the config; refreshes only the catalog subtree when the stamp is older than the CLI version or is missing; when the stamp is newer and a developer runs it interactively, updates the CLI itself to exactly that version, verified as an authentic release (signature or provenance, details in the TRD), and continues, or stops with the exact upgrade command if it cannot (PRD O12), never refreshing the catalog backwards; a non-interactive run never self-updates: with a newer stamp it changes nothing, prints the upgrade command and exits 1; CI and the reviewer never self-update; with a catalog and stamp but no config, refreshes an older catalog, prints the setup prompt again and exits 0; independent of the stamp, installs the reviewer skill when it is missing or differs from the embedded one; otherwise changes nothing and exits 0. A refresh replaces the copied catalog folder wholesale: skills removed in the new version are deleted and local edits to copied skills are overwritten, because customization belongs in adapters, not in the catalog. The same version always produces the same files (updated issue #74; the entry file is superseded) | PRD R32 |
| `gabo check` | Read-only, engine-neutral validation of five things: the global user file and `gabo.yaml` against their JSON Schemas, every agent of the neutral agent spec present, slot coverage, the catalog version stamp matching the CLI version (a missing stamp is a mismatch), and every agent's version line present and matching the catalog stamp (updated issue #74, PRD O11; no template hashes). Writes nothing; non-zero exit on any failure. Never self-updates: when the stamp is newer than the CLI it fails with the exact upgrade command (PRD O12). CI and the reviewer run the version pinned in the base branch's stamp (the CLI they have when the base has no stamp); on an upgrade PR (the stamp changes) they skip only the two version-match checks and report `upgrade PR: version checks skipped, human merge required`. Technical details: TRD (PR #54). Usable by the user, agents, the reviewer and CI | PRD R33 |
| `gabo version` | Print the version | — |

The CLI also carries the setup prompt as embedded text, identical to the one published in the documentation (PRD R36); this adds no command. Command flags and how the embedded text is exposed are TRD details.

**Superseded (owner 2026-10-27), command table of 2026-10-26.** Approved (owner 2026-10-26, TRD D18). The CLI is an agent compiler for the initial configuration; nothing in Go runs during the work itself.

| command | purpose | source |
|---|---|---|
| `gbai init [--runtime claude\|pi\|both] [--scope user\|project] [--dry-run] [--force]` | The first generate: validates the config and installs the per-runtime artifacts from the installer table (skills, including the user-level `gbai-reviewer` skill with its scripts, adapters, agents and tool registrations); registers no gbai MCP server | D18, D21 |
| `gbai generate [--check]` | Agent compiler: validates both config files against their JSON Schemas and adapter coverage, writes runtime-native agent files for `generation.targets` and installs the per-runtime artifacts from the installer table (embedded skills, the user-level `gbai-reviewer` skill, the user's adapters, MCP servers, plugins or packages); `--check` fails on any diff | D3, D18, D20, D21, TRD 9.4 |
| `gbai config set <key> <value> [--global]` | Owner convenience at configuration time: write one key to the right file after schema validation; follow with `gbai generate` to recompile. Agents edit the YAML directly instead (owner 2026-10-26) | R12, D18 |
| `gbai version` | Print the version | — |

### 2.2 Removed commands

Approved (owner 2026-10-26, TRD D18–D20; extended owner 2026-10-27). Each command below is removed from Go; its job moves to an agent using its own tools.

| removed command | where its job goes |
|---|---|
| `gbai generate [--check]` (removed 2026-10-27) | Agent building: the runtime skill of the engine in use, from fixed templates (PRD R34). Validation and drift detection: `gbai check` (PRD R33) |
| `gbai config set` (removed 2026-10-27) | The config-writer skill, or the owner editing the YAML; `gbai check` validates |
| Go generation targets and per-runtime installer table (removed 2026-10-27) | Runtime skills for Claude Code and Pi; the runtime builder skill for other engines (PRD R34, R35) |
| `gbai status` (removed) | The assistant reads the tracker, the PRs and the `memory` slot |
| `gbai audit` (removed) | The assistant checks merges with gh and git per its skill |
| `gbai sessions start\|stop` (removed) | The `direct_sessions` adapter (Orca CLI) |
| `gbai provision`, `gbai secrets set` (removed) | The `agent_host` adapter (for example Railway MCP), configured in a separate assistant session |
| `gbai repo setup` (removed) | The `repository_host` adapter (gh) |
| `gbai worktree close` (removed) | The cycle skill, with git worktree and branch commands |
| `gbai mcp` (removed: no gbai MCP server) | Skills and adapters |
| `gbai doctor` (removed) | Config validation folds into `gbai check` (updated 2026-10-27; earlier `gbai generate`) |
| `gbai config [--json]` (removed 2026-10-26) | The assistant reads the config files; `gbai check` reports schema and agent errors (updated 2026-10-27; earlier `gbai generate`) |
| `gbai plan new\|status\|story\|check` (removed) | Templates and the traceability check live in the directiva skills (D19) |
| `gbai review request\|run\|ci\|notify` (removed) | Scripts of the user-level `gbai-reviewer` skill, installed by `gbai init` (updated 2026-10-27; earlier the compiler); the reviewer agent invokes them (D20) |

Auto-mode decisions (R30) are queried through the assistant and the memory slot, not a compiled command, because memory access is tool-specific (D2). The same holds for wave work: the next wave, the resource ledger, cleanup and resume are done by the cycle through its skill (TRD D12, D15, D17).

## 3. Terminal UI (TUI)

Withdrawn (owner 2026-10-26). gbai has no terminal UI: no Bubble Tea dependency, no `gbai` home screen and no setup wizard. The former sections 3.1–3.5 (gentle-ai TUI findings, structure, screens, keybindings, theming) and proposals U9–U14 no longer apply. Setup is done by the user's own agent from a pasted prompt (section 1.8); the former milestone M4 and tasks T035–T037 are removed from the implementation plan.

## 4. Messages and errors

Proposed (auto) U15. Tone: neutral, direct, no blame, no exclamation marks; English for CLI output, the owner's language in conversation.

Approved (owner 2026-10-26): no work-time message comes from Go (TRD D18). Messages come from the CLI (`gbai init` and `gbai check` output; updated owner 2026-10-27, earlier config and generate errors), the scripts of the user-level `gbai-reviewer` skill (merge gate, TRD D20) and agents in the conversation and the wave report (cleanup, blocked tasks, setup verify, traceability), all in the same shape.

Format: `✗ <what happened>` on one line, then `why:` and `next:` lines, then optional details. Non-zero exit for every CLI or script failure.

| case | example |
|---|---|
| Refusal (auto-mode limit, R23, 10.2; coder at task close or cycle agent at the wave-close sweep) | `✗ refused to delete worktree feat-x` · `why: it was not created by this workflow` · `next: remove it yourself or add it to worktree.keep` |
| Refusal (merge; reviewer script merge gate) | `✗ refused to merge into main` · `why: auto mode merges only into the integration branch (dev)` |
| Blocked task (cycle agent, wave report) | `✗ task #42 BLOCKED` · `why: 4 review rounds without approval` · `next: owner decision needed` (also in the wave report) |
| Verification failure (setup; assistant after the subagent smoke test) | `✗ tasks slot not verified` · `why: "read one task" failed: gh: not authenticated` · `next: run gh auth login, then retry verify` |
| Coverage gap (`gabo check`, slot coverage; also raised by the setup agent; earlier `gbai generate`) | `✗ adapter tasks-gh.md does not cover: link PR, verify by re-read` · `next: add those operations or write them with the assistant` |
| Traceability gap (reported by the directiva agent, D19) | `✗ planning step blocked` · `why: R7 has no task; US-012 has no requirement` · `next: close the gaps; the check reruns before the next step` |
| Missing config (`gabo check`) | `✗ no gabo.yaml found` · `next: run gabo init, then ask your agent to configure gabo-ai` (updated owner 2026-10-27; earlier: paste the setup prompt) |
| Catalog refreshed (`gabo init`, exit 0) | `catalog refreshed from <old> to <new>` (plus `reviewer skill installed` when it was missing or differed) · `next: ask your agent to rebuild the agents, then run gabo check` |
| Already up to date (`gabo init`, exit 0) | `catalog already at <version>; nothing changed` |
| Version stamp mismatch, older stamp (`gabo check`) | `✗ catalog version <stamp> does not match gabo <version>` · `next: run gabo init, then ask your agent to rebuild` |
| CLI self-updated (`gabo init`, interactive only) | `gabo updated to <v>` (release authenticity verified), then the command continues (PRD O12) |
| Cannot self-update (`gabo init`, non-zero exit) | `✗ cannot self-update to <v>` · `why: <installed by a package manager, or no network>` · `next: run <exact upgrade command>` |
| Stamp newer than the CLI (`gabo check`, non-zero exit; never self-updates) | `✗ catalog version <w> is newer than gabo <v>` · `next: run <exact upgrade command>` |
| No stamp (`gabo check`) | `✗ no stamp: run gabo init` |
| Stamp newer, non-interactive (`gabo init`, exit 1) | `✗ catalog version <w> is newer than gabo <v>; nothing changed` · `next: run <exact upgrade command>` |
| Upgrade PR (CI and reviewer, `gabo check`) | `upgrade PR: version checks skipped, human merge required` (all other checks still run) |
| Missing agent (`gabo check`) | `✗ agent gabo-reviewer is missing` · `next: ask your agent to build it with the runtime skill, then rerun gabo check` |
| Agent not rebuilt (`gabo check`) | `✗ agent <id> built with <v>, catalog is <w>: ask your agent to rebuild` · `next: ask your agent to rebuild the agents, then rerun gabo check` (also when the agent's version line is missing; PRD O11) |
| Engine not supported (runtime builder) | `✗ cannot add a runtime for <engine>` · `why: it has no subagent and skill support, or an equivalent` · `next: use Claude Code, Pi or another engine with subagents and skills` |
| Invalid config (agent edited the YAML during the work) | `✗ gbai.yaml: plan_change.scope: "teams" is not one of task, group, global` · `next: fix the key, then rerun gbai check` |

Messages never include secret values (R21); they name the env var or adapter instead.

## 5. Open questions

Questions 1–3 resolved (owner 2026-10-26). The 2026-10-27 decision leaves two product questions open in the PRD (section 13, questions 10 and 11): the role of the published setup prompt, and how a project with a config refreshes the copied skill catalog after an upgrade.

1. **TUI scope.** Resolved: the TUI is removed (section 3); setup starts from a pasted prompt (section 1.8). Milestone M4 and tasks T035–T037 are removed.
2. **Read-only `gbai config [--json]`.** Resolved: removed; the assistant reads the files and `gbai generate` reports schema and coverage errors.
3. **Reviewer scripts location.** Resolved: not in the repository; the scripts live in the `scripts/` folder of the `gbai-reviewer` skill, installed at user level so PR code cannot modify them (TRD D20).
