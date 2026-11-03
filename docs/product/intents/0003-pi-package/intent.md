# Intent 0003 Pi package

| field | value |
|---|---|
| intent id | 0003-pi-package |
| status | Draft |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-03 |

## Problem

A Pi user cannot install gabo with `pi install`. There is no gabo Pi package, so gabo's skills and its check hooks do not reach Pi, and a Pi setup without subagents gets no clear guidance about what gabo needs.

## Who is affected

Owners and teams who run gabo inside Pi. It happens every time they set up gabo in a new project, and every time gabo runs on a Pi install that lacks a subagent package.

## Desired outcome

- A Pi user can install gabo with `pi install`, from npm or git.
- The installed package makes gabo's skills and extensions available, as listed in its manifest.
- On a Pi install without a subagent package, gabo refuses to run and recommends gentle-pi plus the full Gentle AI framework.

## Constraints

- No subagent fallback: gabo refuses on Pi without a subagent package and recommends gentle-pi plus the full Gentle AI framework (owner decision, 2026-11-03).
- The package manifest lists skills and extensions (owner brief).
- Hooks on Pi are TypeScript extensions (owner brief).

## Open questions

| # | question | status |
|---|---|---|
| 1 | Package scope: what does the gabo package ship itself, and what does it rely on gentle-pi for? | open |
| 2 | Is the package distributed through npm, git, or both in the first release? | open |
| 3 | How does gabo detect that a subagent package is present before refusing? | carried to spec |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#174 I0003: Pi package](https://github.com/testeos-z/gabo-ai/issues/174) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| related intent | 0001-claude-code-plugin |
