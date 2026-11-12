# tasks-gh: example adapter for the gabo tasks contract

Fills the `tasks` slot with GitHub Issues through the `gh` CLI. Copy it to `.gabo/adapters/tasks-gh.md` and point `slots.tasks.adapter` at it. Repository `R=<owner>/<repo>` (`repo.name` in `gabo.yaml`).

Notes:

- Sub-issue and dependency endpoints need the issue `id` (global integer), not its number: `gh api repos/$R/issues/<n> --jq .id`.
- PRs target the integration branch, so closing keywords do not close issues; close them explicitly.
- Status labels: `status:in-progress`, `status:blocked`, `needs-user`. Create a missing label with `gh label create "<name>" -R $R`.
- Hierarchy: product card issue → intent sub-issue (`I####: <title>`, body links `docs/product/intents/<NNNN-slug>/`) → task sub-issues (`T###: <title>`). An issue has at most one parent, so a task has exactly one intent; `blocked_by` edges may cross intents. Create an intent once, when its folder is created:

  ```bash
  gh issue create -R $R --title "I0001: <title>" --body "Intent folder: docs/product/intents/0001-<slug>/"
  gh api -X POST repos/$R/issues/<card>/sub_issues -F sub_issue_id=<intent id>
  ```

## Operations

### create

```bash
gh issue create -R $R --title "T###: <title>" --body-file <tmp> --label <labels>
gh api -X POST repos/$R/issues/<n>/dependencies/blocked_by -F issue_id=<blocker id>
```

`unblocks` is the inverse edge: add `<n>` as `blocked_by` on each task it unblocks.

Parent intent (optional input `parent`, the intent issue number `<i>`); `replace_parent=true` moves a task that already has another parent:

```bash
gh api -X POST repos/$R/issues/<i>/sub_issues -F sub_issue_id=<task id>
gh api -X POST repos/$R/issues/<i>/sub_issues -F sub_issue_id=<task id> -F replace_parent=true
```

### read unblocked

Open issues without a `status:*` or `needs-user` label whose open blockers count is zero, sorted by the `T###` id in the title (plan order):

```bash
gh issue list -R $R --state open --limit 500 --json number,title,labels --jq 'sort_by(.title)'
gh api repos/$R/issues/<n>/dependencies/blocked_by --jq '[.[]|select(.state=="open")]|length'
```

With the optional `intent` filter (intent issue `<i>`), take the candidates from its sub-issues instead of the whole repo, then apply the same label and blocker checks; a blocker in another intent still counts:

```bash
gh api --paginate repos/$R/issues/<i>/sub_issues --jq '[.[]|select(.state=="open")|{number,title,labels:[.labels[].name]}]|sort_by(.title)'
```

### read one

```bash
gh issue view <n> -R $R --json number,title,state,labels,assignees,body,comments
```

Parent intent (`gh issue view` does not return it; a 404 means no parent):

```bash
gh api repos/$R/issues/<n>/parent --jq '{number,title}'
```

### claim

```bash
gh issue edit <n> -R $R --add-assignee @me --add-label status:in-progress
gh issue comment <n> -R $R --body "claimed by <agent> (wave <w>)"
```

### log progress

```bash
gh issue comment <n> -R $R --body "<!-- gabo-progress --> <one line>"
```

### change status

- DONE, only after the PR merged: `gh issue edit <n> -R $R --remove-label status:in-progress,status:blocked,needs-user`, then `gh issue close <n> -R $R --reason completed --comment "DONE via PR #<pr>"`. The removal covers every status label, not only `status:in-progress`, so a closed task keeps none.
- BLOCKED: `gh issue edit <n> -R $R --add-label status:blocked --remove-label status:in-progress`, then a comment with the reason.

### link PR

The PR body carries `Refs #<n>`; comment `PR #<pr>` on the issue.

### verify by re-read

Repeat `read one` (or `gh api repos/$R/issues/<n>/dependencies/blocked_by` for dependencies, `gh api repos/$R/issues/<n>/parent` for the parent intent) and compare the changed field. Retry a missing write once, then report it as unverified.
