---
name: gabo-eval-runner
description: "Trigger: eval runner, eval set, evals/*.json, agent eval task, run the eval set, eval pass rate. Validates the agent eval task set and runs its accepted tasks in scratch copies, eval.runs_per_task times."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.1"
---
## Activation Contract

Load when a gabo agent proposes, validates or runs the agent eval set of intent 0030. This skill owns the task file format, the set rules and the runner for one run of the set. The CLI gains no command for it (intent 0030 S12); the scripts live in this skill folder. The fixture tests run with no engine: they use a fake engine command, and a real engine is probed only by the intent's own runner probe task.

## Task Set

- The set is `evals/*.json` at the repository root, one JSON object per task, reviewed like code (PD of intent 0030, D2). A baseline and run reports are not tasks and never live there.
- Each task holds these fields:
  - `id`: the task id, lowercase letters, digits and hyphens, for example `eval-001`; unique in the set.
  - `source_issue`: the task issue number the work came from.
  - `source_pr`: the PR number that carried the finished work.
  - `source_wave`: the wave record id the work came from, for example `I0030-20261114T184447Z`.
  - `starting_commit`: the 40-hex commit the coder started from.
  - `task_input`: the task input the coder received, verbatim.
  - `spec_refs`: the S# ids the task links, for example `["S1", "S4"]`.
  - `outcome_checks`: one or more check commands a script can run.
  - `acceptance`: the owner's record, `{"accepted_by": "<login>", "accepted_on": "<YYYY-MM-DD>"}`.
- A task with no `acceptance` (or `acceptance: null`) is never run. The assistant proposes it; only the owner's acceptance adds the record. A proposal is still validated: a malformed one is refused, not skipped.

## Rules

- A run counts only accepted tasks. A task with no acceptance record that is well formed reads `not accepted` and is skipped.
- A task missing a required field is refused and the field is named: `refused <file>: missing <field>`. A `starting_commit` that is not 40 hex, an `id` or `accepted_on` in the wrong form, a `spec_refs` entry that is not `S<n>`, or an acceptance record without `accepted_by` or `accepted_on`, or whose `accepted_by` is not a string, is refused the same way (`acceptance.accepted_by must be a string`). A second file with an id already in the set is refused as `duplicate id`.
- The minimum comes from `eval.min_tasks` (default 2) and the maximum from `eval.max_tasks` (default 10), read from `gabo.yaml`. Both must be at least 1 and the minimum must not be above the maximum, as config validation requires. The run is refused with `set below minimum: <accepted> of <min>` when accepted tasks are fewer, and with `set above maximum: <tasks> of <max>` when the set holds more files than the maximum.
- The number of runs per task comes from `eval.runs_per_task` (default 1), which must be at least 1 (`refused config: eval.runs_per_task must be at least 1`).
- The script reads the eval block in block style only (`eval:` then indented keys). An inline `eval: {...}` value, a key with no value, a `--config` path that does not exist, or any value the rules above reject is refused with `refused config: <reason>`. It never falls back to the defaults when it cannot read the config, so the counts it applies are the counts `config.Load` gives.
- A proposal that would add a task beyond `eval.max_tasks` is refused with `maximum reached: <n> of <max> tasks (eval.max_tasks); the proposal is not added`. The owner raises the maximum or removes a task; the assistant never does either.
- Secrets: a task file holding a token or key pattern (GitHub tokens, `sk-` keys, AWS access keys, Slack tokens, private key headers) is refused with `refused <file>: token or key pattern`. The check runs on the raw bytes and on every decoded string and key, so a JSON unicode escape does not hide a token. The matched value is never printed, copied or written anywhere. A task, baseline or report never holds a credential (PRD R21).

## Run

- `eval-run.sh run` runs every accepted task `eval.runs_per_task` times, each run with the coder engine of `runtimes.coder` (`claude`, `codex`, `pi` or `opencode`; any other value is refused). The set check runs first and a refused set starts no engine.
- Each run works in its own `mktemp -d` scratch directory. Its `tree` holds the task's `starting_commit` tree with the PR head's changed configuration applied: the diff from `starting_commit` to `--head`, limited to the configuration paths (`gabo.yaml`, `.gabo`, `.claude`, `AGENTS.md` by default, or the paths named with `--config-path`). Other changed paths are not applied.
- The engine runs in `tree` with the task input as its prompt. Its GitHub token is removed from the environment and `gh` reads an empty config directory of the scratch. It is bounded by `GABO_EVAL_ENGINE_TIMEOUT` seconds (default 600) when `timeout` exists.
- Then each outcome check runs as a script (`bash -c`) in `tree`. Exit 0 reads `pass`, any other exit `fail`. A check the runner did not execute reads `not run (<reason>)`, never `pass`.
- A run passes when the engine exits 0 and every check passes. A task passes only when all of its runs pass. With two runs and one failing it reads `fail (1 of 2 runs)`, and both run lines are kept.
- The report names the engine and `runs per task: <n>`, lists each run with its scratch path, and lists each check. The runner prints every scratch path and never removes one: the parent removes them after use.
- Command: `bash <skill dir>/scripts/eval-run.sh run evals --head <pr head> --config gabo.yaml [--repo <dir>] [--config-path <path>]...`. Exit 0 when every task passes, 1 on a refusal or a failing task, 2 on bad usage.
- The per-engine command lines are not yet probed against a real engine; that probe is a later task of the same intent. The fixture tests use `GABO_EVAL_ENGINE_CMD`, a fake command that the report labels as such.

## Commands

Both validation commands print one line per file and exit 0 when the set may run or the proposal may be added, 1 on any refusal, 2 on bad usage. They write nothing.

- `bash <skill dir>/scripts/eval-set.sh check evals --config gabo.yaml`: validates the whole set. The last line is `set: <accepted> accepted of <tasks> tasks, minimum <min>, maximum <max>` when the set may run.
- `bash <skill dir>/scripts/eval-set.sh propose evals <candidate.json> --config gabo.yaml`: validates one proposed task before it is added to the set for review. A proposal needs no acceptance record yet.
- `bash <skill dir>/scripts/eval-set.sh config <min_tasks|max_tasks|runs_per_task> --config gabo.yaml`: prints one validated count (the runner reads its runs per task this way).

## Hard Rules

- Never write a token, key or credential into a task file, a proposal, a baseline or a report. When a source PR printed a secret, the task is not proposed with that text.
- Never accept a task on the owner's behalf. The owner's acceptance is a human decision.
- The runner never pushes, opens a PR, comments, calls `gh` or writes to the tracker. It writes only inside its scratch directories, and it never runs an engine in the repository.
- Values the owner sets (the pass threshold and the cap) are never written by this skill.
