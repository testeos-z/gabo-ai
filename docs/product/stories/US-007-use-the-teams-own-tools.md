# US-007 Use the team's own tools

| field | value |
|---|---|
| status | Approved (auto proposal, owner 2026-10-26) |
| user | tech lead |
| priority | must |
| requirements | R8, R11, R34, R35 |
| depends on | US-001 |

## Story

As a tech lead, I want gbai to use my team's methodology and tracker, so that I add agents without changing how the team works.

## Acceptance criteria

- Given the project config declares the team's coder methodology, When coders work, Then they load it under the fixed name `gbai-methodology`.
- Given the team switches methodology, When I edit one line, ask my agent to rebuild the affected agents and run `gabo check`, Then the change is validated and the coders use the new methodology (updated owner 2026-10-27; earlier: rerun `gbai generate`).
- Given my team uses Claude Code, Pi or both, When my agent configures gabo-ai, Then the shipped, tested runtime skill for each engine builds the agents of the neutral agent spec in that engine's format from fixed templates, and for Pi it first requires a subagent package (gentle-pi recommended, another acceptable) and refuses with what to install when none is present (Approved (owner 2026-10-27), R34).
- Given my team uses an engine with subagents and skills but no runtime skill, When I ask my agent to add it, Then the runtime builder researches the engine's official documentation first, checks it against the installed version and its CLI help, cites each source and version, asks me one concrete question when the docs are unclear (or asks for documentation links when it has no web access or documentation MCP), tests a probe agent and `gabo check`, and writes a reusable runtime skill, with no gabo-ai code change or release (Approved (owner 2026-10-27), R35).
- Given my engine has no subagent and skill support or an equivalent, When I ask my agent to add it, Then the runtime builder refuses and says why (Approved (owner 2026-10-27), R35).
- Superseded (owner 2026-10-27): Given `generation.targets` lists Claude Code, Pi or both, When `gbai generate` runs, Then it installs each runtime's skills, agents and tool registrations (MCP servers, plugins or packages, as that runtime supports them) from one installer table (Approved (owner 2026-10-26), TRD D21); for a Pi target it first requires a subagent package (gentle-pi recommended, another acceptable) and refuses with what to install when none is present (Approved (owner 2026-10-26)).
- Given the team uses GitHub issues, When gbai runs a wave, Then tasks are read and updated in GitHub issues.
- Given the team needs another tracker, When a provider skill plus a plugin or MCP is added, Then it works without changing the agents' skills.

## Out of scope

- Plane, Trello and other tracker providers in the first release (Later).
- Imposing a development methodology (non-goal).
- Replacing the tracker or being one (non-goal).

## Notes

- App flow journey: [J7 in APP_FLOW](../APP_FLOW.md).
- Approved (owner 2026-10-26): acceptance criteria aligned with TRD D18–D20 (the Go CLI only compiles the initial configuration; work is done by agents guided by skills). Superseded in part (owner 2026-10-27): runtime support lives in runtime skills and the runtime builder; no Go installer table (PRD R34, R35, issue #51).
