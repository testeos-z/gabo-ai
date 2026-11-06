---
name: gabo-cycle
description: "Trigger: gabo cycle, run wave, resume wave, wave close, launch coders, wave ledger, wave report. Runs one wave (one intent's unblocked tasks) with coder subagents and cleans up."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "2.3"
---
## Activation Contract

Load in the `gabo-cycle` subagent. The assistant gives you a wave id and the mode; you run that one wave and return the wave report. A wave is exactly one accepted intent: the wave id is the intent id (`I####`), and the wave runs only that intent's tasks. You use the `tasks`, `repository_host` and `memory` slot adapters. The whole wave runs through this skill and those adapters: no `gabo` command and no state dir (D15, D17).

## Hard Rules

- Target `repo.integration_branch` (`dev`); never `main`.
- Launch at most `parallelism` coder subagents at once (`gabo-coder`), one per task, each in its own worktree: `parallelism` bounds the coders in parallel within this intent. Never exceed it while a review runs. When several waves run at once (one per session), `parallelism` bounds this wave only; there is no project-wide counter (spec 0013 open question 3, owner).
- Session engine (intent 0013 S7): you run the wave on the engine of your session, the `gabo-engine:` value of your built `gabo-cycle` agent (`{{engine}}`, intent 0013 S6). When `runtimes.coder` is `session` or equal to your session engine, coders are in-process subagents as today. When it is another fixed engine, coders still run in-process on the session engine (subagents cannot change engine, TRD 2) and the wave report carries `coders ran on <session engine> (runtimes.coder is <engine>; subagents inherit the session engine)`. The wave report always carries `session engine: <engine>`. An agent file with no engine line (built before the slot) gives `session engine: unknown (rebuild the agents)`; then a `session` value in `runtimes.coder` or `runtimes.reviewer` cannot be resolved: start no task, claim nothing and return that as the question for the assistant.
- One intent per wave: never take a task of another intent. A task of this intent blocked by a task of another intent stays blocked: it is reported, never pulled in, and that other task is not started.
- Reviewer isolation gate: a wave, auto mode included, never starts with a `codex` or `opencode` reviewer that has no isolation test pass (intent 0002 S12): when `runtimes.reviewer` in `gabo.yaml` is `codex` or `opencode`, run the reviewer preflight (step 1) before you claim any task. On failure start no task, claim nothing and launch no coder. There is no other way around the check: the only fix is the owner running the `gabo-reviewer-config` isolation test at configuration time, never during a wave. `claude` and `pi` reviewers start as before, without this step. Under `runtimes.reviewer` `session`, resolve the review engine first and apply the same gate to the resolved engine (step 1, intent 0013 S11).
- Launch coders as plain in-process subagents, never as named teammates in their own panes, unless the owner asks.
- Resource ledger (D15, D23): every worktree, branch, container or test database a coder reports, and the runtime handle that resumes the coder, go into `gabo/ledger/<wave>` through the `memory` slot as soon as they are known. One line per resource: kind, name or path, creator, task, state (live, removed, leftover).
- The coder owns its task close and is the only writer of DONE (D23); the cycle never writes DONE. A coder that gets `merge=merged` closes on its own; after any other merge the assistant resumes it by the handle you recorded. A coder without a handle, or one the runtime cannot resume, leaves its entry to the fallback sweep; its merged task stays open and goes in the wave report.
- Your own cleanup (fallback sweep) runs only from the ledger, only for entries no coder could close, only after `repository_host` shows the PR merged, with ordinary commands and no `gabo` command: `git worktree remove <path>` (never `--force`), branches through `repository_host` (delete branch), `docker rm` for ledger containers, the database's own drop command for ledger test databases. Skip `worktree.keep` entries. A removal that fails is REFUSED: leave it and list it under leftovers. Never delete anything outside the ledger; a resource you find that is not in it is reported, never removed.
- Operating rules (PRD 8.1, TRD 9.5):
  - rule 1: a mid-wave task change waits for the round in progress. Re-read the task through `tasks` before each coder round (launch, fix round, resume) and pass the current text; never interrupt a round.
  - rule 2: during a plan-change debate mark only the affected task BLOCKED; the rest of the wave continues.
  - rule 3: a coder that reports BLOCKED after 4 fix rounds leaves its task BLOCKED; its question goes to the wave report for the assistant. Never launch a fifth round.
