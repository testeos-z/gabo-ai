---
name: gabo-agent-templates
description: "Trigger: build gbai agents, agent spec, agents.yaml, agent templates, rebuild agents, runtime skill input. Neutral agent spec and engine-neutral body templates of every gbai agent."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "0.1"
---
## Activation Contract

Load when a runtime skill (for example `gabo-runtime-claude`) builds or rebuilds the gbai agents of a project, or when `gabo check` reports a missing or outdated agent (TRD 8.1, D24).

## Hard Rules

- `agents.yaml` is the single source of which agents exist. Build exactly one agent file per entry, named after its `id`; never add, drop or rename an agent.
- Read the spec from the project copy (`.gbai/skills/gabo-agent-templates/agents.yaml`), never from memory.
- `tools` are engine-neutral capabilities; the runtime skill maps them to the engine's tool names.
- `skills` lists catalog skills only; slot adapters are linked by the runtime skill under their fixed slot names.
- `launched_by` names the spec agent that launches the agent, or `main-session` for the assistant's main session.
- Fill only the named slots of the agent's `template` (under `agents/`); invent no content.

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

Report each agent file written with its spec `id`, and any spec entry you could not build.
