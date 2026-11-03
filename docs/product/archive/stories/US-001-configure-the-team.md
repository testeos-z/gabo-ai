# US-001 Configure the team

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | must |
| requirements | R12, R16, R8, R31, R32, R33, R34, R36, R39 |
| depends on | — |

## Story

As a solo developer, I want the assistant to guide me through configuring my agent team, so that it works with my identity and my methodology without silent defaults.

## Acceptance criteria

- Given gabo-ai is installed, When I run `gabo init` in a project without a config, Then it copies the skill catalog byte for byte into a neutral project location, writes a version stamp, installs the user-level `gabo-reviewer` skill, writes no config, builds no agent and prints the embedded setup prompt (Approved (owner 2026-10-27), R32, updated issue #74).
- Given I paste the published setup prompt into my agent, When the agent runs it, Then it installs the gabo CLI, runs `gabo init` and continues with the same setup as when I run `gabo init` myself, and the prompt I pasted is the same literal text embedded in the CLI (Approved (owner 2026-10-27), R36).
- Given the project has a config and its catalog stamp is older than the CLI version or is missing, When I run `gabo init`, Then it leaves the config untouched, replaces the copied catalog folder wholesale (skills removed in the new version are deleted and local edits to copied skills are overwritten; customization belongs in adapters), updates the stamp and tells me to ask my agent to rebuild (Approved (owner 2026-10-27), R32, issue #74).
- Given the project has a config and its catalog stamp matches the CLI version and the user-level `gabo-reviewer` skill is current, When I run `gabo init`, Then it changes nothing and exits 0 with a short message (Approved (owner 2026-10-27), R32, issue #74).
- Given I ask my agent (Claude Code, Pi or another engine with a runtime skill) to configure gabo-ai, When it follows the setup prompt, Then the setup skill asks me the questions one at a time, the config-writer skill writes `gabo.yaml` and the global user file, the runtime skill builds every agent of the neutral agent spec from fixed templates without inventing content, and the agent runs `gabo check` and fixes every failure before reporting setup as done (Approved (owner 2026-10-27), R34).
- Given a project with adapters and a config, When `gabo init` refreshes the catalog, Then only the catalog subtree is removed and rewritten, the adapters and the config are byte-for-byte unchanged, and the copied skills are byte-identical to the embedded catalog (Approved (owner 2026-10-27), R32, issue #74).
- Given a teammate cloned a configured repository whose stamp was written by a newer gabo, When they run `gabo init` interactively with their older CLI, Then gabo asks "this repo needs gabo <w>, you have <v>. Update? This affects all your projects."; on yes it updates itself to exactly the stamp version after verifying the release is authentic (signature or provenance), prints `gabo updated to <v>` and continues; on no it changes nothing, prints the upgrade command and exits 1; the catalog is never refreshed backwards; if it cannot self-update it stops and prints the exact upgrade command (Approved (owner 2026-10-27), R32, O12, issue #74).
- Given the stamp is newer than the running CLI, When `gabo init` runs non-interactively (an agent or CI), Then it changes nothing, prints the upgrade command and exits 1 (Approved (owner 2026-10-27), R32, O12, issue #74).
- Given a PR that changes the stamp (an upgrade PR), When CI and the reviewer run `gabo check` with the CLI they have (at least the base stamp), Then they validate only the config schemas, skip every spec-dependent check (agents, slot coverage, both version checks), report "upgrade PR: only config validated, human merge required" with the changed paths outside the catalog subtree and the agent directories, and the owner merges it; when the base branch has no stamp they use the CLI they have (Approved (owner 2026-10-27), R33, O12, issue #74; O13, issue #79).
- Given an upgrade PR whose new gabo version changes the agent list, When CI and the reviewer run the CLI they have (at least the base stamp), Then the run passes with "upgrade PR: only config validated, human merge required" and the owner merges the PR (Approved (owner 2026-10-27), R33, O12, issue #74; O13, issue #79).
- Given a PR changes the stamp `.gabo/VERSION` (`.gabo/VERSION` until the rename), When the reviewer evaluates it in auto mode, Then it treats the stamp as human-only whatever `review.human_only_paths` says and never merges the PR; only the owner merges it (Approved (owner 2026-10-27), R33, O12, issue #74).
- Given two projects on one machine with different stamps and a CLI matching the newer one, When I run `gabo check` in the older project, Then it only warns "this project uses gabo <stamp>; run gabo init to upgrade it", its CI also only warns and passes with any gabo at least as new as its base stamp, and nothing is upgraded until I choose to (Approved (owner 2026-10-27), R33, O12, issue #74; O13, issue #79).
- Given a project stamped with an older gabo and a newer CLI, When `gabo check` runs, Then it reads the neutral agent spec and the contract operations from the project's copied catalog, not from the binary, validates the project correctly and only warns about the older stamp (Approved (owner 2026-10-27), R33, O13, issue #79).
- Given the stamp is newer than the running CLI, When `gabo check` runs, Then it does not self-update, stays read-only and fails with the exact upgrade command; and CI and the reviewer need gabo at least as new as the base branch's stamp, never the PR head's, failing with the upgrade command otherwise (O13) (Approved (owner 2026-10-27), R33, O12, issue #74; O13, issue #79).
- Given a PR edits the catalog subtree `.gabo/skills/` (`.gabo/skills/` until the rename), When the reviewer evaluates it in auto mode, Then it treats the catalog as human-only whatever `review.human_only_paths` says and never merges the PR; only the owner merges it (Approved (owner 2026-10-27), R33, O13, issue #79).
- Given `gabo init` ran but setup did not (catalog and stamp, no config), When I run `gabo init` again, Then it refreshes the catalog if the stamp is older, prints the setup prompt again and exits 0 (Approved (owner 2026-10-27), R32, issue #74).
- Given the runtime skill builds the agents, When I open any built agent file, Then it carries the `gabo-version` line with the gabo version it was built from (Approved (owner 2026-10-27), R34, O11).
- Given a second developer clones a configured repository without the user-level `gabo-reviewer` skill, or with one that differs from the embedded skill, When they run `gabo init`, Then it installs that skill whatever the stamp says, and leaves the config untouched (Approved (owner 2026-10-27), R32, issue #74).
- Given I upgraded and ran `gabo init` but my agent has not rebuilt the agents yet, When `gabo check` runs, Then it fails for each agent whose version line differs from the catalog stamp with "agent <id> built with <v>, catalog is <w>: ask your agent to rebuild" (Approved (owner 2026-10-27), R33, PRD O11).
- Given a configured project, When I, an agent, the reviewer or CI runs `gabo check`, Then it validates the config files against their JSON Schemas, confirms every agent of the neutral agent spec is present, checks slot coverage and checks that the catalog version stamp matches the CLI version, warning locally without failing on an older stamp with "this project uses gabo <stamp>; run gabo init to upgrade it", on a missing stamp with "no stamp: run gabo init" and on a newer stamp with the exact upgrade command, and checks that every agent's version line exists and matches the catalog stamp (failing with "agent <id> built with <v>, catalog is <w>: ask your agent to rebuild"), writes nothing and exits non-zero on any failure (Approved (owner 2026-10-27), R33).
- Superseded (Proposed (auto), issue #148): Given I list the gabo CLI commands, When I read the help, Then only `gabo init`, `gabo check` and `gabo version` exist (Approved (owner 2026-10-27), R31).
- Given I list the gabo CLI commands, When I read the help, Then only `gabo init`, `gabo check`, `gabo version` and `gabo prompt` exist (Proposed (auto), owner request 2026-10-29, issue #148, R31).
- Given I need the setup prompt again (for example in a new agent session), When I run `gabo prompt` in any directory, Then it prints exactly the setup prompt embedded in the CLI, the same text `gabo init` prints and the documentation publishes, prints nothing else and exits 0 (Proposed (auto), issue #148, R39, R36).
- Given a project with or without a catalog, stamp or config, When I run `gabo prompt`, Then no file, config, catalog, stamp or reviewer skill is created or changed (Proposed (auto), issue #148, R39).
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

- App flow journey: [J1 in APP_FLOW](../APP_FLOW-0.7.md).
- Approved (owner 2026-10-26): acceptance criteria aligned with TRD D18–D20 (the Go CLI only compiles the initial configuration; work is done by agents guided by skills).
- Approved (owner 2026-10-26): no setup terminal UI; setup starts from a pasted prompt (UX_UI 1.8, TRD D21); agents write config by editing the YAML and `gbai generate` validates it (TRD D18). Superseded in part (owner 2026-10-27): `gabo init` starts setup, the user's agent builds config and agents from skills, and `gabo check` validates (PRD R31–R34, issue #51).
