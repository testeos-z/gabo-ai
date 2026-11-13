# Spec 0027 Coders follow gentle-ai ODD and RDD

| field | value |
|---|---|
| intent | [0027-coders-follow-gentle-ai-odd-and-rdd](intent.md) |
| status | Draft (not accepted) |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: sections 3 and 4 are not applicable) |
| date | 2026-11-13 (rewritten for the wider scope: coder and reviewer) |

Mode is `auto`. Every open product decision carries its recommended option, marked **Proposed (auto)** with a one-line reason (section 6). None is decided: the owner has not answered any question. Nothing here is accepted: section 7 stays empty until the owner accepts the spec. An agent never accepts a spec.

Evidence base:

- **Coder side, checked 2026-11-13 against `origin/dev` at `a2a049a`** (worktree `docs/i0027-intent-spec`, unchanged for these bullets).
- **Reviewer side, checked 2026-11-13 at `406015f`** (worktree `docs/i0027-qa-reviewer-scope`, the base of this change).
- **Wiring.** `gabo.yaml:27-28` sets `slots.methodology.adapter: .claude/skills/gabo-methodology-gentle-ai/SKILL.md`; `.claude/skills/gabo-methodology` is a symlink to it. `.claude/agents/gabo-coder.md:8` preloads `gabo-methodology`; line 14 says to follow the methodology slot adapter. `apps/cli/internal/skills/gabo-coder/SKILL.md:11` says to load the adapter before reading code and that the skill carries no methodology of its own.
- **The adapter (72 lines).** `.claude/skills/gabo-methodology-gentle-ai/SKILL.md`: ODD explore-first and RED/GREEN (line 15); RDD only when `gentle-ai review mode status` is on (line 17); after each commit, assess and follow `review_due` and `next_transition` (lines 24 and 46-52); never toggle RDD (line 18); the per-task report and its five outcome values (line 56).
- **The assess command on committed work.** Run on 2026-11-13 with gentle-ai 4.4.0 (`~/.local/bin/gentle-ai`) in a clean worktree at `origin/dev`: `gentle-ai review assess --cwd <wt> --json` returned `"risk": "high"`, `"reason": "unassessable"`, `"review_due": true`, `"review_due_reason": "high_risk"`, and exited with an error that names `gentle-ai review assess --base-ref <commit>` as the way to assess committed work. `gentle-ai review assess --help` lists `--base-ref` and `--committed-only`.
- **Host RDD state.** `gentle-ai review mode status` reports `on` (`global: on`, `clone-local: unset`), as the assistant reported and as seen on this host.
- **Observed coder runs (reported, not re-checked).** Session claude-b2e28fa5, waves I0023, I0025 and I0026, as reported by the assistant in the intent's Problem: 13 runs, adapter preloaded in all 13; one run ran `review start` and `status` (I0025 T3, PR #706) and stopped on the consent envelope; one ran only `review mode status` (I0026 T8, PR #739); eleven ran no gentle-ai review command; no coder created `odd/tasks/`.
- **Human-only paths.** `gabo.yaml:53` lists `.claude/` in `review.human_only_paths`; the user-level reviewer scripts treat such paths as human-merged (`common.sh:256-257`, `is_human_only`). The adapter and the coder agent live under `.claude/`. The catalog subtree `.gabo/skills/` is human-only by built-in rule (TRD D24).
- **Reviewer input (`apps/cli/internal/skills/gabo-reviewer/scripts/review-head.sh`).** Lines 299-306: the review input is `git diff --stat` and `git diff` from the merge base to the head, and the reviewer is told to read other checkout files for context. Lines 324-328: the prompt asks for a review "against the project rules in AGENTS.md and the product documents under docs/product/", for correctness bugs, security issues, architecture boundary violations and missing tests, ignoring pure formatting. Lines 335-343: the verdict contract (`VERDICT_CLEAN` or `VERDICT_BLOCKING` on the first line). The owner's framing "the reviewer does not read the spec" is only partly exact: `docs/product/` is named as a rule source, so a spec can be read, but the prompt does not ask for a check of any requirement's acceptance criteria, does not pass the task issue, and does not ask for a run of the product.
- **Reviewer gates.** `gabo.yaml:45-48` `review.checks` (`go -C apps/cli build ./...`, `vet`, `test`, run on the merge result); `gabo.yaml:52` `review.auto_merge: true`; `merge-head.sh:42` merges only when `REVIEWER_AUTO_MERGE=true`. `merge-head.sh:9-12` refuses a change to a human-only path.
- **Acceptance criteria in accepted specs.** Spec 0026 section 1 states each requirement with Given/When/Then acceptance criteria, which is the form S1 to S15 use.
- **Harness article.** [Anthropic, Harness design for long-running apps](https://www.anthropic.com/engineering/harness-design-long-running-apps), read 2026-11-13. The evaluator "used the Playwright MCP to click through the running application the way a user would"; the generator "was instructed to self-evaluate its work at the end of each sprint before handing off to QA"; the two "agree[d] on what 'done' looked like for that chunk of work before any code was written". Caveat: these describe the article's sprint-based harness. The article later removes the sprint construct for its updated harness and does not say whether the contracts stay. The stale-assumption point is the one cited in spec 0022.
- **TRD and PRD.** PRD R9 (`docs/product/PRD.md:148`); PRD R26 (`:165`); TRD 9.1 methodology row (`docs/product/TRD.md:372`); 9.2 coder outcome (`:383`); D6 reviewer (`:588`); D20 reviewer execution (`:602`); the gentle-ai split "RDD is the coder-side validation layer, the gabo reviewer stays the PR-side merge gate" (`:675`, to verify, not decided).
- **Methodology contract.** `apps/cli/internal/skills/gabo-contract-methodology/SKILL.md`: an adapter declares coder-side review explicitly; coder-side review never replaces the gabo reviewer; the methodology never weakens the coder's hard rules.

## Log

- 2026-11-13, owner, relayed by the owner's local assistant (verbatim): "quiero que los coder usen gentle-ai lo mas apegado a ODD y RDD". RDD is in scope for coders and is not dropped. Draft written by gabo-pm from the intent's evidence; section 6 carries the decisions the owner must make. Section 7 stays empty.
- 2026-11-13, owner, relayed by the owner's local assistant (approximate voice transcription, quoted as relayed; English gloss in brackets): "yo creo que podemos hacerlo en un solo intent, que organice bien el tema" [I think we can do it in one intent that organizes the topic well]; the gabo reviewer "no valida código. Valida es cómo interactúa el usuario con ese código, o sea, el producto" [does not validate code; it validates how the user interacts with that code, that is, the product]; it must be well defined "qué debe cumplir el código, o sea, cuál es la funcionalidad final" [what the code must meet, that is, the final functionality]. The owner names two roles: (a) the coder does the engineering and its own code review with gentle-ai ODD and RDD, with well-written code and unit and integration tests; (b) the gabo reviewer becomes a product QA that uses the product as a user would. The owner cited the Anthropic harness article (see the evidence base above). Effect: the spec grows by S11 to S15, the done contract (S12), and the open questions 3, 6 and 7 to 11; S1 to S10 are kept, with S10 widened by one sentence. Nothing is decided; every new question is open with a Proposed (auto) option.

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | When a coder starts a task, the coder shall read the task issue and the spec requirements the task links (S#), explore the code they touch, and then observe a RED test before its GREEN change when a relevant deterministic test exists; otherwise the coder shall state the exception and the proportional checks it ran (ODD, the methodology adapter's coder instructions). | Given a task with a runnable deterministic test, When the coder reports, Then the report names that test, its RED failure and its GREEN run. Given a task with none, Then the report states the exception and the checks it ran, and no RED or GREEN is claimed. | skill review of the adapter's coder instructions and of `gabo-coder`; report fields of S8, checked on the probe wave |
| S2 | Each coder report shall list, for the task, every spec requirement id (S#) the task links, with `covered` and the acceptance criteria verified, or `not covered: <reason>`. | Given a task linked to S2 and S4, Then the report has one line for S2 and one for S4, each `covered` or `not covered: <reason>`. | skill review of the report contract; wave report check (S9) |
| S3 | When `gentle-ai review mode status` reports on, the coder shall assess each RDD candidate with `gentle-ai review assess --cwd <worktree> --base-ref <base> --committed-only --json`, and shall follow the returned `review_due` and `next_transition` verbatim. The candidate (proposed: each work-unit commit before its push) and the base are decided by OQ 4. | Given the mode on and a committed work unit, Then the report carries the RDD outcome and, for an assessed commit, its `risk` and `review_due_reason`. Given the mode off, Then the outcome reads `off` and no assess command runs for that task. | adapter text test (S10); probe wave: one task with the mode on, its assess command and envelope in the report |
| S4 | A coder shall never answer, grant, decline or synthesize a consent envelope (`gentle-ai.review-integration.consent/v3`), and shall never run `gentle-ai review mode enable` or `disable`. A consent envelope is returned unchanged to the assistant as an open question of the task; the route decided in OQ 1 resumes the task. Consent is candidate-scoped: global RDD on never grants it. | Given a consent envelope in an assess or transition result, Then the outcome carries it verbatim under open questions and the coder runs no START with an answer. Given the adapter, Then it contains no `review mode enable` or `review mode disable` command outside a prohibition line. | grep test over the adapter (S10); probe wave observing one consent envelope or a recorded `not run: no consent envelope in this wave` |
| S5 | RDD never replaces the gabo reviewer. After every push, whatever the RDD outcome, the coder shall run the reviewer script for the pushed head and act on its `VERDICT` line, as `gabo-coder` already requires. The reviewer stays the only merge gate (PRD R9), and it now also runs product QA (S11). | Given an RDD outcome of `granted`, `passive` or `off`, Then the report still carries one `VERDICT` line per pushed head. No merge is made by the coder. | existing `gabo-coder` rule, unchanged; grep test that the adapter names the reviewer as the merge gate (S10) |
| S6 | An assessment that fails, is unavailable, or returns `unassessable` shall never be read as low risk. The coder shall treat that commit as due and report the RDD outcome `unavailable` with the error line. OQ 5 decides what happens next. | Given the committed-work `unassessable` error recorded in the evidence base, Then the outcome reads `unavailable` with that line and never `passive`. | skill review of the adapter; probe wave; grep test of the wording (S10) |
| S7 | The adapter's assess command for committed work shall name the base revision and `--committed-only`. The tree-only form shall not be the documented command for committed work. | Given the adapter, Then its assess command includes `--base-ref` and `--committed-only` (grep test). Given a committed worktree in a `mktemp -d` scratch repository, Then the command returns a `gentle-ai.review-assessment/v1` envelope with a risk tier, not `unassessable`. | Go test in `apps/cli/internal/skills` (S10); manual run in the scratch repository during the plan, cleaned by literal path |
| S8 | Each coder task's outcome shall carry the methodology report block: the ODD evidence of S1, the spec lines of S2, the done contract status of S12, each `review.checks` command with its result, the commits (SHA and message), the RDD outcome of S3 and, for an assessed commit, `risk` and `review_due_reason`. A field the coder did not produce reads `not run: <reason>`, never as a pass. | Given a task returned HANDED or BLOCKED, Then every field is present, each one with a value or `not run: <reason>`. | skill text test of the outcome contract in `gabo-coder` and of the adapter's `report per task` (S10) |
| S9 | The cycle shall write each task's RDD outcome and ODD line to the wave trace on each outcome (step 6 of `gabo-cycle`), and the wave report's `Tasks` line shall show `RDD: <outcome>` and `ODD: <RED/GREEN or exception>` for every task. A resumed wave keeps the trace lines (PRD R25). | Given a wave of three tasks with RDD `granted`, `declined` and a missing report, Then the wave report has three `Tasks` lines, each with its RDD outcome, and the missing one reads `not run: <reason>`. Given an interrupted wave resumed from its trace, Then the lines are unchanged. | skill review of `gabo-cycle` step 6 and `gabo-wave-report` section 2; the wave report of the probe wave |
| S10 | Tests shall fail when the coder stops preloading `gabo-methodology`, when the adapter loses the RDD rule (S3), the no-consent and no-toggle rules (S4), the reviewer gate (S5), the never-lower rule (S6), the committed-work selectors (S7) or the report fields (S8). Once OQ 9 is decided, they also fail when the reviewer comment template loses the product QA and checks sections of S13. | Given the agent template in `apps/cli/internal/skills/gabo-agent-templates/agents/gabo-coder.md`, Then it preloads `gabo-methodology`. Given the adapter text, Then each named rule and selector is present. | Go test in `apps/cli/internal/skills` (location of the adapter read by the test decided in the plan) |
| S11 | Product QA (owner role b; OQ 7 and OQ 8). For each pushed head that changes product behavior (a `gabo` command, option, message, output, exit code, or skill text), the gabo reviewer shall run the build under test as a user would, in a scratch project created with `mktemp -d`, and check each done-contract line of the task (S12). Its comment shall report one line per contract line: `pass`, `fail: <observed>` or `not run: <reason>`. A head with no product behavior change reads `not applicable: no product behavior change`. The reviewer never reports `pass` for a line it did not run. | Given a PR that changes the output or exit code of a `gabo` command, When the reviewer runs, Then its QA section names the command it ran and its stdout and exit code against each contract line. Given a contract line the reviewer cannot run, Then the line reads `not run: <reason>`. Given a docs-only PR, Then the section reads `not applicable: no product behavior change`. | reviewer script test with a fixture PR (location decided in the plan); probe on one product PR (scope decided by OQ 8) |
| S12 | Done contract (owner item A; OQ 7). Before its first commit, a coder shall write the done contract of its task on the task issue: one line per linked S# with the observable check that proves it (command, input, expected output or exit code). A requirement that no observable check can prove shall read `not provable: <reason>`. | Given a task linked to S2 and S4, When its first commit is pushed, Then the task issue carries two contract lines, one for S2 and one for S4, each with a check or `not provable: <reason>`. Given a task whose first commit is pushed with no contract, Then the report reads `contract: not run: <reason>`. | skill text test of the coder rule; read-back of the task issue in the probe wave |
| S13 | Verdict split (OQ 9). The reviewer comment shall keep one `VERDICT` line per pushed head (the rule of S5, unchanged) and carry the product QA result and the checks result in separate sections. A failed product line and a failed `review.checks` command are distinct, so each names its own fix. | Given a product QA `fail` and all checks passing, Then the VERDICT is blocking, the QA section names the failed contract line and the checks section reads pass. Given a `review.checks` failure and a QA `pass`, Then the checks section names the failed command. Given all pass, Then the VERDICT is clean. | reviewer script test of the comment template with a fixture PR |
| S14 | Code read (OQ 3 and OQ 10). Whether the reviewer reads the diff for code defects, and in which cases, follows the owner's decision. Until the owner decides, the prompt at `review-head.sh:324-328` is unchanged. After the decision, the prompt change and its tests ship in one PR, and the user-level copy reaches users through `gabo init` (intent 0016 S4). | Given the owner's decision, Then the prompt asks for the code read only in the cases decided. Given no decision, Then the change to that prompt is empty. | text test of the prompt in the reviewer scripts; grep of the diff |
| S15 | Coder tests (owner role a). The coder's work shall carry unit tests for each changed rule and, where the task crosses a boundary (a CLI command, a file, a skill that runs a command), an integration test through the public interface. The report lists each test with its result, or `integration: not applicable: <reason>`. | Given a task that adds a CLI option, Then the report lists a unit test of the rule and an integration test of the command, each with its result. Given a task with no boundary, Then the report says `integration: not applicable: <reason>`. | skill text test of the outcome contract (S8); report of the probe wave |

## 2. Design

The change keeps the existing homes of the coder rules. The methodology adapter (the slot, `gabo-methodology`) tells the coder how to build and how to run RDD. The coder skill and its agent template make the report and the preload binding. The cycle and the wave report carry the outcome to the owner. The reviewer side changes in the `gabo-reviewer` skill, whose scripts are installed at user level (TRD D20), not in the repository catalog. No new CLI command is added (PRD R31).

Affected files:

- the adapter `.claude/skills/gabo-methodology-gentle-ai/SKILL.md` (human merge, section 5);
- the coder agent `.claude/agents/gabo-coder.md` only if the plan finds its line 8 or 14 insufficient (human merge);
- `apps/cli/internal/skills/gabo-coder/SKILL.md` and `apps/cli/internal/skills/gabo-agent-templates/agents/gabo-coder.md` (S8, S12, S15);
- `apps/cli/internal/skills/gabo-cycle/SKILL.md` step 6 (S9);
- `apps/cli/internal/skills/gabo-wave-report/SKILL.md` section 2 (S9);
- `apps/cli/internal/skills/gabo-reviewer/scripts/review-head.sh` and the comment it posts (S11, S13, and S14 once decided); the user-level copy reaches users through `gabo init`, and `gabo init` runs its refresh when the copy differs (intent 0016 S4);
- the `gabo-reviewer-notes` skill, which explains the review layers to other agents (its catalog copy and its `.gabo/skills/` copy);
- TRD 9.2 and the reviewer rows of TRD 9.1 and D6 (section 5).

Each catalog file has its byte-for-byte copy under `.gabo/skills/`, which the plan updates in the same change. A change under `.gabo/skills/` is human-only (TRD D24), so the plan's PR is merged by a human.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Where the RDD and ODD rules live | (a) keep them in the gentle-ai adapter in the methodology slot; (b) move them into `gabo-coder`; (c) a gabo CLI command that wraps gentle-ai | (a). (b) would make the coder skill carry a methodology, which its own text forbids; (c) is excluded by PRD R31 | The adapter change merges by a human (`.claude/` is human-only). TRD TD5 stays open (section 5) |
| D2 | Outcome vocabulary for RDD | (a) the adapter's five values `granted`, `declined`, `passive`, `off`, `unavailable`; (b) the five plus gentle-ai's `review_due_reason` verbatim when no review was due; (c) free text | (b) | A commit gentle-ai reports as not due (for example `under_budget` or `already_reviewed`) still gets a closed-list outcome, so the owner can count it |
| D3 | Where the outcome travels | (a) the coder report only; (b) report, wave trace and wave report line; (c) the tracker task comment | (b). (a) is lost on resume (PRD R25); (c) puts a second report in the tracker | Each outcome adds one trace line and one wave report field; TRD 9.2 gains the field (section 5) |
| D4 | Selector for assessing committed work | (a) the tree-only command of the adapter today, which returns `unassessable` on committed work (observed); (b) `--base-ref <base> --committed-only`, which gentle-ai's help documents; (c) a gabo wrapper script | (b) | The base choice follows OQ 4. (c) adds a script the owner did not ask for |
| D5 | Who answers a consent envelope | (a) the coder answers it; (b) the coder declines it itself; (c) the coder returns it to the assistant | (c) | gentle-ai makes consent candidate-scoped and owner-given, and a coder cannot give it. The route is OQ 1 |
| D6 | Whether the reviewer or RDD gates the merge | (a) the reviewer (unchanged); (b) RDD replaces the reviewer for granted candidates | (a) | PRD R9 and the methodology contract. Option (b) would need a PRD change and is not offered as a question |
| D7 | Where product QA runs (owner role b) | (a) a second section of the existing reviewer run, same script, same clean environment, same verdict and merge gate; (b) a separate QA script in the `gabo-reviewer` skill, run by the same gate; (c) the coder runs QA before its push | (a), Proposed (auto); depends on OQ 8 and OQ 9 | One run and one verdict per pushed head (PRD R26). Product QA needs the build under test, which touches the environment and binary rules of section 5. (c) moves the product check back to the coder, which the owner's split does not allow: the reviewer validates the product |

## 3. UX and app flow (optional)

Not applicable: the change has no screen and no user-facing surface. The owner reads the wave report, which is skill text (`gabo-wave-report` section 2), and the new `RDD:` and `ODD:` fields are plain text lines in it. The reviewer's product QA result is a plain-text section of its PR comment.

## 4. DB migration steps (optional)

Not applicable: the change alters no stored data. The wave trace gains lines of the existing shape, written by the cycle skill; no migration.

## 5. Policy conflicts

Nothing in this section is resolved by this spec. Each row names the owner decision that would resolve it.

| policy | conflict | resolution |
|---|---|---|
| PRD R15: in auto mode the human receives no prompts, only the final answer | A per-candidate consent envelope needs the owner | Open: OQ 1. Nothing is decided here; the proposal keeps the answer for the final answer and does not decline the candidate |
| PRD R5 and R29: questions reach the assistant and the wave report | The envelope is a question with no answerer in the coder | Consistent with S4 and D5: the envelope goes to the assistant as a question |
| PRD R9 and the methodology contract: RDD never replaces the reviewer | None: S5 states it | None |
| PRD R9 (`PRD.md:148`): "a mandatory reviewer-side merge gate (tests, build, lint and an independent agent review)" | Product QA changes what the independent agent review judges: from the diff to the running product | Open: OQ 3 and OQ 10. Not resolved here |
| TRD D6 (`TRD.md:588`) and D20 (`:602`): the reviewer runs checks and an agent review over the diff, from user-level scripts | Product QA adds a run of the build under test; the reviewer's role is no longer only the diff | Open: OQ 8 and OQ 9. Not resolved here |
| TRD D20 and the reviewer's clean, credential-free environment (`review.checks` run with a scrubbed environment) | A headless engine session for QA (OQ 8 option b) needs model credentials in the reviewer's environment | Open: OQ 8. Option (a) of OQ 8 needs none. Not resolved here |
| TRD D24: the reviewer runs the installed gabo, at least the base's `.gabo/VERSION`, and never self-updates | Product QA runs the build under test from the merge result, which is a different binary from the installed one | Open: OQ 8. Not resolved here. The PR's build is already the code `review.checks` runs on the merge result |
| PRD R26: the reviewer reviews each pushed head at most once | Product QA repeated per command or per fix round would multiply the runs | Consistent with D7 (a): one run per head. The cost is OQ 8 |
| PRD R9 separation: the owner named it as "separation of duties"; PRD and TRD do not use that phrase | The rule that matters is R9's: PR code cannot change the scripts that review it | Product QA runs PR code (the build under test), as `review.checks` already does. The rule keeps PR code out of the scripts, not out of the run. Open: OQ 8 |
| `gabo.yaml:45-48` `review.checks` and `gabo.yaml:53` `review.human_only_paths` (`gabo.yaml` is human-only) | OQ 9 may move the checks to CI or change them | Not changed here. A change to `gabo.yaml` merges by a human |
| TRD 9.2: the coder outcome and wave report fields | Gains the RDD and ODD fields and the done contract line | Additive. The TRD 9.2 amendment is written in the plan after the owner accepts the spec |
| `gabo.yaml:53` `review.human_only_paths` includes `.claude/` | The adapter and the coder agent are human-merged | Not a conflict: the plan's PR says a human merges it. The coder never merges it |
| TRD 675: gentle-ai's own workflow may overlap the cycle (to verify) | Adopting ODD and RDD tightens the overlap | The intended split is kept: RDD is the coder-side layer, the reviewer the merge gate. Checked on the probe wave (S9): no second merge decision |
| TRD TD5: the coder skill hardcodes ODD test-first | Not resolved by this intent | S1 adds the report, not the move. Stays open |

## 6. Open questions

Each question carries a **Proposed (auto)** option with a one-line reason. The owner decides each one; the spec is not accepted until the owner does. Questions 1, 2, 4 and 5 are the coder-side questions of the first draft, kept with their options. Questions 3 and 6 are rewritten for the reviewer's new role. Questions 7 to 10 are the owner's items A to D. Question 11 is the title.

| # | question | status |
|---|---|---|
| 1 | How does a per-candidate RDD consent reach the owner when a coder cannot give it, and what happens to the task meanwhile? | open: **Proposed (auto)**, option (a) |
| 2 | Does the ODD feature document (`odd/tasks/<feature>.md` and its Engram mirror) get created per task, or do the gabo plan row and the task issue act as it? | open: **Proposed (auto)**, option (b) |
| 3 | Who reads the code: does coder-side RDD own the code read while the reviewer runs product QA only, or do both still read the code? | open: **Proposed (auto)**, option (a) |
| 4 | Is the RDD candidate each work-unit commit before the push, or the PR head? | open: **Proposed (auto)**, option (a) |
| 5 | What happens when RDD is unavailable or its assessment fails? | open: **Proposed (auto)**, option (c) |
| 6 | Beyond visibility, does a gate enforce the coder's ODD, RDD and done contract? | open: **Proposed (auto)**, option (a) |
| 7 | Where does the product QA get what the product must do: the spec's acceptance criteria, the task issue, a done contract agreed before coding, or a combination? (owner item A) | open: **Proposed (auto)**, option (d) |
| 8 | How does QA test a product that is a CLI plus skills with no web UI, and at what cost per PR? (owner item B) | open: **Proposed (auto)**, option (a) on every product PR, (c) per wave, (b) only under a cost cap |
| 9 | Who gates the merge when QA does not read the code, and what happens to `review.checks` (build, vet, test)? (owner item C) | open: **Proposed (auto)**, option (a) |
| 10 | What happens when RDD did not run or was not due, and nobody reviewed the code? (owner item D) | open: **Proposed (auto)**, option (c) |
| 11 | The intent title no longer names both roles. Keep "Coders follow gentle-ai ODD and RDD", or propose a new title with the folder unchanged? | open: **Proposed (auto)**, option (b) |

### 6.1 Consent routing (question 1)

- **(a) Relay and wait.** The coder returns the envelope unchanged. The assistant relays it losslessly to the owner, and the task waits while other tasks continue. The assistant resumes the coder by its ledger handle with the owner's answer. Tradeoff: follows gentle-ai's candidate-scoped rule; in auto mode the wave can close with a task still waiting.
- **(b) Time-boxed decline.** The assistant declines after a fixed wait or at wave end, and the outcome is `declined`. Tradeoff: the wave never waits; RDD becomes optional in auto mode, which defeats the binding of this intent.
- **(c) Standing grant in `gabo.yaml`.** Not offered as an option: gentle-ai's consent is candidate-scoped, and a standing grant would grant consent for candidates the owner has not seen.

**Proposed (auto): (a).** In auto mode the task is not declined. The wave report shows `RDD: unavailable: consent pending` for that task, and the owner answers in the final answer, which the assistant relays. Reason: keeps consent with the owner and costs no prompt during the run (PRD R15); the gabo reviewer still gates the merge.

### 6.2 Specification artifact (question 2)

- **(a) ODD feature document per task.** `odd/tasks/<feature>.md` and its Engram mirror, as gentle-ai's ODD describes. Tradeoff: closest to the wording; a second tracker beside the plan and the task issues, with its own drift.
- **(b) Plan row and task issue act as the document.** The spec's requirements (S#) are the specification, the plan's spec refs link each task to them (`gabo-pm` spec template, section 6 wording), and the task issue carries the coder's report and its done contract (S12) as progress comments. Tradeoff: no duplicate tracker, which the intents README and the PRD require; the mapping of ODD names to gabo names must be accepted.
- **(c) Engram mirror only.** Tradeoff: memory is a trace, not a tracker; resume reads it, but the owner does not see it in the tracker.

**Proposed (auto): (b).** Reason: the tracker is the one global view (PRD section 7, intents README), and S2 already gives the traceability that ODD's document provides.

### 6.3 Who reads the code (question 3)

This question is rewritten for the reviewer's new role. The reviewer today reads the diff for code defects (`review-head.sh:324-328`).

- **(a) Coder-side RDD owns the code read; the reviewer runs product QA only.** The reviewer reads code only in the cases question 10 names. Tradeoff: matches the owner's split (`no valida código`); a commit that RDD did not assess (not due, unavailable, declined) gets no code read unless question 10 adds one.
- **(b) Both read the code; product QA is added on top.** Tradeoff: two code reads per due commit and the reviewer's diff read kept; the owner's split is only partly applied, and the cost is one more agent read per PR.
- **(c) Split by risk.** RDD reads the commits assessed as due, and the reviewer reads the diff for every other head. Tradeoff: cost follows risk and no commit is unread; the reviewer keeps a code read the owner asked to remove for the commits RDD covers.

**Proposed (auto): (a).** Reason: it is the owner's stated split, and question 10 covers the commits RDD does not reach, so every head keeps one code read.

### 6.4 Timing of the RDD candidate (question 4)

- **(a) Each work-unit commit before its push.** The assess command names the base (the branch point for the first commit, then the last reviewed commit, as gentle-ai's ODD states). Tradeoff: closest to gentle-ai's ODD; more candidates and consent envelopes per task.
- **(b) Once on the PR head before the first reviewer request.** A fix round that moves the head assesses again. Tradeoff: one candidate per head and fewer consents; the candidate can be a large accumulated diff, which weakens the per-slice review ODD asks for.
- **(c) Both.** Tradeoff: most cost.

**Proposed (auto): (a).** Reason: the owner asked for the closest match to gentle-ai's ODD and RDD, and gentle-ai reviews work-unit commits; the consent cost is question 1's.

### 6.5 RDD unavailable or the assessment fails (question 5)

- **(a) Report `unavailable`, treat the commit as due, continue.** The gabo reviewer still gates the merge, and the wave report shows the gap. Tradeoff: waves keep moving; the gap is visible but not blocking.
- **(b) Block the task until RDD works.** Tradeoff: strongest binding; one tool outage stops delivery of an informational layer.
- **(c) Retry the same assess once on the same commit, then (a).** Tradeoff: one more call; catches transient failures.

**Proposed (auto): (c).** Reason: transient failures are common and a retry costs one call; a persistent failure must not stop delivery, which the reviewer gates. The tier is never lowered in any option (S6).

### 6.6 Enforcement beyond visibility (question 6)

This question is rewritten to include the done contract (S12) and the reviewer's product QA.

- **(a) Visibility only.** The report fields, the trace and the wave report line, the done contract line in the task issue, and the pinned text tests (S8 to S10). Tradeoff: no change to the merge or to DONE; a coder can still skip RDD or the contract, but the wave report shows it.
- **(b) The cycle holds a task whose report has no RDD outcome or no done contract.** Tradeoff: stronger; a report gap pauses a wave.
- **(c) The reviewer refuses a PR whose task issue has no done contract, or whose report has no RDD line.** Tradeoff: moves the check into the merge gate; changes PRD R9 and the reviewer's job, so it needs an owner decision and a plan change, not this intent alone.

**Proposed (auto): (a) for this intent.** Reason: a gate changes the merge path or the wave flow, which is the owner's decision. Revisit (b) after two waves whose reports show the RDD line and the contract.

### 6.7 Source of the definition of done (question 7, owner item A)

- **(a) The spec's acceptance criteria only.** The Given/When/Then criteria of the S# the owner accepted. Tradeoff: one source, already human-accepted; the criteria are written before the product is built, but they are not turned into runnable checks for a given task.
- **(b) The task issue's acceptance section only.** The change request template already has `### Acceptance criteria`. Tradeoff: no new artifact; the criteria are written per task and can drift from the spec.
- **(c) A done contract written by the coder before coding, one observable check per S#.** Tradeoff: the agreement is explicit before code, as the harness article describes; the coder writes its own check, so the reviewer tests what the coder declared, and the contract needs a rule to stop it from weakening the spec.
- **(d) Combination of (a) and (c).** The spec defines what must hold; the coder turns each S# into a runnable check on the task issue before the first commit (S12), and the reviewer runs that contract (S11). Tradeoff: the most text; the spec stays the only requirement source and the task issue carries the proof.

**Proposed (auto): (d).** Reason: the accepted spec stays the only source of requirements, and the contract makes the agreement explicit before code, as the article describes for its sprint contracts. The article's contracts belong to a sprint-based harness it later replaces, so this is an adaptation, not a copy.

### 6.8 How QA tests a CLI plus skills product (question 8, owner item B)

The product is the `gabo` binary and skill text that an engine runs. It has no web UI for a browser test.

- **(a) Run the built `gabo` binary in a `mktemp -d` scratch project on every product PR.** Compare stdout, stderr and exit code with the contract line. Skills are not run. Tradeoff: deterministic and cheap; does not test what a skill does when an engine runs it.
- **(b) Run a skill through a headless engine session (for example `claude --print`) in a scratch repository.** Check the files and commands the skill produces, not the model's wording. Tradeoff: tests the skill as a user runs it; costs model time and tokens per run (not measured here), its output varies between runs, and it needs model credentials that the reviewer's clean, credential-free environment does not give (section 5).
- **(c) Probes on the seeded `testeos-z/gabo-repo`, reset to seed before and after, as intent 0026 S9 does.** Real PRs, tracker and comments, run per wave, not per PR. Tradeoff: the closest to the real flow; slow, and only as often as the wave runs.

**Proposed (auto): (a) on every product PR, (c) once per wave, and (b) only for PRs that change skill text and only under a cost cap the owner sets.** Reason: (a) is cheap and always runs; (b) is the only way to test a skill, but it costs model time and needs credentials, so it waits for the owner's cap; (c) reuses the probe method already in intent 0026.

### 6.9 Merge gate and `review.checks` (question 9, owner item C)

- **(a) Keep `review.checks` as a mechanical gate inside the reviewer.** The reviewer adds the product QA section (S13) to the same verdict; the reviewer still merges (`review.auto_merge: true`). Tradeoff: no infrastructure change; the checks already run on the merge result in a clean environment (`gabo.yaml:45-48`).
- **(b) Move build, vet and test to CI (GitHub Actions), and have the reviewer require the CI status.** Tradeoff: standard CI and its logs; changes `.github/`, which is human-only (`gabo.yaml:53`), and adds a second place where the checks run.
- **(c) Keep the checks in the reviewer and also require the coder's RDD `granted` outcome before merge.** Tradeoff: a stronger gate; turns an informational layer into a merge precondition, which question 10 and PRD R9 make an owner decision.

**Proposed (auto): (a).** Reason: the checks already run mechanically on the merge result, so keeping them costs no change; moving them (b) changes human-only infrastructure, and (c) is the owner's call (question 6 option c).

### 6.10 RDD not run and nobody reviewed the code (question 10, owner item D)

This applies when RDD is off, unavailable, declined, or not due for a commit.

- **(a) Merge on product QA and checks alone.** The report reads `code read: not run: <reason>`. Tradeoff: delivery is not blocked and the gap is visible; a defect in the diff can reach main unread, which is the owner's stated split taken to its end.
- **(b) Hold the merge until a coder review outcome of `granted` or a human reads the diff.** Tradeoff: no unread code; one unavailable RDD stops delivery of an informational layer (as question 5 option b does).
- **(c) The reviewer reads the diff for code defects (today's prompt) for each pushed head whose commits have no `granted` coder review, and product QA runs as well.** Tradeoff: no unread code and no blocked delivery; the reviewer keeps a code read the owner asked to drop, but only where the coder side did not run.

**Proposed (auto): (c).** Reason: a code read happens wherever the coder-side review did not, and an unavailable RDD does not block delivery, which is the rule question 5 chose; where the coder reviewed, the owner's split applies.

### 6.11 Intent title (question 11)

The title "Coders follow gentle-ai ODD and RDD" no longer names the reviewer's role. Per the intent rules, the folder slug is not renamed here: a new title is proposed and the owner decides.

- **(a) Keep the current title.** Tradeoff: no change to links or the README row; the title names only the coder side.
- **(b) "Coders and product QA share one definition of done", folder and slug unchanged.** Tradeoff: the title names both roles; the folder name then differs from the title, but the slug is the key every link and the README row use.
- **(c) New title and a renamed folder with a new slug.** Tradeoff: the title and slug match; every link, the README row and the issue links change. Owner decision only.

**Proposed (auto): (b).** Reason: the title should name both roles, and keeping the slug keeps every link and the README row valid. The intent's own title line and the issue title stay unchanged until the owner decides.

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
