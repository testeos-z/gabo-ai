# Intent 0031 Automerge resumes after a pending review check turns to success

| field | value |
|---|---|
| intent id | 0031-automerge-resumes-after-pending-review |
| status | Draft |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed; intent issue linked under it, as [#767](https://github.com/testeos-z/gabo-ai/issues/767) was) |
| intent issue | [#910](https://github.com/testeos-z/gabo-ai/issues/910) |
| date | 2026-11-26 |

Mode is `auto`. Every proposal the owner did not make carries a **Proposed (auto)** mark with a one-line reason.

## Problem

With `review.auto_merge: true`, the reviewer returns a signed clean verdict for the exact PR head, but the immediate merge check still reads `reviewer/review` as pending and skips the merge — and then nothing ever resumes it. The PR stays open even after GitHub shows both required checks as `SUCCESS` on the same head SHA. Asking for another review of that same head changes nothing: the head is no longer due because it already carries a final `reviewer/review` status and a verdict comment that signs it, so it is skipped. This is an automerge handoff problem, not a missing GitHub Approve action: gabo-reviewer never approves (the owner authors the PRs, so the verdict comment is the approval).

Observed twice in downstream runs, reported in [#908](https://github.com/testeos-z/gabo-ai/issues/908). Owner words from the issue (Context and problem): "The review produced a signed clean verdict for the exact head and the logs showed `REVIEWED … tests=success review=success`. The immediate merge check then logged `MERGE-SKIPPED … reviewer/review is pending`. GitHub later showed `reviewer/tests` and `reviewer/review` as `SUCCESS`, with the same head SHA, but the PR stayed open. Requesting review again for that same head skipped it because it already had a final `reviewer/review` status."

Reproduction sequence (from #908):

1. Enable `review.auto_merge` and run the configured reviewer against an open, otherwise eligible PR to the integration branch.
2. Have the reviewer return a clean verdict, but have the status read used by the immediate merge check still return `pending` for `reviewer/review`.
3. Let that status become `success` without changing the head, then invoke the normal reviewer entry point or its scheduled equivalent again.

Actual: the first run skips merging; the unchanged head is no longer eligible for review and automerge is not resumed. Expected: the reviewer can finish automerge for the already-reviewed head once all gates pass, without re-reviewing it or requiring a manual merge.

Evidence, checked 2026-11-26 on `dev` at `2687c7c`:

- **The skip is real.** `reviewer.sh` treats a head as not due when "its newest reviewer/review status from the bot login is final (success or failure, or error after REVIEWER_MAX_ATTEMPTS attempts) or a verdict comment by the bot login already signs it" (`reviewer.sh:20-21`), printing `SKIP #<pr> <sha> already has a final reviewer/review status` (`reviewer.sh:192`) or `already has a verdict comment` (`reviewer.sh:202`).
- **The gate is exact-head and status-based.** `merge-head.sh <pr> <sha>` merges only when the PR is open, targets the integration branch, is not a draft, comes from this repository, its head is still the reviewed SHA, and the newest `reviewer/tests` and `reviewer/review` statuses by the bot login on that SHA are success — plus a clean verdict comment signing that SHA, no human-only paths and no eval-notice hold (`merge-head.sh`, header and gate lines).
- **No continuation exists.** `reviewer.sh` offers `preflight|prune|once|list-due|due|head` (`reviewer.sh:33-36`); there is no mode that re-runs only the merge gate for an already-reviewed head.
- **A downstream local patch confirms the shape.** The #908 thread (comment 2026-11-24) reports a local fix worked on a real PR via a `reviewer.sh resume <pr-number> <full-reviewed-sha>` entry point that reused the existing merge gate without another model review or verdict; that patch is local to the downstream checkout, not an upstream release.

## Who is affected

- The owner and every team running gabo-reviewer with `review.auto_merge: true`: each PR whose required checks settle after the clean verdict stays open and needs a manual merge today.
- The reviewer schedule: every normal or scheduled invocation after the verdict skips the head instead of finishing the merge, so the stuck state repeats on every PR where the status read races the verdict.
- Frequency: observed twice downstream; it strikes whenever the merge check reads a still-pending status for a head that later settles without moving.

## Desired outcome

- A clean head skipped only because a required reviewer status was still pending is reconsidered once both required reviewer statuses are successful on that same head, and merges without a new model review and without a manual merge.
- The continuation reuses the existing merge gate unchanged in strictness: moved heads, pending or failed checks, disabled automerge, human-only paths and eval holds still prevent the automatic merge.
- Measure: no PR stays open in the "clean verdict, both checks green, same head, unmerged" state after a reviewer run; no head merges twice and no extra verdict is posted.

## Constraints

- Preserve the signed clean verdict check, the required checks, exact-head pinning, the configured integration branch, the automerge setting, human-only paths and eval holds (scope of #908; the gate lines of `merge-head.sh`).
- No coder or assistant merge permission changes; no GitHub Approve requirement; no duplicate review comments (#908 scope and non-goals).
- The merge stays exactly `gh pr merge <pr> --merge --match-head-commit <sha>` on the integration branch, never main (the gate's merge line).
- Test-first for behavior changes (`AGENTS.md`, conventions): the fix ships with its deterministic regression test.
- Planning only in this intent: no code, no task issues; the plan and its tasks come after the owner accepts the spec.

## Open questions

| # | question | status |
|---|---|---|
| 1 | How is the continuation triggered: a new reviewer entry point the owner or schedule invokes, the scheduled run picking up skipped-clean heads on its own, or both? | carried to spec (Q3, D1) |
| 2 | Where does the reviewer record that a head was already reviewed clean, so the continuation trusts the earlier verdict without re-running the model? | carried to spec (Q3, D2) |
| 3 | Which status contexts count as "both required reviewer statuses" if a project renames them? | carried to spec (Q3, D3) |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed) |
| bug report | [#908 Automerge does not resume after a clean review check changes from pending to success](https://github.com/testeos-z/gabo-ai/issues/908) |
| sibling intent | [0030-agent-evals-when-skills-change](../0030-agent-evals-when-skills-change/intent.md) (its S10 eval-notice hold is one of the gates the continuation must preserve) |
| sibling intent | [0032 intent landing branches](../0032-intent-landing-branches/intent.md) (planned in parallel; this intent does not touch it) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) (none yet; written by `gabo-dev-lead` after the owner accepts the spec) |
| sources | `apps/cli/internal/skills/gabo-reviewer/scripts/reviewer.sh:20-21,33-36,150,163-206`; `apps/cli/internal/skills/gabo-reviewer/scripts/merge-head.sh` (header and gate lines); `gabo.yaml:52` (`review.auto_merge: true`); #908 body and its 2026-11-24 confirmation comment; all checked on `dev` at `2687c7c` except the #908 thread |
