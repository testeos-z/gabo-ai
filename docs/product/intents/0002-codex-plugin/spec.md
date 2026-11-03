# Spec 0002 Codex plugin

| field | value |
|---|---|
| intent | [0002-codex-plugin](intent.md) |
| status | Draft |
| owner | gabo-pm |
| contributors | — |
| date | 2026-11-03 |

## Evidence

Every Codex fact below was checked on 2026-11-03 against the owner's local install, `codex-cli 0.163.0`, and the official documentation. A fact marked **unverified** is checked by S10 before the plan relies on it.

| fact | source |
|---|---|
| A plugin is a folder with `.codex-plugin/plugin.json`. Documented keys: `name`, `version`, `description`, `skills` (for example `"./skills/"`), `hooks`, `apps`, `interface`. MCP servers come from `.mcp.json` next to it. | https://developers.openai.com/codex/plugins/build; installed plugins `engram@0.1.1`, `warp@0.4.3`, `supabase@0.1.16` under `~/.codex/plugins/cache/` |
| An enabled plugin loads `hooks/hooks.json` from its root by default; a manifest `hooks` entry overrides the path, which must stay inside the plugin root. | https://learn.chatgpt.com/docs/hooks |
| A marketplace is `<repo>/.agents/plugins/marketplace.json`; Codex also reads the legacy `<repo>/.claude-plugin/marketplace.json`. | https://developers.openai.com/codex/plugins/build; `codex plugin marketplace list` shows the git marketplaces `codex-warp` (`.agents/plugins/`) and `voicemode`, `claude-plugins-official` (`.claude-plugin/`) |
| The CLI installs and refreshes plugins: `codex plugin marketplace add <path or git URL>`, `codex plugin add PLUGIN@MARKETPLACE`, `codex plugin marketplace upgrade`, `codex plugin remove`. Installed plugins are cached in `~/.codex/plugins/cache/<marketplace>/<plugin>/<version>/`. | `codex plugin --help`, `codex plugin marketplace --help` (0.163.0); local cache layout |
| Custom agents are TOML files in `.codex/agents/` (project) or `~/.codex/agents/` (user), one agent per file. `name`, `description` and `developer_instructions` are required; `name` is the identity, not the file name. Optional: `model`, `model_reasoning_effort`, `model_reasoning_summary`, `model_verbosity`, `features.*` capability switches, `skills.config`. `sandbox_mode`, `approval_policy`, `model_provider` and `mcp_servers` do not override the parent's configuration from an agent file. | https://learn.chatgpt.com/docs/agent-configuration/subagents; 21 TOML agents in the owner's `~/.codex/agents/` |
| A custom agent has no per-agent tool allowlist and no skill preload list; skills are inherited from the parent session and loaded by name on demand. | same page |
| Whether a plugin can ship custom agents is not documented. The reference `ghiaog123/ai-native-sdlc` declares `"subagents": "./agents/"` with `.toml` agents; no official page confirms the key. **Unverified.** | https://developers.openai.com/codex/plugins/build (silent); `gh api repos/ghiaog123/ai-native-sdlc` |
| Skills are discovered in `.agents/skills/` of every directory from the working directory up to the repository root, in `$HOME/.agents/skills`, in `/etc/codex/skills` and in bundled system skills; symlinked skill folders are followed. `agents/openai.yaml` inside a skill folder is optional UI metadata, not an agent. | https://learn.chatgpt.com/docs/build-skills; `bashebr/ai-native-sdlc` `skills/ai-native-sdlc/agents/openai.yaml` |
| Hook events include `SessionStart`, `PreToolUse`, `PostToolUse`, `Stop`, `SubagentStart`, `SubagentStop`. File edits match `apply_patch`, also as `Edit` or `Write`; the edit's input is the patch text in `tool_input.command` (no file-path field). `PostToolUse` can return `hookSpecificOutput.additionalContext` or `decision: "block"` with a `reason`. | https://learn.chatgpt.com/docs/hooks |
| Plugin hook commands get `PLUGIN_ROOT` and `PLUGIN_DATA`; Codex also sets `CLAUDE_PLUGIN_ROOT` and `CLAUDE_PLUGIN_DATA` for compatibility. Non-managed hooks run only after the user trusts them (`/hooks`), and a changed hook is skipped until trusted again. | https://learn.chatgpt.com/docs/hooks; `[hooks.state.*]` entries with `trusted_hash` in the owner's `~/.codex/config.toml` |
| gabo today: no `gabo-runtime-codex` skill; `runtimes.*` accepts only `claude` or `pi` (`apps/cli/internal/config/config.go` line 192, `gabo.project.schema.json` line 252); `gabo check` finds agents as `<id>.<any extension>` in the runtime skill's `metadata.agent_dir` and reads `gabo-version:` anywhere in the file (`check/usecase.go` lines 333, 340). | repository at `5ced3b7` |

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | The plugin directory shall carry a Codex manifest `.codex-plugin/plugin.json` that names the plugin `gabo`, declares its `version`, its `skills` path and, when hooks ship, a `hooks` path to the Codex hook file. | Given the plugin directory When the Go test parses `.codex-plugin/plugin.json` Then `name` is `gabo`, `version` matches `x.y.z`, `skills` and `hooks` resolve to existing paths inside the plugin root, and the manifest declares no `apps` and no `mcpServers`. | Go unit test on the plugin tree. |
| S2 | The repository shall be a Codex marketplace named `gabo-ai` that lists the plugin `gabo`. | Given the repository When `codex plugin marketplace add <repository path or git URL>` runs Then `codex plugin marketplace list` shows `gabo-ai`, and `codex plugin list` shows `gabo` from it. | Go unit test on `.agents/plugins/marketplace.json`; manual check S10. |
| S3 | When a user runs `codex plugin add gabo@gabo-ai`, Codex shall install the plugin with the same skills as the Claude Code plugin of intent 0001 (its S3 and S10, owner decision 2026-11-03): every skill `gabo init` copies, minus every skill that runs reviews (`gabo-reviewer`, `gabo-reviewer-placement-local`), plus a reviewer-configuration skill when the catalog has one. The plugin never ships a skill that runs reviews; the skill that runs the reviewer scripts stays user level only. | Given the marketplace is added When `codex plugin add gabo@gabo-ai` runs Then `~/.codex/plugins/cache/gabo-ai/gabo/<version>/` holds each plugin skill's `SKILL.md`, a new `codex` session lists them, and neither `gabo-reviewer` nor `gabo-reviewer-placement-local` is in the plugin. | Go unit test on the skill set; manual check S10. |
| S4 | The catalog shall ship a `gabo-runtime-codex` skill that builds every agent of the neutral agent spec as `.codex/agents/<id>.toml` from the `gabo-agent-templates` templates, in the shape of `gabo-runtime-claude` and `gabo-runtime-pi`. | Given a project after `gabo init` with `runtimes.coder: codex` When the agent follows `gabo-runtime-codex` Then `.codex/agents/` holds one TOML file per spec id (`gabo-pm`, `gabo-dev-lead`, `gabo-db-master`, `gabo-frontend`, `gabo-plan-reviewer`, `gabo-cycle`, `gabo-coder`, `gabo-reviewer`), each parses as TOML with `name` equal to its id, `description` and `developer_instructions`, and carries a `# gabo-version: <x.y.z>` line equal to `.gabo/VERSION`. | Skill test in `apps/cli/internal/skills` (like `runtimeclaude_test.go`); probe in S10. |
| S5 | `gabo-runtime-codex` shall link the catalog skills and the fixed-name adapter links (`gabo-tasks`, `gabo-methodology`, `gabo-repository-host`, `gabo-memory`, …) under `.agents/skills/`, link the user-level `gabo-reviewer` copy, and never place `gabo-reviewer` in the project. | Given the build of S4 When `gabo check` runs Then it reports no `✗ skill link` and no `✗ reviewer link` failure, and `.agents/skills/gabo-reviewer` does not exist in the project. | `gabo check` integration test with a fixture project; manual check S10. |
| S6 | The configuration shall accept `codex` for `runtimes.coder` and `runtimes.assistant`, and `runtimes.reviewer` shall stay `claude` or `pi` until the reviewer scripts support more engines (D27, owner decision 2026-11-03). This is the same enum edit as intent 0004 S1 (`opencode`): one change to `apps/cli/internal/config/config.go` (the runtime check near line 192) and both schema copies (`apps/cli/internal/config/schemas/gabo.project.schema.json`, `schemas/gabo.project.schema.json`) that adds `codex` and `opencode` to `runtimes.coder` and `runtimes.assistant` only; the plan makes that edit once. Accepting any engine `gabo-runtime-builder` generates is intent 0005, not this one. | Given `gabo.yaml` with `runtimes.coder: codex` and `runtimes.assistant: codex` When `gabo check` runs Then the config passes. Given `runtimes.reviewer: codex` Then `gabo check` fails with a line naming `runtimes.reviewer` and the allowed values `claude` and `pi`. Given both schema copies Then `runtimes.coder` and `runtimes.assistant` list `codex` and `runtimes.reviewer` does not. | Go unit tests on `config` and the JSON Schema (both copies). |
| S7 | Where the hooks ship (open question 3), when Codex edits a gabo file in a project that has `.gabo/VERSION`, the plugin shall run `gabo check` in the project root and return its failure lines to Codex as additional context. | Given a gabo project whose intent `0009-x` has a `plan.md` and a `spec.md` with an empty Acceptance table When Codex applies a patch to `docs/product/intents/0009-x/plan.md` Then the hook output carries the `✗ … spec.md is not accepted` line from `gabo check`. | Shell test of the shared hook script with recorded Codex `PostToolUse` input; manual check S10. |
| S8 | The hook shall stay silent and never block when the edited file is not a gabo file, the project has no `.gabo/VERSION`, `gabo` is not on `PATH`, or `gabo check` cannot run. | Given each of these four cases When the hook receives a Codex `PostToolUse` input Then it exits 0 with no output. | Shell test of the shared hook script. |
| S9 | The plugin shall not change `gabo init`, `gabo check`, the project's catalog copy, `.codex/agents/`, `.agents/skills/`, `gabo.yaml` or any user-level file. | Given a gabo project that passes `gabo check` When the plugin is installed and a Codex session runs with the hooks trusted Then `git status` shows no change made by the plugin and `gabo check` still passes. | Manual check S10. |
| S10 | The repository documentation shall give the Codex install, update and local verification steps, and the owner shall run them on the local Codex before the plugin is announced. | Given the README Then it names `codex plugin marketplace add`, `codex plugin add gabo@gabo-ai`, `codex plugin marketplace upgrade`, `/hooks` to trust the hooks, and that `gabo init` plus `gabo-runtime-codex` build the agents. Given the owner's Codex When the steps in section 2 "Local verification" run Then every step passes or the failing fact is reported as a blocker. | Doc review; manual run on the owner's Codex with the result recorded in the plan. |
| S11 | The Codex manifest `version` shall equal the gabo CLI release version and the Claude Code manifest `version` of intent 0001. | Given a release tag `apps/cli/v<x.y.z>` When both manifests are read Then each `version` is `x.y.z`. | Go unit test comparing both manifests; release checklist. |

