---
name: gabo-methodology-gentle-ai
description: "Trigger: gabo methodology, ODD, test-first, RDD, receipt-driven review, gentle-ai update. gentle-ai methodology adapter for gabo coders: ODD plus coder-side RDD."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "0.1"
---
## Activation Contract

Load in every coder as the `methodology` slot (`gabo-methodology`, TRD 9.1). gentle-ai is the recommended, not required, stack (D21); a team may point the slot at another methodology adapter.

## Hard Rules

- **ODD.** Explore the task's code first. Test-first when a relevant deterministic test exists: observe RED, implement GREEN, refactor green. Otherwise state the exception and run proportional checks.
- Work-unit commits with Conventional Commit messages, tests and docs with the behavior, no AI attribution lines. About 400 changed lines per PR is a heuristic, not a cap.
- **RDD (coder-side review).** Runs only when `gentle-ai review mode status` reports it on. Follow its returned transitions verbatim; a declined candidate continues under ordinary policy. RDD never replaces the gabo reviewer: the gabo reviewer is the only merge gate, and after every push you run it and act on its `VERDICT` line, whatever the RDD outcome.
- Never toggle RDD (`review mode enable|disable`) on your own; that switch belongs to the owner. Never run `gentle-ai review mode enable` or `gentle-ai review mode disable`.
- A consent envelope (`gentle-ai.review-integration.consent/v3`) is never yours to answer: never answer, grant, decline or synthesize it, and run no START with an answer. Return it unchanged to the assistant as an open question of the task.

## Coder Steps

1. Read the task, its acceptance criteria and the code it touches.
2. RED → GREEN → refactor per behavior; commit each work unit.
3. If RDD is on: `gentle-ai review assess --cwd <worktree> --base-ref <base> --committed-only --json` on the commit and follow `review_due` / `next_transition`.
4. Run the repository checks the reviewer will run (`review.checks` in `gabo.yaml`) before pushing.

## Operations

Derived from the Hard Rules and Coder Steps above and the `methodology` contract; `gentle-ai` commands as named in this adapter.

### coder instructions

1. Read the task issue, its acceptance criteria and the code it touches (ODD: explore first).
2. When a relevant deterministic test exists: write it and observe RED, implement GREEN, refactor while green; the report names that test, its RED failure and its GREEN run. Otherwise the report states the exception and the checks it ran, and claims no RED or GREEN.
3. Commit each work unit with a Conventional Commit message, tests and docs with the behavior, no AI attribution lines (`git commit -m "<type>(<scope>): <summary>"`).
4. Before pushing, run every command of `review.checks` in `gabo.yaml` (split on whitespace, as the reviewer runs them) and fix failures.

### coder-side review

RDD, only when it is on. Check, read-only:

```bash
gentle-ai review mode status
```

When it reports on, after each work-unit commit and before its push run

```bash
gentle-ai review assess --cwd <worktree> --base-ref <base> --committed-only --json
```

The base is the branch point for the first commit and the last reviewed commit after it. Follow `review_due` / `next_transition` verbatim (execute the returned command unchanged); a declined candidate continues under ordinary policy. When the mode is `off`, the outcome is `off` and no assess command runs for that task.

An assessment that fails, is unavailable or returns `unassessable` is never read as low risk and never reported as `passive`: retry the same assess once on the same commit; if it still fails or returns `unassessable`, treat that commit as due and report the RDD outcome `unavailable` with the error line.

If the assess result or a transition carries a consent envelope (`gentle-ai.review-integration.consent/v3`), return it unchanged under open questions; consent is candidate-scoped and global RDD on never grants it. Never run `gentle-ai review mode enable` or `gentle-ai review mode disable`. RDD never replaces the gabo reviewer, the only merge gate, which still runs after the push.

### report per task

Add to the coder outcome: RED/GREEN evidence or the stated exception, commits (SHA and message), each `review.checks` command with its result, and the RDD outcome per assessed commit: one of `granted`, `declined`, `passive`, `off`, `unavailable`, with its `risk` and, when no review was due, gentle-ai's `review_due_reason` verbatim (for example `under_budget` or `already_reviewed`).

## Keeping gentle-ai current

Only at session start or between waves, never mid-wave (a version change mid-wave would split one wave across two toolchains):

| step | command | effect |
|---|---|---|
| check | `gentle-ai update` | read-only: lists installed vs latest |
| apply | `gentle-ai upgrade` | upgrades managed tools; ask the owner first in manual mode |
| refresh configs | `gentle-ai sync` | re-syncs agent configs and skills to the installed version |

Record an upgrade as a decision through the memory slot.

## Output Contract

Report per task: RED/GREEN evidence or the exception, commits, checks run with results, RDD outcome (`granted`, `declined`, `passive`, `off`, `unavailable`, plus `risk` and `review_due_reason` verbatim).
