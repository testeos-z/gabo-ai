# Intent 0010 gabo init writes the main-session key

| field | value |
|---|---|
| intent id | 0010-gabo-init-writes-the-main-session-key |
| status | Accepted |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-06 |

## Problem

`gabo init` writes some engine config keys, and the runtime skills write the others, so setup is not complete after `gabo init` and fails where an agent may not write engine settings. The vulcam session reported it (cappy-lab/vulcam PRs #650 and #652). Owner's brief, verbatim: "Reported by the vulcam session (cappy-lab/vulcam PRs #650 and #652): gabo init 0.2.8 merges the Claude integration keys (extraKnownMarketplaces gabo-ai, enabledPlugins gabo@gabo-ai) into .claude/settings.json but not "agent": "gabo-assistant"; that key is left to the agent during the runtime rebuild (gabo-runtime-claude Main-Session Entry step 3). In projects whose hooks stop agents from writing Claude settings (vulcam), the agent cannot write it, gabo check reports "gabo assistant entry incomplete for claude" with "next: ask your agent to rebuild the agents", and the owner loops until adding it by hand."

The `gabo check` hint makes it worse: it sends the owner back to a rebuild that cannot write the key, so following the hint never clears the failure.

## Who is affected

Every owner of a gabo project whose `runtimes.assistant` is `claude` or `opencode`, on every setup and every change of `runtimes.assistant`; most of all projects whose own hooks stop agents from writing engine settings, such as cappy-lab/vulcam. The same split affects OpenCode projects, whose runtime skill writes `default_agent` and `experimental.subagent_depth` into the project OpenCode config.

## Desired outcome

Owner's scope, verbatim: "Scope: when runtimes.assistant is claude, gabo init merges agent: "gabo-assistant" into .claude/settings.json with the same rule as runtime step 3 (keep a user own value and report it; invalid JSON: change nothing, report); OpenCode equivalent default_agent when runtimes.assistant is opencode; remove the gabo value when the assistant engine changes; runtime skills stop writing engine config keys and only build agent files and links; gabo check next: hint names gabo init (or the exact key and file) when that key is the only gap."

- One `gabo init` run writes every engine config key gabo needs; the owner never adds a key by hand.
- Setup works in projects whose hooks stop agents from writing engine settings.
- `gabo init` is the only writer of engine config keys; the runtime skills only build agent files and links.
- When the main-session key is the gap, `gabo check` names `gabo init` (or the exact key and file), so following its hint clears the failure.
- The change ships in gabo release 0.2.10. The version bump is not part of this intent.

Rationale to record, owner's brief, verbatim: "Rationale to record: one command completes setup; works with guard hooks; gabo init is the single writer of engine config; clear guidance, no loop."

## Constraints

- `gabo init` writes only project-level files, inserts only the keys it owns, keeps every other byte and every user-set value (reported as kept), and never touches the user's global engine config or `gabo.yaml` (PRD R32, TRD 9.4, intent 0006).
- `gabo check` stays read-only and engine-neutral outside its engine-key rules (PRD R33, TRD D24, D28).
- Every change assesses whether `gabo check` must change (TRD 8.5, intent 0008 S10).
- `.claude/`, `.agents/`, `.codex/`, `plugins/`, `AGENTS.md`, `CLAUDE.md` and `gabo.yaml` are `review.human_only_paths`: catalog skill copies under `plugins/` and this repository's own `.claude/settings.json` need a human-only merge.
- About 400 changed lines per reviewable task (TRD, dev-lead plan template).

## Open questions

| # | question | status |
|---|---|---|
| 1 | The scoped block that the Codex and Pi runtime skills write into `AGENTS.md` (or `AGENTS.override.md`, `CLAUDE.md`) is an instructions file, not an engine config key. Does it stay with the runtime skills, or does `gabo init` write it too? | answered 2026-11-06: it stays with the runtime skills (spec D3 A) |
| 2 | A flexible-tier engine's runtime skill (written with `gabo-runtime-builder`) writes its own main-session key today, and `gabo init` writes nothing for a flexible-tier engine (PRD R32, TRD D28). Who writes that key now? | answered by the owner on 2026-11-06: the runtime skill keeps writing it (spec D4 A) |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#354 I0010: gabo init writes the main-session key](https://github.com/testeos-z/gabo-ai/issues/354) |
| PRD | [PRD](../../PRD.md) R32, R33 |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| review findings | cappy-lab/vulcam PRs #650 and #652 (the vulcam session's report) |
| related intent | 0009-main-session-defaults-to-the-gabo-assistant (the main-session entry), 0006-engine-integration-at-init (the `gabo init` engine keys), 0004-opencode-runtime (`experimental.subagent_depth`), 0008-stricter-gabo-check (TRD 8.5) |
