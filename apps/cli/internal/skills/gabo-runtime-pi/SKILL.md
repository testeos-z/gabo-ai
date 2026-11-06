---
name: gabo-runtime-pi
description: "Trigger: build gabo agents for Pi, rebuild Pi agents, runtimes pi, .pi/agents, gentle-pi, link reviewer into Pi. Builds every spec agent as a Pi subagent and places skills and links."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
  agent_dir: .pi/agents
  agent_skills: "line:Skills:"
  agent_name: frontmatter:name
---
## Activation Contract

Load when a value of `runtimes.*` in `gabo.yaml` is `pi` and the gabo agents must be built or rebuilt: during setup, after `gabo init` refreshed the catalog, or when `gabo check` reports a Pi agent missing or built with another version (TRD 8.1, 9.4, D21, D24). `gabo check` reads `metadata.agent_dir` above and looks for the agents only in `.pi/agents`.

## Hard Rules

- Pi core has no subagents. Build nothing until a subagent package is present (Prerequisite); then refuse and say what to install.
- Engine config (intent 0004 S8, D7): gabo writes only the engine keys it needs to operate, at project level only, merged with the existing project config (every unrelated key kept), reports what it wrote, and has the value verified by `gabo check`; never the user's global engine config: no write to `settings.json`, `models.json`, `mcp.json` or anything under the agent home except the one `skills/gabo-reviewer` link. `gabo init` writes the project-level gabo integration keys (intent 0006); this skill writes no Pi config, and never runs `pi install`. Never edit `subagents.json`: the subagent package owns it. Never run `pi install`, `pi config` or `npm i` yourself; the user installs.
- Never place `gabo-reviewer` in the project, not as a copy and not as a link (D20): PR code must not reach the scripts that review it.
- Read the spec, templates and version from the project copy (`.gabo/`), never from memory or the binary. Fill only the template slots (`gabo-agent-templates`); invent no rule, tool or skill.
- Write only `.pi/agents/<id>.md` for spec ids, `.pi/skills/` links, and the user-level reviewer link. Leave every other file in `.pi/` untouched.

## Prerequisite

Run in the repository root, read-only:

