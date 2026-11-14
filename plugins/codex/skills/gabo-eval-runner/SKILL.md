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
- The engine runs in `tree` with the task input as its prompt. The GitHub tokens (github.com and enterprise) and `SSH_AUTH_SOCK` are removed from its environment, and `gh` reads an empty config directory of the scratch. It is bounded by `GABO_EVAL_ENGINE_TIMEOUT` seconds (default 600) when `timeout` exists.
- Not a sandbox: the engine keeps the owner's `HOME`, its login and the filesystem, and the head's `.claude` hooks run with the owner's privileges. Run the set only on the owner's machine, for a configuration the owner is reviewing (intent 0030 S3).
- Then each outcome check runs as a script (`bash -c`) in `tree`. Exit 0 reads `pass`, any other exit `fail`. A check the runner did not execute reads `not run (<reason>)`, never `pass`.
- A run passes when the engine exits 0, every check passes and the coder's report holds the method evidence (see Method evidence). A task passes only when all of its runs pass. With two runs and one failing it reads `fail (1 of 2 runs)`, both run lines are kept, and the task line names the reason of the first failing run.
- The report names the engine and `runs per task: <n>`, lists each run with its scratch path, and lists each check. The runner prints every scratch path and never removes one: the parent removes them after use.
- Command: `bash <skill dir>/scripts/eval-run.sh run evals --head <pr head> --config gabo.yaml [--repo <dir>] [--config-path <path>]...`. Exit 0 when every task passes, 1 on a refusal or a failing task, 2 on bad usage, 3 when a cap paused the run and no task failed.
- The per-engine command lines are not yet probed against a real engine; that probe is a later task of the same intent. The fixture tests use `GABO_EVAL_ENGINE_CMD`, a fake command that the report labels as such.

## Caps (intent 0030 S8, under the rule of intent 0020)

- Two optional caps, set by the owner by hand in `gabo.yaml` under `eval.cap`: `eval.cap.turns_per_task_run` (the turns one task run may take) and `eval.cap.task_runs_per_eval_run` (the task runs one eval run may start). Each has no default: unset, or `null`, means no cap, and the report reads `cap: none set` when neither is set. A value must be a whole number of at least 1; an inline `eval.cap` value is refused. The runner writes no cap value.
- Enforcement follows intent 0020 (S7, D2). A turn cap is enforced by the engine where the engine documents one: claude gets `--max-turns`. Other engines get the limit in their prompt (`Turn limit for this run: <n> turns`), an advisory cap. The report says so per engine and says that gabo does not cap spend. `eval.cap.task_runs_per_eval_run` is enforced by the runner: it counts task runs started, and the next one is not started once the count reaches the cap.
- A cap that fires pauses the run. The task it cut, and every task after it, reads `not run: cap <name>`, never `fail` or `pass`. A task the cap stops before its first run reads only the `not run` line. A run that failed before the cut still makes the run exit 1, and the cut task's line says so. The report reads `status: paused by cap <name>`, gives no pass rate for the set, and ends with `continue or stop?`. The runner never continues by itself: a continue is a new run that the owner starts.
- A turn-cap stop is recognized only for claude: a line of its log that starts with `Error: Reached max turns` (its print-mode message). Any other engine that prints the words is a failing run. The real claude phrase and the advisory engines' compliance are probed by I0030/T17; no engine runs in the fixture tests.

## Method evidence

- The coder's report is its engine output. A run passes only when that report holds the fields of the report gate of intent 0027 (S19), judged as `gabo-cycle` step 6 judges them: `odd:` (RED and GREEN, or the exception), one `spec S<n>:` line for each of the task's `spec_refs` (covered or not covered), `RDD:`, `contract:`, `tests:`, `integration:`, `commits:` and `verifier:`.
- A missing field reads `fail: <field> line missing`, for example `fail: RDD line missing` or `fail: spec S5 line missing`. A field present as `not run: <reason>` or `not applicable: <reason>` counts as present: the rule asks for the line, not for a pass.
- A failing outcome check with complete evidence reads `fail: check <cmd> failed (exit <n>)` with the observed exit code. The checks are judged before the report, so a failing check gives its own reason.
- A line counts with or without a list marker and backticks, as a coder writes it. The field list is pinned to the gate text by a Go test (`evalreport_test.go`), so the runner and the gate cannot drift.

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
