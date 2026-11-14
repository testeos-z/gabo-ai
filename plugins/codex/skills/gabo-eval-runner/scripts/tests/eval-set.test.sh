#!/usr/bin/env bash
# Fixture tests for eval-set.sh (intent 0030 S1, S2, S11, S12): the task file
# format, the acceptance record, required fields, the minimum and maximum of
# the set, the config refusals and the secret refusal. No engine runs; the
# fixtures are only read.
# Run: bash tests/eval-set.test.sh
set -uo pipefail
here="$(cd "$(dirname "$0")/.." && pwd -P)"
script="$here/eval-set.sh"
work="$(mktemp -d "${TMPDIR:-/tmp}/gabo-eval-set-test.XXXXXX")"
trap 'rm -rf "$work"' EXIT
failures=0
passes=0
commit=1b5c1a57d667744802aec1690bedd7cfe94803c2

ok() { echo "ok    $1"; passes=$((passes + 1)); }
bad() { echo "FAIL  $1"; failures=$((failures + 1)); }

# valid_task: one accepted task body with every required field.
valid_task() {
  jq -n --arg c "$commit" '{
    id: "eval-001", source_issue: 821, source_pr: 849, source_wave: "I0030-20261114T184447Z",
    starting_commit: $c, task_input: "Implement the runner format.", spec_refs: ["S1", "S2"],
    outcome_checks: ["bash checks/accepted.sh"],
    acceptance: {accepted_by: "owner", accepted_on: "2026-11-14"}
  }'
}

# fresh <name>: an empty evals dir under the test scratch, printed.
fresh() {
  mkdir -p "$work/$1/evals"
  echo "$work/$1/evals"
}

min1="$work/min1.yaml"
printf 'eval:\n  min_tasks: 1\n' >"$min1"
max2="$work/max2.yaml"
printf 'eval:\n  min_tasks: 1\n  max_tasks: 2\n' >"$max2"
flow="$work/flow.yaml"
printf 'eval: {min_tasks: 1}\n' >"$flow"
flowmax="$work/flowmax.yaml"
printf 'eval: {max_tasks: 20}\n' >"$flowmax"
zero="$work/zero.yaml"
printf 'eval:\n  min_tasks: 0\n' >"$zero"
inverted="$work/inverted.yaml"
printf 'eval:\n  min_tasks: 5\n  max_tasks: 3\n' >"$inverted"
empty="$work/empty.yaml"
printf 'eval:\n  min_tasks:\n' >"$empty"
missing="$work/missing.yaml"

# expect <label> <want-exit> <want-line> <command...>: runs the command and
# checks its exit status and that one output line equals want-line exactly.
expect() {
  local label="$1" want_exit="$2" want_line="$3"
  shift 3
  local out code
  out="$("$@" 2>&1)"
  code=$?
  if [[ "$code" == "$want_exit" ]] && grep -Fxq -- "$want_line" <<<"$out"; then
    ok "$label"
  else
    bad "$label (exit $code, want $want_exit; want line: $want_line)"
    sed 's/^/      | /' <<<"$out"
  fi
}

# --- S1: the accepted task format -------------------------------------------

e="$(fresh accepted)"
valid_task >"$e/eval-001.json"
expect "S1 accepted task with source issue, PR, wave, commit, S# and an outcome check" 0 "accepted eval-001.json" \
  bash "$script" check "$e" --config "$min1"

e="$(fresh not-accepted)"
valid_task | jq 'del(.acceptance)' >"$e/eval-001.json"
valid_task | jq '.id = "eval-002"' >"$e/eval-002.json"
expect "S1 task with no acceptance record is skipped and reported not accepted" 0 "not accepted eval-001.json" \
  bash "$script" check "$e" --config "$min1"
expect "S1 an accepted sibling still counts when another task is not accepted" 0 "accepted eval-002.json" \
  bash "$script" check "$e" --config "$min1"

