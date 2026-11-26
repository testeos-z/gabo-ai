# Spec 0031 Automerge resumes after a pending review check turns to success

| field | value |
|---|---|
| intent | [0031-automerge-resumes-after-pending-review](intent.md) |
| status | Accepted |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: sections 3 and 4 are not applicable) |
| date | 2026-11-26 |

Mode is `auto`. Every decision the owner did not make carries a **Proposed (auto)** mark with a one-line reason. No agent accepts a spec: section 7 is the owner's. This spec encodes #908 and reopens none of its scope; #908's four acceptance criteria appear verbatim in S1–S4.

## 0. Terms and evidence

Terms used below:

- **Clean head:** a PR head for which the newest reviewer verdict comment by the bot login is clean and signs that head SHA (`merge-head.sh` verdict lines).
- **Pending skip:** a reviewer run that ends `verdict=clean merge=skipped` because the merge gate read a required reviewer status as still pending.
- **Continuation:** a later reviewer invocation for the same PR and head that runs only the existing merge gate, without a new model review and without a new verdict comment.

Evidence, checked 2026-11-26 on `dev` at `2687c7c` (except the #908 thread):

- **The gate to reuse.** `merge-head.sh <pr> <sha>` merges only when `REVIEWER_AUTO_MERGE=true`, the PR is open, targets the integration branch (`BASE_BRANCH`, never main), is not a draft, comes from this repository, its head is still `<sha>`, the newest `reviewer/tests` (`TESTS_CONTEXT`) and `reviewer/review` (`REVIEW_CONTEXT`) statuses by the bot login on that SHA are success, the newest verdict comment by the bot login newer than the head commit opens with the clean verdict and signs that SHA, no changed path is human-only, and no eval notice holds the head. Anything else ends `MERGE-SKIPPED <reason>`, `NEEDS-HUMAN-MERGE` or `MERGE-FAILED <reason>`.
- **The dead end today.** After a pending skip, the head carries a final `reviewer/review` status and a verdict comment signing it, so every later `reviewer.sh head <pr> <sha>` (or `once`) prints `SKIP` and never reaches the merge gate (`reviewer.sh:163-206`).
- **The confirmed shape.** The #908 thread (2026-11-24 comment) confirms a downstream local patch with a `reviewer.sh resume <pr> <sha>` entry point returned `merge=merged` into the configured integration branch once both checks showed `SUCCESS` for the same SHA, reusing the existing merge gate with no new model review or verdict; the reporter invoked the gate through `bash` because installed scripts land at `0644`, and notes the regression test covers those permissions. That patch is local, not an upstream release.

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | When a reviewer run ends in a pending skip for a PR head, the reviewer shall reconsider that same head on a later invocation once the required reviewer statuses settle, without requiring a new review request for it. | A clean head skipped for a pending reviewer status is reconsidered when both required reviewer statuses become successful on that same head. | Deterministic reviewer-script regression test with mocked GitHub status reads switching from pending to success while the PR head stays fixed: the first execution ends `verdict=clean merge=skipped` and the continuation merges only the pinned head once. |
| S2 | The continuation shall run the existing merge gate (`merge-head.sh`) for the pinned head and shall not start a new model review nor post another verdict comment. | That continuation uses the existing merge gate and does not re-run the model review or post another verdict. | Same regression test with the model-review path stubbed to fail if entered: the continuation merges while the stub records zero model calls and zero new verdict comments. |
| S3 | The continuation shall refuse exactly where the merge gate refuses today: a head that moved since the review, any required check still pending or failed, disabled automerge, a human-only path, or an eval-notice hold. | Moved heads, pending or failed checks, disabled automerge, human-only paths and eval holds still prevent the automatic merge. | Regression-test matrix over the ineligible cases (moved head, pending check, failed check, `REVIEWER_AUTO_MERGE` not true, human-only path, eval notice): each ends unmerged with the gate's existing outcome (`MERGE-SKIPPED` or `NEEDS-HUMAN-MERGE`), and the existing gates keep refusing them after the change. |
| S4 | Running the continuation for a PR that already merged shall change nothing: no second merge, no new comment, no new status. | Running the continuation after the PR merges does not issue another merge. | Regression test invoking the continuation after the mocked merge: the PR mock reports merged/closed, the gate is not re-entered for a merge, and exactly one merge call exists in the log. |
| S5 | The fix shall ship with a deterministic reviewer-script regression test that replays the #908 sequence with mocked GitHub status reads: pending then success on a fixed head, asserting first-run skip, one continuation merge of the pinned head, the S3 refusals, and the S4 repeat-invocation silence. | Given the mocked pending-to-success sequence on a fixed head, When the suite runs, Then the first execution does not merge, the continuation merges only the pinned head once, the ineligible cases stay unmerged, and a repeat after the merge merges nothing. | The new test file under `apps/cli/internal/skills/gabo-reviewer/scripts/tests/` passes in `review.checks` (`go -C apps/cli test ./...` covers the skill-text side; the script suite covers the behavior). |

## 2. Design

The change adds one continuation path to the reviewer that reaches the existing merge gate for an already-reviewed clean head, plus its regression test. It changes no gate condition: every refusal in `merge-head.sh` keeps firing exactly as today, and the merge command stays `gh pr merge <pr> --repo <repo> --merge --match-head-commit <sha>`.

Flow: a reviewer invocation for a PR whose head already carries a final clean verdict and a signed head either (D1, option chosen) runs the continuation entry point for that `<pr> <sha>`, or reaches the gate through the normal path when the head is otherwise due. The continuation re-reads the PR (open, branch, draft, head still `<sha>`), re-reads the two required statuses, re-checks the signed clean verdict, the human-only paths and the eval hold through `merge-head.sh` unchanged, and merges only when every check passes. A head that moved, a check still pending or failed, disabled automerge, a human-only path or an eval hold ends exactly where the gate ends today.

Affected files (final paths are the plan's): `apps/cli/internal/skills/gabo-reviewer/scripts/reviewer.sh` (the continuation entry point and its due/skip wording) and its copy installed by `gabo init`; `apps/cli/internal/skills/gabo-reviewer/scripts/tests/` (the S5 regression test and its stubs); reviewer docs if they enumerate the entry points.

Every decision below is **Proposed (auto)**; the owner's acceptance of this spec confirms or changes each.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Form and trigger of the continuation | (a) a new entry point `reviewer.sh resume <pr> <sha>` invoked by the owner or the schedule; (b) the scheduled/`once` run automatically re-entering the gate for skipped-clean heads; (c) a flag on `head` | (a). Reason: it is the shape the #908 thread already confirmed working downstream, and it keeps the normal due/skip path untouched so S3 behavior cannot shift silently. (b) stays possible later as a scheduler change. | The entry point validates its two arguments like `head` does and ends with the same one-line `VERDICT …` shape; docs list it |
| D2 | What the continuation trusts | (a) re-read the signed clean verdict comment and the exact-head pinning through `merge-head.sh` at continuation time, no new stored state; (b) a new local record of "reviewed clean" heads | (a). Reason: the gate already performs both checks on live data, so there is no second source of truth to go stale or to secure. | No new file, no new state, no migration; a verdict comment removed or superseded is handled by the gate's existing refusal |
| D3 | Which statuses count as "both required reviewer statuses" | (a) the two contexts the gate already requires (`TESTS_CONTEXT` / `REVIEW_CONTEXT`, i.e. `reviewer/tests` and `reviewer/review`); (b) a new configurable list | (a). Reason: #908 names exactly these two, and the gate already reads them from configuration. | S1's "both required reviewer statuses" is exactly the gate's pair; a project that reconfigures the contexts gets the continuation for its own pair with no spec change |
| D4 | Installed-script permissions | (a) cover the `0644`-installed invocation (`bash <script>`) in the regression test, as the #908 reporter notes; (b) change install permissions | (a). Reason: the behavior must hold as installed, and permission changes are a separate packaging concern. | The S5 test invokes the scripts the way the installed copies run |

## 3. UX and app flow (optional)

Not applicable: the change has no screen and posts no new comment. The owner sees the existing one-line outcomes (`MERGE-SKIPPED` with its reason, `MERGED`) in the reviewer log.

## 4. DB migration steps (optional)

Not applicable: the change alters no stored data (D2: no new state).

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| #908 scope: preserve the signed clean verdict check, required checks, exact-head pinning, integration branch, automerge setting, human-only paths and eval holds | A continuation that merges could weaken any of them | No conflict by construction: the continuation calls the unchanged gate, so every refusal in S3 fires exactly as today |
| #908 non-goals: no coder or assistant merge permission changes, no GitHub Approve requirement, no duplicate review comments | A new entry point could widen who merges | No conflict: the entry point only reaches the existing gate, which still merges only through `gh pr merge --merge --match-head-commit` on the integration branch with no approval action and no new comment |
| Intent 0030 S10: an eval notice holds the head for a person | The continuation must not release that hold | No conflict: the hold is inside the reused gate, so S3 keeps refusing held heads |
| Test-first for behavior changes (`AGENTS.md`) | A gate change without a replay of the incident | S5: the deterministic pending-to-success regression test ships with the fix |
| PRD R31 (no new CLI command, per spec 0030) | A continuation could be built as a command | No conflict: D1 is a reviewer-script entry point, not a `gabo` command |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Intent question 1 (trigger: explicit entry point vs automatic pickup by the scheduled run) | answered: D1 chooses the explicit `resume` entry point; automatic pickup is a later scheduler change, not this intent |
| 2 | Intent question 2 (where the clean-verdict record lives) | answered: D2, no new record; the gate re-reads the signed verdict at continuation time |
| 3 | Intent question 3 (which contexts count if renamed) | answered: D3, the gate's configured pair |
| 4 | Should the continuation also cover the `NEEDS-HUMAN-MERGE` and `MERGE-FAILED` exits, or only pending-skip heads? | open, carried to the owner. **Proposed (auto): only pending-skip heads in this intent.** Reason: #908 is exactly the pending-to-success race; human-only and eval holds need a person by design, and failures need diagnosis. Tradeoff: a transient `could not read` skip stays manual until the scheduler change |
| 5 | Does the owner want the scheduled reviewer run to invoke the continuation on its own once this intent lands, or keep it manual until the landing-branch intent (0032) settles scheduling? | open, carried to the owner. **Proposed (auto): keep it manual (owner or explicit schedule entry), and decide automatic pickup together with 0032.** Reason: avoids two intents changing scheduler behavior at once. Tradeoff: stuck heads wait for an explicit invocation until then |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
| leomerida15 (owner, alias Leo) | 2026-11-26 | Acceptance of the spec as written in the main session. Owner, verbatim (Spanish, English gloss in brackets): "capto los spec," [I accept the specs]. Context: the assistant asked for acceptance of the 0031 and 0032 specs, including the proposed D6 of 0032. Not covered: the intent and the plan. |
