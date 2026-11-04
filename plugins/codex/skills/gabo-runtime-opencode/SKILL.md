---
name: gabo-runtime-opencode
description: "Trigger: build gabo agents for OpenCode, rebuild OpenCode agents, runtimes opencode, .opencode/agents, .opencode/skills, gabo check reports an OpenCode agent or link. Builds the gabo agents as OpenCode subagents and places skills and links."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
  agent_dir: .opencode/agents
  skill_dir: .opencode/skills
---
## Activation Contract

Load when any of `runtimes.coder`, `runtimes.reviewer` or `runtimes.assistant` in `gabo.yaml` is `opencode` and the gabo agents must be built or rebuilt: after setup, after `gabo init` refreshed the catalog, or when `gabo check` reports `✗ agent <id> missing` or an agent built with another version (TRD 8.1, 9.4, D24; intent 0004 S2 to S6). Load `gabo-agent-templates` with it: that skill owns the spec, the templates and the slot rules; this skill only turns them into OpenCode files.

`metadata.agent_dir` above is the directory `gabo check` searches for the agent files of this runtime (it reads `gabo-version:` anywhere in the file, so also in the body comment of build step 6); `metadata.skill_dir` is where it requires the links of build steps 2 and 3. There is no `metadata.reviewer_link`: OpenCode gets no reviewer link (Hard Rules). Never change them without changing where you build. `gabo check` reports a missing or wrong link as `✗ skill link <path> …`: rebuild with this skill.

## Hard Rules

- Build every agent of `.gabo/skills/gabo-agent-templates/agents.yaml`, each as `.opencode/agents/<id>.md`, whatever runtime each agent plays: `gabo check` requires all of them in `.opencode/agents`.
- Read the spec, the templates and `.gabo/VERSION` from the project; never build from memory. A missing or empty `.gabo/VERSION` or catalog: stop and tell the user to run `gabo init`.
- Fill only the template slots, as below, and invent nothing: no frontmatter key, permission, skill, model or rule the spec and template do not give. OpenCode passes unknown agent options to the provider as model options, so the frontmatter holds `description`, `mode` and `permission` and no other key; the version goes in a body comment (D2).
- Permissions only restrict (D3): a capability the spec lists is left at the owner's setting, a capability it lacks is set to `deny`. Never loosen the owner's own rules.
- Never place `gabo-reviewer` in the project (D5, TRD D20): never create `.opencode/skills/gabo-reviewer`, no copy of its scripts, and never in an agent's `Skills:` line. OpenCode gets no reviewer link at all, neither in the project nor under `~/.config/opencode/skills`: the reviewer agent runs the user-level script by path, and every agent but `gabo-reviewer` sets `permission.skill` `gabo-reviewer: deny`, so a `gabo-reviewer` skill OpenCode finds elsewhere (for example in `~/.claude/skills`) cannot start a second review. The `gabo-reviewer-config` link of build step 4 is not a reviewer link: that skill configures the reviewer engine and never runs a review (intent 0004 S14).
- Write only `.opencode/agents/<id>.md` for spec ids, `.opencode/skills/` links, the stage-gate plugin `.opencode/plugins/gabo-stage-gate.js`, the probe files you delete, and the project-level key `experimental.subagent_depth` of the Prerequisites, merged into one project config file. Nothing else: never change any other key of a project config (intent 0004 S9).
- Never write under `~/.config/opencode` (no global config, agents, skills or plugins) and never write `.opencode/package.json` (D8). Name nothing outside the spec ids, all `gabo-*`, so no gabo agent collides with gentle-ai's or the owner's agents. Every gabo agent is `mode: subagent` and no build sets a default agent, so the owner's default agent (for example gentle-ai's `gentle-orchestrator`) stays the default.
- Overwrite only the agent files named after a spec id and the links this skill makes. Any other existing entry at a path you would write (a real file or folder, or a link to another target): leave it and report it; never delete or move it.
- Make every link of steps 2, 3 and 4 on every build, also when the agent files are already current; a build that skips one is not finished.
- Write no secret into any file (R21).

## Prerequisites

Run in the repository root before anything is built or written (intent 0004 S8, D7):

