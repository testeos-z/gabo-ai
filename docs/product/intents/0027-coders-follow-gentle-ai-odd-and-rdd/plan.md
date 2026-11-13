# Plan 0027 Coders and product QA share one definition of done

| field | value |
|---|---|
| intent | [0027-coders-follow-gentle-ai-odd-and-rdd](intent.md) |
| spec | [spec](spec.md) (accepted by the owner on 2026-11-13, section 7, row "Ok, aprueba el spec veintisiete") |
| status | Draft. Not started: no wave runs until the owner orders it. Tasks are in the tracker under [#758](https://github.com/testeos-z/gabo-ai/issues/758). |
| owner | gabo-dev-lead |
| date | 2026-11-13 |

Mode is `auto` (owner's standing order, 2026-11-13: apply the recommended option to every doubt and record it; merges to `main`, versions, publishing, secrets and the owner's own values stay with the owner). Every fixed question below is answered by `gabo-dev-lead` and marked **Proposed (auto)** with a one-line reason. The plan is not accepted by any agent: it changes nothing until a human reviews the plan PR.

The spec is not edited. Two items need the owner before their tasks can start: the TRD D24 reconciliation that product QA needs (task T11) and the owner's permission to use `testeos-z/gabo-repo` for the final probe (task T14). One item waits for intent 0028 (task T12). The blocked tasks are marked in their titles and in the tracker.

Human merge: every task that touches `.claude/` or `.gabo/skills/` is merged by a person, not by the reviewer. Those paths are human-only (`gabo.yaml:53`, and `.gabo/skills/` by built-in rule, TRD D20 and 9.4). Each such row says so in its acceptance cell.

## Fixed questions

| id | answer |
|---|---|
| Q1 | **Proposed (auto)**: see Steps. Three lines of work start together: the coder rules (T1 and T2, with T3 after T2), the gates and the trace (T4, then T5, then the TRD note T6), and the reviewer policy (T7, then T8, T9, T10 and T13). Reason: the three lines touch disjoint files, and each line's own order follows its shared file. |
| Q2 | **Proposed (auto)**: fourteen tasks, each one reviewable PR of about 400 changed lines or less, test-first where a runnable test exists, plus the final probe (T14). Reason: one behavior per task, split at the shared file that forces an order. |
| Q3 | **Proposed (auto)**: see the `spec refs` column and the Spec map below. Reason: each row names the S# and D# its acceptance proves. |
| Q4 | **Proposed (auto)**: see `depends_on` and `unblocks`, acyclic (checked with a script on 2026-11-13). There is no edge outside this intent. S29 waits for intent 0028, which has no spec and no task yet, so T12 carries status `blocked` and no `blocked_by` edge. Reason: a `blocked_by` edge needs a real task id, and 0028 has none. |
| Q5 | **Proposed (auto)**: see `acceptance`. Each acceptance starts with its failing test. Skill-text rules are pinned by Go text tests in `apps/cli/internal/skills` (the precedent of intent 0026 and spec S10). Reviewer rules are pinned by shell fixture tests in `apps/cli/internal/skills/gabo-reviewer/scripts/tests`. Size is `S` (about 100 changed lines), `M` (about 200 to 400) or `L` (more than 400, to split at unblock). Reason: the spec's own verification column names these forms. |
| Q6 | **Proposed (auto)**: yes. Every spec requirement has at least one task: see the Spec map (all of S1 to S15, S18, S19 and S25 to S29; the withdrawn ids are not in the plan). Reason: checked against the spec table. |
| Q7 | **Proposed (auto)**: see Risks to the plan. |
| Q8 | **Proposed (auto)**: see Open questions. |
| Q9 | **Proposed (auto)**: no `gabo check` update is needed. The change adds no config key, no CLI command, no agent file under `.claude/agents/` and no schema; it edits skill text, the reviewer scripts, a root `REVIEW.md`, and the TRD. Each catalog task keeps the `.gabo/skills/` copy byte-identical with its catalog file and runs `gabo check` (exit 0). Reason: intent 0008 S10 and TRD 8.5; spec S10 and PRD R31 (no new command). |

## Steps

1. **Coder rules first, in parallel.** T1 (methodology adapter, human merge) and T2 (`gabo-coder` report contract and done contract, human merge) start together. T3 (the verifier rule, S28) follows T2, because both edit the coder skill and its template.
2. **Gates and trace.** T4 (cycle gates, S18 and S19) follows T2, since the gates read the report fields. T5 (trace and wave report lines, S9) follows T4, since both edit step 6 of `gabo-cycle`. T6 (TRD 9.2 fields) follows T5.
3. **Review policy before review logic.** T7 (`REVIEW.md` at the repository root) must merge to `dev` before T8. Reason: T8 makes the reviewer read `REVIEW.md` from `dev` and fail closed when it is missing, so merging T8 first would refuse every head.
4. **Reviewer logic in one chain.** T8 (policy read from `dev`), then T9 (three passes, Important-only verdict, nit cap), then T10 (comment order and the `code read:` line), then T13 (docs). All four edit `review-head.sh` or its tests, so they run one at a time.
5. **Product QA and the test-edit rule wait.** T11 (product QA, S11) waits for the owner's D24 decision, then runs after T10. T12 (test-edit rule, S29) waits for intent 0028's record, then runs after T9.
6. **Final probe.** T14 runs one wave on `testeos-z/gabo-repo` with RDD on, after every other task. It needs the owner's permission to use that repository. Before it runs, the owner refreshes the user-level reviewer install (`gabo init`, intent 0016 S4), so the reviewer runs the merged scripts.

The spec's end-to-end acceptance is checked by T14, not by any single task.

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. The tracker titles are `I0027/T<n>: <title>` (the 0026 convention). Tracker issues (verified by re-read, parent #758, blocked-by links match `depends_on`): T1 #779, T2 #780, T3 #781, T4 #782, T5 #783, T6 #784, T7 #785, T8 #786, T9 #787, T10 #788, T11 #789, T12 #790, T13 #791, T14 #792. Labels: `m1-local-core`, the size label, `flow-improvements`; `status:blocked` and `needs-user` on the blocked rows.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | Coder RDD rules in the methodology adapter | S1, S3, S4, S5, S6, S7, S10; D2, D4, D5 | — | T14 | Human merge: `.claude/skills/gabo-methodology-gentle-ai/SKILL.md` is human-only. Test first: a new Go text test in `apps/cli/internal/skills` reads that file and fails on today's text: the assess command lacks `--base-ref` and `--committed-only` (S7); there is no retry-once-then-`unavailable` rule (S6); no rule to return a consent envelope unchanged (S4); no `review mode enable` or `disable` prohibition line (S4); the reviewer is not named as the merge gate (S5); no RED and GREEN or exception rule (S1); the outcome vocabulary lacks the five values and gentle-ai's `review_due_reason` (D2). The change makes it pass. One manual run in a `mktemp -d` scratch repository of the new assess command: the envelope has a risk tier, not `unassessable`; the command and its result go in the PR body. | S |
| T2 | Coder report contract: ODD evidence, covered S#, done contract, tests, checks output | S1, S2, S8, S10, S12, S15, S18 (fields) | — | T3, T4, T11 | Human merge: `.gabo/skills/gabo-coder/` is human-only; the catalog copy under `apps/cli/internal/skills/gabo-coder/` and its codex copy stay byte-identical. Test first: text tests (a new file next to `coder_test.go`) fail on today's `gabo-coder` SKILL.md and on the agent template for each report field: RED and GREEN for a deterministic test, or the stated exception (S1); one line per linked S# with `covered` or `not covered: <reason>` (S2); the S8 block with `not run: <reason>` for any field not produced; the done contract on the task issue before the first commit, or `contract: not run: <reason>` (S12); each test with its result, or `integration: not applicable: <reason>` (S15); each `review.checks` command line with its pasted output and an `exit: <n>` line (S18 fields). If the rebuilt `.claude/agents/gabo-coder.md` differs from the template after the change, it is rebuilt in the same PR (human-only, same human merge). The catalog test and `gabo check` pass. | M |
| T3 | Fresh-context verifier before the PR | S28 | T2 | T14 | Human merge: `.gabo/skills/gabo-coder/` is human-only. No file under `.claude/agents/` is added (PD4). Test first: the text test of T2 fails until the coder rule names a verifier that starts without the coder's conversation, exercises the changed behavior and the two nearest neighboring flows, reports what it ran and saw, changes no file, and never approves or blocks; the report field list gains `verifier: <commands and what it saw>` or `verifier: not applicable: no behavior change`. The rule is a fixed gabo rule (Q33). | S |
| T4 | Cycle gates: the done gate and the report gate refuse a close | S18, S19 | T2 | T5 | Human merge: `.gabo/skills/gabo-cycle/` is human-only. Mechanism: the cycle reads the coder's outcome text before it writes DONE (D8 a, Q26), so the gate is a rule of step 6 of `gabo-cycle/SKILL.md`, pinned by text tests (PD1). Test first: a Go text test fails until step 6 states both gates and their refusal strings: `refused: <command>` for a missing `review.checks` command (S18), `refused: exit line missing for <command>`, `refused: RDD line missing` and `refused: verifier line missing` (S19); `not run: <reason>` passes the gate; a refused task does not close. Probe fixtures run through the rule by hand (the S18 and S19 examples of spec section 1) and their results go in the PR body. `cyclesentences_test.go` stays green. | M |
| T5 | Cycle trace and wave report show RDD, ODD and gates for every task | S9; D3 | T4 | T6 | Human merge: `.gabo/skills/gabo-cycle/` and `.gabo/skills/gabo-wave-report/` are human-only. Test first: text tests over step 6 of `gabo-cycle` and section 2 of `gabo-wave-report` fail until each outcome is written to the wave trace (RDD outcome, ODD line, both gate outcomes) and the `Tasks` line shows `RDD: <outcome>`, `ODD: <RED/GREEN or exception>` and `gates: done <passed or refused>, report <passed or refused>` for every task; a missing value reads `not run: <reason>`. Resume keeps the trace lines (PRD R25). The probe (T14) shows the rendered lines. | M |
| T6 | TRD 9.2 records the coder outcome fields and the wave report line | S9 (TRD 9.2 gains the field, D3) | T5 | T14 | Docs only (`docs/product/TRD.md`), not human-only. Test first: a doc check fails until section 9.2 lists the S8 fields and the `RDD:`, `ODD:` and `gates:` wave report fields; the diff touches no other TRD line (D6, D20 and D24 wait for T11). `gabo check` exits 0. | S |
| T7 | Review policy `REVIEW.md` at the repository root | S25 (file and missing-file rule), S26 (three passes), S27 (Important, nit cap, do-not-report) | — | T8 | Not human-only, and it is read from `dev` only (D9). Test first: a Go text test in `apps/cli/internal/skills` reads `REVIEW.md` at the repository root and fails until it has the three pass headings `bugs`, `security`, `compliance` (S26); the compliance pass names spec.md, plan.md and the design principles; a `What Important means here` section with the playbook's definition (S27); a `Cap the nits` section of five with the count of the rest; a `Do not report` section (generated files and what `review.checks` already enforces). Merge before T8 (Steps, step 3). | S |
| T8 | Reviewer reads the policy and its rule sources from `dev` | S25, S14 (dev commit named); Q22, Q23 (D9) | T7 | T9 | Reviewer scripts, not human-only (user-level, D20). Test first: fixture tests in `apps/cli/internal/skills/gabo-reviewer/scripts/tests` (a fixture repository with an integration branch and a PR head) fail until: `REVIEW.md` is read with `git show <integration branch>:REVIEW.md` at the dev commit named in the comment, never from the PR head, so a PR that edits it is judged by the dev copy; `AGENTS.md` and `docs/product/` are read from `dev` too (Q23); a missing `REVIEW.md` on `dev` gives `review policy: not run: no policy file on dev` and no `VERDICT_CLEAN` (fail closed). The prompt names the policy file. `prune.test.sh` and `preflight.test.sh` stay green. `REVIEW.md` is not added to `review.human_only_paths` (PD5). | M |
| T9 | Reviewer prompt: three passes, Important-only blocking, nit cap | S26, S27, S14 (`code read:` result) | T8 | T10, T12 | Reviewer scripts. Test first: text tests of the prompt block of `review-head.sh` and fixture reviews fail until: every finding carries one pass tag, bugs, security or compliance (S26); the compliance pass names spec.md and plan.md or `compliance: not run: <reason>`; a review with no finding states that the three passes ran; only an Important finding caused by the PR makes the verdict blocking (S27, Q24), so the mapping at `review-head.sh:335-343` no longer blocks on High or Medium alone; at most five nits are listed, with the count of the rest; a fixture with seven nits and no Important gives five listed, a count of two, and `VERDICT_CLEAN`. | M |
| T10 | Reviewer comment order and the `code read:` line | S13 (order, checks section), S14 (`code read:` line) | T9 | T11, T13 | Reviewer scripts. Test first: a comment-template test on a fixture PR fails until the order is the `VERDICT` line, then the checks section, then the code read section, which ends with `code read: pass`, `code read: fail: <finding>` or `code read: not run: <reason>`; a failed `review.checks` command is named in the checks section; a code read finding is named on its line; all pass gives `VERDICT_CLEAN`. The `VERDICT` line stays the first line, and the merge gate in `merge-head.sh` is unchanged (a test pins that). The product QA section is added by T11 between the verdict and the checks. | M |
| T11 | Product QA: the reviewer runs the build under test as a user would [blocked-on-owner: D24] | S11, S13 (QA section), S12 (reads the done contract) | T10, T2 | T14 | **Status `blocked` and `needs-user` until the owner decides the D24 question (Open questions 1).** Human merge: no. Once unblocked, test first: a fixture PR that changes a `gabo` command's output is judged by a reviewer test that fails until the QA section names each contract line of the task issue as `pass`, `fail: <observed>` or `not run: <reason>`; a docs-only fixture reads `not applicable: no product behavior change`; a line that the reviewer did not run never reads `pass`; the two nearest neighboring flows are named, or `not run: <reason>` for each. The build runs in a `mktemp -d` scratch project and never in the live checkout. The same task updates TRD D6, D20 and D24 for the decided option. Size `L`: when unblocked, split the contract read from the build run if it exceeds about 400 changed lines. | L |
| T12 | Test-edit rule: an existing test loosened without a reason is an Important finding [blocked on intent 0028] | S29, S26, S27 | T9 | T14 | **Status `blocked`** until intent 0028 publishes the hook record that S29 reads (its spec and plan are not accepted; PD3). Human merge: no. Once unblocked, test first: a text test of the prompt and a fixture PR with one loosened test and no reason fail until the comment carries an Important finding for that test with its pass tag, and the verdict is blocking; an edit that does not loosen a test, or loosens it with a stated reason, raises no finding. | S |
| T13 | Reviewer docs and notes describe the policy, the passes and the code read | S5 (RDD never replaces the reviewer), S13, S14, S25 to S27 | T10 | T14 | Human merge: `.gabo/skills/gabo-reviewer-notes/` is human-only; `apps/cli/internal/skills/gabo-reviewer/SKILL.md` and `README.md` are not. Test first: a doc test over `gabo-reviewer-notes/SKILL.md`, the reviewer `SKILL.md` and `README.md` fails until they name `REVIEW.md` on `dev`, the three passes, the Important-only blocking rule, the `code read:` line, the comment order, and that RDD never replaces the reviewer (S5). The QA paragraph is added by T11. | S |
| T14 | Final probe on `testeos-z/gabo-repo` (the 0027 acceptance run) [blocked-on-owner: repository permission] | S1, S2, S3, S5, S9, S11, S12, S18, S19, S28, S29 | T1, T3, T6, T11, T12, T13 | — | **Status `blocked` and `needs-user` until the owner grants permission to use `testeos-z/gabo-repo` (Open questions 2).** Not human-only (a disposable repository). Once granted: one wave with RDD on, its probe tasks run the new rules: a task report shows RDD outcome, ODD line, covered lines, verifier line and checks output; the wave report shows the `Tasks` lines (S9); the gates refuse one deliberately incomplete fixture report and pass a complete one (S18, S19); the reviewer comment shows `VERDICT`, QA, checks and code read in that order (S13); the consent envelope is observed, or recorded as `not run: no consent envelope in this wave` (S4). The installed reviewer is the merged one (the `gabo init` refresh is done first). Results are posted on #758, read back. | M |

## Spec map

How each spec item lands on the tree at `dev` (`fb6fe19`, 2026-11-13). The spec is not edited. The withdrawn ids (S16, S17, S20 to S24) have no task.

| spec | what changes | task |
|---|---|---|
| S1 | ODD RED/GREEN rule in the adapter; report field in the coder contract | T1 (adapter), T2 (report) |
| S2 | one covered or not-covered line per linked S# | T2 |
| S3 | assess in the adapter, outcome when mode on or off | T1 (text), T14 (probe) |
| S4 | no consent answer, no mode toggle, envelope returned unchanged | T1 |
| S5 | the reviewer named as the merge gate, unchanged rule | T1, T13 |
| S6 | retry once, then `unavailable`, never `passive` | T1 |
| S7 | assess names `--base-ref` and `--committed-only` | T1 (text and the manual scratch run) |
| S8 | the methodology block fields of the coder outcome | T2 |
| S9 | trace lines and the `Tasks` line of the wave report | T5, T6 (TRD) |
| S10 | each rule above has its test; the reviewer template and prompt have theirs | every task's acceptance (T1 to T13) |
| S11 | product QA section, build under test, scratch project | T11 (blocked on D24) |
| S12 | done contract on the task issue before the first commit | T2 (rule), T11 (reviewer reads it) |
| S13 | comment order: verdict, QA, checks, code read | T10, T11 |
| S14 | code read of every pushed head under the policy, `code read:` line | T9, T10 |
| S15 | unit and integration tests in the report | T2 |
| S18 | done gate: `review.checks` output and `exit:` line | T2 (fields), T4 (gate) |
| S19 | report gate: required fields, refused or passed | T2 (fields), T4 (gate) |
| S25 | `REVIEW.md` read from `dev`, fail closed | T7 (file), T8 (read) |
| S26 | three passes tagged, one reviewer run | T7 (text), T9 (prompt) |
| S27 | Important blocks, nit cap of five, do-not-report | T7 (text), T9 (verdict and cap) |
| S28 | fresh-context verifier before the PR | T3 |
| S29 | loosened test without a reason is an Important finding | T12 (blocked on 0028) |

## Plan decisions

All are **decided by the owner's standing rule** (resolver: agent, owner's rule of 2026-11-13: apply the recommended option to every doubt and record it).

| id | what | question | choice | reason | task |
|---|---|---|---|---|---|
| PD1 | Mechanism and test form of the gates (S18, S19) | The spec's verification column says "Go unit tests over fixture reports". A Go checker needs a caller, and PRD R31 forbids a new CLI command; hooks belong to 0028. | The gate is a rule of the cycle's step 6 (D8 a), pinned by Go text tests that name each refusal string. The fixture reports are run through the rule by hand in the PR body and in the probe. | Keeps the spec's mechanism; the spec left the test form to the plan. gabo-pm confirms that the verification wording may read "text test plus fixture run" (Open questions 4). | T4 |
| PD2 | Order of the reviewer changes | Several reviewer tasks edit `review-head.sh` and its tests. | One chain: T8, then T9, then T10, then T13; T11 and T12 branch from it. | One owner per shared file at a time (the 0026 PD2 rule). | T8 to T13 |
| PD3 | Trigger of S29 | The spec ties the rule to the hook record of intent 0028. No 0028 spec or task exists. | T12 waits for 0028 and carries `blocked`, with no edge. A diff-based check now is not chosen. | A diff-based check would widen the trigger beyond the owner's decision. | T12 |
| PD4 | Where the verifier (S28) is defined | Q33 (a) fixed gabo rule; the playbook's example is a subagent file under `.claude/agents/`. | The rule goes in the coder skill and its template. No file under `.claude/agents/` is added. | Q33 (a) is decided; a new file under `.claude/` would be one more human merge with no added behavior. | T3 |
| PD5 | Whether `REVIEW.md` joins `review.human_only_paths` (D9) | Adding it is a `gabo.yaml` change, which a person merges. | Not added. The reviewer reads the file from `dev`, so a PR edit does not reach the review. | A `dev` read already gives PR code no effect on the rule; adding a path is a config change the owner may still choose. | T7, T8 |
| PD6 | TRD changes | TRD 9.2 gains the fields (D3). D6, D20 and D24 change with product QA. | 9.2 in T6 now. D6, D20 and D24 in T11, after the owner's D24 decision. | The 9.2 fields are settled by the spec; D24 is a convention the owner decides. | T6, T11 |
| PD7 | PRD R9 wording (spec sections 2 and 5) | The PRD is gabo-pm's document, not the dev lead's. | No task here. The assistant returns the wording to gabo-pm. | Not a requirement of the spec; the spec names it as an affected file for the PRD owner. | none (note) |
| PD8 | Human merges | Six tasks change `.claude/` or `.gabo/skills/` (T1, T2, T3, T4, T5, T13) and the reviewer merges clean heads by itself. | Each such row says "human merge". The reviewer's `merge-head.sh` stops a merge that touches them. | `gabo.yaml:53` and TRD D20 (`.gabo/skills/` is human-only by built-in rule). | T1 to T5, T13 |
| PD9 | Tracker | The intent item #758 exists under #492 (closed). Its body still says "Section 7 is empty". | No new item. A comment on #758 links the spec's acceptance and this plan PR, read back. The body is not edited. | The owner's order: the intent exists; a body edit is not asked. | — |

## Parallel groups

Edit surfaces are disjoint between groups. Inside a group the order is the `depends_on` column.

| group | tasks | files |
|---|---|---|
| G1 coder rules | T1 (independent), T2, then T3 | `.claude/skills/gabo-methodology-gentle-ai/SKILL.md` (T1); `apps/cli/internal/skills/gabo-coder/SKILL.md`, `apps/cli/internal/skills/gabo-agent-templates/agents/gabo-coder.md`, their `plugins/codex/skills/` copies and the `.gabo/skills/gabo-coder/` copy (T2, T3); one new test file (T1), one extension of `coder_test.go` (T2) |
| G2 cycle and report | T4, then T5, then T6 | `apps/cli/internal/skills/gabo-cycle/SKILL.md` and its copies (T4, T5), `gabo-wave-report/SKILL.md` and its copies (T5), `docs/product/TRD.md` (T6) |
| G3 reviewer | T7, then T8, then T9, then T10, then T13; T11 after T10; T12 after T9 | `REVIEW.md` (T7); `apps/cli/internal/skills/gabo-reviewer/scripts/` and its tests (T8 to T11); `apps/cli/internal/skills/gabo-reviewer/SKILL.md` and `README.md` and `gabo-reviewer-notes` and its copy (T13) |

Runnable at once (pairwise disjoint): T1, T2, T7. T3 follows T2. T4 follows T2. T5 follows T4. T6 follows T5. T8 follows T7. T9 follows T8. T10 follows T9. T11 waits for the owner. T12 waits for 0028. T13 follows T10. T14 waits for all (its `depends_on` lists the leaves T1, T3, T6, T11, T12 and T13; the rest are reached through them).

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| `REVIEW.md` is not on `dev` when T8 merges | every review fails closed (`review policy: not run`) and the auto-merge stops | Steps, step 3: T7 merges first; T8's fixture tests cover the missing case |
| The owner's machine keeps the old reviewer scripts | the probe checks the old rules, and the new ones never run | the `gabo init` refresh (intent 0016 S4) before T14; T14 records the installed version |
| Six human-merged PRs create a queue | T2, T3, T4, T5 and T13 wait for a person | T1 and T2 first (both start now); the rest follow by `depends_on` |
| Text tests pin wording, and wording drifts | tests fail on harmless edits, or pass on a changed rule | tests pin the rule words and the refusal strings, not whole paragraphs (the 0026 precedent) |
| The Important-only verdict depends on the model tagging findings | an under-tagged defect does not block | T9's fixture with seven nits and one Important; T14 probe on one seeded defect |
| Cost per PR rises (three passes, QA, a code read on every head) | the owner's budget | not measured here; T14 records the reviewer's run time and count per PR (Q31: no target until two waves) |
| D24 blocks product QA (T11) | S11 has no build under test until the owner decides | owner question 1; T11 is `blocked`; the other tasks do not depend on it except T14, which waits |
| S29 waits for 0028 (T12) | the probe (T14) waits too | owner question 3; T14 may run without S29 only if the owner says so |

## Open questions

1. **D24 for product QA (T11).** Product QA must run the build under test; TRD D24 says the reviewer runs the installed `gabo` and never self-updates. Option (a), recommended: the reviewer builds the merge result in a `mktemp -d` scratch project and never installs it, and D24 gains that clause (owner decides; the TRD convention is the owner's). Option (b): QA runs only the installed binary, and any contract line that needs the build under test reads `not run: installed binary is not the build under test` until a release. Tradeoff: (a) gives real coverage at the cost of one D24 amendment; (b) keeps D24 and gives little QA. (owner)
2. **Permission to use `testeos-z/gabo-repo` for the final probe (T14).** The owner's permission is still open (spec section 7, Log). Nothing runs against that repository until it is granted. (owner)
3. **The S29 trigger (T12).** Intent 0028 must define the hook record S29 reads, and its spec and plan must be accepted. (owner and assistant, through intent 0028)
4. **Verification wording of S18 and S19 (PD1).** The plan tests the cycle rule by text and by fixture runs, not by Go unit tests over a checker, because a checker needs a caller and PRD R31 forbids a new command. (gabo-pm, to confirm the wording)
5. **The `REVIEW.md` human-only entry (PD5).** Not added. The owner may add it to `review.human_only_paths` in `gabo.yaml`. (owner)
6. **PRD R9 wording.** The PRD is gabo-pm's document; the wording named in spec sections 2 and 5 has no task here (PD7). (gabo-pm)
