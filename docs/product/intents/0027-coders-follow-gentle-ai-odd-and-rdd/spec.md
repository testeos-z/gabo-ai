# Spec 0027 Coders follow gentle-ai ODD and RDD

| field | value |
|---|---|
| intent | [0027-coders-follow-gentle-ai-odd-and-rdd](intent.md) |
| status | Draft (not accepted) |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: sections 3 and 4 are not applicable) |
| date | 2026-11-13 |

Mode is `auto`. Every open product decision carries its recommended option, marked **Proposed (auto)** with a one-line reason (section 6). Nothing here is accepted: section 7 stays empty until the owner accepts the spec. An agent never accepts a spec.

Evidence base (checked 2026-11-13 against `origin/dev` at `a2a049a`, worktree `docs/i0027-intent-spec`):

- **Wiring.** `gabo.yaml:27-28` sets `slots.methodology.adapter: .claude/skills/gabo-methodology-gentle-ai/SKILL.md`; `.claude/skills/gabo-methodology` is a symlink to it. `.claude/agents/gabo-coder.md:8` preloads `gabo-methodology`; line 14 says to follow the methodology slot adapter. `apps/cli/internal/skills/gabo-coder/SKILL.md:11` says to load the adapter before reading code and that the skill carries no methodology of its own.
- **The adapter (72 lines).** `.claude/skills/gabo-methodology-gentle-ai/SKILL.md`: ODD explore-first and RED/GREEN (line 15); RDD only when `gentle-ai review mode status` is on (line 17); after each commit, assess and follow `review_due` and `next_transition` (lines 24 and 46-52); never toggle RDD (line 18); the per-task report and its five outcome values (line 56).
- **The assess command on committed work.** Run on 2026-11-13 with gentle-ai 4.4.0 (`~/.local/bin/gentle-ai`) in a clean worktree at `origin/dev`: `gentle-ai review assess --cwd <wt> --json` returned `"risk": "high"`, `"reason": "unassessable"`, `"review_due": true`, `"review_due_reason": "high_risk"`, and exited with an error that names `gentle-ai review assess --base-ref <commit>` as the way to assess committed work. `gentle-ai review assess --help` lists `--base-ref` and `--committed-only`.
- **Host RDD state.** `gentle-ai review mode status` reports `on` (`global: on`, `clone-local: unset`), as the assistant reported and as seen on this host.
- **Observed coder runs (reported, not re-checked).** Session claude-b2e28fa5, waves I0023, I0025 and I0026, as reported by the assistant in the intent's Problem: 13 runs, adapter preloaded in all 13; one run ran `review start` and `status` (I0025 T3, PR #706) and stopped on the consent envelope; one ran only `review mode status` (I0026 T8, PR #739); eleven ran no gentle-ai review command; no coder created `odd/tasks/`.
- **Human-only paths.** `gabo.yaml:53` lists `.claude/` in `review.human_only_paths`; the user-level reviewer scripts treat such paths as human-merged (`common.sh:256-257`, `is_human_only`). The adapter and the coder agent live under `.claude/`.
- **Wave outputs.** `.gabo/skills/gabo-cycle/SKILL.md:49` (step 6: on each outcome, update the trace); `apps/cli/internal/skills/gabo-wave-report/SKILL.md:23` (one `Tasks` line per task).
- **TRD.** 9.1 methodology row (`docs/product/TRD.md:372`); 9.2 coder outcome (`:383`); D6 reviewer (`:588`); the gentle-ai split "RDD is the coder-side validation layer, the gabo reviewer stays the PR-side merge gate" (`:675`, to verify, not decided).
- **Methodology contract.** `apps/cli/internal/skills/gabo-contract-methodology/SKILL.md`: an adapter declares coder-side review explicitly; coder-side review never replaces the gabo reviewer; the methodology never weakens the coder's hard rules.

## Log

