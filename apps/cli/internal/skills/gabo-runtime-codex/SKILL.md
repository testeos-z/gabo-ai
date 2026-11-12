---
name: gabo-runtime-codex
description: "Trigger: build gabo agents for Codex, rebuild Codex agents, runtimes codex, .codex/agents, .agents/skills, gabo check reports a Codex agent or link. Builds the gabo agents as Codex TOML agents and places skills and links."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
  agent_dir: .codex/agents
  skill_dir: .agents/skills
  agent_skills: "line:Load these skills by name before work:"
  agent_name: toml:name
  reviewer_link: $HOME/.agents/skills/gabo-reviewer
  main_session:
    kind: block
    files: [AGENTS.override.md, AGENTS.md]
---
## Activation Contract

Load when any of `runtimes.coder`, `runtimes.reviewer` or `runtimes.assistant` in `gabo.yaml` is or lists `codex` and the gabo agents must be built or rebuilt: after setup, after `gabo init` refreshed the catalog, or when `gabo check` reports `✗ agent <id> missing` or an agent built with another version (TRD 8.1, 9.4, D24; intent 0002 S4, S5). Load `gabo-agent-templates` with it: that skill owns the spec, the templates and the slot rules; this skill only turns them into Codex files, and the main-session entry of its table row into the Codex block (intent 0009).

`metadata.agent_dir` above is the directory `gabo check` searches for the agent files of this runtime (it accepts any extension and reads `gabo-version:` in a TOML comment); `metadata.skill_dir` is where it requires the links of build steps 2 and 3, and `metadata.reviewer_link` the user-level link of step 4 (outside CI). Never change them without changing where you build. `gabo check` reports a missing or wrong link as `✗ skill link <path> …` or `✗ reviewer link <path> …`: rebuild with this skill.

## Hard Rules

- Build every agent of `.gabo/skills/gabo-agent-templates/agents.yaml`, each as `.codex/agents/<id>.toml`, whatever runtime each agent plays: `gabo check` requires all of them in `.codex/agents` (D2).
- Read the spec, the templates and `.gabo/VERSION` from the project; never build from memory. A missing or empty `.gabo/VERSION` or catalog: stop and tell the user to run `gabo init`.
- Fill only the template slots, as below, and invent nothing: no TOML key, skill, model or rule the spec and template do not give. No `model`, `sandbox_mode` or `features` key is written, also not for the read-only agent `gabo-plan-reviewer`: its limits stay in their instructions (D6).
- No built agent's instructions tell it to launch an agent (intent 0015 S6, D7): the main session launches each spec agent and no agent launches another; fill the templates and add no launch sentence.
- Never place `gabo-reviewer` in the project: no `.agents/skills/gabo-reviewer`, no copy of its scripts, and never in an agent's skill line. Only link the user-level copy that `gabo init` installs (D20).
- Engine config (intent 0004 S8, D7): gabo writes only the engine keys it needs to operate, at project level only, merged with the existing project config (every unrelated key kept), reports what it wrote, and has the value verified by `gabo check`; never the user's global engine config. `gabo init` writes every engine config key (intent 0010 S5); this skill writes none and only builds agent files, links and the scoped block of the Main-Session Entry.
- Never write under `~/.codex` or `$CODEX_HOME` (no `config.toml`, no user agents, no hook trust): this skill writes no user-level file but the reviewer link of step 4.
- Overwrite only the agent files named after a spec id, the links this skill makes, the probe file it writes and the gabo main-session entries of the Main-Session Entry section. Any other existing entry at a path you would write (a real file or folder, or a link to another target): leave it and report it; never delete or move it.
- Make every link of steps 2 to 4 on every build, also when the agent files are already current; a build that skips one is not finished.
- Write no secret into any file (R21).
- `gabo.yaml` is a human-only path (intent 0024 S5, D5). Before the rebuild, record `git hash-object gabo.yaml` (`absent` when the file does not exist); compare it after the rebuild. A rebuild writes no key of `gabo.yaml`. When the two differ, add `✗ gabo.yaml changed during the rebuild; next: review it with git diff gabo.yaml` to the report; when equal, print nothing. Report it and leave the file as it is: never restore or edit `gabo.yaml`.

## Sources

