#!/usr/bin/env bash
# Tests for the Codex copy of gabo-pre-write.sh run as
# `gabo-pre-write.sh codex`, the Codex plugin's PreToolUse apply_patch hook
# (intent 0002 S8, S13, D5; intent 0028 S3, S7, S12, S15; plan T8, T11). A fake
# gabo stands on PATH and records its working directory, argv, the engine it
# sees and its stdin outside the project; every case runs in a scratch gabo
# project of its own under one mktemp -d. The inputs follow the Codex
# PreToolUse shape (https://learn.chatgpt.com/docs/hooks): the patch text is in
# tool_input.command, with no file-path field. The deny output is the
# documented example kept in fixtures/pre-tool-use-deny.json, with gabo's lines
# in place of its reason.
# Run: bash plugins/codex/scripts/tests/gabo-pre-write.test.sh
set -uo pipefail

tests="$(cd "$(dirname "$0")" && pwd -P)"
dir="${tests%/*}"
script="$dir/gabo-pre-write.sh"
helper="$dir/gabo-hook-filter.sh"
fixture="$tests/fixtures/pre-tool-use-deny.json"
fixture_reason='"Destructive command blocked by hook."'
bash_bin="$(command -v bash)"
failures=0
passes=0
work="$(mktemp -d "${TMPDIR:-/tmp}/gabo-codex-pre-write-test.XXXXXX")"
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

# Fake gabo: appends "<cwd> <argv> engine=<GABO_ENGINE>" to $FAKE_GABO_LOG,
# copies its stdin to $FAKE_GABO_STDIN, prints $FAKE_GABO_OUT to stdout (as the
# real gabo check --pre-write does), exits $FAKE_GABO_EXIT or, with
# FAKE_GABO_KILL=1, kills itself with SIGKILL.
fakebin="$work/bin"
mkdir -p "$fakebin" "$work/empty"
cat >"$fakebin/gabo" <<'GABO'
#!/usr/bin/env bash
echo "$PWD $* engine=${GABO_ENGINE-unset}" >>"$FAKE_GABO_LOG"
cat >"$FAKE_GABO_STDIN"
[[ -n "${FAKE_GABO_OUT:-}" ]] && printf '%s\n' "$FAKE_GABO_OUT"
[[ "${FAKE_GABO_KILL:-}" == 1 ]] && kill -KILL $$
exit "${FAKE_GABO_EXIT:-0}"
GABO
chmod +x "$fakebin/gabo"