for field in id source_issue source_pr source_wave starting_commit task_input spec_refs outcome_checks; do
  e="$(fresh "missing-$field")"
  valid_task | jq --arg f "$field" 'del(.[$f])' >"$e/eval-001.json"
  expect "S1 task missing $field is refused with the field named" 1 "refused eval-001.json: missing $field" \
    bash "$script" check "$e" --config "$min1"
done

e="$(fresh empty-wave)"
valid_task | jq '.source_wave = ""' >"$e/eval-001.json"
expect "S1 an empty source_wave is refused as missing" 1 "refused eval-001.json: missing source_wave" \
  bash "$script" check "$e" --config "$min1"

e="$(fresh missing-accepted-by)"
valid_task | jq 'del(.acceptance.accepted_by)' >"$e/eval-001.json"
expect "S1 acceptance without accepted_by is refused with the field named" 1 \
  "refused eval-001.json: missing acceptance.accepted_by" bash "$script" check "$e" --config "$min1"

# R1-acceptance-type-lax (#821, fixed in I0030/T3): accepted_by must be a string.
i=0
for by in 42 true '["owner"]' '{"login": "owner"}'; do
  i=$((i + 1))
  e="$(fresh accepted-by-type-$i)"
  valid_task | jq --argjson v "$by" '.acceptance.accepted_by = $v' >"$e/eval-001.json"
  expect "S1 acceptance.accepted_by that is not a string is refused by name ($by)" 1 \
    "refused eval-001.json: acceptance.accepted_by must be a string" bash "$script" check "$e" --config "$min1"
done

e="$(fresh bad-date)"
valid_task | jq '.acceptance.accepted_on = "14/11/2026"' >"$e/eval-001.json"
expect "S1 accepted_on that is not YYYY-MM-DD is refused by name" 1 \
  "refused eval-001.json: acceptance.accepted_on must be YYYY-MM-DD" bash "$script" check "$e" --config "$min1"

e="$(fresh bad-id)"
valid_task | jq '.id = "Eval 1"' >"$e/eval-001.json"
expect "S1 an id that is not a lowercase slug is refused by name" 1 \
  "refused eval-001.json: id must be lowercase letters, digits and hyphens, for example eval-001" \
  bash "$script" check "$e" --config "$min1"

e="$(fresh duplicate-id)"
valid_task >"$e/eval-001.json"
valid_task | jq '.id = "eval-001"' >"$e/eval-002.json"
expect "S1 a second file with the same id is refused as a duplicate" 1 \
  "refused eval-002.json: duplicate id eval-001" bash "$script" check "$e" --config "$min1"

e="$(fresh unaccepted-malformed)"
valid_task | jq 'del(.acceptance) | del(.starting_commit)' >"$e/eval-001.json"
expect "S1 a proposal with no acceptance but a missing field is refused, not skipped" 1 \
  "refused eval-001.json: missing starting_commit" bash "$script" check "$e" --config "$min1"

e="$(fresh bad-commit)"
valid_task | jq '.starting_commit = "main"' >"$e/eval-001.json"
expect "S1 starting_commit that is not a 40-hex commit is refused by name" 1 \
  "refused eval-001.json: starting_commit is not a 40-hex commit" bash "$script" check "$e" --config "$min1"

e="$(fresh bad-spec)"
valid_task | jq '.spec_refs = ["X1"]' >"$e/eval-001.json"
expect "S1 spec_refs that are not S# ids are refused by name" 1 \
  "refused eval-001.json: spec_refs must be S<n> ids" bash "$script" check "$e" --config "$min1"

e="$(fresh bad-json)"
echo '{"id": "eval-001",' >"$e/eval-001.json"
expect "S1 a file that is not JSON is refused" 1 "refused eval-001.json: not valid JSON" \
  bash "$script" check "$e" --config "$min1"

# --- S2: the minimum and the maximum of the set -----------------------------

e="$(fresh below-minimum)"
valid_task >"$e/eval-001.json"
expect "S2 one accepted task under the default minimum of 2" 1 "set below minimum: 1 of 2" \
  bash "$script" check "$e"