## 2. Design

The change adds a Codex manifest beside the Claude Code manifest of intent 0001 in one repository, shares the skills and the hook script, and adds one runtime skill that builds the agents per project. Agents are not shipped in the plugin.

```text
repository root
├─ .agents/plugins/marketplace.json     Codex marketplace "gabo-ai" (new, thin)
├─ .claude-plugin/marketplace.json      Claude Code marketplace (intent 0001)
└─ plugins/<plugin dir>/                one plugin root, see D4 and open question 5
   ├─ .claude-plugin/plugin.json        Claude Code manifest (intent 0001)
   ├─ .codex-plugin/plugin.json         Codex manifest (new, thin)
   ├─ skills/<catalog skill>/           shared, one source with apps/cli/internal/skills
   └─ hooks/
      ├─ hooks.json                     Claude Code hook config (intent 0001)
      ├─ hooks.codex.json               Codex hook config (new)
      └─ gabo-check.sh                  shared script, reads both input shapes
```

**Where things come from.**

| piece | shared with the Claude Code plugin | Codex only |
|---|---|---|
| Skills | yes: same folders, same SKILL.md format, same single source | — |
| Manifest | — | `.codex-plugin/plugin.json` |
| Marketplace file | — (Codex can read `.claude-plugin/marketplace.json`, but see D3) | `.agents/plugins/marketplace.json` |
| Hook config | — | `hooks/hooks.codex.json` (Codex would otherwise load the Claude `hooks/hooks.json` by default) |
| Hook script | yes: `hooks/gabo-check.sh`, which reads `PLUGIN_ROOT` or `CLAUDE_PLUGIN_ROOT` and both input shapes | — |
| Agents | the neutral spec and templates (`gabo-agent-templates`) | `gabo-runtime-codex` renders them to TOML in the project |
| Version | yes: one version for both manifests (S11) | — |

