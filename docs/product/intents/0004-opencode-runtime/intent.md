# Intent 0004 OpenCode runtime

| field | value |
|---|---|
| intent id | 0004-opencode-runtime |
| status | Draft |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-03 |

## Problem

gabo has no runtime support for OpenCode. An OpenCode user cannot run gabo's agents and skills with the runtime guidance gabo gives other runtimes, and the intent stage gate is not enforced while they work.

## Who is affected

Owners and teams who run gabo inside OpenCode. It happens on every gabo session in OpenCode, including waves that use worktrees.

## Desired outcome

- An OpenCode user can run gabo through a `gabo-runtime-opencode` skill, using OpenCode's native subagents and native SKILL.md skills.
- Optionally, a small OpenCode plugin enforces the intent stage gate on events such as `tool.execute.before`.
- The gabo reviewer runs once per review on OpenCode.

## Constraints

- OpenCode has native subagents (`.opencode/agents/*.md`, `mode: subagent`, the Task tool) and native SKILL.md skills (owner brief, 2026-11-03).
- The optional plugin is a JS/TS module whose functions fire on events (owner brief).
- No third-party dependency (owner brief).
- A skill name must equal its folder name, which conflicts with gabo's fixed-name adapter links (OpenCode behavior, owner brief).
- The Task tool has no working-directory argument, which affects worktree-based work (OpenCode behavior, owner brief).
- `external_directory` defaults to ask (OpenCode behavior, owner brief).
- OpenCode also reads `.claude/skills`, so the reviewer could appear twice (OpenCode behavior, owner brief).

## Open questions

| # | question | status |
|---|---|---|
| 1 | How do gabo's fixed-name adapter links coexist with OpenCode's rule that a skill name equals its folder name? | open |
| 2 | How do coders work in their own worktree when the Task tool has no working-directory argument? | open |
| 3 | How is the `external_directory` ask default handled for worktrees outside the project? | open |
| 4 | How is the reviewer prevented from loading twice when OpenCode reads both its own skills and `.claude/skills`? | open |
| 5 | Is the stage-gate plugin in the first release, or later? | open |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#175 I0004: OpenCode runtime](https://github.com/testeos-z/gabo-ai/issues/175) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| related intent | 0001-claude-code-plugin |
