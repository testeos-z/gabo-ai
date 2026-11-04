#!/usr/bin/env bash
# Tests for the Codex copy of gabo-check.sh run as `gabo-check.sh codex`, the
# Codex plugin's PostToolUse apply_patch hook (intent 0002 S7, S8, S9, D5;
# plan T7). A fake gabo stands on PATH and records its argv and working
# directory outside the project; every case runs in a scratch gabo project of
# its own under one mktemp -d. The inputs follow the Codex PostToolUse shape
# (https://learn.chatgpt.com/docs/hooks): the patch text is in
# tool_input.command, with no file-path field.
# Run: bash plugins/codex/scripts/tests/gabo-check.test.sh
set -uo pipefail

script="$(cd "$(dirname "$0")/.." && pwd -P)/gabo-check.sh"
bash_bin="$(command -v bash)"
failures=0
passes=0
work="$(mktemp -d "${TMPDIR:-/tmp}/gabo-codex-check-test.XXXXXX")"
trap 'rm -rf "$work"' EXIT

expect() {
  if [[ "$2" == "$3" ]]; then
    echo "ok    $1"
    passes=$((passes + 1))
  else
    echo "FAIL  $1"
    echo "      want: $(printf '%q' "$3")"
    echo "      got:  $(printf '%q' "$2")"
    failures=$((failures + 1))
  fi
}

# Fake gabo: appends "<cwd> <argv>" to $FAKE_GABO_LOG, prints $FAKE_GABO_OUT
# to stdout (as the real gabo check does) and exits $FAKE_GABO_EXIT.
fakebin="$work/bin"
mkdir -p "$fakebin" "$work/empty"
cat >"$fakebin/gabo" <<'GABO'
#!/usr/bin/env bash
echo "$PWD $*" >>"$FAKE_GABO_LOG"
[[ -n "${FAKE_GABO_OUT:-}" ]] && printf '%s\n' "$FAKE_GABO_OUT"
exit "${FAKE_GABO_EXIT:-0}"
GABO
chmod +x "$fakebin/gabo"

# new_project <name> [no-version]: a gabo project with an intent 0009-x.
new_project() {
  local p="$work/$1"
  mkdir -p "$p/docs/product/intents/0009-x" "$p/src"
  [[ "${2:-}" == no-version ]] || { mkdir -p "$p/.gabo" && echo 0.1.0 >"$p/.gabo/VERSION"; }
  printf '# plan\n' >"$p/docs/product/intents/0009-x/plan.md"
  printf '# spec\n' >"$p/docs/product/intents/0009-x/spec.md"
  printf 'x\n' >"$p/src/main.go"
  echo "$p"
}

# snapshot <dir>: every path with its size and mtime, to prove the hook
# writes nothing.
snapshot() { (cd "$1" && find . -printf '%p %s %T@\n' | sort); }

# json_string <text>: <text> as a JSON string literal, escaped the way
# Codex's serializer does (backslash, quote, newline, carriage return, tab).
json_string() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '"%s"' "$s"
}

# codex_input <project> <patch>: a recorded-shape Codex PostToolUse input for
# an apply_patch call whose tool_input.command is <patch>. tool_response
# comes first and holds a "command" key of its own, so only the tool input's
# command may count.
codex_input() {
  printf '{"session_id":"s1","transcript_path":null,"cwd":"%s","hook_event_name":"PostToolUse","model":"gpt-5","turn_id":"t1","permission_mode":"default","tool_name":"apply_patch","tool_use_id":"call_1","tool_response":{"command":"*** Begin Patch\\n*** Update File: gabo.yaml\\n*** End Patch","output":"Success."},"tool_input":{"command":%s}}' \
    "$1" "$(json_string "$2")"
}

# patch_for <path>...: an apply_patch text that updates each <path>.
patch_for() {
  local p
  printf '*** Begin Patch\n'
  for p in "$@"; do
    printf '*** Update File: %s\n@@\n-a "quoted" \\ line\n+b\tline\n' "$p"
  done
  printf '*** End Patch\n'
}

