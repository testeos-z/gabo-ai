---
name: gabo-runtime-opencode
description: "Trigger: build gabo agents for OpenCode, rebuild OpenCode agents, runtimes opencode, .opencode/agents, .agents/skills, gabo check reports an OpenCode agent or link. Builds the gabo agents as OpenCode subagents and places skills and links."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
  agent_dir: .opencode/agents
  skill_dir: .agents/skills
  former_skill_dir: .opencode/skills
  agent_skills: "line:Skills:"
  main_session:
    kind: agent
    config: [opencode.json, opencode.jsonc, .opencode/opencode.json, .opencode/opencode.jsonc]
    key: default_agent
---
## Activation Contract

Load when `runtimes.coder` or `runtimes.reviewer` in `gabo.yaml` is `opencode`, or `runtimes.assistant` lists `opencode`, and the gabo agents must be built or rebuilt: after setup, after `gabo init` refreshed the catalog, or when `gabo check` reports `✗ agent <id> missing` or an agent built with another version (TRD 8.1, 9.4, D24; intent 0004 S2 to S6). Load `gabo-agent-templates` with it: that skill owns the spec, the templates and the slot rules; this skill only turns them into OpenCode files.

`metadata.agent_dir` above is the directory `gabo check` searches for the agent files of this runtime (it reads `gabo-version:` anywhere in the file, so also in the body comment of build step 7); `metadata.skill_dir` is where it requires the links of build steps 2 and 3, the `.agents/skills` that Codex and Pi share (intent 0008 S3); `metadata.former_skill_dir` is where this skill made its links before the move, which `gabo check` reports when a gabo link is left there (S4). There is no `metadata.reviewer_link`: OpenCode gets no reviewer link (Hard Rules). Never change them without changing where you build. `gabo check` reports a missing or wrong link as `✗ skill link <path> …`: rebuild with this skill.

## Hard Rules

- Build every agent of `.gabo/skills/gabo-agent-templates/agents.yaml`, each as `.opencode/agents/<id>.md`, whatever runtime each agent plays: `gabo check` requires all of them in `.opencode/agents`.
- Read the spec, the templates and `.gabo/VERSION` from the project; never build from memory. A missing or empty `.gabo/VERSION` or catalog: stop and tell the user to run `gabo init`.
- Fill only the template slots, as below, and invent nothing: no frontmatter key, permission, skill, model or rule the spec and template do not give. OpenCode passes unknown agent options to the provider as model options, so the frontmatter holds `description`, `mode` and `permission` and no other key; the version goes in a body comment (D2).
- Permissions only restrict (D3): a capability the spec lists is left at the owner's setting, a capability it lacks is set to `deny`. Never loosen the owner's own rules.
- Never place `gabo-reviewer` in the project (D5, TRD D20): never create `.agents/skills/gabo-reviewer`, no copy of its scripts, and never in an agent's `Skills:` line. OpenCode gets no reviewer link at all, neither in the project nor under `~/.config/opencode/skills`: the reviewer agent runs the user-level script by path, and every agent but `gabo-reviewer` sets `permission.skill` `gabo-reviewer: deny`, so a `gabo-reviewer` skill OpenCode finds elsewhere (for example in `~/.claude/skills`) cannot start a second review. The `gabo-reviewer-config` link of build step 4 is not a reviewer link: that skill configures the reviewer engine and never runs a review (intent 0004 S14).
- Engine config (intent 0004 S8, D7): gabo writes only the engine keys it needs to operate, at project level only, merged with the existing project config (every unrelated key kept), reports what it wrote, and has the value verified by `gabo check`; never the user's global engine config. `gabo init` writes every engine config key (intent 0010 S5), `experimental.subagent_depth` and `default_agent` included; this skill writes none and only builds agent files, links and the stage-gate plugin.
- Write only `.opencode/agents/<id>.md` for spec ids and `.opencode/agents/gabo-assistant.md`, `.agents/skills/` links, the stage-gate plugin `.opencode/plugins/gabo-stage-gate.js`, the probe files you delete and the removals of other engines' main-session entries of build step 9. Nothing else: never write or remove a key of a project config (intent 0004 S9, intent 0010 S5). Remove only the `.opencode/skills/gabo-*` links of build step 5, the `.agents/skills/gabo-*` links the shared-directory rule of build step 3 allows and the main-session entries of build step 9; leave every other entry of `.opencode/` and `.agents/` untouched.
- Never write under `~/.config/opencode` (no global config, agents, skills or plugins) and never write `.opencode/package.json` (D8). Name nothing outside the spec ids, all `gabo-*`, so no gabo agent collides with gentle-ai's or the owner's agents. Every spec agent is `mode: subagent`; the only primary agent a build writes is `gabo-assistant`, and only when `opencode` is listed in `runtimes.assistant` (build step 9), so otherwise the owner's default agent (for example gentle-ai's `gentle-orchestrator`) stays the default.
- Overwrite only the agent files named after a spec id, `.opencode/agents/gabo-assistant.md` when it carries a `gabo-version:` line, the links this skill makes and `.opencode/plugins/gabo-stage-gate.js`. Any other existing entry at a path you would write (a real file or folder, or a link to another target): leave it and report it; never delete or move it.
- Make every link of steps 2, 3 and 4 on every build, also when the agent files are already current; a build that skips one is not finished.
- Write no secret into any file (R21).

