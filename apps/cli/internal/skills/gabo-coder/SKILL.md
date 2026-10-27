---
name: gabo-coder
description: "Trigger: gbai coder, implement issue, worktree coder, open PR, request review, fix review findings. Rules for a coder subagent working one task in its own worktree."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.1"
---
## Activation Contract

Load in a `gabo-coder` subagent launched by `gabo-cycle` for one task in one git worktree. Before reading code, load the `methodology` slot adapter under its fixed name `gabo-methodology` (`slots.methodology.adapter`) and build the way it says; this skill carries no methodology of its own. Use the `tasks` and `repository_host` adapters for issue and PR operations.

## Hard Rules

- Never merge, approve or set commit statuses; the reviewer does that.
- Never post a comment, PR body or review containing the reviewer marker (`gbai-reviewer-bot`), even quoted: the reviewer posts with the same login and only its marker comment is the verdict.
- Open the PR to `repo.integration_branch` (`dev`), never `main`, through `repository_host` (open PR), with a body that links the task as `Closes #<n>`. On a base that is not the default branch the keyword closes nothing; the Task Close does. Never force-push; delete only your own task branch, and only in the Task Close.
- After every push, request one review from the `gabo-reviewer` subagent with exactly `PR #<n> head <40-hex pushed sha>`. Act on the `VERDICT` line it returns; the PR comment holds the findings.
- Questions go back to the cycle in your outcome; never ask the owner directly.
- You own your task to the end (D23): after the merge you close it yourself (Task Close), and you are the only writer of DONE. Between hand-off and merge stay resumable: end your turn, never poll or wait for the merge.

## Clean Worktree and Resources

- Write every scratch, draft, debug or probe file in a `mktemp -d` dir, never inside the worktree.
- Before opening or updating the PR, run `git -C <wt> status --porcelain --ignored=matching`. The worktree may hold only committed work plus rebuildable build output. Report anything else under `leftovers:`; never delete or move it.
- Test containers: reuse one per task. List every container you created under `containers:`. Never remove containers; the cycle does it from the ledger.
- Heavy suites: never run the same long suite twice at once.
- Committed migrations are immutable: fix them with a new migration, never by rewriting history.

## Decision Gates

| VERDICT | Action |
|---|---|
| `verdict=clean merge=merged` | Run the Task Close now. |
| `verdict=clean` other merge | Report "needs human merge"; HANDED. You are resumed with `PR #<n> merged` once it merges. |
| `verdict=blocking` | Fix round: read the latest marker comment, fix, push, request review for the new head. |
| `verdict=error` | Request once more; a second error ⇒ BLOCKED. |
| `verdict=skipped` | The head moved; request review for the current head. |

Max 4 fix rounds; then label the issue `needs-user`, report BLOCKED with the question.

## Task Close

Run when resumed with `PR #<n> merged`, or after `verdict=clean merge=merged`. This is the one exception to working only inside the worktree: run it from the main checkout, the parent of `git -C <wt> rev-parse --path-format=absolute --git-common-dir`. Touch only the worktree and branch of this task that you reported in your outcome. Other coders share that checkout: on a git lock error (`*.lock` exists), retry once before you report a leftover.

1. Verify through `repository_host` (read status) that the PR is merged into `repo.integration_branch` from your branch, and note its merged head SHA. Not merged, or another base or head branch ⇒ change nothing, report HANDED "not merged".
2. Close the task issue (`tasks` change status DONE) and re-read it: it must show closed. Retry once, then report it under `unverified:`.
3. Remove the worktree only if all hold, else keep it and its local branch, list them under `leftovers:` and skip the local branch in step 4:
   - it is not in `worktree.keep` (`gbai.yaml`);
   - `git -C <wt> status --porcelain --ignored=matching`, run before any removal, shows only committed work and rebuildable output (removal deletes ignored files);
   - `git worktree remove <wt>` succeeds, never `--force`.
4. Delete the branch through `repository_host` (delete branch). A refused branch goes under `leftovers:`.
5. Containers stay: the cycle removes them from the ledger.

## Output Contract

The outcome to the cycle (TRD 9.2) holds, in this order:

- `status`: `HANDED` (review clean, you stay resumable), `BLOCKED` (with the reason) or, only after the Task Close, `DONE`;
- task id and the PR link (number and URL, the PR that carries `Closes #<n>`), with the last reviewed SHA;
- a one-paragraph summary of the change;
- open questions for the cycle, each with what it blocks;
- resources created: worktrees, branches, containers and test databases, so the cycle can add them to the wave ledger;
- fix rounds used (at most 4);
- the `methodology` adapter's per-task report.

DONE carries the close report: merged SHA, issue closed (verified or not), worktree removed or left, local and remote branch deleted or left. Then always:

```
leftovers: <none | one path per line from status --ignored=matching that is not committed work or rebuildable output>
containers: <none | name and image of every container you created>
unverified: <none | each tracker or host write whose re-read did not show it>
resources: <worktree path, branch, and any test database you created>
```
