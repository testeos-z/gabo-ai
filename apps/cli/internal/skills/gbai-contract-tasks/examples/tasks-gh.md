# tasks-gh: example adapter for the gbai tasks contract

Fills the `tasks` slot with GitHub Issues through the `gh` CLI. Copy it to `.gbai/adapters/tasks-gh.md` and point `slots.tasks.adapter` at it. Repository `R=<owner>/<repo>` (`repo.name` in `gbai.yaml`).

Notes:

- Sub-issue and dependency endpoints need the issue `id` (global integer), not its number: `gh api repos/$R/issues/<n> --jq .id`.
- PRs target the integration branch, so closing keywords do not close issues; close them explicitly.
- Status labels: `status:in-progress`, `status:blocked`, `needs-user`. Create a missing label with `gh label create "<name>" -R $R`.

## Operations

### create

```bash
gh issue create -R $R --title "T###: <title>" --body-file <tmp> --label <labels>
gh api -X POST repos/$R/issues/<n>/dependencies/blocked_by -F issue_id=<blocker id>
```

`unblocks` is the inverse edge: add `<n>` as `blocked_by` on each task it unblocks. Optional parent link: `gh api -X POST repos/$R/issues/<parent>/sub_issues -F sub_issue_id=<id>`.

### read unblocked

Open issues without a `status:*` or `needs-user` label whose open blockers count is zero:

```bash
gh issue list -R $R --state open --json number,labels
gh api repos/$R/issues/<n>/dependencies/blocked_by --jq '[.[]|select(.state=="open")]|length'
```

### read one

```bash
gh issue view <n> -R $R --json number,title,state,labels,assignees,body,comments
```

### claim

```bash
gh issue edit <n> -R $R --add-assignee @me --add-label status:in-progress
gh issue comment <n> -R $R --body "claimed by <agent> (wave <w>)"
```

### log progress

```bash
gh issue comment <n> -R $R --body "<!-- gbai-progress --> <one line>"
```

### change status

- DONE, only after the PR merged: `gh issue close <n> -R $R --reason completed --comment "DONE via PR #<pr>"`.
- BLOCKED: `gh issue edit <n> -R $R --add-label status:blocked --remove-label status:in-progress`, then a comment with the reason.

### link PR

The PR body carries `Refs #<n>`; comment `PR #<pr>` on the issue.

### verify by re-read

Repeat `read one` (or `gh api repos/$R/issues/<n>/dependencies/blocked_by` for dependencies) and compare the changed field. Retry a missing write once, then report it as unverified.
