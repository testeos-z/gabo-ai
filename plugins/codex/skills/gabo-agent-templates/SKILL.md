---
name: gabo-agent-templates
description: "Trigger: build gabo agents, agent spec, agents.yaml, agent templates, rebuild agents, runtime skill input. Neutral agent spec and engine-neutral body templates of every gabo agent."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.2"
---
## Activation Contract

Load when a runtime skill (for example `gabo-runtime-claude`) builds or rebuilds the gabo agents of a project, or when `gabo check` reports a missing or outdated agent (TRD 8.1, D24).

## Hard Rules

- `agents.yaml` is the single source of which agents exist. Build exactly one agent file per entry, named after its spec `id`; never add, drop or rename an agent.
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

The Go loader (`internal/agentspec`) validates the spec: unique ids, every `launched_by` and `template` resolves, every listed skill exists in the catalog, and no cycle in `launched_by`.

## Output Contract

Report each agent file written with its spec `id` and the version it carries, and any spec entry or slot you could not fill.
