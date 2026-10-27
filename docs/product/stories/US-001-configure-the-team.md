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

- Given gbai is installed, When I run `gabo init` in a project without a config, Then it copies the skill catalog byte for byte into a neutral project location, writes a version stamp, installs the user-level `gabo-reviewer` skill, writes no config, builds no agent and prints the embedded setup prompt (Approved (owner 2026-10-27), R32, updated issue #74).
- Given I paste the published setup prompt into my agent, When the agent runs it, Then it installs the gabo CLI, runs `gabo init` and continues with the same setup as when I run `gabo init` myself, and the prompt I pasted is the same literal text embedded in the CLI (Approved (owner 2026-10-27), R36).
- Given the project has a config and its catalog stamp differs from the CLI version or is missing, When I run `gabo init`, Then it leaves the config untouched, replaces the copied catalog folder wholesale (skills removed in the new version are deleted and local edits to copied skills are overwritten; customization belongs in adapters), updates the stamp and tells me to ask my agent to rebuild (Approved (owner 2026-10-27), R32, issue #74).
- Given the project has a config and its catalog stamp matches the CLI version, When I run `gabo init`, Then it changes nothing and exits 0 with a short message (Approved (owner 2026-10-27), R32, issue #74).
- Given I ask my agent (Claude Code, Pi or another engine with a runtime skill) to configure gabo-ai, When it follows the setup prompt, Then the setup skill asks me the questions one at a time, the config-writer skill writes `gabo.yaml` and the global user file, the runtime skill builds every agent of the neutral agent spec from fixed templates without inventing content, and the agent runs `gabo check` and fixes every failure before reporting setup as done (Approved (owner 2026-10-27), R34).
- Given a project with adapters and a config, When `gabo init` refreshes the catalog, Then only the catalog subtree is removed and rewritten, the adapters and the config are byte-for-byte unchanged, and the copied skills are byte-identical to the embedded catalog (Approved (owner 2026-10-27), R32, issue #74).
- Given a teammate cloned a configured repository whose stamp was written by a newer gabo, When they run `gabo check` with their older CLI, Then it fails with the stamp mismatch and tells them to upgrade gabo; CI and the reviewer run the stamped version, pinned (Approved (owner 2026-10-27), R33, issue #74).
- Given a second developer clones a configured repository without the user-level `gabo-reviewer` skill, or with one that differs from the embedded skill, When they run `gabo init`, Then it installs that skill whatever the stamp says, and leaves the config untouched (Approved (owner 2026-10-27), R32, issue #74).
- Given I upgraded and ran `gabo init` but my agent has not rebuilt the agents yet, When `gabo check` runs, Then it fails for each agent whose version line differs from the catalog stamp with "agent <id> built with <v>, catalog is <w>: ask your agent to rebuild" (Approved (owner 2026-10-27), R33, PRD O11).
- Given a configured project, When I, an agent, the reviewer or CI runs `gabo check`, Then it validates the config files against their JSON Schemas, confirms every agent of the neutral agent spec is present, checks slot coverage and checks that the catalog version stamp matches the CLI version, a missing stamp counting as a mismatch (failing with "run `gabo init`, then ask your agent to rebuild" on a mismatch), and checks that every agent's version line exists and matches the catalog stamp (failing with "agent <id> built with <v>, catalog is <w>: ask your agent to rebuild"), writes nothing and exits non-zero on any failure (Approved (owner 2026-10-27), R33).
- Given I list the gabo CLI commands, When I read the help, Then only `gabo init`, `gabo check` and `gabo version` exist (Approved (owner 2026-10-27), R31).
- Superseded (owner 2026-10-27): Given I paste the setup prompt from the gbai documentation into Claude Code or Pi, When the agent runs it, Then it installs gbai, asks the setup questions, writes the config files and runs `gbai generate`, which installs the agents and per-runtime artifacts for each runtime in `generation.targets` (Approved (owner 2026-10-26)). Since 2026-10-27 the pasted prompt is the second entry, with the text embedded in the CLI (R36).
- Given I run setup for the first time, When the assistant configures a slot, Then it asks me before configuring it and only suggests options, applying no silent default.
- Given I am configuring the team, When the assistant asks where each agent runs and which tracker is used, Then it explains whether and how the chosen agents can communicate.
- Given Pi is one of the chosen runtimes and no Pi subagent package is installed, When setup runs, Then the assistant explains that Pi core has no subagents, offers to install the recommended gentle-pi and `gentle-shell`, installs them only if I accept, accepts another subagent package I already use, and never edits that package's own configuration (Approved (owner 2026-10-26), TRD D21).
- Given no Pi subagent package is present, When the Pi runtime skill is asked to build Pi agents, Then it refuses and says what to install (updated owner 2026-10-27; earlier `gbai generate` refused the Pi target).
- Given I run setup, When the assistant reaches the methodology and memory slots, Then it offers the gentle-ai stack (ODD with RDD as coder-side validation, Engram) as the recommended, not required, option and installs it only if I accept (Approved (owner 2026-10-26)).
- Given my git user has no alias in the developer catalog, When setup runs, Then the assistant asks once how to call me, stores the mapping and from then on addresses me by that alias.
- Given the project config declares `slots.methodology.adapter`, When setup runs, Then the runtime skill links the methodology under the fixed name `gabo-methodology`, and changing it requires editing one line in `gabo.yaml`, having the agent rebuild the affected agents and a passing `gabo check` (updated owner 2026-10-27; earlier `gbai config set` and `gbai generate`).

## Out of scope

- Company-internal organization of developers (R16).
- Imposing a development methodology (non-goal).
- Providing its own infrastructure (non-goal).

## Notes

- App flow journey: [J1 in APP_FLOW](../APP_FLOW.md).
- Approved (owner 2026-10-26): acceptance criteria aligned with TRD D18–D20 (the Go CLI only compiles the initial configuration; work is done by agents guided by skills).
- Approved (owner 2026-10-26): no setup terminal UI; setup starts from a pasted prompt (UX_UI 1.8, TRD D21); agents write config by editing the YAML and `gbai generate` validates it (TRD D18). Superseded in part (owner 2026-10-27): `gbai init` starts setup, the user's agent builds config and agents from skills, and `gbai check` validates (PRD R31–R34, issue #51).