expect "S2 eval.min_tasks read from gabo.yaml lowers the minimum" 0 \
  "set: 1 accepted of 1 tasks, minimum 1, maximum 10" bash "$script" check "$e" --config "$min1"

e="$(fresh empty-set)"
expect "S2 an empty set is below the minimum" 1 "set below minimum: 0 of 2" bash "$script" check "$e"

e="$(fresh proposal-at-maximum)"
for n in 1 2; do valid_task | jq --arg n "$n" '.id = "eval-00\($n)"' >"$e/eval-00$n.json"; done
valid_task | jq '.id = "eval-003"' >"$work/candidate-over.json"
expect "S2 a proposal that would pass eval.max_tasks is refused and says the maximum is reached" 1 \
  "maximum reached: 2 of 2 tasks (eval.max_tasks); the proposal is not added" \
  bash "$script" propose "$e" "$work/candidate-over.json" --config "$max2"

e="$(fresh proposal-below-maximum)"
valid_task >"$e/eval-001.json"
valid_task | jq '.id = "eval-002"' >"$work/candidate-under.json"
expect "S2 a proposal below eval.max_tasks is validated and accepted for review" 0 \
  "proposal ok: candidate-under.json" bash "$script" propose "$e" "$work/candidate-under.json" --config "$max2"

e="$(fresh proposal-duplicate)"
valid_task >"$e/eval-001.json"
valid_task >"$work/candidate-dup.json"
expect "S2 a proposal that reuses an id in the set is refused by name" 1 \
  "refused candidate-dup.json: duplicate id eval-001" bash "$script" propose "$e" "$work/candidate-dup.json"

e="$(fresh above-maximum)"
for n in 1 2 3; do valid_task | jq --arg n "$n" '.id = "eval-00\($n)"' >"$e/eval-00$n.json"; done
expect "S2 a set above eval.max_tasks is refused" 1 "set above maximum: 3 of 2" \
  bash "$script" check "$e" --config "$max2"

e="$(fresh flow-style)"
valid_task >"$e/eval-001.json"
expect "S2 a flow-style eval block is refused, not read as the defaults" 1 \
  "refused config: eval must be a block (eval: then indented keys), not an inline value" \
  bash "$script" check "$e" --config "$flow"
expect "S2 a flow-style eval.max_tasks is refused, not read as the default of 10" 1 \
  "refused config: eval must be a block (eval: then indented keys), not an inline value" \
  bash "$script" check "$e" --config "$flowmax"

expect "S2 eval.min_tasks of 0 is refused (the loader requires at least 1)" 1 \
  "refused config: eval.min_tasks must be at least 1" bash "$script" check "$e" --config "$zero"
expect "S2 eval.min_tasks above eval.max_tasks is refused" 1 \
  "refused config: eval.min_tasks is above eval.max_tasks" bash "$script" check "$e" --config "$inverted"
expect "S2 an eval.min_tasks key with no value is refused" 1 \
  "refused config: eval.min_tasks is not a whole number" bash "$script" check "$e" --config "$empty"
expect "S2 a --config path that does not exist is refused" 1 \
  "refused config: $missing is not a file" bash "$script" check "$e" --config "$missing"

# --- S11: no credential or token in a task file -----------------------------

e="$(fresh token)"
valid_task | jq '.task_input = "use ghp_0123456789abcdefghijklmnopqrstuvwxyz"' >"$e/eval-001.json"
out="$(bash "$script" check "$e" --config "$min1" 2>&1)"
code=$?
if [[ "$code" == 1 ]] && grep -Fxq "refused eval-001.json: token or key pattern" <<<"$out" && ! grep -q "ghp_" <<<"$out"; then
  ok "S11 a task holding a GitHub token is refused and the value is not printed"
else
  bad "S11 a task holding a GitHub token is refused and the value is not printed (exit $code)"
  sed 's/^/      | /' <<<"$out"
fi

