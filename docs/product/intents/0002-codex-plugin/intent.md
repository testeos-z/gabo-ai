# Intent 0002 Codex plugin

| field | value |
|---|---|
| intent id | 0002-codex-plugin |
| status | Draft |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-03 |

## Problem

A Codex user has no installable gabo plugin. gabo's catalog skills, its agents and the hooks for its deterministic checks cannot be installed into Codex the way they could be into Claude Code, so Codex users depend on per-project setup alone.

## Who is affected

Owners and teams who run gabo inside Codex. It happens every time they set up gabo in a new project or update its skills, agents or checks.

## Desired outcome

- A Codex user can install gabo's catalog skills as a Codex plugin.
- The plugin provides gabo's agents in a form Codex runs.
- Optionally, the plugin provides hooks for the deterministic checks (`gabo check`, the intent stage gate).

## Constraints

- Codex needs its own plugin manifest (owner brief, 2026-11-03).
- Agents must be in Codex's format, which is distinct from Claude's markdown agents (owner brief). Reference: repository ghiaog123/ai-native-sdlc ships both `.claude-plugin` and `.codex-plugin`.
- Hooks have their own Codex config but share the check scripts with the other runtimes (owner brief).
- The plugin complements `gabo init`, as in intent 0001 (owner brief).

## Open questions

| # | question | status |
|---|---|---|
| 1 | Which gabo agents must exist in Codex's format for the first release? | carried to spec |
| 2 | Can one source of truth generate both the Claude and the Codex agent formats? | open |
| 3 | Are the hooks in the first release, or later? | open |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#173 I0002: Codex plugin](https://github.com/testeos-z/gabo-ai/issues/173) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| related intent | 0001-claude-code-plugin |