## Prerequisites

Run in the repository root before anything is built or written (intent 0004 S8, D7):

1. `opencode --version` must succeed; else refuse: build nothing, write nothing and report that `opencode` is missing or fails.

Then read the project config: the first existing of `opencode.json`, `opencode.jsonc`, `.opencode/opencode.json`, `.opencode/opencode.jsonc` (OpenCode's own lookup order, the same one `gabo check` reads). Call it `<file>`. Never write `~/.config/opencode/` or `.opencode/package.json`, and never read the global value as the project's. `gabo init` writes `experimental.subagent_depth` (at least 3) and `default_agent` into `<file>` (intent 0010 S3, S4); this skill only reads them and writes no key. When no `<file>` exists, it does not parse, or its `experimental.subagent_depth` is missing, not a whole number or below 3 (assistant → cycle → coder → test runner, TRD 2), report `run gabo init` with the file and the value read, and go on with the build.

## Sources

Checked on 2026-11-03 against OpenCode 2.0.32 (`opencode --version`), its `--help`, and these pages (intent 0004 spec, Evidence). Rows marked **(binary)** come from the 2.0.32 binary and are not in the docs.

| fact | used for | source |
|---|---|---|
| Project agents are Markdown files in `.opencode/agents/`; the file name is the agent name; `description` is required; `mode` is `primary`, `subagent` or `all`; `permission` replaces the deprecated `tools`. | agent files | https://opencode.ai/docs/agents |
| Unknown agent options are passed through directly to the provider as model options. | no `gabo-version` key (D2) | https://opencode.ai/docs/agents |
| Skills are found in `.opencode/skills`, `.claude/skills` and `.agents/skills` of the project and in `~/.config/opencode/skills`, `~/.claude/skills`, `~/.agents/skills`; they load on demand through the `skill` tool; there is no per-agent preload field. | catalog skills, `Skills:` line, reviewer deny | https://opencode.ai/docs/skills |
| **(binary)** With `gabo-pm` linked in both `.claude/skills` and `.agents/skills` to the same `.gabo/skills/gabo-pm`, OpenCode 2.0.32 lists it once, with no warning or error, and the `skill` tool loads it: OpenCode builds its skill list from `.claude/skills` before `.agents/skills` and keeps one entry per name (`ConfigSkillPlugin.refresh`), so the `.agents/skills` copy replaces the `.claude/skills` one (intent 0008 T9 probe, checked 2026-11-06). | links in the shared `.agents/skills` next to Claude Code's `.claude/skills` (intent 0008 S5, D6) | https://github.com/testeos-z/gabo-ai/issues/315#issuecomment-6689595417 |
| Permission keys include `read`, `edit` (covers edit, write, patch), `glob`, `grep`, `bash`, `task`, `skill`, `webfetch`, `websearch`; a key takes `allow`, `ask` or `deny`, or an object of patterns; the last matching rule wins; agent rules merge over global rules. | Tool Map, `skill` patterns | https://opencode.ai/docs/permissions |
| Built-in tools `read`, `edit`, `write`, `grep`, `glob`, `bash`, `task`, `skill`, `webfetch`, `websearch`; MCP tools are named `<server>_<tool>`. | Tool Map | https://opencode.ai/docs/tools |
| `default_agent` in the project config "determines which agent is used when none is explicitly specified", "must be a primary agent" and applies to the TUI, `opencode run`, the desktop app and the GitHub Action; `--agent` picks another agent for one run. | main-session entry (build step 9; `gabo init` writes the key, this skill only reads it) | https://opencode.ai/docs/config, https://opencode.ai/docs/agents |
| **(binary)** Nested subagents stop at `experimental.subagent_depth`, default 1 ("Subagent depth limit reached"); a top-level `subagent_depth` is not read. | Prerequisites (`gabo init` writes the key, this skill only reads it) | OpenCode 2.0.32 binary |
| **(binary)** The tool `opencode.session_move` makes a worktree the session's working directory; the `shell` tool takes `workdir`; the `subagent` tool takes no directory. | coder worktree step | OpenCode 2.0.32 binary |
| Local plugins are JS or TS files in `.opencode/plugins/`, loaded at startup; `tool.execute.before(input, output)` gets the tool name in `input.tool` and its arguments in `output.args`, and blocks the call by throwing; a plugin that loads a module needs a `package.json` that OpenCode installs with `bun install`. | stage-gate plugin (build step 8) | https://opencode.ai/docs/plugins |
| **(binary)** v2 renamed tools: `bash` is `shell`, `task` is `subagent`, `apply_patch` is `patch`; `edit` and `write` take `path` and `patch` takes `patchText`; it emits permission actions named `shell` and `subagent`. | Tool Map aliases, stage-gate plugin targets | OpenCode 2.0.32 binary |

Recheck before you build: run `opencode --version`. When it is not 2.0.32, read the six pages and `opencode --help` again and confirm each row; recheck each **(binary)** row on the installed binary. A row that no longer holds: stop, build nothing and report the row with what you found.

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

Each slot of `gabo.yaml` (`slots.<slot>.adapter`) whose adapter is a `<folder>/SKILL.md` gets one link `.agents/skills/gabo-<slot>` (`_` written `-`) to `<folder>`, as a relative symlink (relative to `.agents/skills/`): `gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory` and so on. An adapter that is not a skill folder (for example the reviewer agent file) gets no link: report it. Agents read the linked skills by path (`Skills:` line of build step 7), so a link whose folder name differs from the adapter's frontmatter `name` still works (D4).

## Build Steps

1. Run the Prerequisites; on a refusal stop here. Recheck the sources. Read `gabo.yaml`, `.gabo/VERSION` and the spec.
2. Catalog skills: for each skill in any agent's spec `skills`, make `.agents/skills/<name>` a relative symlink to `../../.gabo/skills/<name>`. An entry that already resolves to that folder is kept (Codex or Pi may have made it); a real folder or file, or a link to another folder that this skill did not make, is left and reported.
3. Fixed-name links as above. Then remove each `.agents/skills/gabo-*` symlink that no spec skill or slot asks for any more and that this skill made: one that points into `../../.gabo/skills/` or to a folder a slot adapter of `gabo.yaml` named before, except `.agents/skills/gabo-reviewer-config` (step 4). `.agents/skills/` is shared by every engine that reads it (Codex, Pi and OpenCode), so keep every link any engine declared in `runtimes.*` needs: an engine uses `.agents/skills` when its runtime skill `.gabo/skills/gabo-runtime-<engine>/SKILL.md` declares `metadata.skill_dir: .agents/skills`, and every such engine needs the same spec skills and fixed-name links (one catalog copy, one `gabo.yaml`); also keep `.agents/skills/gabo-assistant` while `runtimes.assistant` names an engine that uses `.agents/skills` (intent 0009 S1): it is removed only when every engine that uses `.agents/skills` is not listed in `runtimes.assistant` (intent 0013 S4). Leave and report any other `gabo-*` symlink, and never remove a real folder or file.
4. Reviewer configuration (intent 0004 S14, D12): make `.agents/skills/gabo-reviewer-config` a relative symlink to `../../.gabo/skills/gabo-reviewer-config`, as in step 2, on every build, also when it already exists, so the main OpenCode session finds it; no agent's `Skills:` line names it. No reviewer link: never create `.agents/skills/gabo-reviewer` (Hard Rules).
5. Migration from `.opencode/skills` (intent 0008 S4): earlier builds placed the links in `.opencode/skills/` (intent 0004). On every build, after the links of steps 2 to 4, remove each `.opencode/skills/gabo-*` symlink that this skill made, `.opencode/skills/gabo-reviewer-config` included: one that points into `../../.gabo/skills/` or to a slot adapter folder of `gabo.yaml` (relative to `.opencode/skills/`). Leave and report any other entry of `.opencode/skills/`, and never remove a real folder or file. A missing `.opencode/skills/` needs nothing; leave the directory itself even when it is empty. `gabo check` fails on such a leftover: `✗ .opencode/skills/<name>: gabo link left after the move to .agents/skills`, with `next: ask your agent to rebuild the agents`.
6. For each spec agent, fill its template:

   | slot | value |
   |---|---|
   | `{{id}}`, `{{role}}` | the spec `id` and `role` |
   | `{{gabo_version}}` | the single line of `.gabo/VERSION` |
   | `{{engine}}` | `opencode`, the engine name as written in `gabo.yaml` |
   | `{{methodology}}` | `gabo-methodology` |
   | `{{adapter:tasks}}`, `{{adapter:repository_host}}`, `{{adapter:memory}}` | `gabo-tasks`, `gabo-repository-host`, `gabo-memory` |
   | `{{reviewer_script}}` | `${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh`, written literally: the shell expands it on each machine, so the committed file holds no personal path |

   A slot whose link does not exist (its slot has no skill adapter) cannot be filled: skip that agent and report it.
7. Write `.opencode/agents/<id>.md` in exactly this shape: frontmatter with only `description`, `mode: subagent` and `permission`, no other key, then the body.

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
   <!-- gabo-engine: opencode -->
   Skills: before work read .agents/skills/<name>/SKILL.md for each of <spec skills, then the fixed-name links the body names>.

   <filled body>
   ```

   - `description` is the spec `role` in double quotes, each `\` written `\\` and each `"` written `\"`, so a colon in it stays plain text.
   - `permission` follows the Tool Map, including its `skill` rule; omit `permission` only when it would be empty, which no spec agent is.
   - `<!-- gabo-version: <x.y.z> -->` is the template's version line moved into an HTML comment, with the value of `.gabo/VERSION`; `gabo check` compares it with the stamp.
   - `<!-- gabo-engine: opencode -->` is the template's engine line moved into an HTML comment, with `{{engine}}` filled (intent 0013 S6); `gabo check` fails a file whose engine is not `opencode`.
   - The `Skills:` line lists the spec `skills` in spec order, then each fixed-name link the body names (`gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory`, in this order). Omit it when the list is empty (the reviewer agent).
   - `<filled body>` is the filled template without its first two lines `gabo-version: …` and `gabo-engine: …` and the blank line after them.
   - For `gabo-coder` only, write this paragraph between the `Skills:` line and `<filled body>`, with one blank line before and after it (intent 0004 S7, D6: the `subagent` tool takes no working directory):

     ```text
     First step, before any other tool call: call `opencode.session_move` with the worktree path from your launch prompt, so that worktree becomes your working directory. If `opencode.session_move` is not callable here, pass that worktree path as `workdir` on every `shell` call and use absolute paths under it for every `read`, `edit`, `write` and `patch`. Then check that `git rev-parse --show-toplevel` prints the worktree path; if it does not, stop and report BLOCKED.
     ```
8. Stage gate (intent 0004 S10, D10): copy `.gabo/skills/gabo-runtime-opencode/plugins/gabo-stage-gate.js` byte for byte to `.opencode/plugins/gabo-stage-gate.js`, on every build, also when it already exists; create `.opencode/plugins/` when it is missing. Copy it unchanged and write no `package.json`: the plugin is plain JavaScript that loads no module. Its `tool.execute.before` hook passes each target path of an `edit`, `write` or `patch` call to `gabo check --pre-write <path...>` and fails the call with gabo's output when gabo exits 1; when `gabo` is not on `PATH` or exits otherwise, the call runs and the plugin logs one warning. The stage rule lives in gabo, not in the plugin. A `shell` call that writes files is not gated. OpenCode loads plugins at startup: tell the user to restart OpenCode when the file was new or changed.
9. Main-session entry (intent 0009 S1, S2, S4, D3): read the Main-Session Entry table of `gabo-agent-templates` and the `main_session` entry of its `agents.yaml`; that table is the one definition of every engine's entry. `runtimes.assistant` is one engine or a list (intent 0013 S4). A build removes another full-tier engine's entry only when it is not listed in `runtimes.assistant`, never the entry of another listed engine, so rebuilding the listed engines in any order leaves every listed entry in place.
   - When `opencode` is listed in `runtimes.assistant`, on every build: make `.agents/skills/gabo-assistant` a relative symlink to `../../.gabo/skills/gabo-assistant`, as in step 2; write `.opencode/agents/gabo-assistant.md` from `main_session.template` in the shape of step 7 with `mode: primary` in place of `mode: subagent`: `description` is the `main_session.role`, `permission` holds only the `skill` object with `gabo-reviewer: deny` (the entry declares no `tools`, so nothing else is denied), the `Skills:` line names `gabo-assistant`, and the body is the template filled with `{{gabo_version}}` and `{{engine}}` (`opencode`). `gabo init` writes `default_agent` with the value `"gabo-assistant"` into `<file>` (intent 0010 S4); this skill only reads it: when it is missing, report `run gabo init`. A `default_agent` with another value is the owner's own: leave it and report it, and the entry stays incomplete until the owner changes it. An existing `.opencode/agents/gabo-assistant.md` without a `gabo-version:` line is not gabo's: leave it and report it.
   - When `opencode` is not listed in `runtimes.assistant`: remove `.opencode/agents/gabo-assistant.md` when it carries a `gabo-version:` line. This skill removes the shared `.agents/skills/gabo-assistant` link only when every declared engine using `.agents/skills` is not listed in `runtimes.assistant` (step 3), and only a symlink to the catalog folder.
   - On every build, remove the gabo main-session entry of every other full-tier engine of that table that is not listed in `runtimes.assistant`, recognized only by its gabo identity: for Claude Code, `.claude/agents/gabo-assistant.md` when it carries a `gabo-version:` line, and the `.claude/skills/gabo-assistant` symlink to the catalog folder; for Codex and Pi, the bytes from `<!-- GABO_ASSISTANT_START -->` to `<!-- GABO_ASSISTANT_END -->`, with the one blank line before them, in `AGENTS.override.md`, `AGENTS.md` and `CLAUDE.md`, every other byte kept, writing through a symlinked file only when its target stays inside the repository; keep a block in a file the entry of an engine listed in `runtimes.assistant` uses (Codex and Pi share `AGENTS.md`). Their `.agents/skills/gabo-assistant` link follows the shared rule above. Anything else at those paths (a file without the version line, a link to another target): leave it and report it. Never remove Claude Code's `agent` key or OpenCode's `default_agent` key: `gabo init` removes them (intent 0010 S4), and this skill writes and removes no engine config key.
10. Verify: run `gabo check`, which also checks the links of steps 2 and 3 and any gabo link left in `.opencode/skills/`. Fix every failure this skill caused and rerun until `gabo check` passes, except the `✗ agent <id> missing` lines of agents you skipped and reported in step 6; report any other failure.

## Output Contract

Report the OpenCode version and whether the sources were rechecked; the project config file `<file>` read and, for an `experimental.subagent_depth` or `default_agent` that is missing or low, the `run gabo init` reported (`gabo init` writes both keys), or the refusal printed; each agent file written with its id, `gabo-version`, `gabo-engine` and denied permission keys; each link made (catalog, fixed-name and the `gabo-reviewer-config` link); each link removed; each `.opencode/skills/` link removed and each entry left with why; that no reviewer link was made; whether `.opencode/plugins/gabo-stage-gate.js` was copied, new, changed or unchanged; whether the `gabo-coder` worktree step was written; the main-session entry: the `gabo-assistant` link, `.opencode/agents/gabo-assistant.md` and `default_agent` in `<file>` present, missing (`run gabo init`) or left with why, and each other engine's entry removed or left with why; each entry left or slot not filled, with why; and the `gabo check` result.
