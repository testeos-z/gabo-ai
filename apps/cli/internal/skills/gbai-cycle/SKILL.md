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
- Cleanup: remove coder branches and worktrees only after GitHub confirms the PR is merged. Never delete unmerged work.

## Execution Steps

1. Reconcile once: open PRs, claims, `needs-user` issues (`gbai status`).
2. Eligible = open, in the roadmap, all `blockedBy` closed, not `needs-user`, not claimed. Claim it (label + marker comment).
3. Run the wrk chain; launch coder sessions (up to the configured parallelism), each with `gbai-coder`.
4. On a verdict message: blocking ⇒ fix session; merged ⇒ advance and prune, then recompute eligibility (merges unblock issues).
5. Nothing eligible ⇒ end the turn and wait for the next inbox message.

## Output Contract

Per cycle: parent issue, sub-issues with PR and verdict, merged/blocked/needs-user, cleanup done.
