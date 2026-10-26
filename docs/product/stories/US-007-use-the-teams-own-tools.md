# US-007 Use the team's own tools

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | tech lead |
| priority | must |
| requirements | R8, R11 |
| depends on | US-001 |

## Story

As a tech lead, I want gbai to use my team's methodology and tracker, so that I add agents without changing how the team works.

## Acceptance criteria

- Given the project config declares the team's coder methodology, When coders work, Then they load it under the fixed name `gbai-methodology`.
- Given the team switches methodology, When I edit one line and rerun `gbai generate`, Then the change is validated and the coders use the new methodology.
- Given `generation.targets` lists Claude Code, Pi or both, When `gbai generate` runs, Then it installs each runtime's skills, agents and tool registrations (MCP servers, plugins or packages, as that runtime supports them) from one installer table (Approved (owner 2026-10-26), TRD D21); for a Pi target it first requires a subagent package (gentle-pi recommended, another acceptable) and refuses with what to install when none is present (Approved (owner 2026-10-26)).
- Given the team uses GitHub issues, When gbai runs a wave, Then tasks are read and updated in GitHub issues.
- Given the team needs another tracker, When a provider skill plus a plugin or MCP is added, Then it works without changing the agents' skills.

## Out of scope

- Plane, Trello and other tracker providers in the first release (Later).
- Imposing a development methodology (non-goal).
- Replacing the tracker or being one (non-goal).

## Notes

- App flow journey: [J7 in APP_FLOW](../APP_FLOW.md).
- Approved (owner 2026-10-26): acceptance criteria aligned with TRD D18–D20 (the Go CLI only compiles the initial configuration; work is done by agents guided by skills).
