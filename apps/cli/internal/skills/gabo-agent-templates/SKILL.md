---
name: gabo-agent-templates
description: "Trigger: build gabo agents, agent spec, agents.yaml, agent templates, rebuild agents, runtime skill input. Neutral agent spec and engine-neutral body templates of every gabo agent."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.3"
---
## Activation Contract

Load when a runtime skill (for example `gabo-runtime-claude`) builds or rebuilds the gabo agents of a project, or when `gabo check` reports a missing or outdated agent (TRD 8.1, D24).

## Hard Rules

- `agents.yaml` is the single source of which agents exist. Build exactly one agent file per `agents` entry, named after its spec `id`; never add, drop or rename an agent. The `main_session` entry is no spec agent: build it only as the Main-Session Entry below.
- Read the spec and templates from the project copy (`.gabo/skills/gabo-agent-templates/`), never from memory.
- `tools` are engine-neutral capabilities; the runtime skill maps them to the engine's tool names.
- `skills` lists catalog skills only; slot adapters are linked by the runtime skill under their fixed slot names.
- `launched_by` names the spec agent that launches the agent, or `main-session` for the assistant's main session.
- Fill only the named slots of the agent's `template` (under `agents/`) and invent nothing: copy every other byte of the body unchanged, and add no rule, step, tool or skill the spec and template do not state. A slot you cannot fill is a failure to report, not a guess.
- The templates hold no engine-specific syntax (no frontmatter, engine tool names or engine paths); the runtime skill adds the engine's frontmatter, tool list and skill list from the spec.

## Slots

Each template is a Markdown body. A slot is written `{{name}}`; these are the only slots:

| slot | filled with |
|---|---|
| `{{id}}` | the agent's spec `id` (identity) |
| `{{role}}` | the agent's spec `role` (identity) |
| `{{gabo_version}}` | the single line of `.gabo/VERSION`, the bare version (for example `0.4.0`) |
| `{{methodology}}` | the fixed-name link of the methodology slot adapter (`gabo-methodology`) |
| `{{adapter:tasks}}` | the fixed-name link of the tasks slot adapter |
| `{{adapter:repository_host}}` | the fixed-name link of the repository host slot adapter |
| `{{adapter:memory}}` | the fixed-name link of the memory slot adapter |
| `{{reviewer_script}}` | the path of `scripts/reviewer.sh` in the user-level reviewer skill the runtime skill links (D20); never a copy inside the project |

Every template starts with the version line `gabo-version: {{gabo_version}}` (TRD 9.4, no hashes). The runtime skill writes it with the filled version where the engine keeps agent metadata (for example the frontmatter), and `gabo check` compares it with `.gabo/VERSION`. A runtime skill that links an adapter also makes the agent load it, so the slot name in the body names a skill the agent has.

## Spec Fields

| field | meaning |
|---|---|
| `id` | agent file base name, unique |
| `role` | one line: what the agent does |
| `skills` | catalog skills the agent loads |
| `tools` | capabilities: `read`, `search`, `edit`, `shell`, `git`, `tracker`, `repository-host`, `memory`, `web`, `skills`, `subagents` |
| `launched_by` | launching spec agent or `main-session` |
| `template` | body template path relative to this skill |

The top-level `main_session` entry has `id`, `role`, `skills` and `template` as above, plus `block`: the scoped block template for an instructions file. It is never in the agent list.

The Go loader (`internal/agentspec`) validates the spec: unique ids, every `launched_by` and `template` resolves, every listed skill exists in the catalog, and no cycle in `launched_by`; a `main_session` entry needs an id that is no spec agent, a role, and a `template` and `block` that resolve.

## Main-Session Entry

The owner's main session runs as the gabo assistant (spec 0009 S1 to S4). Build the entry only for `runtimes.assistant`: the runtime skill of that engine writes and refreshes its row on every build, and every runtime skill removes the row of each other full-tier engine it finds, recognized only by its gabo identity (the block markers, a key whose value is exactly `gabo-assistant`, an agent file `gabo-assistant` carrying a `gabo-version:` line, the `gabo-assistant` link). This table is the one place each engine's entry is defined.

| engine | skill link | main-session entry |
|---|---|---|
| Claude Code | `.claude/skills/gabo-assistant` | agent file `.claude/agents/gabo-assistant.md` from `main_session.template`, and the one key `agent: "gabo-assistant"` merged into `.claude/settings.json` |
| OpenCode | `.agents/skills/gabo-assistant` | agent file `.opencode/agents/gabo-assistant.md` with `mode: primary` from `main_session.template`, and the one key `default_agent: "gabo-assistant"` merged into the project config file the runtime skill resolves |
| Codex | `.agents/skills/gabo-assistant` | `main_session.block` in `AGENTS.override.md`, else `AGENTS.md` (created when neither exists) |
| Pi | `.agents/skills/gabo-assistant` | `main_session.block` in `AGENTS.override.md`, else `AGENTS.md`, else an existing `CLAUDE.md` (`AGENTS.md` created when none exists) |

- Every skill link points to the catalog folder `.gabo/skills/gabo-assistant`, and every entry names `.gabo/skills/gabo-assistant/SKILL.md`. In the shared `.agents/skills`, keep the link while any declared engine using that directory is `runtimes.assistant`.
- The agent file follows the agent build rules above (frontmatter from the spec, version line, body from the template). Merge only the one key; keep every other byte, and keep and report a user's own value of that key.
- The block (`main_session.block`) is copied verbatim, from `<!-- GABO_ASSISTANT_START -->` to `<!-- GABO_ASSISTANT_END -->`, at most 2048 bytes. Replace only the bytes between the markers, append it after one blank line when absent, keep every other byte, and write through a symlinked file only when its target stays inside the repository. Its first paragraph keeps role-prompt and headless sessions in their role.

## Output Contract

Report each agent file written with its spec `id` and the version it carries, and any spec entry or slot you could not fill.