# run_hook <project> <json> [env...]: feeds <json> to `gabo-check.sh codex`
# from the project root (Codex runs hooks in the session cwd and sets no
# CLAUDE_PROJECT_DIR) and sets out, err, code, calls.
run_hook() {
  local project="$1" json="$2"
  shift 2
  local log="$work/gabo.log"
  : >"$log"
  local before after
  before=$(snapshot "$project")
  out=$(cd "$project" && env -i HOME="$work" PATH="$fakebin:/usr/bin:/bin" \
    FAKE_GABO_LOG="$log" "$@" \
    "$bash_bin" "$script" codex <<<"$json" 2>"$work/stderr")
  code=$?
  err=$(cat "$work/stderr")
  calls=$(cat "$log")
  after=$(snapshot "$project")
  [[ "$before" == "$after" ]] || { echo "FAIL  hook changed $project"; failures=$((failures + 1)); }
}

fail_line='✗ docs/product/intents/0009-x/plan.md: spec.md is not accepted (Acceptance table has no filled row)'
plan=docs/product/intents/0009-x/plan.md

# S7: a patch updating an intent plan.md whose spec is unaccepted -> exit 0,
# Codex JSON with gabo's line as hookSpecificOutput.additionalContext.
p=$(new_project failing)
run_hook "$p" "$(codex_input "$p" "$(patch_for "$plan")")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="$fail_line"
expect "S7 failing check exits 0" "$code" 0
expect "S7 failing check prints the Codex additionalContext JSON" "$out" \
  "{\"hookSpecificOutput\":{\"hookEventName\":\"PostToolUse\",\"additionalContext\":$(json_string "$fail_line")}}"
expect "S7 failing check is silent on stderr" "$err" ""
expect "S7 runs gabo check once in the project root" "$calls" "$p check"

# Several gabo lines, with quotes and backslashes, stay one valid JSON string.
two=$'✗ a "quoted" path\\x: one\n✗ b\tsecond'
run_hook "$p" "$(codex_input "$p" "$(patch_for "$plan")")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="$two"
expect "S7 multi-line output is escaped as one JSON string" "$out" \
  "{\"hookSpecificOutput\":{\"hookEventName\":\"PostToolUse\",\"additionalContext\":$(json_string "$two")}}"
if command -v python3 >/dev/null; then
  ctx=$(printf '%s' "$out" | python3 -c 'import json,sys; print(json.load(sys.stdin)["hookSpecificOutput"]["additionalContext"], end="")')
  expect "S7 output parses as JSON and round-trips gabo's lines" "$ctx" "$two"
fi

# S7: passing project -> exit 0, silent.
p=$(new_project passing)
run_hook "$p" "$(codex_input "$p" "$(patch_for "$plan")")" FAKE_GABO_EXIT=0
expect "S7 passing check is silent" "$code:$out:$err" "0::"
expect "S7 passing check ran gabo check" "$calls" "$p check"

# Every header kind and every gabo path triggers the check, relative or
# absolute, also when a non-gabo file comes first in the patch.
for rel in gabo.yaml .gabo/VERSION .claude/agents/x.md .claude/skills/x/SKILL.md docs/product/prd.md; do
  run_hook "$p" "$(codex_input "$p" "$(patch_for src/main.go "$rel")")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
  expect "gabo path $rel runs the check" "$code:$calls" "0:$p check"
done
run_hook "$p" "$(codex_input "$p" "$(patch_for "$p/$plan")")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "an absolute gabo path runs the check" "$calls" "$p check"
add=$(printf '*** Begin Patch\r\n*** Add File: %s\r\n+# plan\r\n*** End Patch\r\n' "$plan")
run_hook "$p" "$(codex_input "$p" "$add")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "an Add File header with CRLF lines runs the check" "$calls" "$p check"
mv_patch=$(printf '*** Begin Patch\n*** Update File: src/old.md\n*** Move to: %s\n@@\n-a\n+b\n*** End Patch\n' "$plan")
run_hook "$p" "$(codex_input "$p" "$mv_patch")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "a Move to gabo path runs the check" "$calls" "$p check"
del=$(printf '*** Begin Patch\n*** Delete File: docs/product/intents/0009-x/spec.md\n*** End Patch\n')
run_hook "$p" "$(codex_input "$p" "$del")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "a Delete File of a gabo path runs the check" "$calls" "$p check"

