# US-001 Configure the team

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | solo developer |
| priority | must |
| requirements | R12, R16, R8 |
| depends on | — |

## Story

As a solo developer, I want the assistant to guide me through configuring my agent team, so that it works with my identity and my methodology without silent defaults.

## Acceptance criteria

- Given I paste the setup prompt from the gbai documentation into Claude Code or Pi, When the agent runs it, Then it installs gbai, asks the setup questions, writes the config files and runs `gbai generate`, which installs the agents and per-runtime artifacts for each runtime in `generation.targets` (Approved (owner 2026-10-26)).
- Given I run setup for the first time, When the assistant configures a slot, Then it asks me before configuring it and only suggests options, applying no silent default.
- Given I am configuring the team, When the assistant asks where each agent runs and which tracker is used, Then it explains whether and how the chosen agents can communicate.
- Given Pi is in `generation.targets` and no Pi subagent package is installed, When setup runs, Then the assistant explains that Pi core has no subagents, offers to install the recommended gentle-pi and `gentle-shell`, installs them only if I accept, accepts another subagent package I already use, and never edits that package's own configuration (Approved (owner 2026-10-26), TRD D21).
- Given no Pi subagent package is present, When `gbai generate` compiles a Pi target, Then it refuses and prints what to install (Approved (owner 2026-10-26)).
- Given I run setup, When the assistant reaches the methodology and memory slots, Then it offers the gentle-ai stack (ODD with RDD as coder-side validation, Engram) as the recommended, not required, option and installs it only if I accept (Approved (owner 2026-10-26)).
- Given my git user has no alias in the developer catalog, When setup runs, Then the assistant asks once how to call me, stores the mapping and from then on addresses me by that alias.
- Given the project config declares `slots.methodology.adapter`, When setup runs, Then the methodology is linked under the fixed name `gbai-methodology` and changing it requires editing one line (directly in `gbai.yaml`, or with `gbai config set`) and rerunning `gbai generate`, which validates the change.

## Out of scope

- Company-internal organization of developers (R16).
- Imposing a development methodology (non-goal).
- Providing its own infrastructure (non-goal).

## Notes

- App flow journey: [J1 in APP_FLOW](../APP_FLOW.md).
- Approved (owner 2026-10-26): acceptance criteria aligned with TRD D18–D20 (the Go CLI only compiles the initial configuration; work is done by agents guided by skills).
- Approved (owner 2026-10-26): no setup terminal UI; setup starts from a pasted prompt (UX_UI 1.8, TRD D21); agents write config by editing the YAML and `gbai generate` validates it (TRD D18).
