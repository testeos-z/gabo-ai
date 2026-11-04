---
name: gabo-runtime-claude
description: "Trigger: build gabo agents for Claude Code, rebuild agents after gabo init, runtime claude, gabo check reports a missing agent, skill link or reviewer link on claude. Builds the gabo agents, skills and fixed-name links for Claude Code."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
  agent_dir: .claude/agents
  skill_dir: .claude/skills
  reviewer_link: ${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gabo-reviewer
---
## Activation Contract

Load when any of `runtimes.coder`, `runtimes.reviewer` or `runtimes.assistant` in `gabo.yaml` is `claude` and the gabo agents must be built or rebuilt: after setup, after `gabo init` refreshed the catalog, or when `gabo check` reports `✗ agent <id> missing` or an agent built with another version (TRD 8.1, 9.4, D24). Load `gabo-agent-templates` with it: that skill owns the spec, the templates and the slot rules; this skill only turns them into Claude Code files.

`metadata.agent_dir` above is the directory `gabo check` searches for the agent files of this runtime; `metadata.skill_dir` is where it requires the links of build steps 2 and 3, and `metadata.reviewer_link` the user-level link of step 4 (outside CI). Never change them without changing where you build. `gabo check` reports a missing or wrong link as `✗ skill link <path> …` or `✗ reviewer link <path> …`: rebuild with this skill.

## Hard Rules

- Build every agent of `.gabo/skills/gabo-agent-templates/agents.yaml`, each as `.claude/agents/<id>.md`, whatever runtime each agent plays: `gabo check` requires all of them in `.claude/agents`.
- Read the spec, the templates and `.gabo/VERSION` from the project; never build from memory. A missing or empty `.gabo/VERSION` or catalog: stop and tell the user to run `gabo init`.
- Fill only the template slots, as below, and invent nothing: no frontmatter key, tool, skill, model or rule the spec and template do not give.
- Never place `gabo-reviewer` in the project: no `.claude/skills/gabo-reviewer`, no copy of its scripts, and never in an agent's `skills`. Only link the user-level copy that `gabo init` installs (D20).
- Overwrite only the agent files named after a spec id, the links this skill makes and the probe file it writes. Any other existing entry at a path you would write (a real file or folder, or a link to another target): leave it and report it; never delete or move it.
- Make every link of steps 2 to 4 on every build, also when the agent files are already current, and run the probe of step 8; a build that skips one is not finished.
- Write no secret into any file (R21).
- Never install or upgrade Claude Code. Engine config (intent 0004 S8, D7): gabo writes only the engine keys it needs to operate, at project level only, merged with the existing project config (every unrelated key kept), reports what it wrote, and has the value verified by `gabo check`; never the user's global engine config (`~/.claude/settings.json` or `$CLAUDE_CONFIG_DIR/settings.json`). Claude Code needs no engine key today, so this skill writes no settings file.

## Sources

Checked on 2026-10-28 against Claude Code 2.1.302 (`claude --version`), its `claude --help` and `claude plugin validate --help`, and these pages (former TRD 9.6 rows):

| fact | used for | source |
|---|---|---|
| Project agents are Markdown files with YAML frontmatter in `.claude/agents/<id>.md`; the directory is scanned recursively and identity comes only from `name`. | agent files | https://code.claude.com/docs/en/sub-agents |
| `name` and `description` are required; a file without them, or whose YAML does not parse, is skipped silently. | frontmatter | https://code.claude.com/docs/en/sub-agents |
| `tools` is a comma-separated allowlist of tool names; `mcp__<server>__*` gives one MCP server's tools; `Agent` spawns subagents. | tool list | https://code.claude.com/docs/en/sub-agents |
| `skills` preloads each listed skill's full content at startup; a missing skill is skipped with a debug-log warning. | skill list | https://code.claude.com/docs/en/sub-agents |
| Project skills live in `.claude/skills/<name>/SKILL.md`; a `.claude/skills/<name>` entry may be a symlink to a folder, and the folder name also invokes the skill. | catalog skills, fixed-name links | https://code.claude.com/docs/en/skills |
| A subagent's `skills` entry naming a `.claude/skills/<link>` symlink preloads the target folder's skill even when its frontmatter `name` differs. Verified by probe on Claude Code 2.1.302 (four links `gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory` to folders named `sample-*`); the probe in step 8 rechecks it on every build. | fixed-name preload | https://code.claude.com/docs/en/skills, probe |
| Personal skills live in `~/.claude/skills/<name>/SKILL.md` and win over a project skill of the same name. | reviewer link | https://code.claude.com/docs/en/skills |
| `CLAUDE_CONFIG_DIR`, when set in the environment Claude Code starts from, replaces `~/.claude` as the user configuration directory. | reviewer link | https://code.claude.com/docs/en/env-vars |
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
2. Catalog skills: for each skill in any agent's spec `skills`, make `.claude/skills/<name>` a relative symlink to `../../.gabo/skills/<name>`, so a later `gabo init` refresh reaches it. An entry that already resolves to that folder is kept. A symlink this skill made earlier to another folder (for example the CLI source `apps/cli/internal/skills/<name>`) is replaced; a real folder or file is left and reported.
3. Fixed-name links as above. Then remove each `.claude/skills/gabo-*` symlink that no spec skill or slot asks for any more; never remove a real folder or file.
4. Reviewer link: make `~/.claude/skills/gabo-reviewer` (with `CLAUDE_CONFIG_DIR` set: `$CLAUDE_CONFIG_DIR/skills/gabo-reviewer`) a symlink to `${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer` with the absolute path. That folder missing: stop and tell the user to run `gabo init`. An existing entry with another target: leave it and report it.
5. For each spec agent, fill its template:

   | slot | value |
   |---|---|
   | `{{id}}`, `{{role}}` | the spec `id` and `role` |
   | `{{gabo_version}}` | the single line of `.gabo/VERSION` |
   | `{{methodology}}` | `gabo-methodology` |
   | `{{adapter:tasks}}`, `{{adapter:repository_host}}`, `{{adapter:memory}}` | `gabo-tasks`, `gabo-repository-host`, `gabo-memory` |
   | `{{reviewer_script}}` | `${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gabo-reviewer/scripts/reviewer.sh`, written literally: the shell expands it on each machine |

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
7. Verify: run `claude plugin validate .claude/agents`, then `gabo check`, which also checks every link of steps 2 to 4. Fix every failure this skill caused and rerun until `gabo check` passes, except the `✗ agent <id> missing` lines of agents you skipped and reported in step 5; report any other failure.
8. Probe, on every build. When `.claude/agents` or `.claude/skills` did not exist when this Claude Code session started (you created it in this build), the session cannot see the new agent or links: skip the probe, change nothing, and report `probe not run: restart Claude Code and rebuild`. Otherwise write `.claude/agents/gabo-probe.md` in the same format, with `tools: Bash`, `skills:` listing every fixed-name link made in step 3, and the body: run `git rev-parse --show-toplevel` once and reply with its output, then one line per preloaded skill with the name it was listed under and its first Markdown heading. Launch it once, then delete the file whatever the result. It passes when it ran the command and every link shows its adapter's heading. Only a probe that ran in such a session can fail a link. A link that does not preload: in `skills:` and in its slot fill, use the adapter's frontmatter `name` instead of the link name, rebuild the affected agents, repeat step 7 and probe once more; still failing, stop and report the link. A `gabo-probe.md` that this skill did not write: leave it and report it, and report the probe as not run.

## Output Contract

Report the Claude Code version and whether the sources were rechecked; each agent file written with its id and `gabo-version`; each link made (catalog, fixed-name and reviewer); each entry left or slot not filled, with why; the `claude plugin validate` and `gabo check` results; and the probe result when it ran.
