---
name: gbai-reviewer-notes
description: "Trigger: how the reviewer works, review verdict, reviewer marker comment, review run, why blocked, dirty reviewer clone. Explains the gbai reviewer side for other agents."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use when an agent needs to understand or explain a review result. Agents never run or influence the reviewer themselves.

## Facts

- Single-user mode: the reviewer runs as the same OS user as the sessions and posts with the owner's `gh` login; no sudo, no extra users.
- `gbai review request` opens a detached Orca terminal that runs `gbai review run head <pr> <sha>` directly, then pipes the summary line to the coder's mailbox.
- The reviewer works from its own state dir (`~/.local/state/gbai/reviewer`: `clone/`, `worktrees/`, `ci/`) and refuses to run unless its clone is a clean checkout of the integration branch. Never edit or build inside that dir.
- It re-reads the PR on GitHub and skips it when the head moved; it merges the pinned head with a fresh integration branch in a private worktree. A merge conflict is blocking.
- Checks and the review model run in a workspace fed by `git archive`, with GitHub credentials stripped from their environment; the model runs read-only with a fresh empty HOME. This is trust-based, not an OS-user boundary.
- The verdict is computed by the script from local evidence (checks, conflict, findings), never by a model's or agent's message.
- It posts `reviewer/tests` and `reviewer/review` statuses and one comment whose first line is `No blocking issues found.` or `Blocking issues found.`, ending with the visible footer `🤖 gbai reviewer` and the hidden marker `<!-- gbai-reviewer-bot sha=<head> -->`. It never approves: GitHub forbids approving your own PR, and the ruleset requires no approval.
- Merge gate: the newest marker comment by the owner's login must sign the current head and open with `No blocking issues found.`; both statuses must pass. Marker comments by other logins are ignored.
- It merges only clean `mode:cloud` PRs into the integration branch, pinned to the reviewed SHA, and never PRs that touch human-only paths.

## Hard Rules

- Do not argue with a verdict in messages; fix the findings and push a new head.
- Read the reviewer's marker comment on GitHub for the findings; the message only says the verdict.
- The coder posts with the same login, so it must never post a comment, PR body or review containing the marker, even quoted; a forged marker could sign a head the reviewer never checked.
