---
name: gabo-runtime-claude
description: "Trigger: build gabo agents for Claude Code, rebuild agents after gabo init, runtime claude, gabo check reports a missing agent on claude. Builds the gabo agents, skills and fixed-name links for Claude Code."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
  agent_dir: .claude/agents
---
## Activation Contract

Load when any of `runtimes.coder`, `runtimes.reviewer` or `runtimes.assistant` in `gabo.yaml` is `claude` and the gabo agents must be built or rebuilt: after setup, after `gabo init` refreshed the catalog, or when `gabo check` reports `✗ agent <id> missing` or an agent built with another version (TRD 8.1, 9.4, D24). Load `gabo-agent-templates` with it: that skill owns the spec, the templates and the slot rules; this skill only turns them into Claude Code files.

`metadata.agent_dir` above is the directory `gabo check` searches for the agent files of this runtime. Never change it without changing where you build.

## Hard Rules

- Build every agent of `.gabo/skills/gabo-agent-templates/agents.yaml`, each as `.claude/agents/<id>.md`, whatever runtime each agent plays: `gabo check` requires all of them in `.claude/agents`.
- Read the spec, the templates and `.gabo/VERSION` from the project; never build from memory. A missing or empty `.gabo/VERSION` or catalog: stop and tell the user to run `gabo init`.
- Fill only the template slots, as below, and invent nothing: no frontmatter key, tool, skill, model or rule the spec and template do not give.
- Never place `gabo-reviewer` in the project: no `.claude/skills/gabo-reviewer`, no copy of its scripts, and never in an agent's `skills`. Only link the user-level copy that `gabo init` installs (D20).
- Overwrite only the agent files named after a spec id and the links this skill makes. Any other existing entry at a path you would write (a real file or folder, or a link to another target): leave it and report it; never delete or move it.
- Write no secret into any file (R21).

## Sources

Checked on 2026-10-28 against Claude Code 2.1.302 (`claude --version`), its `claude --help` and `claude plugin validate --help`, and these pages (former TRD 9.6 rows):

| fact | used for | source |
|---|---|---|
| Project agents are Markdown files with YAML frontmatter in `.claude/agents/<id>.md`; the directory is scanned recursively and identity comes only from `name`. | agent files | https://code.claude.com/docs/en/sub-agents |
| `name` and `description` are required; a file without them, or whose YAML does not parse, is skipped silently. | frontmatter | https://code.claude.com/docs/en/sub-agents |
| `tools` is a comma-separated allowlist of tool names; `mcp__<server>__*` gives one MCP server's tools; `Agent` spawns subagents. | tool list | https://code.claude.com/docs/en/sub-agents |
| `skills` preloads each listed skill's full content at startup; a missing skill is skipped with a debug-log warning. | skill list | https://code.claude.com/docs/en/sub-agents |
| Project skills live in `.claude/skills/<name>/SKILL.md`; a `.claude/skills/<name>` entry may be a symlink to a folder, and the folder name also invokes the skill. | catalog skills, fixed-name links | https://code.claude.com/docs/en/skills |
| Personal skills live in `~/.claude/skills/<name>/SKILL.md` and win over a project skill of the same name. | reviewer link | https://code.claude.com/docs/en/skills |
| Edits under `.claude/agents/` load without restart, unless the folder did not exist when the session started. | rebuild | https://code.claude.com/docs/en/sub-agents |
| `claude plugin validate <dir>` reports agent files whose frontmatter does not parse (2.1.233 or later); `--strict` turns the unknown `gabo-version` key into an error, so never pass it. | verify | https://code.claude.com/docs/en/sub-agents |

Recheck before you build: run `claude --version`. When it is not 2.1.302, read both pages and `claude --help` again and confirm each row. A row that no longer holds: stop, build nothing and report the row with what you found.

## Tool Map

The spec's engine-neutral capabilities become these Claude Code tools. Write them in this table's order, each tool once.

| capability | Claude Code tools |
|---|---|
| `read` | Read |
| `search` | Grep, Glob |
| `edit` | Edit, Write |
| `shell` | Bash |
| `git` | Bash |
| `tracker` | Bash |
| `repository-host` | Bash |
| `memory` | `mcp__<server>__*` for each MCP server (from `claude mcp list`) whose tools the memory adapter names; Bash when it names none, reported as unwired |
| `web` | WebFetch, WebSearch |
| `skills` | Skill |
| `subagents` | Agent |

## Fixed-Name Links

Each slot of `gabo.yaml` (`slots.<name>.adapter`) whose adapter is a `<folder>/SKILL.md` gets one link `.claude/skills/gabo-<name>` (underscores become hyphens) to `<folder>`, as a relative symlink: `gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory` and so on. An adapter that is not a skill folder (for example the reviewer agent file) gets no link: report it. When the adapter folder already is `.claude/skills/gabo-<name>`, there is nothing to link.

## Build Steps

1. Recheck the sources. Read `gabo.yaml`, `.gabo/VERSION` and the spec.
2. Catalog skills: for each skill in any agent's spec `skills`, make `.claude/skills/<name>` a relative symlink to `../../.gabo/skills/<name>`, so a later `gabo init` refresh reaches it. An entry that already resolves to that folder is kept.
3. Fixed-name links as above.
4. Reviewer link: make `~/.claude/skills/gabo-reviewer` a symlink to `${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer` with the absolute path. That folder missing: stop and tell the user to run `gabo init`. An existing entry with another target: leave it and report it.
5. For each spec agent, fill its template:

   | slot | value |
   |---|---|
   | `{{id}}`, `{{role}}` | the spec `id` and `role` |
   | `{{gabo_version}}` | the single line of `.gabo/VERSION` |
   | `{{methodology}}` | `gabo-methodology` |
   | `{{adapter:tasks}}`, `{{adapter:repository_host}}`, `{{adapter:memory}}` | `gabo-tasks`, `gabo-repository-host`, `gabo-memory` |
   | `{{reviewer_script}}` | `$HOME/.claude/skills/gabo-reviewer/scripts/reviewer.sh`, written literally: the shell expands it on each machine |

   A slot whose link does not exist (its slot has no skill adapter) cannot be filled: skip that agent and report it.
6. Write `.claude/agents/<id>.md`: the frontmatter below, then the filled body without its first line `gabo-version: …` and the blank line after it. Frontmatter keys, in this order and no others:

   ```yaml
   ---
   name: <id>
   description: '<role, with each single quote doubled>'
   tools: <Tool Map of the spec tools, comma-separated>
   skills:
     - <each spec skill, in spec order>
     - <each fixed-name link the body names: gabo-methodology, gabo-tasks, gabo-repository-host, gabo-memory, in this order>
   gabo-version: <x.y.z, unquoted>
   ---
   ```

   Omit `skills` when the list is empty (the reviewer agent).
7. Verify: run `claude plugin validate .claude/agents`, then `gabo check`. Fix every failure this skill caused and rerun until `gabo check` passes; report any other failure.
8. Probe, on the first build in a repository or after a recheck found a new version: write `.claude/agents/gabo-probe.md` in the same format (`tools: Bash`, `skills: [gabo-methodology]` or another link, body: run `git rev-parse --show-toplevel` and reply with its output and the first heading of each preloaded skill). Launch it once, check that it ran the command and saw the skill, then delete the file.

## Output Contract

Report the Claude Code version and whether the sources were rechecked; each agent file written with its id and `gabo-version`; each link made; each entry left or slot not filled, with why; the `claude plugin validate` and `gabo check` results; and the probe result when it ran.
