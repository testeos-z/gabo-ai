#!/usr/bin/env bash
# Fixture tests for eval-set.sh (intent 0030 S1, S2, S11, S12): the task file
# format, the acceptance record, required fields, the minimum and maximum of
# the set and the secret refusal. No engine runs; the fixtures are only read.
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
    id: "eval-001", source_issue: 821, source_pr: 849, starting_commit: $c,
    task_input: "Implement the runner format.", spec_refs: ["S1", "S2"],
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
expect "S1 accepted task with source issue, PR, commit, S# and an outcome check" 0 "accepted eval-001.json" \
  bash "$script" check "$e" --config "$min1"

e="$(fresh not-accepted)"
valid_task | jq 'del(.acceptance)' >"$e/eval-001.json"
valid_task | jq '.id = "eval-002"' >"$e/eval-002.json"
expect "S1 task with no acceptance record is skipped and reported not accepted" 0 "not accepted eval-001.json" \
  bash "$script" check "$e" --config "$min1"
expect "S1 an accepted sibling still counts when another task is not accepted" 0 "accepted eval-002.json" \
  bash "$script" check "$e" --config "$min1"

for field in id source_issue source_pr starting_commit task_input spec_refs outcome_checks; do
  e="$(fresh "missing-$field")"
  valid_task | jq --arg f "$field" 'del(.[$f])' >"$e/eval-001.json"
  expect "S1 task missing $field is refused with the field named" 1 "refused eval-001.json: missing $field" \
    bash "$script" check "$e" --config "$min1"
done

e="$(fresh missing-accepted-by)"
valid_task | jq 'del(.acceptance.accepted_by)' >"$e/eval-001.json"
expect "S1 acceptance without accepted_by is refused with the field named" 1 \
  "refused eval-001.json: missing acceptance.accepted_by" bash "$script" check "$e" --config "$min1"

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

e="$(fresh above-maximum)"
for n in 1 2 3; do valid_task | jq --arg n "$n" '.id = "eval-00\($n)"' >"$e/eval-00$n.json"; done
expect "S2 a set above eval.max_tasks is refused" 1 "set above maximum: 3 of 2" \
  bash "$script" check "$e" --config "$max2"

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

echo "passes: $passes failures: $failures"
[[ "$failures" == 0 ]]
