---
name: gabo-tasks-github
description: "Trigger: gabo tasks, create issue, sub-issue, blocked-by, claim task, close task, unblocked tasks, tracker. GitHub Issues adapter for the gabo tasks slot via gh."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
---
## Activation Contract

Use whenever a gabo agent reads or writes tasks. This adapter fills the `tasks` slot contract (TRD 9.1) with GitHub Issues through the `gh` CLI. Repository: `repo.name` in `gabo.yaml` (below `R=testeos-z/gabo-ai`).

## Hard Rules

- **Re-read after every write (R24).** After each create, label, claim, comment, dependency or close, read the issue back and compare. A write that does not show up is an *unverified write*: retry once, then report it under `unverified:` in your outcome. Never claim success from the command's exit code alone.
- GitHub REST needs the issue **`id`** (global integer) for sub-issue and dependency bodies, not the `number`: `gh api repos/$R/issues/<n> --jq .id`.
- PRs target `dev`, not the default branch, so closing keywords do not close issues. Close issues explicitly (see Change status).
- Never post the reviewer marker (`gbai-reviewer-bot`) anywhere, even quoted.
- Never close, relabel or reassign an issue you were not asked to work on.

## Labels

Plan labels: `m1-local-core`, `m2-planning-tool`, `m3-waves-autonomy`, `parallel`, `size:S`, `size:M`, `size:L`. Status labels: `status:in-progress`, `status:blocked`, `needs-user`. Create a missing one before using it, then confirm with `gh label list -R $R --search <name>`:

```bash
gh label create "<name>" -R $R --color <hex> --description "<why>"
```

## Operations

Commands come from this adapter's former operation table and the `tasks` contract with its `gh` example adapter (`examples/tasks-gh.md`, same tool); `R=testeos-z/gabo-ai`.

### create

Body follows `.github/ISSUE_TEMPLATE/change_request.yml`: sections `### Change type`, `### Summary`, `### Context and problem`, `### Proposed outcome`, `### Scope and non-goals`, `### Acceptance criteria`, `### Verification plan`, `### Related issues, PRs, or references`.

```bash
gh issue create -R $R --title "T###: <title>" --body-file <tmp> --label <labels>
```

Dependencies use the issue `id` (`gh api repos/$R/issues/<n> --jq .id`):

- blocked by: `gh api -X POST repos/$R/issues/<n>/dependencies/blocked_by -F issue_id=<blocker id>`; remove: `gh api -X DELETE repos/$R/issues/<n>/dependencies/blocked_by/<blocker id>`.
- unblocks: the inverse edge, add `<n>` as `blocked_by` on each task it unblocks; list: `gh api repos/$R/issues/<n>/dependencies/blocking`.
- sub-issue (optional parent): `gh api -X POST repos/$R/issues/<parent>/sub_issues -F sub_issue_id=<child id>`; list: `gh api repos/$R/issues/<parent>/sub_issues`.

Return the new issue number.

### read unblocked

Open issues without a `status:*` or `needs-user` label whose open blockers count is `0`, in plan order (`T###` in the title):

```bash
gh issue list -R $R --state open --limit 500 --json number,title,labels --jq 'sort_by(.title)'
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
gh issue comment <n> -R $R --body "<!-- gabo-progress --> <one line>"
```

### change status

- DONE, only after the PR merged: `gh issue close <n> -R $R --reason completed --comment "DONE via PR #<pr>"`.
- BLOCKED: `gh issue edit <n> -R $R --add-label status:blocked --remove-label status:in-progress`, then `gh issue comment <n> -R $R --body "<reason>"`.

### link PR

The PR body carries `Refs #<n>`; then `gh issue comment <n> -R $R --body "PR #<pr>"`.

### verify by re-read

Repeat `read one` (or `gh api repos/$R/issues/<n>/dependencies/blocked_by` for dependencies) and compare the changed field. Retry a missing write once, then report it as `unverified`.

Endpoints verified against docs.github.com REST "Sub-issues" and "Issue dependencies" on 2026-10-26.

## Output Contract

Return the issue numbers touched, each write with `verified` or `unverified`, and any label created.
