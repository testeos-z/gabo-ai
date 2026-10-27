---
name: gabo-methodology-gentle-ai
description: "Trigger: gbai methodology, ODD, test-first, RDD, receipt-driven review, gentle-ai update. gentle-ai methodology adapter for gbai coders: ODD plus coder-side RDD."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "0.1"
---
## Activation Contract

Load in every coder as the `methodology` slot (`gabo-methodology`, TRD 9.1). gentle-ai is the recommended, not required, stack (D21); a team may point the slot at another methodology adapter.

## Hard Rules

- **ODD.** Explore the task's code first. Test-first when a relevant deterministic test exists: observe RED, implement GREEN, refactor green. Otherwise state the exception and run proportional checks.
- Work-unit commits with Conventional Commit messages, tests and docs with the behavior, no AI attribution lines. About 400 changed lines per PR is a heuristic, not a cap.
- **RDD (coder-side review).** Runs only when `gentle-ai review mode status` reports it on. Follow its returned transitions verbatim; a declined candidate continues under ordinary policy. RDD never replaces the gbai reviewer, which is the mandatory merge gate.
- Never toggle RDD (`review mode enable|disable`) on your own; that switch belongs to the owner.

## Coder Steps

1. Read the task, its acceptance criteria and the code it touches.
2. RED → GREEN → refactor per behavior; commit each work unit.
3. If RDD is on: `gentle-ai review assess --cwd <worktree> --json` on the commit and follow `review_due` / `next_transition`.
4. Run the repository checks the reviewer will run (`review.checks` in `gbai.yaml`) before pushing.

## Keeping gentle-ai current

Only at session start or between waves, never mid-wave (a version change mid-wave would split one wave across two toolchains):

| step | command | effect |
|---|---|---|
| check | `gentle-ai update` | read-only: lists installed vs latest |
| apply | `gentle-ai upgrade` | upgrades managed tools; ask the owner first in manual mode |
| refresh configs | `gentle-ai sync` | re-syncs agent configs and skills to the installed version |

Record an upgrade as a decision through the memory slot.

## Output Contract

Report per task: RED/GREEN evidence or the exception, commits, checks run with results, RDD outcome (granted, declined, passive, off, unavailable).