1. `opencode --version` must succeed; else refuse as in step 4.
2. Find the project config: the first existing of `opencode.json`, `opencode.jsonc`, `.opencode/opencode.json`, `.opencode/opencode.jsonc` (OpenCode's own lookup order, the same one `gabo check` reads), else a new `.opencode/opencode.json`. Call it `<file>`. Never write `~/.config/opencode/` or `.opencode/package.json`, and never read the global value as the project's.
3. Make the project's `experimental.subagent_depth` at least 3 (assistant → cycle → coder → test runner, TRD 2). Write only the key `experimental.subagent_depth`, with the value `3`, into `<file>`:
   - a new `.opencode/opencode.json` holds exactly `{"experimental": {"subagent_depth": 3}}`;
   - in an existing file, merge: edit the text in place, keeping every other key and value, their order and formatting, and the comments of a `.jsonc` file; add `experimental` as an object when it is absent, or the key inside an existing `experimental` object; never create a second config file;
   - a value already at a whole number of 3 or more: leave the file unchanged;
   - Report the file, the key, the old value (or `unset`) and the new value, or that the file was left unchanged with its value.
4. Refuse when `opencode` is missing or fails, or when `<file>` cannot be parsed and merged without loss (it does not parse as JSON with comments and trailing commas, `experimental` is not an object, or the edit would change anything but that key): build nothing, write nothing and print `set experimental.subagent_depth to 3 in <file>`, with the reason and `then ask me to rebuild`.

## Sources

Checked on 2026-11-03 against OpenCode 2.0.32 (`opencode --version`), its `--help`, and these pages (intent 0004 spec, Evidence). Rows marked **(binary)** come from the 2.0.32 binary and are not in the docs.

| fact | used for | source |
|---|---|---|
| Project agents are Markdown files in `.opencode/agents/`; the file name is the agent name; `description` is required; `mode` is `primary`, `subagent` or `all`; `permission` replaces the deprecated `tools`. | agent files | https://opencode.ai/docs/agents |
| Unknown agent options are passed through directly to the provider as model options. | no `gabo-version` key (D2) | https://opencode.ai/docs/agents |
| Skills are found in `.opencode/skills`, `.claude/skills` and `.agents/skills` of the project and in `~/.config/opencode/skills`, `~/.claude/skills`, `~/.agents/skills`; they load on demand through the `skill` tool; there is no per-agent preload field. | catalog skills, `Skills:` line, reviewer deny | https://opencode.ai/docs/skills |
| Permission keys include `read`, `edit` (covers edit, write, patch), `glob`, `grep`, `bash`, `task`, `skill`, `webfetch`, `websearch`; a key takes `allow`, `ask` or `deny`, or an object of patterns; the last matching rule wins; agent rules merge over global rules. | Tool Map, `skill` patterns | https://opencode.ai/docs/permissions |
| Built-in tools `read`, `edit`, `write`, `grep`, `glob`, `bash`, `task`, `skill`, `webfetch`, `websearch`; MCP tools are named `<server>_<tool>`. | Tool Map | https://opencode.ai/docs/tools |
| **(binary)** Nested subagents stop at `experimental.subagent_depth`, default 1 ("Subagent depth limit reached"); a top-level `subagent_depth` is not read. | Prerequisites | OpenCode 2.0.32 binary |
| **(binary)** The tool `opencode.session_move` makes a worktree the session's working directory; the `shell` tool takes `workdir`; the `subagent` tool takes no directory. | coder worktree step | OpenCode 2.0.32 binary |
| **(binary)** v2 renamed tools: `bash` is `shell`, `task` is `subagent`, `apply_patch` is `patch`; it emits permission actions named `shell` and `subagent`. | Tool Map aliases | OpenCode 2.0.32 binary |

Recheck before you build: run `opencode --version`. When it is not 2.0.32, read the five pages and `opencode --help` again and confirm each row; recheck each **(binary)** row on the installed binary. A row that no longer holds: stop, build nothing and report the row with what you found.

## Tool Map

The spec's engine-neutral capabilities map to OpenCode permission keys. A capability the spec lists is left at the owner's setting: write no key for it. For each capability the spec lacks, set each key of its row to `deny` in the agent's `permission`, in this table's order, each key once.

| capability | OpenCode v2 tools | permission keys set to `deny` when the spec lacks the capability |
|---|---|---|
| `read` | `read` | `read` |
| `search` | `grep`, `glob` | `grep`, `glob` |
| `edit` | `edit`, `write`, `patch` | `edit` |
| `shell`, `git`, `tracker`, `repository-host` | `shell` | `bash`, only when the spec lists none of the four (and `shell` too, if the recheck shows v2 does not alias it) |
| `memory` | MCP tools `<server>_*` of the memory adapter's server (for example `engram_*`) | `<server>_*` |
| `web` | `webfetch`, `websearch` | `webfetch`, `websearch` |
| `skills` | `skill` | `skill` |
| `subagents` | `subagent` | `task` (and `subagent` too, as for `shell`) |

`skill` is also where the reviewer rule goes: every agent but `gabo-reviewer` gets `skill` as an object holding `gabo-reviewer: deny`; an agent that also lacks `skills` gets `"*": deny` first, then `gabo-reviewer: deny`. `gabo-reviewer` lacks `skills`, so it gets `skill: deny`.

## Fixed-Name Links

Each slot of `gabo.yaml` (`slots.<slot>.adapter`) whose adapter is a `<folder>/SKILL.md` gets one link `.opencode/skills/gabo-<slot>` (`_` written `-`) to `<folder>`, as a relative symlink: `gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory` and so on. An adapter that is not a skill folder (for example the reviewer agent file) gets no link: report it. Agents read the linked skills by path (`Skills:` line of build step 6), so a link whose folder name differs from the adapter's frontmatter `name` still works (D4).

## Build Steps

1. Run the Prerequisites; on a refusal stop here. Recheck the sources. Read `gabo.yaml`, `.gabo/VERSION` and the spec.
2. Catalog skills: for each skill in any agent's spec `skills`, make `.opencode/skills/<name>` a relative symlink to `../../.gabo/skills/<name>`. An entry that already resolves to that folder is kept; a real folder or file is left and reported.
3. Fixed-name links as above. Then remove each `.opencode/skills/gabo-*` symlink that no spec skill or slot asks for any more, except `.opencode/skills/gabo-reviewer-config` (step 4); never remove a real folder or file.
4. Reviewer configuration (intent 0004 S14, D12): make `.opencode/skills/gabo-reviewer-config` a relative symlink to `../../.gabo/skills/gabo-reviewer-config`, as in step 2, on every build, also when it already exists, so the main OpenCode session finds it; no agent's `Skills:` line names it. No reviewer link: never create `.opencode/skills/gabo-reviewer` (Hard Rules).
5. For each spec agent, fill its template:

   | slot | value |
   |---|---|
   | `{{id}}`, `{{role}}` | the spec `id` and `role` |
   | `{{gabo_version}}` | the single line of `.gabo/VERSION` |
   | `{{methodology}}` | `gabo-methodology` |
   | `{{adapter:tasks}}`, `{{adapter:repository_host}}`, `{{adapter:memory}}` | `gabo-tasks`, `gabo-repository-host`, `gabo-memory` |
   | `{{reviewer_script}}` | `${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh`, written literally: the shell expands it on each machine, so the committed file holds no personal path |

   A slot whose link does not exist (its slot has no skill adapter) cannot be filled: skip that agent and report it.
6. Write `.opencode/agents/<id>.md` in exactly this shape: frontmatter with only `description`, `mode: subagent` and `permission`, no other key, then the body.

   ```markdown
   ---
   description: "<role>"
   mode: subagent
   permission:
     <each key of the Tool Map for a capability the spec lacks>: deny
     skill:
       gabo-reviewer: deny
   ---
   <!-- gabo-version: <x.y.z> -->
   Skills: before work read .opencode/skills/<name>/SKILL.md for each of <spec skills, then the fixed-name links the body names>.

   <filled body>
   ```

   - `description` is the spec `role` in double quotes, each `\` written `\\` and each `"` written `\"`, so a colon in it stays plain text.
   - `permission` follows the Tool Map, including its `skill` rule; omit `permission` only when it would be empty, which no spec agent is.
   - `<!-- gabo-version: <x.y.z> -->` is the template's version line moved into an HTML comment, with the value of `.gabo/VERSION`; `gabo check` compares it with the stamp.
   - The `Skills:` line lists the spec `skills` in spec order, then each fixed-name link the body names (`gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory`, in this order). Omit it when the list is empty (the reviewer agent).
   - `<filled body>` is the filled template without its first line `gabo-version: …` and the blank line after it.
   - For `gabo-coder` only, write this paragraph between the `Skills:` line and `<filled body>`, with one blank line before and after it (intent 0004 S7, D6: the `subagent` tool takes no working directory):

     ```text
     First step, before any other tool call: call `opencode.session_move` with the worktree path from your launch prompt, so that worktree becomes your working directory. If `opencode.session_move` is not callable here, pass that worktree path as `workdir` on every `shell` call and use absolute paths under it for every `read`, `edit`, `write` and `patch`. Then check that `git rev-parse --show-toplevel` prints the worktree path; if it does not, stop and report BLOCKED.
     ```
7. Verify: run `gabo check`, which also checks the links of steps 2 and 3. Fix every failure this skill caused and rerun until `gabo check` passes, except the `✗ agent <id> missing` lines of agents you skipped and reported in step 5; report any other failure.

## Output Contract

Report the OpenCode version and whether the sources were rechecked; the project config file with `experimental.subagent_depth` old and new value, or left unchanged, or the refusal printed; each agent file written with its id, `gabo-version` and denied permission keys; each link made (catalog, fixed-name and the `gabo-reviewer-config` link); each link removed; that no reviewer link was made; whether the `gabo-coder` worktree step was written; each entry left or slot not filled, with why; and the `gabo check` result.
