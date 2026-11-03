# Intent 0001 Claude Code plugin

| field | value |
|---|---|
| intent id | 0001-claude-code-plugin |
| status | Draft |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-03 |

## Problem

A Claude Code user cannot install gabo's catalog skills from a Claude Code plugin marketplace. The skills, and the hooks that could run gabo's deterministic checks (`gabo check`, the intent stage gate), only reach a project through `gabo init`, so there is no marketplace install and no shared update path for them across projects.

## Who is affected

Owners and teams who run gabo inside Claude Code. It happens every time they set up gabo in a new project or want the latest catalog skills and checks.

## Desired outcome

- A Claude Code user can install gabo's catalog skills from a marketplace as a Claude Code plugin.
- Optionally, the same install provides hooks that run the deterministic checks (`gabo check`, the intent stage gate).
- `gabo init` keeps working alongside the plugin for per-project config, slots and adapters.

## Constraints

- The plugin complements `gabo init`; it does not replace per-project config, slots or adapters (owner brief, 2026-11-03).
- Skills already use the shared SKILL.md format; the change reuses it rather than introducing a second skill format (owner brief).
- Hooks are optional and limited to the deterministic checks (owner brief).

## Open questions

| # | question | status |
|---|---|---|
| 1 | Which skills from the catalog go into the plugin, and which stay only in `gabo init`? | carried to spec |
| 2 | Are the hooks for `gabo check` and the intent stage gate in the first release, or later? | open |
| 3 | Which marketplace hosts the plugin (gabo's own repository or another)? | open |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#172 I0001: Claude Code plugin](https://github.com/testeos-z/gabo-ai/issues/172) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| related intent | 0002-codex-plugin, 0003-pi-package, 0004-opencode-runtime |