e="$(fresh escaped-token)"
valid_task | sed 's/Implement the runner format./\\u0067hp_0123456789abcdefghijklmnopqrstuvwxyz/' >"$e/eval-001.json"
out="$(bash "$script" check "$e" --config "$min1" 2>&1)"
code=$?
if [[ "$code" == 1 ]] && grep -Fxq "refused eval-001.json: token or key pattern" <<<"$out" && ! grep -q "hp_0123" <<<"$out"; then
  ok "S11 a token written with a JSON unicode escape is refused and the value is not printed"
else
  bad "S11 a token written with a JSON unicode escape is refused and the value is not printed (exit $code)"
  sed 's/^/      | /' <<<"$out"
fi

e="$(fresh key)"
valid_task | jq '.task_input = "-----BEGIN OPENSSH PRIVATE KEY-----"' >"$e/eval-001.json"
expect "S11 a task holding a private key header is refused" 1 "refused eval-001.json: token or key pattern" \
  bash "$script" check "$e" --config "$min1"

e="$(fresh propose-token)"
valid_task | jq '.task_input = "key sk-ant-api03-0123456789abcdefghijkl"' >"$work/token-candidate.json"
expect "S11 a proposal holding a token pattern is refused" 1 \
  "refused token-candidate.json: token or key pattern" bash "$script" propose "$e" "$work/token-candidate.json"

# --- S12: the check reads the set and changes no file in it -----------------

e="$(fresh read-only)"
valid_task >"$e/eval-001.json"
valid_task | jq '.id = "eval-002"' >"$e/eval-002.json"
before="$(cd "$e" && find . -type f -exec cksum {} + | sort)"
bash "$script" check "$e" --config "$min1" >/dev/null 2>&1
after="$(cd "$e" && find . -type f -exec cksum {} + | sort)"
if [[ "$before" == "$after" ]]; then
  ok "S12 check changes no task file"
else
  bad "S12 check changes no task file"
fi

# --- S4: the run count read from eval.runs_per_task -------------------------

runs3="$work/runs3.yaml"
printf 'eval:\n  runs_per_task: 3\n' >"$runs3"
runs0="$work/runs0.yaml"
printf 'eval:\n  runs_per_task: 0\n' >"$runs0"

expect "S4 config runs_per_task defaults to one run" 0 "1" bash "$script" config runs_per_task --config "$min1"
expect "S4 config runs_per_task reads eval.runs_per_task from gabo.yaml" 0 "3" bash "$script" config runs_per_task --config "$runs3"
expect "S4 config runs_per_task of 0 is refused (the loader requires at least 1)" 1 \
  "refused config: eval.runs_per_task must be at least 1" bash "$script" config runs_per_task --config "$runs0"

# --- S8: the caps are read from eval.cap; unset is a state of its own -------

cap_turns="$work/cap-turns.yaml"
printf 'eval:\n  cap:\n    turns_per_task_run: 4\n' >"$cap_turns"
cap_runs="$work/cap-runs.yaml"
printf 'eval:\n  cap:\n    task_runs_per_eval_run: 6\n' >"$cap_runs"
cap_zero="$work/cap-zero.yaml"
printf 'eval:\n  cap:\n    turns_per_task_run: 0\n' >"$cap_zero"
cap_inline="$work/cap-inline.yaml"
printf 'eval:\n  cap: {turns_per_task_run: 2}\n' >"$cap_inline"
cap_null="$work/cap-null.yaml"
printf 'eval:\n  cap:\n    turns_per_task_run: null\n' >"$cap_null"

expect "S8 config turns_per_task_run reads eval.cap.turns_per_task_run" 0 "4" \
  bash "$script" config turns_per_task_run --config "$cap_turns"
expect "S8 config task_runs_per_eval_run reads eval.cap.task_runs_per_eval_run" 0 "6" \
  bash "$script" config task_runs_per_eval_run --config "$cap_runs"
expect "S8 a cap of 0 is refused (the loader requires at least 1)" 1 \
  "refused config: eval.cap.turns_per_task_run must be at least 1" bash "$script" config turns_per_task_run --config "$cap_zero"
