#!/usr/bin/env bash
# Tests for gabo-check.sh, the plugin's PostToolUse Write|Edit hook
# (intent 0001 S6, S7, S8, D9, D10; plan T7). A fake gabo stands on PATH and
# records its argv and working directory outside the project; every case runs
# in a scratch gabo project of its own under one mktemp -d.
# Run: bash plugins/claude-code/scripts/tests/gabo-check.test.sh
set -uo pipefail

script="$(cd "$(dirname "$0")/.." && pwd -P)/gabo-check.sh"
bash_bin="$(command -v bash)"
failures=0
passes=0
work="$(mktemp -d "${TMPDIR:-/tmp}/gabo-plugin-check-test.XXXXXX")"
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

# run_hook <project> <file_path> [env...]: feeds a Claude Code PostToolUse
# JSON for a Write of <file_path> and sets out, err, code, calls.
run_hook() {
  local project="$1" file="$2"
  shift 2
  local log="$work/gabo.log"
  : >"$log"
  local json
  json=$(printf '{"session_id":"s1","cwd":"%s","hook_event_name":"PostToolUse","tool_name":"Write","tool_input":{"file_path":"%s","content":"say \\"file_path\\": \\"gabo.yaml\\""},"tool_response":{"filePath":"%s","success":true}}' "$project" "$file" "$file")
  local before after
  before=$(snapshot "$project")
  out=$(cd "$project" && env -i HOME="$work" PATH="$fakebin:/usr/bin:/bin" \
    CLAUDE_PROJECT_DIR="$project" FAKE_GABO_LOG="$log" "$@" \
    "$bash_bin" "$script" <<<"$json" 2>"$work/stderr")
  code=$?
  err=$(cat "$work/stderr")
  calls=$(cat "$log")
  after=$(snapshot "$project")
  [[ "$before" == "$after" ]] || { echo "FAIL  hook changed $project"; failures=$((failures + 1)); }
}

fail_line='✗ docs/product/intents/0009-x/plan.md: spec.md is not accepted (Acceptance table has no filled row)'

# S6: failing gabo check on a gabo file -> exit 2, gabo's lines on stderr.
p=$(new_project failing)
run_hook "$p" "$p/docs/product/intents/0009-x/plan.md" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="$fail_line"
expect "S6 failing check exits 2" "$code" 2
expect "S6 failing check relays gabo's line on stderr" "$err" "$fail_line"
expect "S6 failing check prints nothing on stdout" "$out" ""
expect "S6 runs gabo check once in the project root" "$calls" "$p check"

# S6: passing project -> exit 0, silent.
p=$(new_project passing)
run_hook "$p" "$p/docs/product/intents/0009-x/plan.md" FAKE_GABO_EXIT=0 FAKE_GABO_OUT=""
expect "S6 passing check exits 0" "$code" 0
expect "S6 passing check is silent on stderr" "$err" ""
expect "S6 passing check is silent on stdout" "$out" ""
expect "S6 passing check ran gabo check" "$calls" "$p check"

# Each gabo path triggers the check, absolute or relative to the root.
for rel in gabo.yaml .gabo/VERSION .claude/agents/x.md .claude/skills/x/SKILL.md docs/product/prd.md; do
  run_hook "$p" "$p/$rel" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
  expect "gabo path $rel runs the check" "$code:$calls" "2:$p check"
done
run_hook "$p" "docs/product/intents/0009-x/plan.md" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "a relative gabo path runs the check" "$code:$calls" "2:$p check"

# S7: the four silent cases, each exit 0 with no output and no gabo call.
p=$(new_project quiet)
run_hook "$p" "$p/src/main.go" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S7 non-gabo path is silent" "$code:$out:$err:$calls" "0:::"
run_hook "$p" "$p/.claude/settings.json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S7 other .claude file is silent" "$code:$out:$err:$calls" "0:::"
run_hook "$p" "$work/other/docs/product/prd.md" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S7 gabo-like path outside the project is silent" "$code:$out:$err:$calls" "0:::"

nv=$(new_project noversion no-version)
run_hook "$nv" "$nv/docs/product/intents/0009-x/plan.md" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S7 no .gabo/VERSION is silent" "$code:$out:$err:$calls" "0:::"

run_hook "$p" "$p/docs/product/intents/0009-x/plan.md" PATH="$work/empty"
expect "S7 no gabo on PATH is silent" "$code:$out:$err" "0::"

run_hook "$p" "$p/docs/product/intents/0009-x/plan.md" CLAUDE_PLUGIN_OPTION_CHECKS=false FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S7 checks option off is silent" "$code:$out:$err:$calls" "0:::"
run_hook "$p" "$p/docs/product/intents/0009-x/plan.md" CLAUDE_PLUGIN_OPTION_CHECKS=true FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "checks option on runs the check" "$code:$calls" "2:$p check"

# S7: gabo exit 2 -> exit 0 with exactly one warning line.
run_hook "$p" "$p/docs/product/intents/0009-x/plan.md" FAKE_GABO_EXIT=2 FAKE_GABO_OUT="usage: gabo check"
expect "S7 gabo exit 2 exits 0" "$code" 0
expect "S7 gabo exit 2 prints nothing on stdout" "$out" ""
lines=$(printf '%s\n' "$err" | grep -c .)
expect "S7 gabo exit 2 prints one warning line" "$lines" 1
[[ "$err" == *warning* ]] && expect "S7 warning line says warning" ok ok || expect "S7 warning line says warning" "$err" "*warning*"

# Malformed or empty stdin never blocks.
log="$work/gabo.log"; : >"$log"
out=$(cd "$p" && env -i PATH="$fakebin:/usr/bin:/bin" CLAUDE_PROJECT_DIR="$p" FAKE_GABO_LOG="$log" \
  "$bash_bin" "$script" </dev/null 2>"$work/stderr"); code=$?
expect "empty stdin is silent" "$code:$out:$(cat "$work/stderr"):$(cat "$log")" "0:::"

# S8: the script holds no write and no rule of its own.
if [[ ! -f "$script" ]] || grep -nE '>[^&]|\btee\b|\bmkdir\b|\brm\b|\bmv\b|\bcp\b|\btouch\b|Acceptance|plan\.md' "$script" | grep -v '^[0-9]*:#' >/dev/null; then
  echo "FAIL  S8 script holds a write or a rule"
  grep -nE '>[^&]|\btee\b|\bmkdir\b|\brm\b|\bmv\b|\bcp\b|\btouch\b|Acceptance|plan\.md' "$script" | grep -v '^[0-9]*:#'
  failures=$((failures + 1))
else
  echo "ok    S8 script holds no write and no rule"
  passes=$((passes + 1))
fi

echo "passes=$passes failures=$failures"
[[ $failures -eq 0 ]]
