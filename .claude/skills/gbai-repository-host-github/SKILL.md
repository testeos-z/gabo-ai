---
name: gbai-repository-host-github
description: "Trigger: gbai repository host, open PR, PR status, PR checks, merge into dev, delete branch. GitHub adapter for the gbai repository_host slot via gh and git."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "0.1"
---
## Activation Contract

Use whenever a gbai agent works with branches or pull requests. Fills the `repository_host` slot contract (TRD 9.1). Repository `repo.name`, integration branch `repo.integration_branch` from `gbai.yaml` (here `R=testeos-z/gbai-team`, `dev`).

## Hard Rules

- Every PR targets the integration branch `dev`. Never open a PR to `main`, never merge into `main`, never push to `main` or `dev` directly (TRD D13).
- Merge only into `dev`, only pinned to the head SHA that received a clean verdict, and only when the merge is yours to do (reviewer gate or explicit owner request). Coders never merge.
- Never force-push. Delete only branches the workflow created and that are merged (ledger, D15). Force-delete a local branch (`git branch -D`) only when the PR `state` is `MERGED` and the branch tip is the PR's `headRefOid` or its ancestor; squash merges leave it unmerged for git, and anything newer is unmerged work.
- Re-read after every write: `gh pr view` must show the expected state.
- Never post the reviewer marker (`gbai-reviewer-bot`) in a PR body, comment or review.

## Operations

| contract op | command |
|---|---|
| branch | `git switch -c <type>/<issue>-<slug> origin/dev` in the task worktree; `git push -u origin HEAD` |
| open PR | `gh pr create -R $R --base dev --head <branch> --title "<conventional title>" --body-file <tmp>` (body: summary, `Refs #<issue>`, verification) |
| comment | `gh pr comment <pr> -R $R --body-file <tmp>` |
| read status and checks | `gh pr view <pr> -R $R --json state,baseRefName,headRefOid,mergeable,mergeStateStatus,statusCheckRollup,comments` |
| merge into dev | Check `baseRefName == "dev"` first, then `gh pr merge <pr> -R $R --squash --match-head-commit <sha>`; refuse on any other base |
| delete branch | after merge only: `gh pr view <pr> -R $R --json state,headRefOid` must show `MERGED`; `git fetch origin pull/<pr>/head`. Local: `git merge-base --is-ancestor <branch> <headRefOid>` succeeds ⇒ `git branch -D <branch>`, else `git branch -d <branch>` (a refusal is a leftover). Remote: `git push origin --delete <branch>` only if `git ls-remote --heads origin <branch>` shows exactly `<headRefOid>`; absent ⇒ nothing to do; other SHA ⇒ refuse |

## Output Contract

Return PR number, base, head SHA, state, and for merges the merged SHA; report any refused operation with its reason.