# S8: the four silent cases, each exit 0 with no output.
p=$(new_project quiet)
run_hook "$p" "$(codex_input "$p" "$(patch_for src/main.go)")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S8 non-gabo path is silent and calls no gabo" "$code:$out:$err:$calls" "0:::"
run_hook "$p" "$(codex_input "$p" "$(patch_for .claude/settings.json)")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S8 other .claude file is silent" "$code:$out:$err:$calls" "0:::"
run_hook "$p" "$(codex_input "$p" "$(patch_for "$work/other/docs/product/prd.md")")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S8 gabo-like path outside the project is silent" "$code:$out:$err:$calls" "0:::"
body=$(printf '*** Begin Patch\n*** Update File: src/main.go\n@@\n-x\n+*** Update File: gabo.yaml\n*** End Patch\n')
run_hook "$p" "$(codex_input "$p" "$body")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S8 a header-like added line is not a header" "$code:$out:$err:$calls" "0:::"

nv=$(new_project noversion no-version)
run_hook "$nv" "$(codex_input "$nv" "$(patch_for "$plan")")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S8 no .gabo/VERSION is silent" "$code:$out:$err:$calls" "0:::"

run_hook "$p" "$(codex_input "$p" "$(patch_for "$plan")")" PATH="$work/empty"
expect "S8 no gabo on PATH is silent" "$code:$out:$err" "0::"

run_hook "$p" "$(codex_input "$p" "$(patch_for "$plan")")" FAKE_GABO_EXIT=2 FAKE_GABO_OUT="usage: gabo check"
expect "S8 gabo exit 2 is silent" "$code:$out:$err" "0::"
expect "S8 gabo exit 2 still ran gabo check" "$calls" "$p check"

# Malformed, non-patch or empty input never blocks.
run_hook "$p" '{"tool_name":"apply_patch","tool_input":{}}' FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "input without tool_input.command is silent" "$code:$out:$err:$calls" "0:::"
run_hook "$p" 'not json' FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "malformed input is silent" "$code:$out:$err:$calls" "0:::"
log="$work/gabo.log"; : >"$log"
out=$(cd "$p" && env -i PATH="$fakebin:/usr/bin:/bin" FAKE_GABO_LOG="$log" \
  "$bash_bin" "$script" codex </dev/null 2>"$work/stderr"); code=$?
expect "empty stdin is silent" "$code:$out:$(cat "$work/stderr"):$(cat "$log")" "0:::"

# A large patch is read in bounded time.
big=$(printf '*** Begin Patch\n*** Update File: src/main.go\n@@\n'; for _ in $(seq 4000); do printf '+line with "quotes" and \\ slashes\n'; done; printf '*** Update File: %s\n@@\n-a\n+b\n*** End Patch\n' "$plan")
SECONDS=0
run_hook "$p" "$(codex_input "$p" "$big")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "a large patch with a gabo path late runs the check" "$calls" "$p check"
(( SECONDS < 10 )) && expect "a large patch is read in under 10s" ok ok || expect "a large patch is read in under 10s" "${SECONDS}s" "<10s"

# S9: the script holds no write and no rule of its own (the hook runs above
# also prove it leaves the project unchanged).
if [[ ! -f "$script" ]] || grep -nE '>[^&]|\btee\b|\bmkdir\b|\brm\b|\bmv\b|\bcp\b|\btouch\b|Acceptance|plan\.md' "$script" | grep -v '^[0-9]*:#' >/dev/null; then
  echo "FAIL  S9 script holds a write or a rule"
  [[ -f "$script" ]] && grep -nE '>[^&]|\btee\b|\bmkdir\b|\brm\b|\bmv\b|\bcp\b|\btouch\b|Acceptance|plan\.md' "$script" | grep -v '^[0-9]*:#'
  failures=$((failures + 1))
else
  echo "ok    S9 script holds no write and no rule"
  passes=$((passes + 1))
fi

echo "passes=$passes failures=$failures"
[[ $failures -eq 0 ]]
