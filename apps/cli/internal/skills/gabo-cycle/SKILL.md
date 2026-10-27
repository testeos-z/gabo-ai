---
name: gabo-cycle
description: "Trigger: gbai cycle, run wave, resume wave, wave close, launch coders, wave ledger, wave report. Runs one wave of unblocked tasks with coder subagents and cleans up."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.1"
---
## Activation Contract

Load in the `gabo-cycle` subagent. The assistant gives you a wave id and the mode; you run that one wave and return the wave report. You use the `tasks`, `repository_host` and `memory` slot adapters. The whole wave runs through this skill and those adapters: no `gbai` command and no state dir (D15, D17).

## Hard Rules

- Target `repo.integration_branch` (`dev`); never `main`.
- Launch at most `parallelism` coder subagents at once (`gabo-coder`), one per task, each in its own worktree. Never exceed it while a review runs.
- Launch coders as plain in-process subagents, never as named teammates in their own panes, unless the owner asks.
- Resource ledger (D15, D23): every worktree, branch, container or test database a coder reports, and the runtime handle that resumes the coder, go into `gbai/ledger/<wave>` through the `memory` slot as soon as they are known. One line per resource: kind, name or path, creator, task, state (live, removed, leftover).
- The coder owns its task close and is the only writer of DONE (D23); the cycle never writes DONE. A coder that gets `merge=merged` closes on its own; after any other merge the assistant resumes it by the handle you recorded. A coder without a handle, or one the runtime cannot resume, leaves its entry to the fallback sweep; its merged task stays open and goes in the wave report.
- Your own cleanup (fallback sweep) runs only from the ledger, only for entries no coder could close, only after `repository_host` shows the PR merged, with ordinary commands and no `gbai` command: `git worktree remove <path>` (never `--force`), branches through `repository_host` (delete branch), `docker rm` for ledger containers, the database's own drop command for ledger test databases. Skip `worktree.keep` entries. A removal that fails is REFUSED: leave it and list it under leftovers. Never delete anything outside the ledger; a resource you find that is not in it is reported, never removed.
- Operating rules (PRD 8.1, TRD 9.5):
  - rule 1: a mid-wave task change waits for the round in progress. Re-read the task through `tasks` before each coder round (launch, fix round, resume) and pass the current text; never interrupt a round.
  - rule 2: during a plan-change debate mark only the affected task BLOCKED; the rest of the wave continues.
  - rule 3: a coder that reports BLOCKED after 4 fix rounds leaves its task BLOCKED; its question goes to the wave report for the assistant. Never launch a fifth round.
- Every write is re-read (R24): tracker writes through `tasks`, ledger and trace writes through `memory`. A write whose re-read does not show it is retried once, then listed under Unverified writes in the report.
- Board or tracker moves are never skipped silently: a denied write is returned as the exact list (`issue -> Ready`, `issue -> Backlog`) for the assistant to apply.
- Trace (D17): after every state change append the task's status, PR, last verdict, fix rounds and open questions to `gbai/trace/<wave>`, enough to resume.

## Execution Steps

1. Resume first (D17): read `gbai/trace/<wave>` and `gbai/ledger/<wave>`, then re-read every wave task through `tasks`, and recompute the wave from tracker state plus that ledger and trace. The cycle skips DONE tasks (closed issues). An open PR (`repository_host` read status) or a live ledger worktree ⇒ reattach: resume its coder by its handle, or launch a new coder on that worktree and branch. When tracker, host and trace disagree (for example the trace says HANDED and the issue has no PR), record the disagreement in the report and leave that task alone; never guess.
2. Read the open tasks and their blocked-by links through the `tasks` adapter; take the unblocked ones (every blocker closed) up to `parallelism`. Claim each (verified).
3. Per task: re-read it (rule 1), `git worktree add <path> -b <branch> origin/dev`, record worktree and branch in the ledger, launch `gabo-coder` with task, worktree, branch and mode; record its handle.
4. On each outcome (HANDED, DONE, BLOCKED): append its resources to the ledger; update the task (BLOCKED status, PR link; never DONE) and the trace; apply rules 2 and 3; recompute what is unblocked and launch the next task within `parallelism`.
5. Wave close is a fallback sweep: clean the merged entries no coder closed (rules above), without writing DONE; mark every ledger entry removed, live or leftover.
6. Report: build the TRD 9.2 wave report, save the wave report as the last entry of `gbai/trace/<wave>` through `memory` (verified), then return it to the assistant.

## Output Contract

Wave report (TRD 9.2): wave id; per task status, PR, last verdict, fix rounds, coder handle; questions with answer and resolver (R5); interaction count with reasons (R29); Unverified writes (R24); resume disagreements; REFUSED removals and leftovers (paths, containers, test databases), including merged tasks still open because no coder could close them (R13); plan-change proposals; denied board moves.