**Agent rendering (S4).** `gabo-runtime-codex` fills each template exactly as the other runtime skills do and writes:

```toml
# gabo-version: 0.12.0
name = "gabo-coder"
description = "<description from the spec>"
developer_instructions = """
<filled template body>

Load these skills by name before work: gabo-coder, gabo-methodology, gabo-tasks, gabo-repository-host, gabo-memory.
"""
```

The spec's skill list becomes one instruction line, because Codex agents have no preload key. The spec's capability list maps to no TOML key, because Codex has no per-agent tool allowlist; the agent's limits stay in its instructions (D6). No `model`, `sandbox_mode` or `features` key is written: the spec and templates do not give one, and the runtime skill invents nothing.

**Coexistence with `gabo init`.** The plugin offers the skills globally in Codex; `gabo init` still copies the pinned catalog into `.gabo/skills/` and writes `.gabo/VERSION`, and `gabo-runtime-codex` still builds the agents and links in the project. The project copy is what the agents load (`.agents/skills/` links point into `.gabo/skills/`), so a newer or older plugin never changes what a wave runs. The plugin writes no project file (S9).

**Versioning.** Both manifests carry the CLI release version `x.y.z`; the marketplace entries carry none. Codex caches per version, so a user gets an update with `codex plugin marketplace upgrade` once a release bumps the version. The built agent files keep their own `gabo-version` line, which `gabo check` compares with `.gabo/VERSION`, not with the plugin.

