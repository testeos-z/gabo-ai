---
name: gabo-runtime-builder
description: "Trigger: runtime builder, add a runtime, new engine, no runtime skill for my engine, gabo check reports no runtime skill. Researches an engine from its docs, builds and checks its agents, then writes its runtime skill."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
---
## Activation Contract

Load when a value of `runtimes.*` in `gabo.yaml` names an engine (`<engine>`, the value as written, for example `opencode`) with no `gabo-runtime-<engine>` skill in `.gabo/skills/` or `.gabo/runtimes/`, or when `gabo check` reports `✗ runtime <engine> has no runtime skill gabo-runtime-<engine>` (PRD R35, TRD D24). Load `gabo-agent-templates` with it, and read `gabo-runtime-claude` and `gabo-runtime-pi` in `.gabo/skills/`: they are the shape to copy.

The result is `.gabo/runtimes/gabo-runtime-<engine>/SKILL.md`: a runtime skill like those two, which later builds and rebuilds the agents without this skill. It lives outside the catalog copy `.gabo/skills/`, so `gabo init` never removes it, and `gabo check` reads its `metadata.agent_dir` from there. Adding a runtime needs no Go change and no gabo-ai release.

## Hard Rules

- Every engine fact you use (where agents and skills live, file format, frontmatter keys, tool names, how a subagent is launched, the user-level skill directory) comes from the engine's official documentation first, is rechecked against the installed engine (`<engine> --version`, `<engine> --help` and its read-only list or debug commands) and carries its source URL and the version it was checked on. Take nothing from memory, blogs or another engine's skill; never from memory even when the engine looks like Claude Code or Pi.
- With no web access and no documentation MCP, ask the owner for the documentation links in one question and stop until they answer.
- When the docs are unclear, or the installed engine behaves differently from them, ask the owner one concrete question: the fact, the two readings, what you saw and which build step depends on it. One question at a time; build nothing that depends on the answer meanwhile.
- Refuse an engine without subagents and skills, or an equivalent (project-defined agents that a parent agent launches with their own prompt and tools; instruction files an agent loads by name or path on demand). Build nothing and say:

  ```
  ✗ cannot add a runtime for <engine>
  why: it has no subagent and skill support, or an equivalent
  next: use Claude Code, Pi or another engine with subagents and skills
  ```

- Never install or upgrade the engine: run only commands that write nothing (version, help, list, debug). Engine config (intent 0004 S8, D7): gabo writes only the engine keys it needs to operate, at project level only, merged with the existing project config (every unrelated key kept), reports what it wrote, and has the value verified by `gabo check`; never the user's global engine config (for example `~/.config/opencode/opencode.json`, `~/.claude/settings.json`). A key the engine needs that has no `gabo check` verification yet: ask the owner before writing it. Outside the project write only the user-level reviewer link; scratch work goes in `mktemp -d`.
- The build rules of the shipped runtime skills hold here too: read the spec, templates and `.gabo/VERSION` from the project; fill only the template slots and invent nothing; build every spec agent in the one agent directory. Never place `gabo-reviewer` in the project: only link the user-level copy (D20). Overwrite only what this skill writes, and leave and report any other entry; write no secret (R21).
- Order is fixed: probe agent, spec agents, `gabo check`, and only then the runtime skill. A runtime skill is never written from research alone.

## Research

Establish each fact below, one `## Sources` row each, before you build:

| fact | decides |
|---|---|
| installed version, CLI help | the version every row cites |
| project agent directory; agents are flat files named after their id | `metadata.agent_dir` (repository-relative; `gabo check` looks for `<id>.<ext>` directly in it) |
| project skill directory, when the engine links skills as `<dir>/<name>` folders; the user-level path the reviewer agent runs the reviewer skill from | optional `metadata.skill_dir` (repository-relative) and `metadata.reviewer_link` (`${NAME:-default}`, `${NAME}` and `$NAME` are expanded); when set, `gabo check` requires the catalog, fixed-name and reviewer links there |
| agent file format and required keys; a key or field that may hold `gabo-version: <x.y.z>` and that the engine ignores or keeps | the frontmatter of each agent |
| how a parent launches a subagent, and the tool name | the `subagents` row and the probe |
| tool names, and how one agent's tools are restricted | the Tool Map |
| project skill directory; whether linked skill folders are followed; how a skill is named (folder or frontmatter `name`); whether an agent preloads skills or loads them on demand | catalog links, fixed-name links, how an agent learns its skills |
| user-level skill directory and the variable that moves it | the reviewer link and `{{reviewer_script}}` |
| read-only commands that list agents and skills | verification before the probe |

A missing subagent or skill fact after the docs and the installed engine: that is the refusal above, not a guess.

## Build Steps

