# Intent 0008 Stricter gabo check

| field | value |
|---|---|
| intent id | 0008-stricter-gabo-check |
| status | Accepted |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-06 |

## Problem

`gabo check` passes on projects whose agents are not what the neutral agent spec says. It confirms that a file named after each spec agent exists in each runtime's agent directory, but it does not confirm that each agent file loads the skills the spec lists for that agent: the skill links are checked once per skill for the whole runtime, so an agent that lost one of its skills still passes as long as another agent keeps the link. Pi declares no skill directory in its runtime skill, so a Pi project gets no skill check at all. When the catalog version stamp is missing or unreadable, `gabo check` fails on the stamp but stops comparing the agents' version lines, so it says nothing about which agents are stale. The owner, verbatim: "el check debe ser mas estricto y revisar que esten todos los agentes, que tengan las skills correctas, que tengan la version correcta." and "1. haremos check mas estricto y que revise todo".

`gabo check` also falls behind when the product changes: a change that adds or alters something an agent or a runtime file must carry does not have to say whether `gabo check` should verify it. The owner, verbatim: "1. pondremos una regla cada cambio debe revisar si es necesario actualizar el check".

## Who is affected

Every owner and team who builds or rebuilds gabo agents with any of the four full-tier engines (`claude`, `codex`, `pi`, `opencode`), and the reviewer and CI, which trust `gabo check` as the single validation point (PRD R33). It happens on every rebuild after `gabo init`, on every hand edit of an agent file, and on every change to the neutral agent spec or a runtime skill.

## Desired outcome

- `gabo check` fails when any spec agent of any declared full-tier engine is missing, does not load exactly the skills the spec lists for it, or does not carry the catalog version, for all four engines, Pi included.
- Every engine that reads `.agents/skills` (Codex, OpenCode, Pi) uses it as its skill directory, sharing one set of gabo links; only Claude Code keeps `.claude/skills`. Owner, verbatim (2026-11-06): "ademas los agentes que usan .agents/skills/**.md debemos usarlo".
- `gabo check` reports the agent version state even when the catalog stamp is missing or unreadable, instead of skipping it.
- Every change's plan and every coder task say whether `gabo check` must be updated, as a written project convention.
- The change ships in gabo release 0.2.8 (tag `apps/cli/v0.2.8`). Owner, verbatim: "3. subiremos la version a 0.2.8".

## Constraints

- `gabo check` stays read-only and, for a flexible-tier engine, engine-neutral (PRD R33, TRD D24, D28); runtime knowledge stays in the runtime skills (TRD D24), so any per-engine reading is declared by the runtime skill, not hard-coded in Go.
- The neutral agent spec `gabo-agent-templates/agents.yaml` stays the single source of what each agent has (TRD 8.1).
- `.claude/`, `.agents/`, `.codex/`, `plugins/` and `gabo.yaml` are `review.human_only_paths`: rebuilt agent files and plugin skill copies in this repository need a human-only merge.
- One reviewable task per PR of about 400 changed lines (TRD, dev-lead plan template).

## Open questions

| # | question | status |
|---|---|---|
| 1 | Does "revise todo" also cover each agent's tools or permissions per engine? | answered in spec (D4): deferred to a later intent, owner 2026-11-06 |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#304 I0008: Stricter gabo check](https://github.com/testeos-z/gabo-ai/issues/304) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) (not written: waits for the spec's acceptance) |
| related intent | 0009-main-session-defaults-to-the-gabo-assistant (its block check), 0006-engine-integration-at-init (engine keys), 0005-any-runtime-engine (flexible tier) |
