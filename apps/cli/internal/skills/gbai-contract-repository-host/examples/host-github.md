# host-github: example adapter for the gbai repository_host contract

Fills the `repository_host` slot with GitHub through `gh` and `git`. Copy it to `.gbai/adapters/host-github.md` and point `slots.repository_host.adapter` at it. Repository `R=<owner>/<repo>`, integration branch `B=<repo.integration_branch>` (for example `dev`).

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
```

### delete branch

After the merge only:

```bash
git push origin --delete <branch>
git branch -d <branch>
```

Use lower-case `-d`, never `-D`.
