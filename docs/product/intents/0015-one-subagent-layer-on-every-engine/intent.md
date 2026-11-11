# Intent 0015 One subagent layer on every engine

| field | value |
|---|---|
| intent id | 0015-one-subagent-layer-on-every-engine |
| status | Accepted |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-11 |

## Problem

gabo runs a wave through three layers of nested subagents, which is the most an engine allows and more than the work needs, and it already keeps a second, flat path for the one engine that cannot nest.

Owner, 2026-11-11, verbatim: "tenemos un modelo de sub agentes anidado donde cycle invoca coder o dev-lead quiero saber si asi es y si asi lo plantea antropic". Then, after the answer: "ok, si queremnos cumpli mas a raja tabla todo y disminuir la sobre ingenieria que debemos hacer ?"

What the repository shows today (gabo 0.2.10, `dev` at 0fd848e):

- **Three layers.** The neutral agent spec chains `gabo-cycle` (`launched_by: main-session`) → `gabo-coder` (`launched_by: gabo-cycle`) → `gabo-reviewer` (`launched_by: gabo-coder`), so a wave is assistant → cycle → coder → reviewer relay (`apps/cli/internal/skills/gabo-agent-templates/agents.yaml`). TRD D1 accepts "Nesting depth limited to 3 below the assistant" as the consequence of native subagents, and TRD 12 lists the nesting-depth limit as a risk.
- **That is the engine's ceiling, and it has moved.** Claude Code: "By default, a subagent can spawn subagents of its own, up to three layers below the main conversation" (https://code.claude.com/docs/en/sub-agents.md). The limit is `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH`: "set `1` to turn nesting off", and "In v2.1.217 through v2.1.218, the default was 1, so a subagent couldn't spawn its own unless you raised the limit; v2.1.219 raised the default to 3" (https://code.claude.com/docs/en/env-vars). A user on those versions, or with the variable set to 1 or 2, cannot run a gabo wave. OpenCode stops nested subagents at `experimental.subagent_depth`, default 1, so `gabo init` has to raise it to 3 in the project config (TRD 9.4, intent 0010).
- **One engine cannot nest at all.** gentle-pi gives a subagent no `subagent_*` tools, so 0.2.10 had to add a Pi-only path: the Pi main session runs the `gabo-cycle` skill itself and launches coders as first-level subagents (intent 0013 S18, D12), and the Pi coder runs the reviewer script itself through `bash` (intent 0013 S20, D14). gabo now maintains two wave paths, with per-engine branches in `gabo-assistant`, `gabo-cycle`, `gabo-coder` and `gabo-runtime-pi`, and a Pi `gabo-cycle` and `gabo-reviewer` agent that are built and never launched.
- **Two of the layers add no judgment.** The `gabo-reviewer` agent is a relay: it validates one line, runs one script and repeats its last line (TRD D20). The `gabo-cycle` agent runs a skill the main session can load, as the Pi path proves. Anthropic's guidance: "Handing the same task to a subagent doesn't make the answer any better. A fresh reviewer does help, because it hasn't seen your reasoning, which is a different thing from delegating." (https://code.claude.com/docs/en/best-practices) and "you should consider adding complexity only when it demonstrably improves outcomes" (https://www.anthropic.com/research/building-effective-agents).

So every wave pays for two extra agent sessions whose only job is to pass messages on, the product depends on a per-engine depth setting it does not control, and each change to the wave must be written and tested for two paths.

## Who is affected

- Every owner who runs a wave, on every wave and every review round: each task goes through the cycle agent and each pushed head through the relay agent.
- Owners on Claude Code with nesting off or lowered, and owners on OpenCode, whose project config gabo must edit to allow depth 3.
- The gabo team, on every change to the wave: two paths (nested and Pi) to write, test and probe.

## Desired outcome

- A wave runs with one subagent layer on every engine: the main session runs the wave and launches each coder directly; no gabo agent launches another agent. One path, with no Pi exception.
- The agent set shrinks from nine to seven: the owner's graph, approved on 2026-11-11 ("perfecto, podemos crear los issues y sub issues apra este intent ?"), keeps `gabo-assistant` (main session), `gabo-pm`, `gabo-dev-lead`, `gabo-db-master` and `gabo-frontend` on demand, `gabo-plan-reviewer` only on a plan change, and one `gabo-coder` per task; `gabo-cycle` stays as a skill and the `gabo-reviewer` relay agent goes.
- The review stays independent: the reviewer scripts still run as a separate process in their own clean clone, out of reach of PR code.
- What a wave does is unchanged: intent claim, worktrees, ledger, trace, resume, fix rounds, wave report.
- A wave works on Claude Code with nesting turned off, and on OpenCode without gabo raising the engine's depth setting.
- The same wave run on the old and the new graph is compared (tokens, time, fix rounds), so the simplification is measured and not assumed.

## Constraints

- Task handling keeps its three blocks and their owners (PRD R3): creation by directiva, read and status updates by the cycle. The cycle's block is run by the main session.
- The reviewer stays isolated and runs only the user-level scripts (PRD R9, R22; TRD D6, D20, D27).
- Waves delete only the resources they created (PRD R13, R23; TRD D15, D23); the owning coder is still the only writer of DONE (D23).
- The owner performs no launch, relay or copy step (PRD R4).
- The assistant stays the owner's single point of contact and keeps heavy work out of its own thread (PRD R1); the wave's bookkeeping is the exception this intent adds, and it must be stated with its cost.
- `gabo-pm` and `gabo-dev-lead` stay two agents.
- This intent changes TRD conventions (D1, D20, sections 2, 4.3, 5.1, 5.2): it is a plan change of `global` scope (TRD D7) and waits for the owner.
- Every change assesses whether `gabo check` must change (TRD 8.5); `gabo check` stays read-only and engine-neutral outside its engine-key rules (PRD R33, TRD D28).
- `.claude/`, `.agents/`, `.codex/`, `plugins/`, `AGENTS.md`, `CLAUDE.md` and `gabo.yaml` are `review.human_only_paths`.
- Accepted intent folders are not rewritten: intent 0013 stays as accepted; this intent supersedes its S18, S20, D12, D14 and D15 for the other engines.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Which release carries this change? | answered by the owner on 2026-11-11: the next release after 0.2.10, as one release (spec open question 1) |
| 2 | After an upgrade, what happens to the `gabo-cycle` and `gabo-reviewer` agent files an older build left in the project? | answered by the owner on 2026-11-11: a rebuild removes them and `gabo check` fails a leftover one (spec open question 2, D4) |
| 3 | Does gabo stop writing and requiring the OpenCode `experimental.subagent_depth` key once one layer is enough? | answered by the owner on 2026-11-11: yes, and an existing value stays untouched (spec open question 3, D6) |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#474 I0015: One subagent layer on every engine](https://github.com/testeos-z/gabo-ai/issues/474) |
| PRD | [PRD](../../PRD.md) R1, R3, R4, R9, R10, R13, R22, R33 |
| spec | [spec](spec.md) |
| plan | not written yet (gabo-dev-lead; the spec was accepted on 2026-11-11) |
| review findings | the owner's session of 2026-11-11 (questions quoted above and the approved graph) |
| related intent | 0013-several-engines-in-parallel-in-one-project (S18, S20, D12, D14, D15: the Pi path this intent makes the only path), 0010-gabo-init-writes-the-main-session-key (the OpenCode `experimental.subagent_depth` key), 0009-main-session-defaults-to-the-gabo-assistant (the main-session entry), 0008-stricter-gabo-check (TRD 8.5) |