1. **Inputs.** Read `gabo.yaml`, `.gabo/VERSION` and `.gabo/skills/gabo-agent-templates/agents.yaml`. A missing stamp or catalog: stop and tell the user to run `gabo init`.
2. **Stub.** `gabo check` finds the agent directory only through the runtime skill, so write `.gabo/runtimes/gabo-runtime-<engine>/SKILL.md` with the frontmatter only (`name`, `description`, `license`, `metadata.author`, `metadata.version`, `metadata.agent_dir`). If the build stops before the last step, delete the stub you wrote.
3. **Links.** Catalog skills, fixed-name links (`gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory`, one per slot whose adapter is a `SKILL.md`) and the user-level reviewer link, the way `gabo-runtime-claude` and `gabo-runtime-pi` make them: relative links in the engine's project skill directory, and an absolute link from the engine's user-level skill directory to `${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer`. An engine that does not follow linked folders: ask the owner before copying anything.
4. **Probe.** Write `<agent_dir>/gabo-probe.<ext>` with the shell tool only and the body: run `git rev-parse --show-toplevel` once and reply with its output, then the first Markdown heading of each fixed-name link's `SKILL.md`. List the agents with the engine's read-only command, then launch the probe once as a subagent from the parent, the way spec agents are launched, with the prompt `Follow your system instructions exactly and reply with the result.` (a parent that paraphrases the task can make the probe run something else). Delete the file whatever the result. It passes when the command ran and every link shows its adapter's heading. The session cannot see new files: report `probe not run: restart <engine> and rebuild`. The engine cannot reach a model: report the probe as not run, never as passed. A failed probe: recheck the facts, ask the owner when they are unclear, else stop and report.
5. **Spec agents.** For each spec entry write `<agent_dir>/<id>.<ext>`: the engine's frontmatter from your facts, `gabo-version: <x.y.z>` from `.gabo/VERSION` where the engine keeps metadata, the mapped tools, and the filled template body without its first line `gabo-version: …` and the blank line after it. When the engine has no skill list per agent, start the body with one line naming the paths of the skills to load, as `gabo-runtime-pi` does.
6. **Check.** Run `gabo check`, fix every failure this build caused and rerun until it passes. When it rejects the `runtimes.*` value itself (the installed gabo accepts only the runtimes its message lists), stop: report it with the message, keep the agent files and links, delete the stub, and write no runtime skill.
7. **Write the runtime skill.** Replace the stub with the full skill (Runtime Skill Shape), dated today and citing the version checked. Run `gabo check` once more.

## Runtime Skill Shape

The skill written in step 7 follows `gabo-runtime-claude` and `gabo-runtime-pi`:

- Frontmatter: `name: gabo-runtime-<engine>`, `description: "Trigger: build gabo agents for <Engine>, rebuild <Engine> agents, runtimes <engine>. …"` (one line, at most 250 characters), `license: Apache-2.0`, and `metadata` with `author: gabo-ai`, `version` and `metadata.agent_dir`, plus `metadata.skill_dir` and `metadata.reviewer_link` when the facts give them. Also `metadata.agent_skills`, where each agent file lists its skills, and `metadata.agent_name`, where it declares its name (omit it when the engine names an agent by its file name), each as `frontmatter:<key>` (a YAML list or scalar under `<key>` of the file's frontmatter), `toml:<key>` (a top-level TOML string) or `line:<prefix>` (the first line starting with `<prefix>`; the list is the text after ` for each of ` when present, else after `<prefix>`, comma-separated), quoted when the value holds `: `; `gabo check` reads them, and without `metadata.agent_skills` it only warns and skips the per-agent checks for the engine.
- `## Activation Contract`: when to load it (a `runtimes.*` value is `<engine>` and the agents must be built or rebuilt), with `gabo-agent-templates`, and that `gabo check` reads `metadata.agent_dir`.
- `## Hard Rules`: the build rules above, plus the engine's own (for example a package it needs, a config it must not touch).
- `## Sources`: `Checked on <date> against <Engine> <version>`, a table of fact and source URL for every fact used, and the recheck rule: when `<engine> --version` differs, reread the sources and its help, and stop on a row that no longer holds.
- `## Tool Map`: one row per capability, `read`, `search`, `edit`, `shell`, `git`, `tracker`, `repository-host`, `memory`, `web`, `skills`, `subagents`, to the engine's tool names; a capability the engine lacks says so and what the agent loses.
- `## Fixed-Name Links`: the catalog and slot links and how an agent finds a linked skill.
- `## Build Steps`: inputs, links, reviewer link, the slot fills (`{{id}}`, `{{role}}`, `{{gabo_version}}`, `{{methodology}}`, `{{adapter:tasks}}`, `{{adapter:repository_host}}`, `{{adapter:memory}}`, `{{reviewer_script}}` as a path the shell expands, holding no personal path), the agent file with its `gabo-version` line, then verify: probe agent and `gabo check`.
- `## Output Contract`: as in the shipped skills.

## Output Contract

Report the engine and version; the sources read, each with URL and version; the questions asked and their answers; a refusal with its message; the probe result; each agent file and link written and each entry left with why; the `gabo check` result; the runtime skill path written, or why none was written.