- 2026-11-13, owner, relayed by the owner's local assistant (verbatim): "quiero que los coder usen gentle-ai lo mas apegado a ODD y RDD". RDD is in scope for coders and is not dropped. Draft written by gabo-pm from the intent's evidence; section 6 carries the decisions the owner must make. Section 7 stays empty.

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | When a coder starts a task, the coder shall read the task issue and the spec requirements the task links (S#), explore the code they touch, and then observe a RED test before its GREEN change when a relevant deterministic test exists; otherwise the coder shall state the exception and the proportional checks it ran (ODD, the methodology adapter's coder instructions). | Given a task with a runnable deterministic test, When the coder reports, Then the report names that test, its RED failure and its GREEN run. Given a task with none, Then the report states the exception and the checks it ran, and no RED or GREEN is claimed. | skill review of the adapter's coder instructions and of `gabo-coder`; report fields of S8, checked on the probe wave |
| S2 | Each coder report shall list, for the task, every spec requirement id (S#) the task links, with `covered` and the acceptance criteria verified, or `not covered: <reason>`. | Given a task linked to S2 and S4, Then the report has one line for S2 and one for S4, each `covered` or `not covered: <reason>`. | skill review of the report contract; wave report check (S9) |
| S3 | When `gentle-ai review mode status` reports on, the coder shall assess each RDD candidate with `gentle-ai review assess --cwd <worktree> --base-ref <base> --committed-only --json`, and shall follow the returned `review_due` and `next_transition` verbatim. The candidate (proposed: each work-unit commit before its push) and the base are decided by OQ4. | Given the mode on and a committed work unit, Then the report carries the RDD outcome and, for an assessed commit, its `risk` and `review_due_reason`. Given the mode off, Then the outcome reads `off` and no assess command runs for that task. | adapter text test (S10); probe wave: one task with the mode on, its assess command and envelope in the report |
| S4 | A coder shall never answer, grant, decline or synthesize a consent envelope (`gentle-ai.review-integration.consent/v3`), and shall never run `gentle-ai review mode enable` or `disable`. A consent envelope is returned unchanged to the assistant as an open question of the task; the route decided in OQ1 resumes the task. Consent is candidate-scoped: global RDD on never grants it. | Given a consent envelope in an assess or transition result, Then the outcome carries it verbatim under open questions and the coder runs no START with an answer. Given the adapter, Then it contains no `review mode enable` or `review mode disable` command outside a prohibition line. | grep test over the adapter (S10); probe wave observing one consent envelope or a recorded `not run: no consent envelope in this wave` |
| S5 | RDD never replaces the gabo reviewer. After every push, whatever the RDD outcome, the coder shall run the reviewer script for the pushed head and act on its `VERDICT` line, as `gabo-coder` already requires. The reviewer stays the only merge gate (PRD R9). | Given an RDD outcome of `granted`, `passive` or `off`, Then the report still carries one `VERDICT` line per pushed head. No merge is made by the coder. | existing `gabo-coder` rule, unchanged; grep test that the adapter names the reviewer as the merge gate (S10) |
| S6 | An assessment that fails, is unavailable, or returns `unassessable` shall never be read as low risk. The coder shall treat that commit as due and report the RDD outcome `unavailable` with the error line. OQ5 decides what happens next. | Given the committed-work `unassessable` error recorded in the evidence base, Then the outcome reads `unavailable` with that line and never `passive`. | skill review of the adapter; probe wave; grep test of the wording (S10) |
| S7 | The adapter's assess command for committed work shall name the base revision and `--committed-only`. The tree-only form shall not be the documented command for committed work. | Given the adapter, Then its assess command includes `--base-ref` and `--committed-only` (grep test). Given a committed worktree in a `mktemp -d` scratch repository, Then the command returns a `gentle-ai.review-assessment/v1` envelope with a risk tier, not `unassessable`. | Go test in `apps/cli/internal/skills` (S10); manual run in the scratch repository during the plan, cleaned by literal path |
| S8 | Each coder task's outcome shall carry the methodology report block: the ODD evidence of S1, the spec lines of S2, each `review.checks` command with its result, the commits (SHA and message), the RDD outcome of S3 and, for an assessed commit, `risk` and `review_due_reason`. A field the coder did not produce reads `not run: <reason>`, never as a pass. | Given a task returned HANDED or BLOCKED, Then every field is present, each one with a value or `not run: <reason>`. | skill text test of the outcome contract in `gabo-coder` and of the adapter's `report per task` (S10) |
| S9 | The cycle shall write each task's RDD outcome and ODD line to the wave trace on each outcome (step 6 of `gabo-cycle`), and the wave report's `Tasks` line shall show `RDD: <outcome>` and `ODD: <RED/GREEN or exception>` for every task. A resumed wave keeps the trace lines (PRD R25). | Given a wave of three tasks with RDD `granted`, `declined` and a missing report, Then the wave report has three `Tasks` lines, each with its RDD outcome, and the missing one reads `not run: <reason>`. Given an interrupted wave resumed from its trace, Then the lines are unchanged. | skill review of `gabo-cycle` step 6 and `gabo-wave-report` section 2; the wave report of the probe wave |
| S10 | Tests shall fail when the coder stops preloading `gabo-methodology`, when the adapter loses the RDD rule (S3), the no-consent and no-toggle rules (S4), the reviewer gate (S5), the never-lower rule (S6), the committed-work selectors (S7) or the report fields (S8). | Given the agent template in `apps/cli/internal/skills/gabo-agent-templates/agents/gabo-coder.md`, Then it preloads `gabo-methodology`. Given the adapter text, Then each named rule and selector is present. | Go test in `apps/cli/internal/skills` (location of the adapter read by the test decided in the plan) |

## 2. Design

The change keeps the three existing homes of the rules. The methodology adapter (the slot, `gabo-methodology`) tells the coder how to build and how to run RDD. The coder skill and its agent template make the report and the preload binding. The cycle and the wave report carry the outcome to the owner. No new CLI command is added (PRD R31).

Affected files: the adapter `.claude/skills/gabo-methodology-gentle-ai/SKILL.md` (human merge, section 5); the coder agent `.claude/agents/gabo-coder.md` only if the plan finds its line 8 or 14 insufficient (human merge); `apps/cli/internal/skills/gabo-coder/SKILL.md` and `apps/cli/internal/skills/gabo-agent-templates/agents/gabo-coder.md`; `apps/cli/internal/skills/gabo-cycle/SKILL.md` step 6; `apps/cli/internal/skills/gabo-wave-report/SKILL.md` section 2; TRD 9.2. Each catalog file has its byte-for-byte copy under `.gabo/skills/`, which the plan updates in the same change.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Where the RDD and ODD rules live | (a) keep them in the gentle-ai adapter in the methodology slot; (b) move them into `gabo-coder`; (c) a gabo CLI command that wraps gentle-ai | (a). (b) would make the coder skill carry a methodology, which its own text forbids; (c) is excluded by PRD R31 | The adapter change merges by a human (`.claude/` is human-only). TRD TD5 stays open (section 5) |
| D2 | Outcome vocabulary for RDD | (a) the adapter's five values `granted`, `declined`, `passive`, `off`, `unavailable`; (b) the five plus gentle-ai's `review_due_reason` verbatim when no review was due; (c) free text | (b) | A commit gentle-ai reports as not due (for example `under_budget` or `already_reviewed`) still gets a closed-list outcome, so the owner can count it |
| D3 | Where the outcome travels | (a) the coder report only; (b) report, wave trace and wave report line; (c) the tracker task comment | (b). (a) is lost on resume (PRD R25); (c) puts a second report in the tracker | Each outcome adds one trace line and one wave report field; TRD 9.2 gains the field (section 5) |
| D4 | Selector for assessing committed work | (a) the tree-only command of the adapter today, which returns `unassessable` on committed work (observed); (b) `--base-ref <base> --committed-only`, which gentle-ai's help documents; (c) a gabo wrapper script | (b) | The base choice follows OQ4. (c) adds a script the owner did not ask for |
| D5 | Who answers a consent envelope | (a) the coder answers it; (b) the coder declines it itself; (c) the coder returns it to the assistant | (c) | gentle-ai makes consent candidate-scoped and owner-given, and a coder cannot give it. The route is OQ1 |
| D6 | Whether the reviewer or RDD gates the merge | (a) the reviewer (unchanged); (b) RDD replaces the reviewer for granted candidates | (a) | PRD R9 and the methodology contract. Option (b) would need a PRD change and is not offered as a question |

## 3. UX and app flow (optional)

Not applicable: the change has no screen and no user-facing surface. The owner reads the wave report, which is skill text (`gabo-wave-report` section 2), and the new `RDD:` and `ODD:` fields are plain text lines in it.

## 4. DB migration steps (optional)

Not applicable: the change alters no stored data. The wave trace gains lines of the existing shape, written by the cycle skill; no migration.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| PRD R15: in auto mode the human receives no prompts, only the final answer | A per-candidate consent envelope needs the owner | Open: OQ1. Nothing is decided here; the proposal keeps the answer for the final answer and does not decline the candidate |
| PRD R5 and R29: questions reach the assistant and the wave report | The envelope is a question with no answerer in the coder | Consistent with S4 and D5: the envelope goes to the assistant as a question |
| PRD R9 and the methodology contract: RDD never replaces the reviewer | None: S5 states it | None |
| TRD 9.2: the coder outcome and wave report fields | Gains the RDD and ODD fields | Additive. The TRD 9.2 amendment is written in the plan after the owner accepts the spec |
| `gabo.yaml:53` `review.human_only_paths` includes `.claude/` | The adapter and the coder agent are human-merged | Not a conflict: the plan's PR says a human merges it. The coder never merges it |
| TRD 675: gentle-ai's own workflow may overlap the cycle (to verify) | Adopting ODD and RDD tightens the overlap | The intended split is kept: RDD is the coder-side layer, the reviewer the merge gate. Checked on the probe wave (S9): no second merge decision |
| TRD TD5: the coder skill hardcodes ODD test-first | Not resolved by this intent | S1 adds the report, not the move. Stays open |

## 6. Open questions

Each question carries a **Proposed (auto)** option with a one-line reason. The owner decides each one; the spec is not accepted until the owner does.

| # | question | status |
|---|---|---|
| 1 | How does a per-candidate RDD consent reach the owner when a coder cannot give it, and what happens to the task meanwhile? | open: **Proposed (auto)**, option (a) |
| 2 | Does the ODD feature document (`odd/tasks/<feature>.md` and its Engram mirror) get created per task, or do the gabo plan row and the task issue act as it? | open: **Proposed (auto)**, option (b) |
| 3 | When does RDD run next to the gabo reviewer, and which one gates the merge? | open: **Proposed (auto)**, option (a) |
| 4 | Is the RDD candidate each work-unit commit before the push, or the PR head? | open: **Proposed (auto)**, option (a) |
| 5 | What happens when RDD is unavailable or its assessment fails? | open: **Proposed (auto)**, option (c) |
| 6 | Beyond visibility, does a gate enforce the coder's ODD and RDD? | open: **Proposed (auto)**, option (a) |

### 6.1 Consent routing (question 1)

- **(a) Relay and wait.** The coder returns the envelope unchanged. The assistant relays it losslessly to the owner, and the task waits while other tasks continue. The assistant resumes the coder by its ledger handle with the owner's answer. Tradeoff: follows gentle-ai's candidate-scoped rule; in auto mode the wave can close with a task still waiting.
- **(b) Time-boxed decline.** The assistant declines after a fixed wait or at wave end, and the outcome is `declined`. Tradeoff: the wave never waits; RDD becomes optional in auto mode, which defeats the binding of this intent.
- **(c) Standing grant in `gabo.yaml`.** Not offered as an option: gentle-ai's consent is candidate-scoped, and a standing grant would grant consent for candidates the owner has not seen.

**Proposed (auto): (a).** In auto mode the task is not declined. The wave report shows `RDD: unavailable: consent pending` for that task, and the owner answers in the final answer, which the assistant relays. Reason: keeps consent with the owner and costs no prompt during the run (PRD R15); the gabo reviewer still gates the merge.

### 6.2 Specification artifact (question 2)

- **(a) ODD feature document per task.** `odd/tasks/<feature>.md` and its Engram mirror, as gentle-ai's ODD describes. Tradeoff: closest to the wording; a second tracker beside the plan and the task issues, with its own drift.
- **(b) Plan row and task issue act as the document.** The spec's requirements (S#) are the specification, the plan's spec refs link each task to them (`gabo-pm` spec template, section 6 wording), and the task issue carries the coder's report as progress comments. Tradeoff: no duplicate tracker, which the intents README and the PRD require; the mapping of ODD names to gabo names must be accepted.
- **(c) Engram mirror only.** Tradeoff: memory is a trace, not a tracker; resume reads it, but the owner does not see it in the tracker.

**Proposed (auto): (b).** Reason: the tracker is the one global view (PRD section 7, intents README), and S2 already gives the traceability that ODD's document provides.

### 6.3 RDD and the gabo reviewer (question 3)

- **(a) RDD only when due.** RDD runs only when assess returns `review_due: true` (high risk), before the first reviewer request; the outcome is still reported for every commit. Tradeoff: cost follows risk; low-risk commits get no coder-side view.
- **(b) RDD on every assessed commit.** Tradeoff: most outside views, and the most consent envelopes (OQ1's cost).
- **(c) RDD replaces the reviewer for granted candidates.** Not offered: PRD R9 and the methodology contract keep the reviewer the only merge gate.

In every option the gabo reviewer gates the merge. **Proposed (auto): (a).** Reason: gentle-ai's own flow runs its review when it is due, and the reviewer, not RDD, decides the merge.

### 6.4 Timing of the RDD candidate (question 4)

- **(a) Each work-unit commit before its push.** The assess command names the base (the branch point for the first commit, then the last reviewed commit, as gentle-ai's ODD states). Tradeoff: closest to gentle-ai's ODD; more candidates and consent envelopes per task.
- **(b) Once on the PR head before the first reviewer request.** A fix round that moves the head assesses again. Tradeoff: one candidate per head and fewer consents; the candidate can be a large accumulated diff, which weakens the per-slice review ODD asks for.
- **(c) Both.** Tradeoff: most cost.

**Proposed (auto): (a).** Reason: the owner asked for the closest match to gentle-ai's ODD and RDD, and gentle-ai reviews work-unit commits; the consent cost is OQ1's.

### 6.5 RDD unavailable or the assessment fails (question 5)

- **(a) Report `unavailable`, treat the commit as due, continue.** The gabo reviewer still gates the merge, and the wave report shows the gap. Tradeoff: waves keep moving; the gap is visible but not blocking.
- **(b) Block the task until RDD works.** Tradeoff: strongest binding; one tool outage stops delivery of an informational layer.
- **(c) Retry the same assess once on the same commit, then (a).** Tradeoff: one more call; catches transient failures.

**Proposed (auto): (c).** Reason: transient failures are common and a retry costs one call; a persistent failure must not stop delivery, which the reviewer gates. The tier is never lowered in any option (S6).

### 6.6 Enforcement beyond visibility (question 6)

- **(a) Visibility only.** The report fields, the trace and the wave report line, and the pinned text tests (S8 to S10). Tradeoff: no change to the merge or to DONE; a coder can still skip RDD, but the wave report shows it.
- **(b) The cycle holds a task whose report has no RDD outcome.** Tradeoff: stronger; a report gap pauses a wave.
- **(c) The reviewer requires an RDD line before it merges.** Tradeoff: moves the check into the merge gate; changes PRD R9 and the reviewer's job, so it needs a plan change, not this intent.

**Proposed (auto): (a) for this intent.** Reason: a gate changes the merge path or the wave flow, which is the owner's decision. Revisit (b) after two waves whose reports show the RDD line.

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
