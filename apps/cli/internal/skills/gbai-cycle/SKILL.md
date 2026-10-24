---
name: gbai-cycle
description: "Trigger: gbai cycle, cloud cycle, run waves, parent issue cycle, autonomous coding cycle. Cloud variant of pilot-cycle: one parent issue per cycle, event-driven."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use in the cloud coordinator session to run one cycle per parent issue until nothing is eligible.

## Hard Rules

- One parent issue per cycle, chained with the wrk skills in order: purpose → scope (declared ODD/SDD track per sub-issue) → implementation → pr → advance → prune. Stop the chain on any gate failure.
- Event-driven only: react to Orca inbox messages. No timers, no sleep, no polling. A one-shot reconcile at startup is allowed.
- Target the integration branch from config; never `main`.
- The agent runtime is fixed by config; never ask which agent to use per worker.
- Human gates are rules: ambiguous scope or a merge conflict you cannot resolve ⇒ label `needs-user`, comment why, continue with the next eligible issue.
- Cleanup: only after GitHub confirms the PR is merged. Close a coder worktree only with `gbai worktree close <absolute path>` (one line `CLOSED ...` or `REFUSED ...`); it also deletes the merged local branch. Never `git worktree remove` directly, never `--force`, never delete unmerged work. A `REFUSED` worktree stays as is and goes to the owner block.
- Board moves are never skipped silently. A subagent denied the board write never retries another way; it returns the exact `issue -> Ready` list with item ids, and the coordinator applies it and re-reads the board.
- Launch at most `parallelism` coder sessions (config). It already leaves room for the reviewer's checks; never exceed it while a review runs.

## Execution Steps

1. Reconcile once: open PRs, claims, `needs-user` issues (`gbai status`).
2. Eligible = open, in the roadmap, all `blockedBy` closed, not `needs-user`, not claimed. Claim it (label + coordination comment, never the reviewer's verdict marker).
3. Wave planning: move every unblocked card of the wave Backlog → Ready on the project board, then re-read the board to confirm.
4. Run the wrk chain; launch coder sessions (at most `parallelism`), each with `gbai-coder`.
5. On a verdict message: blocking ⇒ fix session; merged ⇒ advance, `gbai worktree close <absolute path>` for the coder worktree, then recompute eligibility (merges unblock issues).
6. Wave close: hand the owner ONE block with every `REFUSED` worktree and its reason plus every test container the coders reported under `containers:` (and their `leftovers:`). Nothing from a finished wave may be left out.
7. Nothing eligible ⇒ end the turn and wait for the next inbox message.

## Output Contract

Per cycle: parent issue, sub-issues with PR and verdict, merged/blocked/needs-user, board moves applied, `CLOSED`/`REFUSED` lines, and the owner block (refused worktrees, containers, leftovers).
