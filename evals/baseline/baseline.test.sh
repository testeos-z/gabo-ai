#!/usr/bin/env bash
# Fixture-free checks of the committed baseline (intent 0030 S6, D2): the file
# the owner accepted holds the commit the first run was taken on, one pass or
# fail per accepted task of the set, and the owner's acceptance. Run from any
# directory: bash evals/baseline/baseline.test.sh
set -u

root="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
evals="$root/evals"
baseline="$evals/baseline/baseline.json"
passes=0
failures=0
ok() { echo "ok    $1"; passes=$((passes + 1)); }
bad() { echo "FAIL  $1"; failures=$((failures + 1)); }

# 1. The baseline file exists and is valid JSON with an object of tasks.
if [[ -f "$baseline" ]] && jq -e 'type == "object" and (.tasks | type) == "object"' "$baseline" >/dev/null 2>&1; then
  ok "S6 the baseline is a JSON object with a tasks object"
else
  bad "S6 the baseline is a JSON object with a tasks object ($baseline)"
  echo "passes: $passes failures: $failures"
  exit 1
fi

# 2. The commit is a 40-hex commit that exists in the repository.
commit="$(jq -r '.commit // ""' "$baseline")"
if [[ "$commit" =~ ^[0-9a-f]{40}$ ]] && git -C "$root" cat-file -e "$commit^{commit}" 2>/dev/null; then
  ok "S6 the baseline commit is a 40-hex commit of this repository"
else
  bad "S6 the baseline commit is a 40-hex commit of this repository"
fi

# 3. Exactly the accepted tasks of the set, each with pass or fail.
set_ids="$(for f in "$evals"/*.json; do jq -r '.id' "$f"; done | sort)"
base_ids="$(jq -r '.tasks | keys[]' "$baseline" | sort)"
if [[ -n "$set_ids" && "$set_ids" == "$base_ids" ]]; then
  ok "S6 the baseline names every task of the set and no other"
else
  bad "S6 the baseline names every task of the set and no other (set: $(tr '\n' ' ' <<<"$set_ids") base: $(tr '\n' ' ' <<<"$base_ids"))"
fi
if jq -e '[.tasks[]] | all(. == "pass" or . == "fail")' "$baseline" >/dev/null 2>&1; then
  ok "S6 every baseline task reads pass or fail"
else
  bad "S6 every baseline task reads pass or fail"
fi

# 4. The owner's acceptance is recorded: the date and the text relayed to the assistant.
if [[ "$(jq -r '.accepted_on // ""' "$baseline")" == "2026-11-15" ]] &&
  [[ "$(jq -r '.acceptance // ""' "$baseline")" == "accepted by the owner 2026-11-15, relayed by the gabo-b assistant" ]]; then
  ok "S6 the baseline records the owner's acceptance of 2026-11-15"
else
  bad "S6 the baseline records the owner's acceptance of 2026-11-15"
fi

# 5. Each accepted task of the set carries the owner's acceptance record.
unaccepted=0
for f in "$evals"/*.json; do
  jq -e '(.acceptance | type) == "object" and (.acceptance.accepted_by | type) == "string" and (.acceptance.accepted_on | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}$"))' "$f" >/dev/null 2>&1 || unaccepted=$((unaccepted + 1))
done
if [[ "$unaccepted" == 0 ]]; then
  ok "S6 every task of the set carries the owner's acceptance record"
else
  bad "S6 every task of the set carries the owner's acceptance record ($unaccepted without)"
fi

echo "passes: $passes failures: $failures"
[[ "$failures" == 0 ]]
