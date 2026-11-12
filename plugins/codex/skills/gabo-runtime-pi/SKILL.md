---
name: gabo-runtime-pi
description: "Trigger: build gabo agents for Pi, rebuild Pi agents, runtimes pi, .pi/agents, .agents/skills, gentle-pi, link reviewer into Pi. Builds every spec agent as a Pi subagent and places skills and links."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
  agent_dir: .pi/agents
  skill_dir: .agents/skills
  former_skill_dir: .pi/skills
  agent_skills: "line:Skills:"
  agent_name: frontmatter:name
  agent_tools: frontmatter:tools
  agent_tool_requirements:
    read: [read]
    skills: [read]
    search: [grep, find, ls]
    edit: [edit, write]
    shell: [bash]
    git: [bash]
    tracker: [bash]
    repository-host: [bash]
    memory: [mem_search, mem_save, mem_update, mem_get_observation]
    web: []
  agent_tools_unavailable: [subagent_*]
  main_session:
    kind: block
    files: [AGENTS.override.md, AGENTS.md, CLAUDE.md]
---
## Activation Contract

Load when a value of `runtimes.*` in `gabo.yaml` is `pi` and the gabo agents must be built or rebuilt: during setup, after `gabo init` refreshed the catalog, or when `gabo check` reports a Pi agent missing or built with another version (TRD 8.1, 9.4, D21, D24). `gabo check` reads `metadata.agent_dir` above and looks for the agents only in `.pi/agents`; `metadata.skill_dir` is where it requires the skill and fixed-name links, the `.agents/skills` that Codex and OpenCode share (intent 0008 S3), and `metadata.former_skill_dir` is where this skill made its links before the move, which `gabo check` reports when a gabo link is left there (S4).

## Hard Rules

- Pi core has no subagents. Build nothing until a subagent package is present (Prerequisite); then refuse and say what to install.
- Engine config (intent 0004 S8, D7): gabo writes only the engine keys it needs to operate, at project level only, merged with the existing project config (every unrelated key kept), reports what it wrote, and has the value verified by `gabo check`; never the user's global engine config: no write to `settings.json`, `models.json`, `mcp.json` or anything under the agent home except the one `skills/gabo-reviewer` link. `gabo init` writes every engine config key (intent 0010 S5); this skill writes none, no Pi config included, and never runs `pi install`. Never edit `subagents.json`: the subagent package owns it. Never run `pi install`, `pi config` or `npm i` yourself; the user installs.
- Never place `gabo-reviewer` in the project, not as a copy and not as a link (D20): PR code must not reach the scripts that review it.
- Read the spec, templates and version from the project copy (`.gabo/`), never from memory or the binary. Fill only the template slots (`gabo-agent-templates`); invent no rule, tool or skill.
- Write only `.pi/agents/<id>.md` for spec ids, `.agents/skills/` links, the `.agents/skills/gabo-assistant` link and the scoped block of the Main-session entry, and the user-level reviewer link; remove only the `.pi/skills/gabo-*` links of the Migration step, the `.agents/skills/gabo-*` links the shared-directory rule allows and the main-session entries of other engines that section removes. Leave every other file in `.pi/` and `.agents/` untouched.

## Prerequisite

Run in the repository root, read-only:

1. `pi --version` must succeed; else refuse: `Pi is not installed: see https://github.com/earendil-works/pi`.
2. `pi list` (lists user and project packages, writes nothing). A subagent package is present when it lists `npm:gentle-pi` or a `git:` source of gentle-pi, or, once `pi list` ran and exited 0, `gentle-shell` is on `PATH` (`npm i -g gentle-pi` ships it with its own Pi); a `pi list` that fails or times out counts as no package.
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
gabo-engine: pi
tools:
  - <pi tool>
---
Skills: before work read `.agents/skills/<name>/SKILL.md` for each of <spec skills, then the fixed-name links the body names>.