# new_project <name> [no-version]: a gabo project with an intent 0009-x whose
# plan.md does not exist yet.
new_project() {
  local p="$work/$1"
  mkdir -p "$p/docs/product/intents/0009-x" "$p/src"
  [[ "${2:-}" == no-version ]] || { mkdir -p "$p/.gabo" && echo 0.1.0 >"$p/.gabo/VERSION"; }
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

# codex_input <project> <patch>: a recorded-shape Codex PreToolUse input for
# an apply_patch call whose tool_input.command is <patch>.
codex_input() {
  printf '{"session_id":"s1","transcript_path":null,"cwd":"%s","hook_event_name":"PreToolUse","model":"gpt-5","turn_id":"t1","permission_mode":"default","tool_name":"apply_patch","tool_use_id":"call_1","tool_input":{"command":%s}}' \
    "$1" "$(json_string "$2")"
}

# add_patch <path>: an apply_patch text that adds <path>.
add_patch() { printf '*** Begin Patch\n*** Add File: %s\n+# plan\n*** End Patch\n' "$1"; }

# deny_for <reason>: the fixture's deny output with <reason> as its reason.
deny_for() {
  # Split around the example reason: a pattern replacement would reinterpret
  # backslashes in <reason>.
  local shape
  shape=$(<"$fixture")
  printf '%s%s%s' "${shape%%"$fixture_reason"*}" "$(json_string "$1")" "${shape#*"$fixture_reason"}"
}

# run_hook <project> <json> [env...]: feeds <json> to
# `gabo-pre-write.sh codex` from the project root (Codex runs hooks in the
# session cwd and sets no CLAUDE_PROJECT_DIR) and sets out, err, code, calls
# and stdin_seen.
run_hook() {
  local project="$1" json="$2"
  shift 2
  local log="$work/gabo.log" seen="$work/gabo.stdin"
  : >"$log"
  rm -f "$seen"
  local before after
  before=$(snapshot "$project")
  out=$(cd "$project" && env -i HOME="$work" PATH="$fakebin:/usr/bin:/bin" \
    FAKE_GABO_LOG="$log" FAKE_GABO_STDIN="$seen" "$@" \
    "$bash_bin" "$script" codex <<<"$json" 2>"$work/stderr")
  code=$?
  err=$(cat "$work/stderr")
  calls=$(cat "$log")
  stdin_seen=$(cat "$seen" 2>/dev/null)
  after=$(snapshot "$project")
  [[ "$before" == "$after" ]] || { echo "FAIL  hook changed $project"; failures=$((failures + 1)); }
}

plan_rel='docs/product/intents/0009-x/plan.md'
fail_line="✗ $plan_rel: spec.md is not accepted (Acceptance table has no filled row)"

# The fixture holds the documented deny shape and its example reason.
[[ -f "$fixture" && "$(<"$fixture")" == *"$fixture_reason"* ]] &&
  expect "fixture holds the documented deny example" ok ok ||
  expect "fixture holds the documented deny example" missing "$fixture with $fixture_reason"

# S13: gabo exit 1 -> Codex PreToolUse deny output with gabo's line as the
# reason, exit 0; the tool-call JSON reaches gabo check --pre-write unchanged,
# in the project root, with GABO_ENGINE=codex.
p=$(new_project denied)
json=$(codex_input "$p" "$(add_patch "$plan_rel")")
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="$fail_line"
expect "S13 denied patch exits 0" "$code" 0
expect "S13 denied patch prints the Codex deny output" "$out" "$(deny_for "$fail_line")"
expect "S13 denied patch is silent on stderr" "$err" ""
expect "S13 runs gabo check --pre-write once in the project root as codex" "$calls" "$p check --pre-write engine=codex"
expect "S13 pipes the tool-call JSON unchanged" "$stdin_seen" "$json"
[[ -e "$p/$plan_rel" ]] && expect "S13 denied plan.md does not exist" exists absent || expect "S13 denied plan.md does not exist" absent absent

# Several gabo lines, with quotes and backslashes, stay one valid JSON string.
two=$'✗ a "quoted" path\\x: one\n✗ b\tsecond'
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="$two"
expect "S13 multi-line reason is escaped as one JSON string" "$out" "$(deny_for "$two")"
if command -v python3 >/dev/null; then
  reason=$(printf '%s' "$out" | python3 -c 'import json,sys; o=json.load(sys.stdin)["hookSpecificOutput"]; print(o["permissionDecision"]+"|"+o["permissionDecisionReason"], end="")' 2>/dev/null)
  expect "S13 output parses as JSON and round-trips gabo's lines" "$reason" "deny|$two"
fi

# S13: gabo exit 0 -> exit 0, silent.
p=$(new_project accepted)
json=$(codex_input "$p" "$(add_patch "$plan_rel")")
run_hook "$p" "$json" FAKE_GABO_EXIT=0
expect "S13 accepted patch exits 0 silent" "$code:$out:$err" "0::"
expect "S13 accepted patch asked gabo as codex" "$calls" "$p check --pre-write engine=codex"
expect "S13 accepted patch piped the JSON unchanged" "$stdin_seen" "$json"

# An Update File of an absolute gabo path after a non-gabo file reaches gabo.
upd=$(printf '*** Begin Patch\n*** Update File: src/main.go\n@@\n-x\n+y\n*** Update File: %s\n@@\n-a\n+b\n*** End Patch\n' "$p/gabo.yaml")
run_hook "$p" "$(codex_input "$p" "$upd")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "an Update File of an absolute gabo path asks gabo" "$code:$out" "0:$(deny_for "✗ x")"

# S7 (intent 0028): every write reaches gabo, gabo path or not. A block on a
# non-gabo path denies it; an allowed write runs silently.
p=$(new_project quiet)
run_hook "$p" "$(codex_input "$p" "$(add_patch src/new.go)")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S7 a non-gabo path reaches gabo and a block denies it" "$code:$out:$err:$calls" "0:$(deny_for "✗ x")::$p check --pre-write engine=codex"
run_hook "$p" "$(codex_input "$p" "$(add_patch src/new.go)")" FAKE_GABO_EXIT=0
expect "S7 an allowed non-gabo path runs silently" "$code:$out:$err:$calls" "0:::$p check --pre-write engine=codex"
run_hook "$p" "$(codex_input "$p" "$(add_patch "$work/other/$plan_rel")")" FAKE_GABO_EXIT=0
expect "S7 an absolute path outside the project reaches gabo" "$code:$out:$err:$calls" "0:::$p check --pre-write engine=codex"

# S15: gabo exit 2 (usage or error) -> the patch is denied and the failure
# named, with gabo's lines after it.
run_hook "$p" "$json" FAKE_GABO_EXIT=2 FAKE_GABO_OUT="usage: gabo check --pre-write"
expect "S15 gabo exit 2 denies the patch naming the failure" "$code:$out" "0:$(deny_for $'gabo plugin: gabo check --pre-write failed (exit 2); the call is blocked\nusage: gabo check --pre-write')"

# S15: a killed gabo denies the patch naming the failure.
run_hook "$p" "$json" FAKE_GABO_KILL=1
expect "S15 a killed gabo denies the patch" "$code" 0
[[ "$out" == *"gabo check --pre-write failed (exit 137); the call is blocked"* ]] &&
  expect "S15 a killed gabo names the failure" ok ok ||
  expect "S15 a killed gabo names the failure" "$out" "*failed (exit 137)*"

# S12 and S10: exit 3 is an ask codex cannot show (probe item (d) is no), so
# the patch is denied and the deny says no prompt is shown.
run_hook "$p" "$json" FAKE_GABO_EXIT=3 FAKE_GABO_OUT="$fail_line"
expect "S12 exit 3 denies the patch and says no prompt is shown" "$code:$out" "0:$(deny_for "$fail_line"$'\n'"gabo plugin: Codex shows no prompt for an ask, so the call is blocked (intent 0028 S12)")"

# S15: gabo exit 1 with no output is still a deny, with a line that says so.
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT=""
expect "S15 a block without output still denies" "$code:$out" "0:$(deny_for "gabo plugin: gabo check --pre-write blocked the call without a message")"

# S15: no gabo on PATH -> the patch runs, with one warning line and no deny.
run_hook "$p" "$(codex_input "$p" "$(add_patch "$plan_rel")")" PATH="$work/empty"
expect "S15 no gabo on PATH runs the patch silently on stdout" "$code:$out" "0:"
lines=$(printf '%s\n' "$err" | grep -c .)
expect "S15 no gabo on PATH prints one warning line" "$lines" 1
[[ "$err" == *warning* ]] && expect "S15 the no-gabo line is a warning" ok ok || expect "S15 the no-gabo line is a warning" "$err" "*warning*"

# Silent cases: checks option off, no .gabo/VERSION, empty stdin.
run_hook "$p" "$json" CLAUDE_PLUGIN_OPTION_CHECKS=false FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "checks option off is silent and skips gabo" "$code:$out:$err:$calls" "0:::"
nv=$(new_project noversion no-version)
run_hook "$nv" "$(codex_input "$nv" "$(add_patch "$plan_rel")")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "no .gabo/VERSION is silent and skips gabo" "$code:$out:$err:$calls" "0:::"

# S15: input gabo cannot read as a tool call is a failure of gabo, so the
# patch is denied and the failure named.
run_hook "$p" '{"tool_name":"apply_patch","tool_input":{}}' FAKE_GABO_EXIT=2 FAKE_GABO_OUT="usage: gabo check --pre-write"
expect "S15 input without tool_input.command is denied as a failure" "$code:$out" "0:$(deny_for $'gabo plugin: gabo check --pre-write failed (exit 2); the call is blocked\nusage: gabo check --pre-write')"
log="$work/gabo.log"; : >"$log"
out=$(cd "$p" && env -i PATH="$fakebin:/usr/bin:/bin" FAKE_GABO_LOG="$log" \
  FAKE_GABO_STDIN="$work/gabo.stdin" "$bash_bin" "$script" codex </dev/null 2>"$work/stderr"); code=$?
expect "empty stdin is silent" "$code:$out:$(cat "$work/stderr"):$(cat "$log")" "0:::"

# S3, S13: the script and its shared filter hold no write and no acceptance or
# path rule of their own; every rule lives in gabo.
for f in "$script" "$helper"; do
  name=$(basename "$f")
  if [[ ! -f "$f" ]] || grep -niE '>[^&]|\btee\b|\bmkdir\b|\brm\b|\bmv\b|\bcp\b|\btouch\b|accept|plan\.md|spec\.md|intents' "$f" | grep -v '^[0-9]*:[[:space:]]*#' >/dev/null; then
    echo "FAIL  S13 $name holds a write or a rule"
    [[ -f "$f" ]] && grep -niE '>[^&]|\btee\b|\bmkdir\b|\brm\b|\bmv\b|\bcp\b|\btouch\b|accept|plan\.md|spec\.md|intents' "$f" | grep -v '^[0-9]*:[[:space:]]*#'
    failures=$((failures + 1))
  else
    echo "ok    S13 $name holds no write and no rule"
    passes=$((passes + 1))
  fi
done

echo "passes=$passes failures=$failures"
[[ $failures -eq 0 ]]
