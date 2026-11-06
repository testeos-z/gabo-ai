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
---
## Activation Contract

Load when any of `runtimes.coder`, `runtimes.reviewer` or `runtimes.assistant` in `gabo.yaml` is `codex` and the gabo agents must be built or rebuilt: after setup, after `gabo init` refreshed the catalog, or when `gabo check` reports `✗ agent <id> missing` or an agent built with another version (TRD 8.1, 9.4, D24; intent 0002 S4, S5). Load `gabo-agent-templates` with it: that skill owns the spec, the templates and the slot rules; this skill only turns them into Codex files, and the main-session entry of its table row into the Codex block (intent 0009).

`metadata.agent_dir` above is the directory `gabo check` searches for the agent files of this runtime (it accepts any extension and reads `gabo-version:` in a TOML comment); `metadata.skill_dir` is where it requires the links of build steps 2 and 3, and `metadata.reviewer_link` the user-level link of step 4 (outside CI). Never change them without changing where you build. `gabo check` reports a missing or wrong link as `✗ skill link <path> …` or `✗ reviewer link <path> …`: rebuild with this skill.

## Hard Rules

- Build every agent of `.gabo/skills/gabo-agent-templates/agents.yaml`, each as `.codex/agents/<id>.toml`, whatever runtime each agent plays: `gabo check` requires all of them in `.codex/agents` (D2).
- Read the spec, the templates and `.gabo/VERSION` from the project; never build from memory. A missing or empty `.gabo/VERSION` or catalog: stop and tell the user to run `gabo init`.
- Fill only the template slots, as below, and invent nothing: no TOML key, skill, model or rule the spec and template do not give. No `model`, `sandbox_mode` or `features` key is written, also not for the read-only agents `gabo-plan-reviewer` and `gabo-reviewer`: their limits stay in their instructions (D6).
- Never place `gabo-reviewer` in the project: no `.agents/skills/gabo-reviewer`, no copy of its scripts, and never in an agent's skill line. Only link the user-level copy that `gabo init` installs (D20).
- Never write under `~/.codex` or `$CODEX_HOME` (no `config.toml`, no user agents, no hook trust): this skill writes no user-level file but the reviewer link of step 4.
- Overwrite only the agent files named after a spec id, the links this skill makes, the probe file it writes and the gabo main-session entries of the Main-Session Entry section. Any other existing entry at a path you would write (a real file or folder, or a link to another target): leave it and report it; never delete or move it.
- Make every link of steps 2 to 4 on every build, also when the agent files are already current; a build that skips one is not finished.
- Write no secret into any file (R21).

## Sources

Checked on 2026-11-03 against codex-cli 0.163.0 (`codex --version`) and these pages (intent 0002 spec, Evidence):

