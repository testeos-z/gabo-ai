---
name: gabo-tasks-github
description: "Trigger: gbai tasks, create issue, sub-issue, blocked-by, claim task, close task, unblocked tasks, tracker. GitHub Issues adapter for the gbai tasks slot via gh."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "0.1"
---
## Activation Contract

Use whenever a gbai agent reads or writes tasks. This adapter fills the `tasks` slot contract (TRD 9.1) with GitHub Issues through the `gh` CLI. Repository: `repo.name` in `gbai.yaml` (below `R=testeos-z/gabo-ai`).

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

| contract op | command |
|---|---|
| create | Body follows `.github/ISSUE_TEMPLATE/change_request.yml`: sections `### Change type`, `### Summary`, `### Context and problem`, `### Proposed outcome`, `### Scope and non-goals`, `### Acceptance criteria`, `### Verification plan`, `### Related issues, PRs, or references`. `gh issue create -R $R --title "T###: <title>" --body-file <tmp> --label <labels>` |
| sub-issue | `gh api -X POST repos/$R/issues/<parent>/sub_issues -F sub_issue_id=<child id>`; list: `gh api repos/$R/issues/<parent>/sub_issues` |
| blocked-by | `gh api -X POST repos/$R/issues/<n>/dependencies/blocked_by -F issue_id=<blocker id>`; remove: `gh api -X DELETE repos/$R/issues/<n>/dependencies/blocked_by/<blocker id>`; `unblocks` = `gh api repos/$R/issues/<n>/dependencies/blocking` |
| read one | `gh issue view <n> -R $R --json number,title,state,labels,assignees,body,comments` |
| read unblocked | open issues without `status:*`/`needs-user` whose `gh api repos/$R/issues/<n>/dependencies/blocked_by --jq '[.[]|select(.state=="open")]|length'` is `0` |
| claim | `gh issue edit <n> -R $R --add-assignee @me --add-label status:in-progress`, then a comment `claimed by <agent> (wave <w>)` |
| log progress | `gh issue comment <n> -R $R --body "<!-- gbai-progress --> <one line>"` |
| link PR | PR body carries `Refs #<n>`; comment `PR #<pr>` on the issue |
| change status DONE | `gh issue close <n> -R $R --reason completed --comment "DONE via PR #<pr>"` (only after the PR merged) |
| change status BLOCKED | `gh issue edit <n> -R $R --add-label status:blocked --remove-label status:in-progress` plus a comment with the reason |

Endpoints verified against docs.github.com REST "Sub-issues" and "Issue dependencies" on 2026-10-26.

## Output Contract

Return the issue numbers touched, each write with `verified` or `unverified`, and any label created.
