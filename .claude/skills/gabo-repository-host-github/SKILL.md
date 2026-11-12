---
name: gabo-repository-host-github
description: "Trigger: gabo repository host, open PR, PR status, PR checks, merge into dev, delete branch. GitHub adapter for the gabo repository_host slot via gh and git."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
---
## Activation Contract

Use whenever a gabo agent works with branches or pull requests. Fills the `repository_host` slot contract (TRD 9.1). Repository `repo.name`, integration branch `repo.integration_branch` from `gabo.yaml` (here `R=testeos-z/gabo-ai`, `dev`).

## Hard Rules

- Every PR targets the integration branch `dev`. Never open a PR to `main`, never merge into `main`, never push to `main` or `dev` directly (TRD D13).
- Merge only into `dev`, only pinned to the head SHA that received a clean verdict, and only when the merge is yours to do (reviewer gate or explicit owner request). Coders never merge.
- Never force-push. Delete only branches the workflow created and that are merged (ledger, D15, D23). Never force-delete a branch (`-D`): delete the local branch with `-d` while the local `origin/<branch>` tracking ref still exists (no `fetch --prune` before it), so git checks it against that upstream ref (the merge commit keeps the pushed head reachable from `dev`), then the remote one. A `-d` refusal means unpushed work: leave it as a leftover.
- Re-read after every write: `gh pr view` must show the expected state.
- Never post the reviewer marker (`gbai-reviewer-bot`) in a PR body, comment or review.

## Operations

Commands come from this adapter's former operation table and the `repository_host` contract; `gh` flags and `--json` fields confirmed with `gh pr view --help` and `gh pr create --help` on gh 2.101.0. `R=testeos-z/gabo-ai` (`repo.name`), integration branch `dev` (`repo.integration_branch`).

### branch

Not a contract operation; kept for the coder. In the task worktree:

```bash
git switch -c <type>/<issue>-<slug> origin/dev
git push -u origin HEAD
```

### open PR

Body: summary, `Refs #<issue>`, verification run.

```bash
gh pr create -R $R --base dev --head <branch> --title "<conventional title>" --body-file <tmp>
```

Read back: `gh pr view <branch> -R $R --json number,state,baseRefName,headRefName,headRefOid` must show `OPEN`, base `dev`, head `<branch>`; return `number`.

### comment on PR

```bash
gh pr comment <pr> -R $R --body-file <tmp>
```

Read back: `gh pr view <pr> -R $R --json comments --jq '.comments[-1].body'` must match the posted body.

### read PR status and checks

```bash
gh pr view <pr> -R $R --json state,baseRefName,headRefOid,mergeable,mergeStateStatus,statusCheckRollup,comments
```

### merge into integration branch

Check `baseRefName == "dev"` first (`gh pr view <pr> -R $R --json baseRefName --jq .baseRefName`); refuse on any other base. Then:

```bash
gh pr merge <pr> -R $R --merge --match-head-commit <sha>
```

Read back: `gh pr view <pr> -R $R --json state,mergeCommit --jq '[.state,.mergeCommit.oid]'` must show `MERGED`; return the merged SHA.

### delete branch

After merge only: `gh pr view <pr> -R $R --json state,baseRefName,headRefName,headRefOid` must show `MERGED`, base `dev` and head `<branch>`; the worktree is already removed. In this order:

1. Local, while the local `origin/<branch>` tracking ref still exists (a refusal is a leftover; never `-D`):

   ```bash
   git branch -d <branch>
   ```

2. Remote, only if `git ls-remote --heads origin <branch>` shows exactly `<headRefOid>`; absent means nothing to do; another SHA means refuse:

   ```bash
   git push origin --delete <branch>
   ```

Read back: `git branch --list <branch>` and `git ls-remote --heads origin <branch>` are empty.

## Output Contract

Return PR number, base, head SHA, state, and for merges the merged SHA; report any refused operation with its reason.
