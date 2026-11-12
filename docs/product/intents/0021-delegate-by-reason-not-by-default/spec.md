# Spec 0021 Delegate by reason, not by default

| field | value |
|---|---|
| intent | [0021-delegate-by-reason-not-by-default](intent.md) |
| status | Accepted (owner, 2026-11-12) |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: section 3 and section 4 are not applicable) |
| date | 2026-11-11 |

Mode is `auto`. Owner's standing rule (2026-11-11): the agent applies its recommended answer to every open question and records it as a decision; only merging to main, tags and secrets stay the owner's. Each decision below is marked **Recorded decision (resolver: agent, rule of the owner)** and is saved through the `memory` slot. The source is the owner's approval of this intent on 2026-11-11 ("si" to creating an intent for each pending idea); on which intent gets its spec next: "vamos en orden numerico".

Evidence base (checked 2026-11-11 on `origin/dev` at `0fd848e`, and the Claude Code documentation):

- **The idea.** Anthropic, Best practices for Claude Code (https://code.claude.com/docs/en/best-practices), the table of common failure patterns: "Handing the same task to a subagent doesn't make the answer any better. A fresh reviewer does help, because it hasn't seen your reasoning, which is a different thing from delegating." On cost, the same page: "What they cost you is mostly tokens, since every subagent sends its own requests and they count against the same usage as your main conversation. Sometimes it's time as well: a fresh subagent first has to gather context your conversation already had, so it can end up *slower* than doing the task where you are."
- **The rule this intent narrows.** `.gabo/skills/gabo-assistant/SKILL.md`, Hard Rules: "Keep the main thread thin: delegate exploring, writing and long commands to subagents; keep short syntheses." Its description: "The owner's single point of contact; delegates all heavy work." The neutral template `.gabo/skills/gabo-agent-templates/agents/gabo-assistant.md` (and `gabo-assistant-block.md`): "Keep the main thread thin and never block on long work: hand writing, exploring and long commands to the gabo agents, `gabo-pm`, `gabo-dev-lead`, `gabo-cycle` and `gabo-plan-reviewer`, and the others the skill names."
- **Where the text lives.** The same wording is in `apps/cli/internal/skills/` (the embedded catalog), `plugins/codex/skills/` (a `review.human_only_paths` entry), `.gabo/skills/` and the built agent `.claude/agents/gabo-assistant.md`. No Go test pins the sentence (searched `apps/cli/internal` tests).
- **The roles.** The assistant's Decision Gates table sends PRD, intent and spec to `gabo-pm`, TRD, plans and task issues to `gabo-dev-lead`, migrations and design norms to `gabo-db-master` and `gabo-frontend`, a wave to `gabo-cycle`, and a critical question with the owner absent to a fresh `gabo-plan-reviewer`. PRD R3 (three task blocks) and R19, TRD D8 (the plan-change reviewer runs in a clean context).
- **The graph this builds on.** Accepted spec 0015 (PR #475): the main session loads the `gabo-cycle` skill and launches each coder as one subagent layer; task T4, issue [#479](https://github.com/testeos-z/gabo-ai/issues/479) (open), moves the wave into the main session and rewrites the assistant's Wave Start. Reviewer independence is out of scope here (it waits on the owner's decision on 0015 S3).

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | The gabo assistant shall hand work to another agent only for a named reason, and shall do the work itself otherwise. The reasons are: (a) independent units of behavior with disjoint edit surfaces, each needing several minutes, launched together; (b) a map that needs more than one bounded read batch (D3); (c) a long suite, build or wait that would block the main thread; (d) an independent verifier, reviewer or plan reviewer, because a fresh context is the point. | Given a small change that the assistant understands, Then it reads and edits inline and launches no agent. Given understanding that needs more than one bounded read batch, Then it launches one read-only explorer. Given a suite or build that takes minutes, Then it runs in a subagent and the assistant does not block. Given two independent units of several minutes with disjoint surfaces, Then it launches one agent per unit together. | skill review against S1; probe (S8) |
| S2 | Every handoff shall name its reason, one of S1 (a) to (d) or the role ownership of S3, in the handoff text, and the assistant shall name it when it tells the owner what it handed off. | Given any handoff, Then its first line names the reason. Given the owner asks why something was delegated, Then the answer is that reason. Given work with no reason, Then no handoff exists. | skill review; probe (S8) reads the handoffs |
| S3 | A document or task block that a role owns shall keep going through that role's agent: `gabo-pm` for the PRD, intents and specs, `gabo-dev-lead` for the TRD, plans and task issues, `gabo-db-master` and `gabo-frontend` for their sections and documents, and a fresh `gabo-plan-reviewer` when the owner is absent. This is role ownership (PRD R3, R19, TRD D8), not delegation by default, and this intent does not change the roles. | Given a request to write a spec, Then it goes to `gabo-pm` and the handoff names "role ownership". Given a critical question with the owner absent, Then a fresh plan reviewer starts, as today. Given the roles in the neutral spec, Then they are unchanged. | skill review; review of the neutral spec diff (none) |
| S4 | When the assistant runs `gabo-cycle` in the main session (intent 0015 T4), the cycle shall launch coders only, each under reason (a), and shall delegate nothing else. | Given a wave with three independent tasks, Then the cycle launches up to `parallelism` coders, one per task worktree, and no other agent. Given a wave with one task, Then one coder launches, since the work is a separate unit in its own worktree. | skill review; probe (S8) |
| S5 | The rule shall be written once and narrowed everywhere the wording lives: the assistant skill (its description, the Hard Rule and the intro of Decision Gates), the neutral assistant template and the main-session block, `gabo-cycle` where it speaks of launching, the docs that describe the assistant, and every catalog copy, with the built `.claude/agents/gabo-assistant.md` rebuilt. The pinned strings of the skill tests stay. | Given a search of the catalog for "delegates all heavy work" and "hand writing, exploring and long commands", Then neither phrase remains. Given `go -C apps/cli test ./...`, Then it passes. Given the catalog copies under `apps/cli/internal/skills/`, `plugins/codex/skills/` and `.gabo/skills/`, Then they carry the same text. | search of the repository; `go -C apps/cli test ./...`; `gabo check` |
| S6 | The change shall add no Go command and change no Go production code, no schema and no `gabo check` output; a test-only addition that pins the new rule text is allowed. | Given the diff, Then it touches skill files, agent notes, docs and at most skill tests. Given `go -C apps/cli build ./...`, `vet` and `test`, Then they pass with no golden file changed. | the three `go -C apps/cli` commands; review of the diff for Go production files |
| S7 | The change shall land after intent 0015 task T4 (#479): every task of this intent that edits the assistant skill, the neutral templates or the cycle skill shall list `0015/T4` as a blocker, since T4 rewrites the same files. | Given the plan of this intent, Then each such task has `0015/T4` in `depends_on` and the tracker shows the same blocked-by link. Given T4 not merged, Then none of those tasks is unblocked. | traceability check of the plan; the tracker's blocked-by links |
| S8 | The change shall be probed on a disposable repository: the same small intent runs once with the old text and once with the new, and the handoffs of each run are listed with their reasons. | Given the new text, Then every handoff carries a reason from S1 or S3, and a task with no reason is done inline. Given the old text, Then the baseline count of handoffs is recorded for comparison. | probe, results in the intent's review findings; the plan assigns it to a task |

## 2. Design

Affected files: `gabo-assistant` (skill, neutral template and block), `gabo-cycle` (the launching text), the docs that describe the assistant, the catalog copies and the built agent. No Go production file.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Who the rule applies to | (a) the assistant, and the cycle when the assistant runs it; (b) also directiva and the coder; (c) the whole method | **Recorded decision (resolver: agent, rule of the owner):** (a), the assistant's recommendation. Reason: the assistant's skill is where "delegates all heavy work" lives, and after intent 0015 the directiva agents and the coder launch no agents, so there is nothing for the rule to narrow there. | The roles stay as they are; their fate is intent 0022's question. |
| D2 | The list of reasons | (a) the four reasons of S1 and role ownership (S3); (b) a longer list; (c) only "heavy" | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: each reason names something a handoff buys that inline work does not: parallelism over disjoint surfaces, a map too big for one batch, an unblocked main thread, or a fresh context. | Adding a reason is a convention change and goes to the owner. The fresh reviewer stays under (d): the docs say a fresh reviewer helps because it hasn't seen the reasoning, which is different from delegating for a better answer. |
| D3 | What "one bounded read batch" is | (a) about 3 reads or searches and about 10k tokens of output; (b) no number; (c) a file count | **Recorded decision (resolver: agent, rule of the owner):** (a), as a starting value. Reason: a handoff costs a fresh context that must reread what the assistant already has, so a number keeps the rule checkable; a file count would make size the criterion. | Tuned later with the data of intent 0022. |
| D4 | Shape of the change | (a) skill and agent-note text, no Go; (b) a gabo-level setting for how much to delegate | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: the rule is judgment the assistant applies, not a value gabo can enforce, and no key could say whether a reason holds. | Nothing for `gabo check` to verify; the probe (S8) is the check. |
| D5 | Order against intent 0015 | (a) after 0015 T4 (#479), as a cross-intent blocker; (b) before it; (c) in the same tasks | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: T4 rewrites the assistant's Wave Start and moves the cycle into the main session, in the same files this intent edits, and S4 only makes sense once the cycle runs there; the owner's order of work puts 0015 first. | The plan carries `0015/T4` as a blocker (S7). If T4 is rewritten, the plan is rechecked. |

## 3. UX and app flow (optional)

Not applicable: no screen or component. The journey the owner sees is the assistant's answer: when it hands something off it says which reason holds, and when it does the work itself it does not announce a handoff.

## 4. DB migration steps (optional)

Not applicable: no stored data changes.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| The assistant skill's Hard Rule and description ("delegate exploring, writing and long commands to subagents", "delegates all heavy work") and the template line | They state delegation as the default. | amended with the spec's acceptance: the default becomes inline work, and the four reasons of S1 name when to hand off. This intent narrows the rule; it does not remove it. |
| The template's "never block on long work" | Inline work could block the main thread. | none: reason (c) of S1 keeps long suites, builds and waits off the main thread. |
| PRD R3 and R19, TRD D8 (roles, plan-change reviewer in a clean context) | Less delegation could read as merging roles. | none: S3 keeps role ownership and the clean-context reviewer; merging roles is intent 0022's question. |
| Intent 0015 (accepted, one subagent layer; T4 rewrites the assistant's Wave Start) | Same files. | none: S7 orders this intent after T4. |
| `review.human_only_paths` (`plugins/`, `.claude/`) | Catalog copies under `plugins/` and the built `.claude/agents/` change. | none: the plan's tasks that touch them go through the human merge path, as in 0015. |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Which reasons justify a delegation? | answered: the four reasons of S1, plus role ownership (S3) (D2). Recorded decision (resolver: agent, rule of the owner). |
| 2 | Does the change apply only to the assistant, or also to directiva and the cycle? | answered: the assistant, and the cycle when the assistant runs it; directiva and the coder launch no agents after 0015 (D1). Recorded decision (resolver: agent, rule of the owner). |
| 3 | In what order does it land relative to intent 0015? | answered: after 0015 T4 (#479), as a cross-intent blocker (D5, S7). Recorded decision (resolver: agent, rule of the owner). |

Nothing is left that needs the owner except the acceptance in section 7.

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