**Local verification (S10)**, on the owner's machine with `codex-cli 0.163.0` or newer, in a scratch copy of a gabo project:

1. `codex plugin marketplace add <repository checkout>`; `codex plugin marketplace list` shows `gabo-ai`.
2. `codex plugin add gabo@gabo-ai`; the cache folder `~/.codex/plugins/cache/gabo-ai/gabo/<version>/` exists with the skills.
3. Start `codex` in the project; the gabo skills are listed; record which copy a session loads when the plugin and `.agents/skills/` both hold one skill name (open question 4).
4. `/hooks` lists the gabo hook; trust it; patch a `plan.md` of an intent whose spec is not accepted; the `gabo check` failure appears as context (S7); patch a non-gabo file; nothing appears (S8).
5. Set `runtimes.coder: codex`, run the agent with `gabo-runtime-codex`, then `gabo check`; it passes (S4, S5, S6).
6. Probe: from the main session spawn `gabo-pm` through Codex's subagent tool with a harmless prompt; it answers and names the skills it loaded.
7. `git status` shows only the files the runtime skill wrote (S9); `codex plugin remove gabo@gabo-ai` leaves the project unchanged.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Where Codex agents come from | (a) ship prebuilt TOML agents in the plugin (`subagents` key, as in `ghiaog123/ai-native-sdlc`); (b) a `gabo-runtime-codex` catalog skill builds them per project | **(b)**: gabo agents are templates with per-project slots (adapters, methodology, identity, D24), `gabo check` needs them in a declared project directory, and plugin-shipped agents are not documented by Codex | The plugin cannot run a wave alone; `gabo init` plus the runtime skill stay the way to get agents, as in intent 0001 D6 |
| D2 | Agent file format and place | `.codex/agents/<id>.toml`; `~/.codex/agents/` (user level); Markdown | **`.codex/agents/<id>.toml`**, committed, `metadata.agent_dir: .codex/agents`: the documented project location; the file name equals the spec id, which `gabo check` requires | `gabo check` works unchanged (any extension, `gabo-version:` in a TOML comment) |
| D3 | Marketplace file | rely on Codex reading `.claude-plugin/marketplace.json`; add `.agents/plugins/marketplace.json` | **add `.agents/plugins/marketplace.json`**: it is the documented Codex location; the `.claude-plugin` path is legacy-compatible and its entries carry Claude-only fields | Two thin marketplace files list the same plugin; a Go test keeps names and source paths equal |
| D4 | Repository layout | (a) one plugin root with both manifests (as `ghiaog123/ai-native-sdlc`); (b) one plugin directory per tool (`plugins/claude-code/`, `plugins/codex/`) | **Assumed (a)**, one repository with thin per-tool manifests over one skills tree; final layout follows intent 0001 (open question 5) | One skill source and one version for both tools |
| D5 | Hook event for the checks | (a) `PostToolUse` on `apply_patch` running `gabo check` and returning failures as `additionalContext`; (b) `PreToolUse` that denies the patch; (c) `Stop` with `decision: "block"` | **(a)**, same as intent 0001 D8: reuses `gabo check` unchanged and never blocks the user; (c) turns the reason into a new prompt and can loop | The script parses `*** Add File:` / `*** Update File:` lines from `tool_input.command` for Codex and `tool_input.file_path` for Claude Code |
| D6 | Read-only agents (`gabo-plan-reviewer`, `gabo-reviewer`) | (a) instructions only; (b) write `sandbox_mode = "read-only"` as the reference repository does; (c) `features.shell_tool = false` | **(a)**: the docs say `sandbox_mode` in an agent file does not override the parent, and both agents need the shell; the templates already forbid edits | Read-only is a prompt rule on Codex, not a tool rule; recorded as a policy conflict |
| D7 | Reviewer runtime | allow `runtimes.reviewer: codex`; keep `claude` or `pi` | **Decided (owner 2026-11-03)**: keep `claude` or `pi` (D27): the reviewer scripts have no Codex engine | A Codex team still needs Claude Code or Pi for the reviewer; a Codex reviewer is a later intent |
| D8 | Version source | one version for both manifests; separate versions | **one version**, the CLI release version | One release updates both plugins; S11 test |