| fact | used for | source |
|---|---|---|
| Custom agents are TOML files in `.codex/agents/` (project) or `~/.codex/agents/` (user), one agent per file; `name`, `description` and `developer_instructions` are required, and `name` is the identity, not the file name. | agent files | https://learn.chatgpt.com/docs/agent-configuration/subagents |
| Optional agent keys are `model`, `model_reasoning_effort`, `model_reasoning_summary`, `model_verbosity`, `features.*` and `skills.config`; there is no per-agent tool allowlist and no skill preload key. `sandbox_mode` and `approval_policy` in an agent file do not override the parent's configuration. | no tool list, skill line, D6 | https://learn.chatgpt.com/docs/agent-configuration/subagents |
| Skills are discovered in `.agents/skills/` of every directory from the working directory up to the repository root and in `$HOME/.agents/skills`; symlinked skill folders are followed. | catalog skills, fixed-name links, reviewer link | https://learn.chatgpt.com/docs/build-skills |
| Codex reads, per directory from the project root (typically the Git root) down to the working directory, `AGENTS.override.md`, else `AGENTS.md`, at most one file per directory, into every run; the combined size is capped by `project_doc_max_bytes` (32 KiB by default). | main-session block file | https://learn.chatgpt.com/docs/agent-configuration/agents-md |
| A TOML comment starts with `#`; a multi-line basic string `"""…"""` trims the newline right after its opening quotes and treats `\` as an escape. | `gabo-version` comment, escaping | https://toml.io/en/v1.0.0 |

Recheck before you build: run `codex --version`. When it is not 0.163.0, read the Codex pages again and confirm each row. A row that no longer holds: stop, build nothing and report the row with what you found.

## Capabilities

Codex has no per-agent tool allowlist, so the spec's engine-neutral `tools` map to no TOML key: every agent gets the parent session's tools, and what an agent may not do stays in its template body (D6). Report this once per build, not per agent.

## Fixed-Name Links

Each slot of `gabo.yaml` (`slots.<name>.adapter`) whose adapter is a `<folder>/SKILL.md` gets one link `.agents/skills/gabo-<name>` (underscores become hyphens) to `<folder>`, as a relative symlink: `gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory` and so on. An adapter that is not a skill folder (for example the reviewer agent file) gets no link: report it. When the adapter folder already is `.agents/skills/gabo-<name>`, there is nothing to link.

## Main-Session Entry

Codex has no project setting that selects the main-session agent, so the owner's main session gets the scoped block of the Codex row in the Main-Session Entry table of `gabo-agent-templates` (intent 0009 S3, D1). Read the table and the templates from the project catalog; never write the block from memory.

- When `runtimes.assistant` is `codex`: make `.agents/skills/gabo-assistant` a relative symlink to `../../.gabo/skills/gabo-assistant`, under the same keep, replace and report rules as build step 2. Then copy the `main_session.block` of `.gabo/skills/gabo-agent-templates/agents.yaml` verbatim, from `<!-- GABO_ASSISTANT_START -->` to `<!-- GABO_ASSISTANT_END -->`, into `AGENTS.override.md` when it exists, else `AGENTS.md`, created when neither exists (both at the repository root). The block names `.gabo/skills/gabo-assistant/SKILL.md` and is at most 2048 bytes; a larger template: stop and report it. Keep exactly one block: remove a gabo block from the other of the two files.
- Writing the block: when the file holds the markers, replace only the bytes between the markers; append it after one blank line when absent (a file not ending in a newline first gets one); keep every other byte, so a second build leaves the file byte-identical. Two blocks in one file: refresh the first and remove the others. A file whose markers are unpaired: change nothing and report it. A symlinked file: write through a symlinked file only when its target stays inside the repository; otherwise leave it and report it.
- When `runtimes.assistant` is not `codex`, remove its block from `AGENTS.override.md` and `AGENTS.md`, except from the file the row of the `runtimes.assistant` engine writes its own block to (Pi uses the same files). Removing a block deletes the markers, the bytes between them and the one blank line written before it, and keeps every other byte; a file that only held the block is deleted. The shared `.agents/skills/gabo-assistant` link: this skill removes the shared link only when no declared engine using .agents/skills is runtimes.assistant (an engine is declared when it is a value of `runtimes.*`, and uses `.agents/skills` when its runtime skill declares `metadata.skill_dir: .agents/skills`), and only when it points to `../../.gabo/skills/gabo-assistant`.
- On every build, remove the gabo entry of every other full-tier engine in the Main-Session Entry table of `gabo-agent-templates` that is not `runtimes.assistant` (TRD D28: Claude Code, OpenCode, Pi), recognized only by its gabo identity: the block markers, a key whose value is exactly `gabo-assistant`, an agent file `gabo-assistant` carrying a `gabo-version:` line, the `gabo-assistant` link to the catalog folder. For a key, remove only that member and its separating comma, keeping every other byte of the file. Anything without that identity (a user's own `agent` or `default_agent` value, an agent file or link of their own): keep and report it.

## Build Steps

1. Recheck the sources. Read `gabo.yaml`, `.gabo/VERSION` and the spec.
2. Catalog skills: for each skill in any agent's spec `skills`, make `.agents/skills/<name>` a relative symlink to `../../.gabo/skills/<name>`, so a later `gabo init` refresh reaches it. An entry that already resolves to that folder is kept. A symlink this skill made earlier to another folder is replaced; a real folder or file is left and reported.
3. Fixed-name links as above. Then remove each `.agents/skills/gabo-*` symlink that no spec skill or slot asks for any more and that this skill made: one that points into `../../.gabo/skills/` or to a folder a slot adapter of `gabo.yaml` named before. `.agents/skills/` is shared by every engine that reads it (Codex, Pi and OpenCode), so keep every link any engine declared in `runtimes.*` needs: an engine uses `.agents/skills` when its runtime skill `.gabo/skills/gabo-runtime-<engine>/SKILL.md` declares `metadata.skill_dir: .agents/skills`, and every such engine needs the same spec skills and fixed-name links (one catalog copy, one `gabo.yaml`); also keep `.agents/skills/gabo-assistant` while `runtimes.assistant` names an engine that uses `.agents/skills` (intent 0009 S1). `.agents/skills/` is shared with other tools, so leave and report any other `gabo-*` symlink, and never remove a real folder or file.
4. Reviewer link: make `$HOME/.agents/skills/gabo-reviewer` a symlink to `${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer` with the absolute path. That folder missing: stop and tell the user to run `gabo init`. An existing entry with another target: leave it and report it.
5. Main-session entry: write, refresh or remove it as the Main-Session Entry section says, on every build.
6. For each spec agent, fill its template:

   | slot | value |
   |---|---|
   | `{{id}}`, `{{role}}` | the spec `id` and `role` |
   | `{{gabo_version}}` | the single line of `.gabo/VERSION` |
   | `{{methodology}}` | `gabo-methodology` |
   | `{{adapter:tasks}}`, `{{adapter:repository_host}}`, `{{adapter:memory}}` | `gabo-tasks`, `gabo-repository-host`, `gabo-memory` |
   | `{{reviewer_script}}` | `$HOME/.agents/skills/gabo-reviewer/scripts/reviewer.sh`, written literally: the shell expands it on each machine |

   A slot whose link does not exist (its slot has no skill adapter) cannot be filled: skip that agent and report it.
7. Write `.codex/agents/<id>.toml` in exactly this shape, these keys in this order and no others:

   ```toml
   # gabo-version: <x.y.z>
   name = "<id>"
   description = "<role>"
   developer_instructions = """
   <filled body>

   Load these skills by name before work: <skills>.
   """
   ```

   - `# gabo-version: <x.y.z>` is the template's version line moved into a TOML comment, with the value of `.gabo/VERSION`; `gabo check` compares it with the stamp.
   - `<filled body>` is the filled template without its first line `gabo-version: …` and the blank line after it.
   - `<skills>` are the spec `skills` in spec order, then each fixed-name link the body names (`gabo-methodology`, `gabo-tasks`, `gabo-repository-host`, `gabo-memory`, in this order), comma-separated. Codex has no skill preload key, so this line is how the agent gets them. Omit the line and the blank line before it when the list is empty (the reviewer agent).
   - Escaping: in `description` write each `\` as `\\` and each `"` as `\"`; in `developer_instructions` write each `\` as `\\` and each `"""` as `""\"`. Copy every other byte unchanged.
8. Verify: each file must parse as TOML (with `python3 -c 'import sys, tomllib; tomllib.load(open(sys.argv[1], "rb"))' <file>` when Python 3.11 or newer is present) with `name` equal to its id; then run `gabo check`, which also checks every link of steps 2 to 4. Fix every failure this skill caused and rerun until `gabo check` passes, except the `✗ agent <id> missing` lines of agents you skipped and reported in step 6; report any other failure.
9. Probe, on every build. When this session is not Codex (for example Claude Code or Pi builds the Codex agents during setup), its subagents prove nothing about Codex: skip the probe, change nothing, and report `probe not run: run gabo-runtime-codex inside Codex`. When `.codex/agents` or `.agents/skills` did not exist when this Codex session started (you created it in this build), the session may not see the new agent or links: skip the probe, change nothing, and report `probe not run: restart Codex and rebuild`. Otherwise write `.codex/agents/gabo-probe.toml` in the same shape, with the skill line listing every fixed-name link made in step 3, and the instructions: run `git rev-parse --show-toplevel` once and reply with its output, then one line per skill of the skill line with the name it was listed under and its first Markdown heading. Launch it once as a subagent, then delete the file whatever the result. It passes when it ran the command and every link shows its adapter's heading. A link that does not load by its link name: in the skill line and in its slot fill, use the adapter's frontmatter `name` instead of the link name, rebuild the affected agents, repeat step 8 and probe once more; still failing, stop and report the link. A `gabo-probe.toml` that this skill did not write: leave it and report it, and report the probe as not run.

## Output Contract

Report the Codex version and whether the sources were rechecked; each agent file written with its id and `gabo-version`; each link made (catalog, fixed-name, reviewer and `gabo-assistant`); the main-session entry written, refreshed or removed, with its file, and each other engine's entry removed or kept; each entry left or slot not filled, with why; that the spec capabilities map to no TOML key; the TOML parse and `gabo check` results; and the probe result when it ran.
