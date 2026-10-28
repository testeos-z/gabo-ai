# host-github: example adapter for the gabo repository_host contract

Fills the `repository_host` slot with GitHub through `gh` and `git`. Copy it to `.gabo/adapters/host-github.md` and point `slots.repository_host.adapter` at it. Repository `R=<owner>/<repo>`, integration branch `B=<repo.integration_branch>` (for example `dev`).

Branches are created in the task worktree with `git switch -c <type>/<issue>-<slug> origin/$B` and pushed with `git push -u origin HEAD`.

## Operations

### open PR

```bash
gh pr create -R $R --base $B --head <branch> --title "<conventional title>" --body-file <tmp>
```

The body holds a summary, `Refs #<issue>` and the verification run.

### comment on PR

```bash
gh pr comment <pr> -R $R --body-file <tmp>
```

### read PR status and checks

```bash
gh pr view <pr> -R $R --json state,baseRefName,headRefOid,mergeable,mergeStateStatus,statusCheckRollup,comments
```

### merge into integration branch

Check `baseRefName` equals `$B` first and refuse otherwise, then:

```bash
gh pr merge <pr> -R $R --squash --match-head-commit <sha>
gh pr view <pr> -R $R --json state,mergeCommit --jq '[.state,.mergeCommit.oid]|@tsv'
```

The read-back must show `MERGED`; its `mergeCommit.oid` is the merged SHA to return.

### delete branch

After the merge only, with the task worktree already removed:

```bash
gh pr view <pr> -R $R --json state,baseRefName,headRefName,headRefOid
git branch -d <branch>
git ls-remote --heads origin <branch>
git push origin --delete <branch>
```

1. The read must show `MERGED`, base `$B` and head `<branch>`; otherwise refuse.
2. Local first, with lower-case `-d` while the local `origin/<branch>` tracking ref still exists (no `fetch --prune` before it): git checks the branch against that upstream ref, so it succeeds after a squash merge and still refuses unpushed commits. A refusal is a leftover; never force it.
3. Remote only if `ls-remote` shows exactly `headRefOid`; no line means nothing to delete; any other SHA is a refusal and a leftover.