## 3. UX and app flow (optional)

Not applicable: the change has no gabo screen or app flow. The user-facing steps are Codex CLI commands, specified in S2, S3 and S10 and documented in the README.

## 4. DB migration steps (optional)

Not applicable: the change alters no stored data. The `runtimes.*` enum gains a value; existing `gabo.yaml` files stay valid, so no config migration is needed.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| TRD 8.1 Agent building: the runtime skill "set[s] tool lists" | Codex agents have no tool allowlist and no skill preload key | D2 and D6: skills become an instruction line, capabilities stay in the template body; the `gabo-runtime-codex` Sources table records the fact; TRD wording follow-up carried to `gabo-dev-lead` |
| TRD 2 "Supported runtimes" (Claude Code and Pi) and PRD R35 / D24 ("adding a runtime needs no Go change") | `runtimes.*` accepts only `claude` and `pi` in Go and in the JSON Schema, so Codex, or any engine `gabo-runtime-builder` adds, fails `gabo check` without a Go change | S6 adds `codex` here, in the same enum edit as intent 0004; the general fix is intent 0005 (owner decision 2026-11-03, open question 9) |
| D20: `gabo-reviewer` never in the project, one user-level copy | A plugin copy would be a second, independently versioned reviewer copy | S3 excludes every skill that runs reviews and allows only a reviewer-configuration skill (owner decision 2026-11-03); S5 links the user-level copy |
| D27: reviewer runtime | — | D7 keeps `runtimes.reviewer` at `claude` or `pi` |
| `review.human_only_paths` (`config/required.go`) lists `.claude/`, `agents/`, `AGENTS.md`, but not `.codex/`, `.agents/` or `.codex-plugin/` | Codex agent files, Codex skill links and the Codex manifest could be merged without a human | Carried to the plan: add `.codex/`, `.agents/` and the plugin directory to `review.human_only_paths` in the same release |
| PRD R31 (CLI surface) | none: no new command | — |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Intent question 1: which gabo agents must exist in Codex's format for the first release? | answered: all eight of the neutral spec, built per project by `gabo-runtime-codex` (S4, D1), because `gabo check` requires every spec agent in the runtime's agent directory |
| 2 | Intent question 2: can one source of truth generate both the Claude and the Codex agent formats? | answered: yes, the neutral spec and templates of `gabo-agent-templates` (D24); `gabo-runtime-claude` renders Markdown, `gabo-runtime-codex` renders TOML; no generator is added |
| 3 | Intent question 3: are the hooks in the first release? Options: (a) yes, the `PostToolUse` check of D5, same as intent 0001; (b) no, skills and runtime skill only, hooks in a later intent; (c) only a `SessionStart` hook that reports `gabo check` once per session. Recommendation: (a), because the script is shared with intent 0001 and stays silent outside gabo projects (S8); Codex already makes the user trust each hook. | carried to owner |
| 4 | Inside a gabo project, Codex sees each skill twice: in the plugin and in `.agents/skills/` (the pinned project copy). Which wins is not documented. Options: (a) accept both, since the agents read the project copy through their instructions; (b) ship only a bootstrap subset in the plugin (`gabo-setup`, `gabo-config-writer`, `gabo-agent-templates`, `gabo-runtime-codex`, `gabo-runtime-builder`, `gabo-doc-guide`, the directiva roles); (c) ship the full set and add a `SessionStart` note that the project copy wins when `.gabo/VERSION` exists. Recommendation: decide together with intent 0001 open question 1 and 5, take (a) for one rule across both tools, and fall back to (b) if step 3 of the local verification shows the plugin copy shadowing the project copy. | **Decided (owner 2026-11-03)** for the skill set, same as intent 0001 open question 1: owner, verbatim: "que estar no es que ese esquiles para revisar el código, ese esquiles para configurar el revisor del código". The plugin ships every skill `gabo init` copies minus every skill that runs reviews, and may ship a reviewer-configuration skill (S3). Which copy wins inside a project stays with intent 0001 open question 5; the fallback to (b) stays tied to step 3 of the local verification. Whether a reviewer-configuration skill exists is intent 0001 open question 6. |
| 5 | Plugin directory layout and skill source. Options: (a) one plugin root with both manifests and one skills tree; (b) `plugins/claude-code/` and `plugins/codex/`, each with symlinks into `apps/cli/internal/skills`; (c) a copied skills tree kept equal by a Go test (the D25 pattern). Recommendation: (a) with the symlinks intent 0001 chose, if step 2 of the local verification shows Codex dereferences symlinks into its cache; otherwise (c). | carried to owner, shared with intent 0001 |
| 6 | Can a Codex plugin ship agents at all (the undocumented `subagents` key)? Options: (a) ignore it, agents come only from `gabo-runtime-codex` (D1); (b) probe it in the local verification and record the result for a later intent. Recommendation: (a) now, (b) as a one-line note in the plan; it does not change this spec. | carried to plan |
| 7 | Should Codex read-only agents also get `sandbox_mode = "read-only"`? Options: (a) no, instructions only (D6); (b) yes, as the reference repository does, though the docs say it is ignored in agent files; (c) probe it and decide. Recommendation: (a); revisit only if Codex documents agent-level sandboxing. | carried to owner |
| 8 | Should `runtimes.reviewer` accept `codex`? Options: (a) no, until the reviewer scripts have a Codex engine (D27); (b) yes, now, with a Codex engine in the scripts. Recommendation: (a); a Codex reviewer engine is its own intent because `skills/gabo-reviewer/` changes are human-merged and need their own tests. | **Decided (owner 2026-11-03)**: (a); `runtimes.reviewer` stays `claude` or `pi` until the reviewer scripts support more engines (S6, D7). |
| 9 | The `runtimes.*` enum blocks every engine that `gabo-runtime-builder` adds, against PRD R35. Options: (a) add only `codex` here (S6); (b) accept any value that has a `gabo-runtime-<engine>` skill in `.gabo/skills/` or `.gabo/runtimes/`, as a new intent. Recommendation: (a) here and open (b) as a new intent under card #171, since it also affects intents 0003 and 0004. | **Decided (owner 2026-11-03)**: (a) here, adding `codex` to `runtimes.coder` and `runtimes.assistant` in the same enum edit as intent 0004 (S6); (b) is intent 0005 (`0005-any-runtime-engine`). |
| 10 | Is the Codex desktop app in scope, or only the CLI? Options: (a) CLI only for the first release; (b) CLI and desktop app. Recommendation: (a): the owner verifies on the CLI, and the docs already say the desktop app installs from the same marketplace. | carried to owner |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
