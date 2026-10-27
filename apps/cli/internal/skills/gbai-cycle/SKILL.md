---
name: gbai-cycle
description: "Trigger: gbai cycle, run wave, resume wave, wave close, launch coders, wave ledger, wave report. Runs one wave of unblocked tasks with coder subagents and cleans up."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.0"
---
## Activation Contract

Load in the `gbai-cycle` subagent. The assistant gives you a wave id and the mode; you run that one wave and return the wave report. You use the `tasks`, `repository_host` and `memory` slot adapters.

## Hard Rules

- Target `repo.integration_branch` (`dev`); never `main`.
- Launch at most `parallelism` coder subagents at once (`gbai-coder`), one per task, each in its own worktree. Never exceed it while a review runs.
- Launch coders as plain in-process subagents, never as named teammates in their own panes, unless the owner asks.
- Resource ledger (D15, D23): every worktree, branch, container or test database a coder reports, and the runtime handle that resumes the coder, go into `gbai/ledger/<wave>` through the `memory` slot as soon as they are known.
- The coder owns its task close and is the only writer of DONE (D23); the cycle never writes DONE. A coder that gets `merge=merged` closes on its own; after any other merge the assistant resumes it by the handle you recorded. A coder without a handle, or one the runtime cannot resume, leaves its entry to the fallback sweep; its merged task stays open and goes in the wave report.
- Your own cleanup (fallback sweep) runs only from the ledger, only for entries no coder could close, only after `repository_host` shows the PR merged: `git worktree remove <path>` (never `--force`), branches through `repository_host` (delete branch), `docker rm` for ledger containers. Skip `worktree.keep` entries. A removal that fails is REFUSED: leave it and list it under leftovers. Never delete anything outside the ledger.
- Mid-wave task change: re-read the task before the coder's next round; never interrupt a round (PRD 8.1 rule 1).
- Plan-change debate: mark only the affected task BLOCKED; the rest of the wave continues (rule 2).
- Every tracker write is re-read (R24); unverified writes go to the report.
- Board or tracker moves are never skipped silently: a denied write is returned as the exact list (`issue -> Ready`, `issue -> Backlog`) for the assistant to apply.

## Execution Steps

1. Resume first (D17): read `gbai/trace/<wave>` and `gbai/ledger/<wave>`; re-read every wave task through `tasks`. DONE ⇒ skip; open PR or worktree ⇒ reattach. Tracker and trace disagree ⇒ report, do not guess.
2. Read the unblocked tasks (all blocked-by closed), claim each (verified).
3. Per task: `git worktree add <path> -b <branch> origin/dev`, record it in the ledger, launch `gbai-coder` with task, worktree, branch and mode; record its handle.
4. On each outcome (HANDED, DONE, BLOCKED): append resources to the ledger; update the task (BLOCKED status, PR link; never DONE) and the trace, recompute what is unblocked, launch the next task within `parallelism`.
5. Wave close is a fallback sweep: clean the merged entries no coder closed (rules above), without writing DONE; mark the rest live or leftover.

## Output Contract

Wave report: wave id; per task status, PR, last verdict, fix rounds, coder handle; questions with answer and resolver; interaction count with reasons; unverified writes; REFUSED removals and leftovers (paths, containers); plan-change proposals.
