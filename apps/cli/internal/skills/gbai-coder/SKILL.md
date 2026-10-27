---
name: gbai-coder
description: "Trigger: gbai coder, implement issue, worktree coder, open PR, request review, fix review findings. Rules for a coder subagent working one task in its own worktree."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.0"
---
## Activation Contract

Load in a `gbai-coder` subagent launched by `gbai-cycle` for one task in one git worktree. Follow the `methodology` slot adapter for how to build; use the `tasks` and `repository_host` adapters for issue and PR operations.

## Hard Rules

- Never merge, approve or set commit statuses; the reviewer does that.
- Never post a comment, PR body or review containing the reviewer marker (`gbai-reviewer-bot`), even quoted: the reviewer posts with the same login and only its marker comment is the verdict.
- Open the PR to `repo.integration_branch` (`dev`), never `main`. Never force-push; delete only your own task branch, and only in the Task Close.
- After every push, request one review from the `gbai-reviewer` subagent with exactly `PR #<n> head <40-hex pushed sha>`. Act on the `VERDICT` line it returns; the PR comment holds the findings.
- Questions go back to the cycle in your outcome; never ask the owner directly.
- You own your task to the end: after the merge you close it yourself (Task Close). Between hand-off and merge stay resumable: end your turn, never poll or wait for the merge.

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

Run when resumed with `PR #<n> merged`, or after `verdict=clean merge=merged`. Run every git command from the main checkout (`git -C <main>`), never from inside the worktree.

1. Verify on the host: `gh pr view <n> --json state` is `MERGED`. Anything else ⇒ change nothing, report HANDED "not merged".
2. Close the task issue (`tasks` change status DONE) and re-read it: it must show `CLOSED`. Retry once, then report it under `unverified:`.
3. `git worktree remove <wt>`, never `--force`. A refusal ⇒ keep the worktree and its local branch, list both under `leftovers:`, skip step 4.
4. `git branch -D <branch>`: allowed only because step 1 saw `MERGED` (squash merges leave the branch unmerged for git).
5. `git push origin --delete <branch>` if `git ls-remote --exit-code --heads origin <branch>` still finds it.
6. Containers stay: the cycle removes them from the ledger.

## Output Contract

End with `HANDED`, `DONE` or `BLOCKED`, task id, PR number, last reviewed SHA, fix rounds used, open questions. DONE carries the close report: issue `CLOSED` (verified or not), worktree removed or left, local and remote branch deleted or left. Then always:

```
leftovers: <none | one path per line from status --ignored=matching that is not committed work or rebuildable output>
containers: <none | name and image of every container you created>
resources: <worktree path, branch, and any test database you created>
```