- Every write is re-read (R24): tracker writes through `tasks`, ledger and trace writes through `memory`. A write whose re-read does not show it is retried once, then listed under Unverified writes in the report.
- Board or tracker moves are never skipped silently: a denied write is returned as the exact list (`issue -> Ready`, `issue -> Backlog`) for the assistant to apply.
- Trace (D17): after every state change append the task's status, PR, last verdict, fix rounds and open questions to `gabo/trace/<wave>`, enough to resume.

## Execution Steps

1. Reviewer preflight, only when `runtimes.reviewer` is `codex` or `opencode`, auto mode included: run the user-level copy, never a copy from a repository checkout, with no extra arguments or environment: `bash "${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh" preflight`. It reads the isolation pass marker of the engine and its installed version (intent 0002 T2). `PREFLIGHT-OK` ⇒ continue. Any other result (a `PREFLIGHT-FAILED` line, a non-zero exit, or a missing script) ⇒ start no task, claim nothing, launch or resume no coder, and return the wave report with the `PREFLIGHT-FAILED` line verbatim, which names the missing isolation test, as the question for the assistant.
   With `runtimes.reviewer` `session`, resolve the review engine first: `runtimes.reviewer` `session` is your session engine; a fixed `runtimes.reviewer` is that engine and keeps the step above unchanged. When the resolved engine is `codex` or `opencode`, run the same user-level copy with `engine <engine>` appended as its only extra argument, only under `session`: `bash "${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh" preflight engine <engine>`, still before any claim, with the same outcomes. A resolved `claude` or `pi` engine starts as before.
2. Resume first (D17): read `gabo/trace/<wave>` and `gabo/ledger/<wave>`, then re-read every wave task through `tasks`, and recompute the wave from tracker state plus that ledger and trace. The cycle skips DONE tasks (closed issues). An open PR (`repository_host` read status) or a live ledger worktree ⇒ reattach: resume its coder by its handle, or launch a new coder on that worktree and branch. When tracker, host and trace disagree (for example the trace says HANDED and the issue has no PR), record the disagreement in the report and leave that task alone; never guess.
3. Read this intent's tasks through the `tasks` adapter: `read unblocked` with the `intent` filter set to the wave id (every one of their blocked-by links counts, a blocker in another intent included); take up to `parallelism`. Claim each (verified). Tasks of this intent whose blocker belongs to another open intent go to the wave report as blocked across intents.
4. Per task: re-read it (rule 1), `git worktree add <path> -b <branch> origin/dev`, record worktree and branch in the ledger, launch `gabo-coder` as an in-process subagent (on your session engine) with task, worktree, branch and mode; record its handle.
5. On each outcome (HANDED, DONE, BLOCKED): append its resources to the ledger; update the task (BLOCKED status, PR link; never DONE) and the trace; apply rules 2 and 3; recompute what is unblocked with the same `intent` filter and launch the next task within `parallelism`.
6. The wave closes when every task of that intent is DONE or BLOCKED (blocked across intents included). Wave close is a fallback sweep: clean the merged entries no coder closed (rules above), without writing DONE; mark every ledger entry removed, live or leftover.
7. Report: build the TRD 9.2 wave report with `gabo-wave-report`, save the wave report as the last entry of `gabo/trace/<wave>` through `memory` (verified), then return it to the assistant.

## Output Contract

The wave report (TRD 9.2) in the format of `gabo-wave-report`, with its comfort answer `pending`, and the save as `verified` or `unverified`.
