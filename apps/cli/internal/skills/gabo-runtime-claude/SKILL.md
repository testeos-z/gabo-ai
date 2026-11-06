---
name: gabo-runtime-claude
description: "Trigger: build gabo agents for Claude Code, rebuild agents after gabo init, runtime claude, gabo check reports a missing agent, skill link or reviewer link on claude. Builds the gabo agents, skills and fixed-name links for Claude Code."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
  agent_dir: .claude/agents
  skill_dir: .claude/skills
  agent_skills: frontmatter:skills
  agent_name: frontmatter:name
  reviewer_link: ${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gabo-reviewer
---
## Activation Contract

Load when any of `runtimes.coder`, `runtimes.reviewer` or `runtimes.assistant` in `gabo.yaml` is `claude` and the gabo agents must be built or rebuilt: after setup, after `gabo init` refreshed the catalog, or when `gabo check` reports `✗ agent <id> missing` or an agent built with another version (TRD 8.1, 9.4, D24). Load `gabo-agent-templates` with it: that skill owns the spec, the templates and the slot rules; this skill only turns them into Claude Code files, including the owner's main-session entry of its Main-Session Entry table (spec 0009).

`metadata.agent_dir` above is the directory `gabo check` searches for the agent files of this runtime; `metadata.skill_dir` is where it requires the links of build steps 2 and 3, and `metadata.reviewer_link` the user-level link of step 4 (outside CI). Never change them without changing where you build. `gabo check` reports a missing or wrong link as `✗ skill link <path> …` or `✗ reviewer link <path> …`: rebuild with this skill.

## Hard Rules

- Build every agent of `.gabo/skills/gabo-agent-templates/agents.yaml`, each as `.claude/agents/<id>.md`, whatever runtime each agent plays: `gabo check` requires all of them in `.claude/agents`.
- Read the spec, the templates and `.gabo/VERSION` from the project; never build from memory. A missing or empty `.gabo/VERSION` or catalog: stop and tell the user to run `gabo init`.
- Fill only the template slots, as below, and invent nothing: no frontmatter key, tool, skill, model or rule the spec and template do not give.
- Never place `gabo-reviewer` in the project: no `.claude/skills/gabo-reviewer`, no copy of its scripts, and never in an agent's `skills`. Only link the user-level copy that `gabo init` installs (D20).
- Overwrite only the agent files named after a spec id, the links this skill makes, the main-session entry below and the probe file it writes. Any other existing entry at a path you would write (a real file or folder, or a link to another target): leave it and report it; never delete or move it.
- Make every link of steps 2 to 4 on every build, also when the agent files are already current, and run the probe of step 8; a build that skips one is not finished.
- Write no secret into any file (R21).
- Never install or upgrade Claude Code. Engine config (intent 0004 S8, D7): gabo writes only the engine keys it needs to operate, at project level only, merged with the existing project config (every unrelated key kept), reports what it wrote, and has the value verified by `gabo check`; never the user's global engine config (`~/.claude/settings.json` or `$CLAUDE_CONFIG_DIR/settings.json`). `gabo init` writes the project-level gabo integration keys (intent 0006); this skill writes only the one main-session key `agent` in `.claude/settings.json` (Main-Session Entry).

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
| `claude --agent <name>` runs the main thread as that agent; `agent` in the project `.claude/settings.json` makes it the default for every session in the project; the CLI flag overrides the setting. A non-empty agent body replaces the default system prompt and reaches the main session, but the agent's `skills` do not preload in this mode (probe of spec 0009 T2 on Claude Code 2.1.302, #321). | main-session entry | https://code.claude.com/docs/en/sub-agents, probe |
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

## Main-Session Entry

The owner's main session runs as the gabo assistant (spec 0009 S1, S2, S4). Read the Main-Session Entry table of `gabo-agent-templates` and the `main_session` entry of its `agents.yaml`; this section only says how the Claude Code row is written and how every other row is removed.

When `runtimes.assistant` is `claude`, on every build:

1. Link `.claude/skills/gabo-assistant` as a relative symlink to `../../.gabo/skills/gabo-assistant` (the catalog folder `.gabo/skills/gabo-assistant`), as build step 2 links a catalog skill (the same keep, replace and report rules). Step 3 never removes it.
2. Write `.claude/agents/gabo-assistant.md`: fill `main_session.template` as build step 5 fills a spec template, then write the frontmatter below and the filled body without its first line `gabo-version: …` and the blank line after it. The body is the short assistant prompt (spec 0009 D2 A: the agent's `skills` do not preload in a main session), never an empty body; it names `.gabo/skills/gabo-assistant/SKILL.md`. Keys, in this order and no others; omit `tools`, so the main session keeps every tool:

   ```yaml
   ---
   name: gabo-assistant
   description: '<main_session.role, with each single quote doubled>'
   skills:
     - <each main_session.skills entry, in order>
   gabo-version: <x.y.z, unquoted>
   ---
   ```

   `name` comes from `main_session.id` and `description` from `main_session.role`.
3. Merge `agent: "gabo-assistant"` into `.claude/settings.json` (created as `{"agent": "gabo-assistant"}` plus a newline when absent): add or set only that one key and keep every other byte. When the file holds a user's own `agent` value (any other value), keep it and report it: the main session then does not run as the assistant. A file that is not valid JSON: change nothing and report it.

When `runtimes.assistant` is not `claude`, on every build remove the Claude Code entry, recognized only by its gabo identity: the `.claude/skills/gabo-assistant` symlink, `.claude/agents/gabo-assistant.md` when it carries a `gabo-version:` line, and in `.claude/settings.json` the `agent` key only when its value is exactly `gabo-assistant` (keep every other byte). Anything else at those paths: leave it and report it.

On every build, also remove the gabo entry of every full-tier engine of the table that is not `runtimes.assistant`, recognized only by its gabo identity, and never a piece the entry of `runtimes.assistant` also uses (the shared `.agents/skills/gabo-assistant` link while Codex, OpenCode or Pi is the assistant; the block while Codex or Pi is):

- OpenCode: `.opencode/agents/gabo-assistant.md` when it carries a `gabo-version:` line, and `default_agent: "gabo-assistant"` (the key only when its value is exactly `gabo-assistant`) in each of `opencode.json`, `opencode.jsonc`, `.opencode/opencode.json`, `.opencode/opencode.jsonc` that holds it, every other byte and comment kept.
- Codex and Pi: in `AGENTS.override.md`, `AGENTS.md` and `CLAUDE.md`, the block from `<!-- GABO_ASSISTANT_START -->` to `<!-- GABO_ASSISTANT_END -->` with the one blank line before it, every other byte kept; never delete the file. Write through a symlinked file only when its target stays inside the repository. A start marker without its end marker, or the reverse: leave the file and report it.
- `.agents/skills/gabo-assistant`, when it is a symlink.

Report each piece written, kept or removed, and each one left.

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

   Main-session check, when `runtimes.assistant` is `claude` and the entry was written: from the project root run `claude --print` with no `--agent`, asking which agent the session runs as and what the `gabo-assistant` skill's Session Start asks first; it passes when the reply names `gabo-assistant` and follows that Session Start. Then run `claude --print --agent gabo-coder` asking which agent it runs as; it passes when it runs as `gabo-coder` (the flag overrides the `agent` key). Each runs once; a failure changes nothing and is reported as `main-session probe failed: <check> <reply>`. A user's own `agent` value kept in step 3 of the Main-Session Entry: skip this check and report it. This check starts new processes, so it runs even when the probe above was skipped for a restart.

## Output Contract

Report the Claude Code version and whether the sources were rechecked; each agent file written with its id and `gabo-version`; each link made (catalog, fixed-name and reviewer); each entry left or slot not filled, with why; the `claude plugin validate` and `gabo check` results; the main-session entry pieces written, kept, removed or left (Main-Session Entry); and the probe result, with the main-session check, when it ran.
