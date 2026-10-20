---
name: gbai-reviewer-notes
description: "Trigger: how the reviewer works, review verdict, reviewer bot, review run, why blocked, reviewer-ci. Explains the gbai reviewer side for other agents."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use when an agent needs to understand or explain a review result. Agents never run or influence the reviewer themselves.

## Facts

- `gbai review request` opens a detached Orca terminal that runs `sudo -n -u reviewer /usr/local/libexec/gbai/review-run head <pr> <sha>`, then pipes the summary line to the coder's mailbox.
- The reviewer re-reads the PR on GitHub and skips it when the head moved; it merges the pinned head with a fresh integration branch in a private worktree. A merge conflict is blocking.
- Checks and the review model run as `reviewer-ci` in a workspace fed by `git archive`, with no GitHub credentials; the model runs read-only with a fresh empty HOME.
- The verdict is computed by the script from local evidence (checks, conflict, findings), never by a model's or agent's message.
- The bot posts `reviewer/tests` and `reviewer/review` statuses and one comment whose first line is `No blocking issues found.` or `Blocking issues found.`.
- It merges only clean `mode:cloud` PRs into the integration branch, pinned to the reviewed SHA, and never PRs that touch human-only paths.

## Hard Rules

- Do not argue with a verdict in messages; fix the findings and push a new head.
- Read the bot's comment on GitHub for the findings; the message only says the verdict.