Checked on 2026-11-03 against codex-cli 0.163.0 (`codex --version`) and these pages (intent 0002 spec, Evidence):

| fact | used for | source |
|---|---|---|
| Custom agents are TOML files in `.codex/agents/` (project) or `~/.codex/agents/` (user), one agent per file; `name`, `description` and `developer_instructions` are required, and `name` is the identity, not the file name. | agent files | https://learn.chatgpt.com/docs/agent-configuration/subagents |
| Optional agent keys are `model`, `model_reasoning_effort`, `model_reasoning_summary`, `model_verbosity`, `features.*` and `skills.config`; there is no per-agent tool allowlist and no skill preload key. `sandbox_mode` and `approval_policy` in an agent file do not override the parent's configuration. | no tool list, skill line, D6 | https://learn.chatgpt.com/docs/agent-configuration/subagents |
| Skills are discovered in `.agents/skills/` of every directory from the working directory up to the repository root and in `$HOME/.agents/skills`; symlinked skill folders are followed. | catalog skills, fixed-name links, reviewer link | https://learn.chatgpt.com/docs/build-skills |
| Codex reads, per directory from the project root (typically the Git root) down to the working directory, `AGENTS.override.md`, else `AGENTS.md`, at most one file per directory, into every run; the combined size is capped by `project_doc_max_bytes` (32 KiB by default). | main-session block file | https://learn.chatgpt.com/docs/agent-configuration/agents-md |
| A TOML comment starts with `#`; a multi-line basic string `"""…"""` trims the newline right after its opening quotes and treats `\` as an escape. | `gabo-version` and `gabo-engine` comments, escaping | https://toml.io/en/v1.0.0 |

Recheck before you build: run `codex --version`. When it is not 0.163.0, read the Codex pages again and confirm each row. A row that no longer holds: stop, build nothing and report the row with what you found.

## Capabilities

Codex has no per-agent tool allowlist, so the spec's engine-neutral `tools` map to no TOML key: every agent gets the parent session's tools, and what an agent may not do stays in its template body (D6). Report this once per build, not per agent.

## Fixed-Name Links

Each slot of `gabo.yaml` (`slots.<name>.adapter`) whose adapter is a `<folder>/SKILL.md` gets one link `.agents/skills/gabo-<name>` (underscores become hyphens) to `<folder>`, as a relative symlink: `gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory` and so on. An adapter that is not a skill folder gets no link: report it. When the adapter folder already is `.agents/skills/gabo-<name>`, there is nothing to link.

## Main-Session Entry

Codex has no project setting that selects the main-session agent, so the owner's main session gets the scoped block of the Codex row in the Main-Session Entry table of `gabo-agent-templates` (intent 0009 S3, D1). Read the table and the templates from the project catalog; never write the block from memory.

- When `codex` is listed in `runtimes.assistant` (one engine or a list, intent 0013 S4): make `.agents/skills/gabo-assistant` a relative symlink to `../../.gabo/skills/gabo-assistant`, under the same keep, replace and report rules as build step 2, and link every other `main_session.skills` entry (`gabo-assistant`, `gabo-cycle` and `gabo-wave-report` in all, intent 0015 D3) the same way as `.agents/skills/<name>`, to `../../.gabo/skills/<name>`. Then copy the `main_session.block` of `.gabo/skills/gabo-agent-templates/agents.yaml` verbatim, from `<!-- GABO_ASSISTANT_START -->` to `<!-- GABO_ASSISTANT_END -->`, into `AGENTS.override.md` when it exists, else `AGENTS.md`, created when neither exists (both at the repository root). The block names `.gabo/skills/gabo-assistant/SKILL.md` and is at most 2048 bytes; a larger template: stop and report it. Keep exactly one block: remove a gabo block from the other of the two files. When `pi` is listed too, the one block serves both: Pi loads the first existing of `AGENTS.override.md`, `AGENTS.md` and `CLAUDE.md`, which is now the file you wrote, so remove a gabo block from `CLAUDE.md` too; neither build removes the other's block.
- Writing the block: when the file holds the markers, replace only the bytes between the markers; append it after one blank line when absent (a file not ending in a newline first gets one); keep every other byte, so a second build leaves the file byte-identical. Two blocks in one file: refresh the first and remove the others. A file whose markers are unpaired: change nothing and report it. A symlinked file: write through a symlinked file only when its target stays inside the repository; otherwise leave it and report it. When the file with the block exceeds 32 KiB, report it: Codex may cut the appended block at `project_doc_max_bytes`.
- When `codex` is not listed in `runtimes.assistant`, remove its block from `AGENTS.override.md` and `AGENTS.md`, but a block is removed only when no engine listed in `runtimes.assistant` uses its file (Pi uses the same files). Removing a block deletes the markers, the bytes between them and the one blank line written before it, and keeps every other byte; a file left empty is kept and reported, never deleted. The shared `.agents/skills/gabo-assistant` link: this skill removes the shared link only when no engine listed in `runtimes.assistant` uses `.agents/skills` (an engine uses `.agents/skills` when its runtime skill declares `metadata.skill_dir: .agents/skills`), and only when it points to `../../.gabo/skills/gabo-assistant`; it removes the `.agents/skills/<name>` link of every other `main_session.skills` entry under the same rule, keeping one while another spec agent's `skills` lists it.
- On every build, remove the gabo entry of every other full-tier engine in the Main-Session Entry table of `gabo-agent-templates` that is not listed in `runtimes.assistant` (TRD D28: Claude Code, OpenCode, Pi), except any link or block a listed engine's row writes (OpenCode and Pi share `.agents/skills/gabo-assistant`, and Pi shares the block files), recognized only by its gabo identity: the block markers, an agent file `gabo-assistant` carrying a `gabo-version:` line, the `gabo-assistant` link to the catalog folder. Anything without that identity (an agent file or link of the user's own): keep and report it. A build never removes the entry of another engine listed in `runtimes.assistant`, so rebuilding the listed engines in any order leaves every listed entry in place. Never remove Claude Code's `agent` key or OpenCode's `default_agent` key: `gabo init` removes them (intent 0010 S4), and this skill writes and removes no engine config key.

## Build Steps

1. Recheck the sources. Read `gabo.yaml`, `.gabo/VERSION` and the spec.
2. Catalog skills: for each skill in any agent's spec `skills`, make `.agents/skills/<name>` a relative symlink to `../../.gabo/skills/<name>`, so a later `gabo init` refresh reaches it. An entry that already resolves to that folder is kept. A symlink this skill made earlier to another folder is replaced; a real folder or file is left and reported.
3. Fixed-name links as above. Then remove each `.agents/skills/gabo-*` symlink that no spec skill or slot asks for any more and that this skill made: one that points into `../../.gabo/skills/` or to a folder a slot adapter of `gabo.yaml` named before. `.agents/skills/` is shared by every engine that reads it (Codex, Pi and OpenCode), so keep every link any engine declared in `runtimes.*` needs: an engine uses `.agents/skills` when its runtime skill `.gabo/skills/gabo-runtime-<engine>/SKILL.md` declares `metadata.skill_dir: .agents/skills`, and every such engine needs the same spec skills and fixed-name links (one catalog copy, one `gabo.yaml`); also keep `.agents/skills/gabo-assistant` while `runtimes.assistant` names an engine that uses `.agents/skills` (intent 0009 S1); with a list, while any engine listed in `runtimes.assistant` uses it (intent 0013 S4); also keep every `main_session.skills` link while `runtimes.assistant` names an engine that uses `.agents/skills` (intent 0015 D3). Former agents (intent 0015 S6, D4): on every build remove `.codex/agents/gabo-cycle.toml` and `.codex/agents/gabo-reviewer.toml`, left by an earlier build, only when the file carries a `gabo-version` line (a TOML comment); a file without one: leave it and report it. `.agents/skills/` is shared with other tools, so leave and report any other `gabo-*` symlink, and never remove a real folder or file.
4. Reviewer link: make `$HOME/.agents/skills/gabo-reviewer` a symlink to `${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer` with the absolute path. That folder missing: stop and tell the user to run `gabo init`. An existing entry with another target: leave it and report it.
5. Main-session entry: write, refresh or remove it as the Main-Session Entry section says, on every build.
6. For each spec agent, fill its template:

   | slot | value |
   |---|---|
   | `{{id}}`, `{{role}}` | the spec `id` and `role` |
   | `{{gabo_version}}` | the single line of `.gabo/VERSION` |
   | `{{methodology}}` | `gabo-methodology` |
   | `{{adapter:tasks}}`, `{{adapter:repository_host}}`, `{{adapter:memory}}` | `gabo-tasks`, `gabo-repository-host`, `gabo-memory` |
   | `{{reviewer_script}}` | `$HOME/.agents/skills/gabo-reviewer/scripts/reviewer.sh`, written literally: the shell expands it on each machine; filled in `gabo-coder`, the only template that carries the slot |
   | `{{engine}}` | `codex`, the engine as written in `gabo.yaml` (intent 0013 S6) |
   | `{{turn_limit_line}}` | with `wave.max_coder_turns` set: `Turn limit: <n> turns; stop at it and report STOPPED at turn limit.`; unset: remove the whole line; filled in `gabo-coder`, the only template that carries the slot |

   A slot whose link does not exist (its slot has no skill adapter) cannot be filled: skip that agent and report it.
7. Write `.codex/agents/<id>.toml` in exactly this shape, these keys in this order and no others:

   ```toml
   # gabo-version: <x.y.z>
   # gabo-engine: codex
   name = "<id>"
   description = "<role>"
   developer_instructions = """
   <filled body>

   Load these skills by name before work: <skills>.
   """
   ```

   - `# gabo-version: <x.y.z>` is the template's version line moved into a TOML comment, with the value of `.gabo/VERSION`; `gabo check` compares it with the stamp.
   - `# gabo-engine: codex` is the template's engine line moved into a TOML comment; `gabo check` fails an agent file whose filled engine is not `codex` (intent 0013 S6).
   - `<filled body>` is the filled template without its first two lines `gabo-version: …` and `gabo-engine: …` and the blank line after them.
   - The TOML has no turn-limit key: the coder gets no engine key for the turn limit; it travels in the launch input as `turn limit: <n>` and is advisory.
   - `<skills>` are the spec `skills` in spec order, then each fixed-name link the body names (`gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory`, in this order), comma-separated. Codex has no skill preload key, so this line is how the agent gets them. Omit the line and the blank line before it when the list is empty.
   - Escaping: in `description` write each `\` as `\\` and each `"` as `\"`; in `developer_instructions` write each `\` as `\\` and each `"""` as `""\"`. Copy every other byte unchanged.
8. Verify: each file must parse as TOML (with `python3 -c 'import sys, tomllib; tomllib.load(open(sys.argv[1], "rb"))' <file>` when Python 3.11 or newer is present) with `name` equal to its id; then run `gabo check`, which also checks every link of steps 2 to 4. Fix every failure this skill caused and rerun until `gabo check` passes, except the `✗ agent <id> missing` lines of agents you skipped and reported in step 6; report any other failure.
9. Probe, on every build. When this session is not Codex (for example Claude Code or Pi builds the Codex agents during setup), its subagents prove nothing about Codex: skip the probe, change nothing, and report `probe not run: run gabo-runtime-codex inside Codex`. When `.codex/agents` or `.agents/skills` did not exist when this Codex session started (you created it in this build), the session may not see the new agent or links: skip the probe, change nothing, and report `probe not run: restart Codex and rebuild`. Otherwise write `.codex/agents/gabo-probe.toml` in the same shape, with the skill line listing every fixed-name link made in step 3, and the instructions: run `git rev-parse --show-toplevel` once and reply with its output, then one line per skill of the skill line with the name it was listed under and its first Markdown heading. launch it once from the main session, the way spec agents are launched, and expect that no agent launches another; then delete the file whatever the result. It passes when it ran the command and every link shows its adapter's heading. A link that does not load by its link name: in the skill line and in its slot fill, use the adapter's frontmatter `name` instead of the link name, rebuild the affected agents, repeat step 8 and probe once more; still failing, stop and report the link. A `gabo-probe.toml` that this skill did not write: leave it and report it, and report the probe as not run.

## Output Contract

Report the Codex version and whether the sources were rechecked; each agent file written with its id and `gabo-version`; each link made (catalog, fixed-name, reviewer and `gabo-assistant`); the main-session entry written, refreshed or removed, with its file, and each other engine's entry removed or kept; each entry left or slot not filled, with why; that the spec capabilities map to no TOML key; the TOML parse and `gabo check` results; and the probe result when it ran.
