# Intent 0014 Agent name and duplicate skill checks

| field | value |
|---|---|
| intent id | 0014-agent-name-and-duplicate-skill-checks |
| status | Specified (spec awaiting the owner's acceptance) |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-07 |

## Problem

`gabo check` passes agent files that their engine will not load, and it gives a misleading fix for a duplicated skill.

- **An agent file with no name passes.** Today the declared-name rule fails only a declared name that differs from the spec id; "a file that declares no name is not a declared name" (`apps/cli/internal/slices/check/agentname.go`, intent 0008 S6, TRD 9.4). But the engines need that name: Codex rejects an agent role file without a non-empty `name` ("agent role file at {} must define a non-empty `name`", `codex-rs/agent-roles/src/agent_role_config.rs`) and only logs a startup warning, and Claude Code requires `name` in a subagent's frontmatter and silently skips a project agent file without it (code.claude.com/docs/en/sub-agents, frontmatter table). So `gabo check` exits 0 while the engine has no such agent, and the owner finds out only when a wave cannot start it.
- **A skill listed twice reads as a wrong skill.** When an agent file lists a spec skill twice, the second copy is reported as `✗ agent <id> (<runtime>): skill <name> not in the agent spec` (`agentskills.go`, `compareSkills`), the same line as a skill that does not belong to the agent. The fix is different: remove the duplicate, not the skill.

## Who is affected

Owners and agents who run `gabo check` after building or editing agents, on every project with a Claude Code, Codex or Pi team (the engines whose runtime skill declares `metadata.agent_name` or `metadata.agent_skills`). It happens whenever a hand edit or a faulty rebuild drops an agent's `name` or repeats a skill.

## Desired outcome

- `gabo check` fails an agent file that declares no name where the engine's runtime skill says where the name lives (owner 2026-11-07, verbatim, asked whether `gabo check` should fail when a Claude Code or Codex agent file declares no `name`: "que falle").
- A skill listed twice in an agent file gets its own failure line saying it is listed twice, not the "not in the agent spec" line (owner 2026-11-07, accepting the proposal, verbatim: "si no la listo tomare tu propuesta para toda opcion no listada").
- Everything else `gabo check` prints stays as it is.
- The change ships in the next gabo release after 0.2.10. The version bump is not part of this intent.

## Constraints

- `gabo check` stays read-only, keeps its output shape (`✗` line, `why:` line, `next:` line) and exit codes, names no engine in Go and reads the engine through what its runtime skill declares (PRD R33, TRD 9.4, intent 0008 D2).
- Every change assesses whether `gabo check` must change (TRD 8.5); this intent is a `gabo check` change.
- OpenCode is out of scope: it names an agent by its file name and its runtime skill declares no `metadata.agent_name` (TRD 9.4). An engine is in scope when its runtime skill declares `metadata.agent_name` (today `claude`, `codex`, `pi`); a flexible-tier engine without `metadata.agent_skills` stays skipped by the per-agent rules (intent 0008 S2).
- The rules are skipped under `GABO_UPGRADE_PR=1` like every per-agent rule (intent 0008 S9).
- About 400 changed lines per reviewable task (TRD, dev-lead plan template).

## Open questions

| # | question | status |
|---|---|---|
| 1 | Do the name rules also cover the main-session agent file `gabo-assistant` (Claude Code), which today gets no name check? | carried to spec open question 1, passed to the owner |
| 2 | Is a fixed-name link listed under both names (link name and its adapter's own name) "listed twice"? | carried to spec open question 2, passed to the owner |
| 3 | Which line does a skill that is not in the spec get when it is listed twice? | carried to spec open question 3, passed to the owner |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#444 I0014: Agent name and duplicate skill checks](https://github.com/testeos-z/gabo-ai/issues/444) |
| PRD | [PRD](../../PRD.md) R33 |
| spec | [spec](spec.md) (awaiting the owner's acceptance) |
| plan | not written yet (gabo-dev-lead, after acceptance) |
| related intent | 0008-stricter-gabo-check (S1, S2, S6, S9, TRD 8.5), 0013-several-engines-in-parallel-in-one-project (engine set of `gabo check`) |
