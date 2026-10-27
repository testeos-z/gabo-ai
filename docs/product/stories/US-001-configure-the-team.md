# US-001 Configure the team

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | must |
| requirements | R12, R16, R8, R31, R32, R33, R34, R36 |
| depends on | — |

## Story

As a solo developer, I want the assistant to guide me through configuring my agent team, so that it works with my identity and my methodology without silent defaults.

## Acceptance criteria

- Given gbai is installed, When I run `gbai init` in a project without a config, Then it copies the skill catalog byte for byte into a neutral project location, writes a short engine-neutral entry file, installs the user-level `gbai-reviewer` skill, writes no config and builds no agent, and tells me to open my agent here and ask it to configure gbai (Approved (owner 2026-10-27), R32).
- Given I paste the published setup prompt into my agent, When the agent runs it, Then it installs the gbai CLI, runs `gbai init` and continues with the same setup as when I run `gbai init` myself, and the prompt I pasted is the same literal text embedded in the CLI (Approved (owner 2026-10-27), R36).
- Given the project already has a config, When I run `gbai init`, Then it writes nothing (no catalog copy, no entry file, no `gbai-reviewer` reinstall), exits non-zero and points me to `gbai check` (Approved (owner 2026-10-27), R32).
- Given I ask my agent (Claude Code, Pi or another engine with a runtime skill) to configure gbai, When it follows the entry file, Then the setup skill asks me the questions one at a time, the config-writer skill writes `gbai.yaml` and the global user file, the runtime skill builds every agent of the neutral agent spec from fixed templates without inventing content, and the agent runs `gbai check` and fixes every failure before reporting setup as done (Approved (owner 2026-10-27), R34).
- Given a configured project, When I, an agent, the reviewer or CI runs `gbai check`, Then it validates the config files against their JSON Schemas, confirms every agent of the neutral agent spec is present, reports any agent stale against its template, writes nothing and exits non-zero on any failure (Approved (owner 2026-10-27), R33).
- Given I list the gbai CLI commands, When I read the help, Then only `gbai init`, `gbai check` and `gbai version` exist (Approved (owner 2026-10-27), R31).
- Superseded (owner 2026-10-27): Given I paste the setup prompt from the gbai documentation into Claude Code or Pi, When the agent runs it, Then it installs gbai, asks the setup questions, writes the config files and runs `gbai generate`, which installs the agents and per-runtime artifacts for each runtime in `generation.targets` (Approved (owner 2026-10-26)). Since 2026-10-27 the pasted prompt is the second entry, with the text embedded in the CLI (R36).
- Given I run setup for the first time, When the assistant configures a slot, Then it asks me before configuring it and only suggests options, applying no silent default.
- Given I am configuring the team, When the assistant asks where each agent runs and which tracker is used, Then it explains whether and how the chosen agents can communicate.
- Given Pi is one of the chosen runtimes and no Pi subagent package is installed, When setup runs, Then the assistant explains that Pi core has no subagents, offers to install the recommended gentle-pi and `gentle-shell`, installs them only if I accept, accepts another subagent package I already use, and never edits that package's own configuration (Approved (owner 2026-10-26), TRD D21).
- Given no Pi subagent package is present, When the Pi runtime skill is asked to build Pi agents, Then it refuses and says what to install (updated owner 2026-10-27; earlier `gbai generate` refused the Pi target).
- Given I run setup, When the assistant reaches the methodology and memory slots, Then it offers the gentle-ai stack (ODD with RDD as coder-side validation, Engram) as the recommended, not required, option and installs it only if I accept (Approved (owner 2026-10-26)).
- Given my git user has no alias in the developer catalog, When setup runs, Then the assistant asks once how to call me, stores the mapping and from then on addresses me by that alias.
- Given the project config declares `slots.methodology.adapter`, When setup runs, Then the runtime skill links the methodology under the fixed name `gbai-methodology`, and changing it requires editing one line in `gbai.yaml`, having the agent rebuild the affected agents and a passing `gbai check` (updated owner 2026-10-27; earlier `gbai config set` and `gbai generate`).

## Out of scope

- Company-internal organization of developers (R16).
- Imposing a development methodology (non-goal).
- Providing its own infrastructure (non-goal).

## Notes

- App flow journey: [J1 in APP_FLOW](../APP_FLOW.md).
- Approved (owner 2026-10-26): acceptance criteria aligned with TRD D18–D20 (the Go CLI only compiles the initial configuration; work is done by agents guided by skills).
- Approved (owner 2026-10-26): no setup terminal UI; setup starts from a pasted prompt (UX_UI 1.8, TRD D21); agents write config by editing the YAML and `gbai generate` validates it (TRD D18). Superseded in part (owner 2026-10-27): `gbai init` starts setup, the user's agent builds config and agents from skills, and `gbai check` validates (PRD R31–R34, issue #51).