1. `pi --version` must succeed; else refuse: `Pi is not installed: see https://github.com/earendil-works/pi`.
2. `pi list` (lists user and project packages, writes nothing). A subagent package is present when it lists `npm:gentle-pi` or a `git:` source of gentle-pi, or `gentle-shell` is on `PATH` (`npm i -g gentle-pi` ships it with its own Pi).
3. Another subagent package is acceptable (for example `pi-subagents-j0k3r`) only after you confirm from its docs that it discovers project agents in `<cwd>/.pi/agents/` and the tool names it registers; else treat it as absent.
4. None present: refuse, build nothing, install nothing, and say:

   > Pi has no subagents, so gabo cannot work on Pi without a subagent package. For subagents I recommend gentle-pi, installed with `npm i -g gentle-pi` (adds the `gentle-shell` launcher) or `pi install npm:gentle-pi` (inside plain Pi); another subagent package is acceptable. As the recommended setup I recommend the full Gentle AI framework (https://github.com/Gentleman-Programming/gentle-ai): its methodology with RDD and the Engram memory stack, set up for Pi with `gentle-ai install --agent pi`. Then ask me to rebuild.

   There is no fallback: without a subagent package, gabo does not build on Pi.

## Inputs

| input | where |
|---|---|
| agent spec | `.gabo/skills/gabo-agent-templates/agents.yaml` (every entry is built) |
| body templates | `.gabo/skills/gabo-agent-templates/<template>` |
| version | `.gabo/VERSION`, one line `x.y.z` |
| adapters | `slots.<slot>.adapter` in `gabo.yaml` (a `SKILL.md` path, or an agent file) |

## Build

### Agents

For each spec entry write `.pi/agents/<id>.md` (gentle-pi reads `name`, `description`, `tools` and the body; other keys are ignored):

```markdown
---
name: <id>
description: "<role>"
gabo-version: <version>
tools:
  - <pi tool>
---
Skills: before work read `.pi/skills/<name>/SKILL.md` for each of <spec skills, then the fixed-name links the body names>.

<template body after its first line `gabo-version: {{gabo_version}}` and the blank line that follows, slots filled>
```

- `description` is the spec `role` in double quotes, so a colon in it stays plain text.
- `gabo-version: <version>` is the template's version line, moved into the frontmatter with the value of `.gabo/VERSION`; `gabo check` compares it with the stamp.
- The `Skills:` line is the spec's `skills` written where Pi can see it (a Pi agent has no skill-list field; the body is its system prompt). Omit the line when the list is empty.
- `tools`: map each spec capability, in spec order, dropping duplicates. List only tools the session really has: an unknown name fails the subagent at launch.

| capability | Pi tool |
|---|---|
| `read`, `skills` | `read` |
| `search` | `grep`, `find`, `ls` |
| `edit` | `edit`, `write` |
| `shell`, `git`, `tracker`, `repository-host` | `bash` |
| `memory` | `bash`; MCP memory tools (Engram) need no entry, Pi keeps MCP tools whatever `--tools` lists |
| `web` | the web tools of an installed package (pi-web-access: `web_search`, `fetch_content`); none: omit and report |
| `subagents` | gentle-pi: `subagent_run`, `subagent_status`, `subagent_result`, `subagent_continue`, `subagent_reply`, `subagent_cancel`, `subagent_list_agents`; another package: its own names |

Slots:

| slot | Pi value |
|---|---|
| `{{methodology}}` | `gabo-methodology` |
| `{{adapter:<slot>}}` | `gabo-<slot>` with `_` written `-` (for example `gabo-repository-host`) |
| `{{reviewer_script}}` | `${GENTLE_PI_AGENT_HOME:-${PI_CODING_AGENT_DIR:-$HOME/.pi/agent}}/skills/gabo-reviewer/scripts/reviewer.sh` (expanded by `bash` inside the template's double quotes, so the committed file holds no personal path; standalone `gentle-shell` sets both variables to its home) |

### Skills and fixed-name links

All links are relative symbolic links, so the committed tree works in every clone. Pi follows linked skill directories and skips a second path to the same file.

- Every skill in any spec entry's `skills`: `.pi/skills/<skill>` → `../../.gabo/skills/<skill>`.
- Every slot whose adapter is a `SKILL.md`: the fixed-name link `gabo-<slot>` (`_` written `-`), for example `.pi/skills/gabo-methodology` → the adapter's directory, relative to `.pi/skills/`. Do not also place the adapter under its own name. Pi lists a skill by its frontmatter `name` with its path, so the agent finds it at `.pi/skills/gabo-<slot>/SKILL.md`.
- A slot whose adapter is an agent file (the reviewer agent covering `reviewer_placement`) gets no link; report it.
- Remove a `.pi/skills/gabo-*` link that no spec skill or slot asks for any more; never touch other entries.

### Reviewer link (user level)

`gabo init` installs the reviewer at `${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`. Link it into the active Pi agent home, detected in this order (gentle-pi resolution): `GENTLE_PI_AGENT_HOME`, then `PI_CODING_AGENT_DIR` (an empty value counts as unset), then `~/.pi/agent`. Standalone `gentle-shell` runs in `~/.gentle-shell/agent` and sets both variables for its Pi, so link there too when the user starts gabo agents from `gentle-shell` without `--link` (`--link` reuses `~/.pi/agent`).

- `<agent-home>/skills/gabo-reviewer` → the user-level copy. An existing link to it is left; anything else at that path: stop and report, never replace it.
- The scripts resolve their own real path, so they run from the trusted user-level copy through the link.

## Verify

1. **Probe agent.** Build one probe agent: write `.pi/agents/gabo-probe.md` (`name: gabo-probe`, `tools:` `read`, `ls`; body: `List the files in .pi/agents and reply with their names.`). Run it once with `subagent_run`; it must reply with the agent names. Then delete the probe file: it is not a spec agent.
2. Run `gabo check`; fix every failure it reports and rerun until it passes.

## Sources

Checked 2026-11-03 against Pi 1.4.0 and gentle-pi 4.5.1 (installed packages and their docs; 4.5.1 still registers the seven `subagent_*` tools above and discovers `<cwd>/.pi/agents/`):

| fact | source |
|---|---|
| project skills in `.pi/skills/` (also `.agents/skills/`), linked directories followed, same file skipped | https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/skills.md; Pi 1.4.0 `dist/core/skills.js` |
| `pi list`, `pi install <source> [-l]`; npm packages under `<agent-home>/npm/`; git packages cloned to `<agent-home>/git/<host>/<path>` (user) and `.pi/git/<host>/<path>` (project, after trust) | https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md; Pi 1.4.0 `dist/core/package-manager.js` `getGitInstallPath`, `getGitInstallRoot`; gentle-pi 4.5.1 `docs/readme-reference.md` (Git-managed `agent-home/git/github.com/Gentleman-Programming/gentle-pi`) |
| agents in `<cwd>/.pi/agents/`, `<cwd>/.pi/subagents/` and the home's `agents/`, `subagents/` (project beats global); frontmatter `name`, `description`, `tools`; `subagent_*` tools; `subagents.json` owned by the package | gentle-pi 4.5.1 `docs/gentle-shell.md` (Subagents), `lib/agents-config.ts` |
| agent home `GENTLE_PI_AGENT_HOME`, then `PI_CODING_AGENT_DIR`, then `~/.pi/agent`; `gentle-shell` home `~/.gentle-shell/agent`, `--link` | gentle-pi 4.5.1 `docs/gentle-shell.md`, `docs/readme-reference.md`, `lib/agent-home.ts` |

Recheck these against `pi --version` and the installed gentle-pi when either is newer, and update this table.

## Output Contract

Report the prerequisite found (package and version), each agent file written with its id and version, each skill and fixed-name link placed, any slot left unlinked, the reviewer link and the agent home it went to, the tools omitted, the probe reply and the `gabo check` result.