<template body after its first two lines `gabo-version: {{gabo_version}}` and `gabo-engine: {{engine}}` and the blank line that follows, slots filled>
```

- `description` is the spec `role` in double quotes, so a colon in it stays plain text.
- The agent file has no turn-limit key: the coder gets no engine key for the turn limit; it travels in the launch input as `turn limit: <n>` and is advisory.
- `gabo-version: <version>` is the template's version line, moved into the frontmatter with the value of `.gabo/VERSION`; `gabo check` compares it with the stamp.
- `gabo-engine: pi` is the template's engine line, moved into the frontmatter with `pi`, the engine as written in `gabo.yaml`; `gabo check` fails an agent file whose filled engine is not `pi` (intent 0013 S6).
- The `Skills:` line is the spec's `skills` written where Pi can see it (a Pi agent has no skill-list field; the body is its system prompt). Omit the line when the list is empty.
- `tools`: map each spec capability, in spec order, dropping duplicates. List only tools the session really has: an unknown name fails the subagent at launch. List every tool the agent needs, extension tools included: gentle-pi starts a subagent as `pi --tools <its tools list>`, and `--tools` with plain names replaces the whole selection, so every tool the list does not name is not declared to the subagent, MCP and extension tools included (intent 0013 S17).
- `metadata.agent_tools` above says where an agent file lists its tools (the frontmatter `tools`), and `metadata.agent_tool_requirements` the tools each spec capability requires; `gabo check` reads both (intent 0013 D11). A required `mem_<x>` is also met by `mcp__<server>__mem_<x>` or a `mcp__<server>__*` entry.
- `metadata.agent_tools_unavailable` the tools a subagent never has (`subagent_*`, a trailing `*` matching a prefix): every `.pi/agents` file runs as a gentle-pi subagent, so `gabo check` fails an agent file that lists one with `✗ <agent file>: tools a subagent never has: <names>` (intent 0013 S20).

| capability | Pi tool |
|---|---|
| `read`, `skills` | `read` |
| `search` | `grep`, `find`, `ls` |
| `edit` | `edit`, `write` |
| `shell`, `git`, `tracker`, `repository-host` | `bash` |
| `memory` | `bash`, `mem_search`, `mem_save`, `mem_update`, `mem_get_observation` when `pi list` shows `npm:gentle-engram`; else `bash`, `mcp__<server>__mem_search`, `mcp__<server>__mem_save`, `mcp__<server>__mem_update`, `mcp__<server>__mem_get_observation` when Pi's MCP config has an Engram server named `<server>`; neither: `bash` only, omit the memory tools and report it |
| `web` | the web tools of an installed package (pi-web-access: `web_search`, `fetch_content`); none: omit and report |

Slots:

| slot | Pi value |
|---|---|
| `{{methodology}}` | `gabo-methodology` |
| `{{adapter:<slot>}}` | `gabo-<slot>` with `_` written `-` (for example `gabo-repository-host`) |
| `{{engine}}` | `pi` (intent 0013 S6) |
| `{{turn_limit_line}}` | with `wave.max_coder_turns` set: `Turn limit: <n> turns; stop at it and report STOPPED at turn limit.`; unset: remove the whole line; filled in `gabo-coder`, the only template that carries the slot |
| `{{reviewer_script}}` | `${GENTLE_PI_AGENT_HOME:-${PI_CODING_AGENT_DIR:-$HOME/.pi/agent}}/skills/gabo-reviewer/scripts/reviewer.sh` (expanded by `bash` inside the template's double quotes, so the committed file holds no personal path; standalone `gentle-shell` sets both variables to its home); filled in `gabo-coder`, the only template that carries the slot |

The Pi coder runs the user-level reviewer script itself like every other engine's coder (intent 0015 D5, which supersedes intent 0013 D15): the Pi coder body is the template with the slots above filled, and this skill rewrites none of its bullets.

### Skills and fixed-name links

All links are relative symbolic links, so the committed tree works in every clone. They go in `.agents/skills/`, which Pi reads as a project skill location besides `.pi/skills/`; Pi follows linked skill directories and skips a second path to the same file.

- Every skill in any spec entry's `skills`: `.agents/skills/<skill>` → `../../.gabo/skills/<skill>`. An entry that already resolves to that folder is kept (Codex or OpenCode may have made it); a real folder or file, or a link to another folder that this skill did not make, is left and reported.
- Every slot whose adapter is a `SKILL.md`: the fixed-name link `gabo-<slot>` (`_` written `-`), for example `.agents/skills/gabo-methodology` → the adapter's directory, relative to `.agents/skills/`. Do not also place the adapter under its own name. Pi lists a skill by its frontmatter `name` with its path, so the agent finds it at `.agents/skills/gabo-<slot>/SKILL.md`.
- A slot whose adapter is an agent file (the reviewer agent covering `reviewer_placement`) gets no link; report it.
- Remove each `.agents/skills/gabo-*` symlink that no spec skill or slot asks for any more and that this skill made: one that points into `../../.gabo/skills/` or to a folder a slot adapter of `gabo.yaml` named before. `.agents/skills/` is shared by every engine that reads it (Codex, Pi and OpenCode), so keep every link any engine declared in `runtimes.*` needs: an engine uses `.agents/skills` when its runtime skill `.gabo/skills/gabo-runtime-<engine>/SKILL.md` declares `metadata.skill_dir: .agents/skills`, and every such engine needs the same spec skills and fixed-name links (one catalog copy, one `gabo.yaml`); also keep `.agents/skills/gabo-assistant` while `runtimes.assistant` names an engine that uses `.agents/skills` (intent 0009 S1); with a list, while any engine listed in `runtimes.assistant` uses it (intent 0013 S4); also keep every `main_session.skills` link while `runtimes.assistant` names an engine that uses `.agents/skills` (intent 0015 D3). Former agents (intent 0015 S6, D4): on every build remove `.pi/agents/gabo-cycle.md` and `.pi/agents/gabo-reviewer.md`, left by an earlier build, only when the file carries a `gabo-version:` line; a file without one: leave it and report it. Leave and report any other `gabo-*` symlink, and never remove a real folder or file.

### Main-session entry

Pi has no project setting that selects the main-session agent, so the owner's main session gets the scoped block of the Pi row in the Main-Session Entry table of `gabo-agent-templates` (intent 0009 S1, S3, S4, D1, D3). Read the table and the templates from the project catalog; never write the block from memory. Run this on every build, after the links above.

- When `pi` is listed in `runtimes.assistant` (one engine or a list, intent 0013 S4): make `.agents/skills/gabo-assistant` a relative symlink to `../../.gabo/skills/gabo-assistant`, under the same keep, replace and report rules as the catalog skill links, and link every other `main_session.skills` entry (`gabo-assistant`, `gabo-cycle` and `gabo-wave-report` in all, intent 0015 D3) the same way as `.agents/skills/<name>`, to `../../.gabo/skills/<name>`. Then copy the `main_session.block` of `.gabo/skills/gabo-agent-templates/agents.yaml` verbatim, from `<!-- GABO_ASSISTANT_START -->` to `<!-- GABO_ASSISTANT_END -->`, into the first existing file of `AGENTS.override.md`, `AGENTS.md` and `CLAUDE.md` at the repository root, in that order (Pi loads only that one file of a directory, see Sources), with `AGENTS.md` created when none exists. When `codex` is listed too, write the block into `AGENTS.override.md` when it exists, else `AGENTS.md`, created when neither exists: the file Codex writes, which Pi then loads first, so the one block serves both and neither build removes the other's block. The block names `.gabo/skills/gabo-assistant/SKILL.md` and is at most 2048 bytes; a larger template: stop and report it. Keep exactly one block: remove a gabo block from the other files of that order.
- Writing the block: when the file holds the markers, replace only the bytes between the markers; append it after one blank line when absent (a file not ending in a newline first gets one); keep every other byte, so a second build leaves the file byte-identical. Two blocks in one file: refresh the first and remove the others. A file whose markers are unpaired: change nothing and report it. A symlinked file: write through a symlinked file only when its target stays inside the repository; otherwise leave it and report it.
- When `pi` is not listed in `runtimes.assistant`, remove its block from `AGENTS.override.md`, `AGENTS.md` and `CLAUDE.md`, but a block is removed only when no engine listed in `runtimes.assistant` uses its file (Codex uses the same files). Removing a block deletes the markers, the bytes between them and the one blank line written before it, and keeps every other byte; a file left empty is kept and reported, never deleted. The shared `.agents/skills/gabo-assistant` link: this skill removes the shared link only when no engine listed in `runtimes.assistant` uses `.agents/skills` (an engine uses `.agents/skills` when its runtime skill declares `metadata.skill_dir: .agents/skills`), and only when it points to `../../.gabo/skills/gabo-assistant`; it removes the `.agents/skills/<name>` link of every other `main_session.skills` entry under the same rule, keeping one while another spec agent's `skills` lists it. With `runtimes.assistant: codex` and `runtimes.coder: pi`, a Pi rebuild keeps `.agents/skills/gabo-assistant` and the Codex block.
- On every build, remove the gabo entry of every other full-tier engine in the Main-Session Entry table of `gabo-agent-templates` that is not listed in `runtimes.assistant` (TRD D28: Claude Code, OpenCode, Codex), except any link or block a listed engine's row writes (Codex and OpenCode share `.agents/skills/gabo-assistant`, and Codex shares the block files), recognized only by its gabo identity: for Claude Code, `.claude/agents/gabo-assistant.md` when it carries a `gabo-version:` line and the `.claude/skills/gabo-assistant` symlink to the catalog folder and the `.claude/skills/<name>` symlinks of the other `main_session.skills` entries; for OpenCode, `.opencode/agents/gabo-assistant.md` when it carries a `gabo-version:` line; for Codex, its block in `AGENTS.override.md` and `AGENTS.md`, removed as above. Anything without that identity (an agent file or link of the user's own): keep and report it. A build never removes the entry of another engine listed in `runtimes.assistant`, so rebuilding the listed engines in any order leaves every listed entry in place. With `runtimes.assistant: [claude, pi]`, a Pi build keeps `.claude/agents/gabo-assistant.md` and the `.claude/skills/gabo-assistant` link. Never remove Claude Code's `agent` key or OpenCode's `default_agent` key: `gabo init` removes them (intent 0010 S4), and this skill writes and removes no engine config key.

### Migration from `.pi/skills`

Earlier builds placed the links in `.pi/skills/` (intent 0003). On every build, after the links above, remove each `.pi/skills/gabo-*` symlink that this skill made: one that points into `../../.gabo/skills/` or to a slot adapter folder of `gabo.yaml` (relative to `.pi/skills/`). Leave and report any other entry of `.pi/skills/`, and never remove a real folder or file. A missing `.pi/skills/` needs nothing; leave the directory itself even when it is empty. `gabo check` fails on such a leftover: `✗ .pi/skills/<name>: gabo link left after the move to .agents/skills`, with `next: ask your agent to rebuild the agents`.

### Reviewer link (user level)

`gabo init` installs the reviewer at `${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`. Link it into the active Pi agent home, detected in this order (gentle-pi resolution): `GENTLE_PI_AGENT_HOME`, then `PI_CODING_AGENT_DIR` (an empty value counts as unset), then `~/.pi/agent`. Standalone `gentle-shell` runs in `~/.gentle-shell/agent` and sets both variables for its Pi, so link there too when the user starts gabo agents from `gentle-shell` without `--link` (`--link` reuses `~/.pi/agent`).

- `<agent-home>/skills/gabo-reviewer` → the user-level copy. An existing link to it is left; anything else at that path: stop and report, never replace it.
- The scripts resolve their own real path, so they run from the trusted user-level copy through the link.

## Grants for an unattended Pi wave

The three grants an unattended Pi wave needs are the owner's to give for the session, not gabo's:

1. the worktree root `worktree.dir` of `gabo.yaml`, where each task worktree is created;
2. the user file `${XDG_CONFIG_HOME:-~/.config}/gabo/config.yaml`, which the assistant reads at session start;
3. the user-level reviewer skill folder `${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer` and its state directory `${XDG_STATE_HOME:-~/.local/state}/gabo/reviewer`, which the coder's review runs from.

The grant is the owner's: gabo grants none of them and writes no permission to the user's engine config. The only file this skill writes under the agent home is the reviewer link above.

## Launching Agents

On Pi the main session launches each coder as one first-level `subagent_run` of `gabo-coder` and resumes it with `subagent_continue`; the run id is the handle. No built agent launches another (intent 0015 S6, D7): the agent files carry no `subagent_*` tool.

## Verify

1. **Probe agent.** Build one probe agent: write `.pi/agents/gabo-probe.md` (`name: gabo-probe`, `tools:` `read`, `ls`; body: `List the files in .pi/agents and reply with their names.`). Run it once with `subagent_run`; it must reply with the agent names. Then delete the probe file: it is not a spec agent.
2. Run `gabo check`; fix every failure it reports and rerun until it passes.

## Sources

Checked 2026-11-03 against Pi 1.4.0 and gentle-pi 4.5.1 (installed packages and their docs; 4.5.1 still registers the seven `subagent_*` tools above and discovers `<cwd>/.pi/agents/`):

| fact | source |
|---|---|
| project skills in `.pi/skills/` and `.agents/skills/`, linked directories followed, same file skipped, the first of two same-name skills kept with a warning | https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/skills.md; Pi 1.4.0 `dist/core/skills.js` |
| `pi list`, `pi install <source> [-l]`; npm packages under `<agent-home>/npm/`; git packages cloned to `<agent-home>/git/<host>/<path>` (user) and `.pi/git/<host>/<path>` (project, after trust) | https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md; Pi 1.4.0 `dist/core/package-manager.js` `getGitInstallPath`, `getGitInstallRoot`; gentle-pi 4.5.1 `docs/readme-reference.md` (Git-managed `agent-home/git/github.com/Gentleman-Programming/gentle-pi`) |
| agents in `<cwd>/.pi/agents/`, `<cwd>/.pi/subagents/` and the home's `agents/`, `subagents/` (project beats global); frontmatter `name`, `description`, `tools`; `subagent_*` tools; `subagents.json` owned by the package | gentle-pi 4.5.1 `docs/gentle-shell.md` (Subagents), `lib/agents-config.ts` |
| Pi 1.4.0 loads only the first existing file per directory of its context files, in the order `AGENTS.override.md`, then `AGENTS.md`, then `CLAUDE.md`, into every Pi process in the project, `pi --print` included (the Main-session entry file order, intent 0009 D6) | Pi 1.4.0 `dist/core/resource-loader.js` (line 116), probed with `pi --print`: https://github.com/testeos-z/gabo-ai/issues/324#issuecomment-6690677042 |
| a subagent starts as `pi --tools <its tools list>` (`childArguments`); `--tools` with plain names replaces the whole selection, so an unnamed tool, MCP or extension, is not declared; the `subagent_*` tools are registered only in the primary session, not in a child (`GENTLE_PI_AGENTS_CHILD=1`) | Pi 1.4.0 `docs/cli.md` (Tools), `docs/mcp.md`; gentle-pi 4.5.1 `lib/agents-runner.ts` `childArguments`, `extensions/gentle-agents.ts`; probe P2a: https://github.com/testeos-z/gabo-ai/issues/414#issuecomment-6701730785 |
| the Engram tools `mem_search`, `mem_save`, `mem_update`, `mem_get_observation` (with the other `mem_*` tools) registered by the gentle-engram Pi package | gentle-engram 0.4.2 `index.ts` `ENGRAM_TOOLS` |
| agent home `GENTLE_PI_AGENT_HOME`, then `PI_CODING_AGENT_DIR`, then `~/.pi/agent`; `gentle-shell` home `~/.gentle-shell/agent`, `--link` | gentle-pi 4.5.1 `docs/gentle-shell.md`, `docs/readme-reference.md`, `lib/agent-home.ts` |

Recheck these against `pi --version` and the installed gentle-pi when either is newer, and update this table.

## Output Contract

Report the prerequisite found (package and version), each agent file written with its id and version, each skill and fixed-name link placed, each `.pi/skills/` link removed and each entry left with why, the main-session entry (the `gabo-assistant` link and the block with its file written, refreshed, removed or left with why) and each other engine's entry removed or kept, any slot left unlinked, the reviewer link and the agent home it went to, the tools omitted, the probe reply and the `gabo check` result.
