# Intent 0013 Several engines in parallel in one project

| field | value |
|---|---|
| intent id | 0013-several-engines-in-parallel-in-one-project |
| status | Accepted |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-06 |

## Problem

The owner cannot run two engines side by side in one gabo project. Owner's need (vulcam), verbatim: "open two terminals in the same project, Claude and Pi, each running its own assistant, coders and reviewer on its own subscription, at the same time."

Owner's account of today, verbatim: "runtimes.* take one engine each (schemas/gabo.project.schema.json:44-52, config.go:72-75,195-197); a rebuild/init removes the main-session entry of any engine that is not runtimes.assistant; gabo-cycle launches coders on the session engine and gabo-agent-host-local uses runtimes.coder for all; nothing stops two sessions taking the same intent."

So a second engine's main session loses its assistant entry on the next `gabo init` or rebuild, the configuration cannot say "run the coders and the reviewer on the engine of the terminal that started the wave", and two terminals can start a wave on the same intent and work against each other.

## Who is affected

Owners who pay for more than one engine subscription and want to use them at the same time in one project, first the owner on cappy-lab/vulcam with Claude Code and Pi. It happens on every working session where both terminals are open, and on every `gabo init` or rebuild, which today removes the other engine's main-session entry.

## Desired outcome

- Two terminals in one project, for example Claude Code and Pi, each start as the gabo assistant and run their own waves at the same time, each on its own engine and subscription.
- `gabo init` and the agent rebuilds keep the main-session entry of every engine the owner lists; none removes another.
- The configuration can say that coders and the reviewer run on the engine of the terminal that runs the wave, or that one fixed engine reviews every team (owner, verbatim: "reviewer may also be a fixed engine such as pi so one engine reviews both teams").
- Two sessions never run the same intent: a wave claims its intent first, and a second session refuses a claimed intent and says which session has it. Each wave keeps its own ledger and worktrees and cleans only its own resources.
- `gabo check` checks every engine the configuration names.
- A project with one engine per role keeps today's behaviour (owner, verbatim: "single values keep today behaviour").
- The change ships in gabo release 0.2.10 (owner, verbatim: "aprobado que entre en 0.2.10"). The version bump is not part of this intent.

## Constraints

- `gabo init` is the single writer of engine config keys, the main-session keys included (intent 0010 S1 to S5), with one exception: a flexible-tier engine's builder-written runtime skill writes its own key (intent 0010 D4).
- `gabo init` removes the gabo main-session value only for an engine that is not the assistant (owner, verbatim, intent 0010: "remove the gabo value when the assistant engine changes"); this intent narrows "not the assistant" to "not listed as an assistant".
- Every change assesses whether `gabo check` must change (TRD 8.5); `gabo check` stays read-only, keeps its output shape and exit codes, and stays engine-neutral outside its engine-key rules (PRD R33, TRD D28).
- Subagents inherit the main session's engine (TRD 2, Runtime inheritance); only separately launched processes, such as the reviewer scripts, run on another engine.
- The reviewer stays isolated and runs only the user-level scripts (PRD R22, TRD D20, D27); a `codex` or `opencode` reviewer still needs its isolation pass marker.
- Waves delete only the resources they created (PRD R13, R23; TRD D15, D23).
- `.claude/`, `.agents/`, `.codex/`, `plugins/`, `AGENTS.md`, `CLAUDE.md` and `gabo.yaml` are `review.human_only_paths`.
- About 400 changed lines per reviewable task (TRD, dev-lead plan template). Owner's estimate, verbatim: "Expected ~12-15 tasks".

## Open questions

| # | question | status |
|---|---|---|
| 1 | How is an intent claim released when its session ends without closing its wave (crash, closed terminal)? | carried to spec open question 1, owner question with a proposal |
| 2 | What runs the coders when `runtimes.coder` is a fixed engine other than the engine of the session running the wave? | carried to spec open question 2, owner question with a proposal |
| 3 | Does `parallelism` bound the coders of each wave or of the whole project when several waves run at once? | carried to spec open question 3, owner question with a proposal |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#393 I0013: Several engines in parallel in one project](https://github.com/testeos-z/gabo-ai/issues/393) |
| PRD | [PRD](../../PRD.md) R4, R13, R22, R23, R32, R33, new R40 |
| spec | [spec](spec.md) |
| plan | not written yet (gabo-dev-lead) |
| review findings | the owner's brief of 2026-11-06 (vulcam need) |
| related intent | 0010-gabo-init-writes-the-main-session-key (single writer of engine keys, removal rule, D4), 0009-main-session-defaults-to-the-gabo-assistant (the main-session entry), 0008-stricter-gabo-check (TRD 8.5), 0002-codex-plugin and 0004-opencode-runtime (reviewer isolation pass marker) |