expect "S8 an inline eval.cap value is refused (block style only)" 1 \
  "refused config: eval.cap must be a block (eval.cap then indented keys), not an inline value" bash "$script" check "$(fresh cap-inline)" --config "$cap_inline"

bash "$script" config turns_per_task_run --config "$min1" >"$work/unset.out" 2>&1
code=$?
if [[ "$code" == 0 && ! -s "$work/unset.out" ]]; then
  ok "S8 an unset cap prints nothing, not even a newline (no number is written by the script)"
else
  bad "S8 an unset cap prints nothing, not even a newline (exit $code, bytes $(wc -c <"$work/unset.out"))"
fi
bash "$script" config turns_per_task_run --config "$cap_null" >"$work/null.out" 2>&1
code=$?
if [[ "$code" == 0 && ! -s "$work/null.out" ]]; then
  ok "S8 a null cap reads as not set, as config.Load reads it"
else
  bad "S8 a null cap reads as not set (exit $code, got: $(cat "$work/null.out"))"
fi

# S8 any indentation width is read the way config.Load reads it (a cap is never dropped)
wide="$work/cap-wide.yaml"
printf 'eval:\n    min_tasks: 1\n    cap:\n        turns_per_task_run: 5\n' >"$wide"
expect "S8 a cap block indented four and eight spaces is read" 0 "5" \
  bash "$script" config turns_per_task_run --config "$wide"
wideinline="$work/cap-wide-inline.yaml"
printf 'eval:\n    cap: {turns_per_task_run: 2}\n' >"$wideinline"
expect "S8 an inline cap under a four-space eval block is refused" 1 \
  "refused config: eval.cap must be a block (eval.cap then indented keys), not an inline value" \
  bash "$script" config turns_per_task_run --config "$wideinline"

# --- S9 config: the pass threshold is read from eval.pass_threshold (S7) -----

thr="$work/threshold.yaml"
printf 'eval:\n  pass_threshold: 0.8\n' >"$thr"
thr_none="$work/threshold-none.yaml"
printf 'eval:\n  min_tasks: 1\n' >"$thr_none"
thr_null="$work/threshold-null.yaml"
printf 'eval:\n  pass_threshold: null\n' >"$thr_null"
thr_high="$work/threshold-high.yaml"
printf 'eval:\n  pass_threshold: 1.5\n' >"$thr_high"
thr_neg="$work/threshold-neg.yaml"
printf 'eval:\n  pass_threshold: -0.1\n' >"$thr_neg"
thr_word="$work/threshold-word.yaml"
printf 'eval:\n  pass_threshold: high\n' >"$thr_word"

expect "S7 config pass_threshold reads eval.pass_threshold" 0 "0.8" \
  bash "$script" config pass_threshold --config "$thr"
thr_one="$work/threshold-one.yaml"
printf 'eval:\n  pass_threshold: 1\n' >"$thr_one"
expect "S7 config pass_threshold of 1 is accepted (the upper bound)" 0 "1" \
  bash "$script" config pass_threshold --config "$thr_one"
for t in thr_none thr_null; do
  bash "$script" config pass_threshold --config "${!t}" >"$work/$t.out" 2>&1
  if [[ $? == 0 && ! -s "$work/$t.out" ]]; then
    ok "S7 an unset pass_threshold prints nothing ($t)"
  else
    bad "S7 an unset pass_threshold prints nothing ($t)"
  fi
done
for t in thr_high thr_neg thr_word; do
  expect "S7 a pass_threshold outside 0 to 1 or not a number is refused ($t)" 1 \
    "refused config: eval.pass_threshold must be from 0 to 1" bash "$script" config pass_threshold --config "${!t}"
done
expect "S7 check refuses the same bad pass_threshold before any count" 1 \
  "refused config: eval.pass_threshold must be from 0 to 1" bash "$script" check "$(fresh threshold-check)" --config "$thr_high"

echo "passes: $passes failures: $failures"
[[ "$failures" == 0 ]]
