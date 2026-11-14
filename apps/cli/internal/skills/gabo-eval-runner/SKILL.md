---
name: gabo-eval-runner
description: "Trigger: eval runner, eval set, evals/*.json, agent eval task, run the eval set, eval pass rate. Validates the versioned agent eval task set: format, acceptance, minimum and maximum, secret refusal."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
---
## Activation Contract

Load when a gabo agent proposes, validates or runs the agent eval set of intent 0030. This skill owns the task file format and the set rules. It runs no task and no engine: running is a later step of the same skill. The CLI gains no command for it (intent 0030 S12); the scripts live in this skill folder.

## Task Set

- The set is `evals/*.json` at the repository root, one JSON object per task, reviewed like code (PD of intent 0030, D2). A baseline and run reports are not tasks and never live there.
- Each task holds these fields:
  - `id`: the task id, for example `eval-001`.
  - `source_issue`: the task issue number the work came from.
  - `source_pr`: the PR number that carried the finished work.
  - `starting_commit`: the 40-hex commit the coder started from.
  - `task_input`: the task input the coder received, verbatim.
  - `spec_refs`: the S# ids the task links, for example `["S1", "S4"]`.
  - `outcome_checks`: one or more check commands a script can run.
  - `acceptance`: the owner's record, `{"accepted_by": "<login>", "accepted_on": "<YYYY-MM-DD>"}`.
- A task with no `acceptance` (or `acceptance: null`) is never run. The assistant proposes it; only the owner's acceptance adds the record.

## Rules

- A run counts only accepted tasks. A task with no acceptance record reads `not accepted` and is skipped.
- A task missing a required field is refused and the field is named: `refused <file>: missing <field>`. A `starting_commit` that is not 40 hex, a `spec_refs` entry that is not `S<n>`, or an acceptance record without `accepted_by` or `accepted_on` is refused the same way.
- The minimum comes from `eval.min_tasks` (default 2) and the maximum from `eval.max_tasks` (default 10), read from `gabo.yaml`. The run is refused with `set below minimum: <accepted> of <min>` when accepted tasks are fewer, and with `set above maximum: <tasks> of <max>` when the set holds more files than the maximum.
- A proposal that would add a task beyond `eval.max_tasks` is refused with `maximum reached: <n> of <max> tasks (eval.max_tasks); the proposal is not added`. The owner raises the maximum or removes a task; the assistant never does either.
- Secrets: a task file holding a token or key pattern (GitHub tokens, `sk-` keys, AWS access keys, Slack tokens, private key headers) is refused with `refused <file>: token or key pattern`. The matched value is never printed, copied or written anywhere. A task, baseline or report never holds a credential (PRD R21).

## Commands

Both print one line per file and exit 0 when the set may run or the proposal may be added, 1 on any refusal, 2 on bad usage. They write nothing.

- `bash <skill dir>/scripts/eval-set.sh check evals --config gabo.yaml`: validates the whole set. The last line is `set: <accepted> accepted of <tasks> tasks, minimum <min>, maximum <max>` when the set may run.
- `bash <skill dir>/scripts/eval-set.sh propose evals <candidate.json> --config gabo.yaml`: validates one proposed task before it is added to the set for review. A proposal needs no acceptance record yet.

## Hard Rules

- Never write a token, key or credential into a task file, a proposal, a baseline or a report. When a source PR printed a secret, the task is not proposed with that text.
- Never accept a task on the owner's behalf. The owner's acceptance is a human decision.
- The runner reads and validates only in this step. It never starts an engine, pushes, opens a PR, comments or writes to the tracker.
- Values the owner sets (the pass threshold and the cap) are never written by this skill.
