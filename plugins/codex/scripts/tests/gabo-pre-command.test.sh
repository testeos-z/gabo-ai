#!/usr/bin/env bash
# Tests for gabo-pre-command.sh run as `gabo-pre-command.sh codex`, the Codex
# plugin's PreToolUse Bash hook (intent 0028 S3, S10, S12, S15, S16; plan T11).
# A fake gabo stands on PATH and records its working directory, argv, the
# engine, task and outcome variables it sees and its stdin outside the project;
# every case runs in a scratch gabo project of its own under one mktemp -d. The
# inputs follow the Codex PreToolUse shape (https://learn.chatgpt.com/docs/hooks):
# the shell command is in tool_input.command, as a string or an argv list. A
# block is Codex's deny output, the documented example kept in
# fixtures/pre-tool-use-deny.json with gabo's lines as its reason.
# Run: bash plugins/codex/scripts/tests/gabo-pre-command.test.sh
set -uo pipefail

tests="$(cd "$(dirname "$0")" && pwd -P)"
dir="${tests%/*}"
script="$dir/gabo-pre-command.sh"
helper="$dir/gabo-hook-filter.sh"
fixture="$tests/fixtures/pre-tool-use-deny.json"
fixture_reason='"Destructive command blocked by hook."'
bash_bin="$(command -v bash)"
failures=0
passes=0
work="$(mktemp -d "${TMPDIR:-/tmp}/gabo-codex-pre-command-test.XXXXXX")"
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

# Fake gabo: appends "<cwd> <argv> engine=<GABO_ENGINE> task=<GABO_TASK>
# outcome=<GABO_OUTCOME_FILE>" to $FAKE_GABO_LOG, copies its stdin to
# $FAKE_GABO_STDIN, prints $FAKE_GABO_OUT to stdout, and exits $FAKE_GABO_EXIT
# or, with FAKE_GABO_KILL=1, kills itself with SIGKILL.
fakebin="$work/bin"
mkdir -p "$fakebin" "$work/empty"
cat >"$fakebin/gabo" <<'GABO'
#!/usr/bin/env bash
echo "$PWD $* engine=${GABO_ENGINE-unset} task=${GABO_TASK-unset} outcome=${GABO_OUTCOME_FILE-unset}" >>"$FAKE_GABO_LOG"
cat >"$FAKE_GABO_STDIN"
[[ -n "${FAKE_GABO_OUT:-}" ]] && printf '%s\n' "$FAKE_GABO_OUT"
[[ "${FAKE_GABO_KILL:-}" == 1 ]] && kill -KILL $$
exit "${FAKE_GABO_EXIT:-0}"
GABO
chmod +x "$fakebin/gabo"

# new_project <name> [no-version]: a gabo project with a src tree.
new_project() {
  local p="$work/$1"
  mkdir -p "$p/docs/product/intents/0009-x" "$p/src"
  [[ "${2:-}" == no-version ]] || { mkdir -p "$p/.gabo" && echo 0.1.0 >"$p/.gabo/VERSION"; }
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

# bash_json <project> <tool_input JSON>: a recorded-shape Codex PreToolUse
# input for a Bash call whose tool_input is <tool_input JSON>.
bash_json() {
  printf '{"session_id":"s1","transcript_path":null,"cwd":"%s","hook_event_name":"PreToolUse","model":"gpt-5","turn_id":"t1","permission_mode":"default","tool_name":"Bash","tool_use_id":"call_1","tool_input":%s}' "$1" "$2"
}

# cmd_json <project> <command text>: bash_json with tool_input.command as a string.
cmd_json() { bash_json "$1" "{\"command\":$(json_string "$2")}"; }

# deny_for <reason>: the fixture's deny output with <reason> as its reason.
deny_for() {
  # Split around the example reason: a pattern replacement would reinterpret
  # backslashes in <reason>.
  local shape
  shape=$(<"$fixture")
  printf '%s%s%s' "${shape%%"$fixture_reason"*}" "$(json_string "$1")" "${shape#*"$fixture_reason"}"
}

# run_hook <project> <json> [env...]: feeds <json> to the hook as
# `gabo-pre-command.sh codex` from the project root (Codex runs hooks in the
# session cwd and sets no CLAUDE_PROJECT_DIR) and sets out, err, code, calls
# and stdin_seen.
run_hook() {
  local project="$1" json="$2"
  shift 2
  local log="$work/gabo.log" seen="$work/gabo.stdin" outcome="$work/outcome.jsonl"
  : >"$log"
  rm -f "$seen" "$outcome"
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

# S3: every shell command reaches gabo check --pre-command, unchanged, in the
# project root, with GABO_ENGINE=codex on the call.
p=$(new_project shell)
json=$(cmd_json "$p" 'git push origin main')
run_hook "$p" "$json" FAKE_GABO_EXIT=0
expect "S3 an allowed shell call exits 0" "$code" 0
expect "S3 an allowed shell call prints nothing" "$out:$err" ":"
expect "S3 runs gabo check --pre-command once in the project root as codex" "$calls" "$p check --pre-command engine=codex task=unset outcome=unset"
expect "S3 pipes the tool-call JSON unchanged" "$stdin_seen" "$json"

# S3: a command with JSON escapes and an argv list reach gabo byte for byte.
json=$(bash_json "$p" '{"command":["bash","-lc","echo \"hi\" \\ ok"]}')
run_hook "$p" "$json" FAKE_GABO_EXIT=0
expect "S3 pipes an argv command and escapes unchanged" "$stdin_seen" "$json"

# S3: the engine is codex on every call, whatever the caller set.
json=$(cmd_json "$p" 'git status')
run_hook "$p" "$json" FAKE_GABO_EXIT=0 GABO_ENGINE=claude
expect "S3 GABO_ENGINE is codex even when the caller sets another" "$calls" "$p check --pre-command engine=codex task=unset outcome=unset"

# The cycle's GABO_TASK and GABO_OUTCOME_FILE pass through when it sets them.
run_hook "$p" "$json" FAKE_GABO_EXIT=0 GABO_TASK=I0028/T11 GABO_OUTCOME_FILE="$work/outcome.jsonl"
expect "S3 GABO_TASK and GABO_OUTCOME_FILE reach gabo" "$calls" "$p check --pre-command engine=codex task=I0028/T11 outcome=$work/outcome.jsonl"

# S15: gabo exit 1 -> Codex deny output with gabo's line as the reason, exit 0.
json=$(cmd_json "$p" 'git push --force origin feat/x')
line='✗ shell_guard: a force push is blocked (S4) next: use git push origin feat/x'
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="$line"
expect "S15 a blocked shell call exits 0 and denies through Codex's output" "$code:$out:$err" "0:$(deny_for "$line"):"
expect "S15 a blocked shell call prints nothing on stderr" "$err" ""

# Two gabo lines both reach the deny reason.
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT=$'✗ a: one\n✗ b: two'
expect "S15 a blocked call relays every gabo line" "$out" "$(deny_for $'✗ a: one\n✗ b: two')"
if command -v python3 >/dev/null; then
  reason=$(printf '%s' "$out" | python3 -c 'import json,sys; o=json.load(sys.stdin)["hookSpecificOutput"]; print(o["permissionDecision"]+"|"+o["hookEventName"], end="")' 2>/dev/null)
  expect "S15 the deny output parses as PreToolUse JSON" "$reason" "deny|PreToolUse"
fi

# S10 and S12: exit 3 (ask) is an ask codex cannot show (probe item (d) is no),
# so the call is denied and the deny says no prompt is shown.
run_hook "$p" "$json" FAKE_GABO_EXIT=3 FAKE_GABO_OUT="✗ shell_guard: a force push is blocked (the state was ask)"
expect "S10 exit 3 denies the call with gabo's line and the no-prompt note" "$code:$out" "0:$(deny_for $'✗ shell_guard: a force push is blocked (the state was ask)\ngabo plugin: Codex shows no prompt for an ask, so the call is blocked (intent 0028 S12)')"

# S15: gabo exit 2 (usage or error) or killed -> deny, failure named.
run_hook "$p" "$json" FAKE_GABO_EXIT=2 FAKE_GABO_OUT="usage: gabo check --pre-command"
expect "S15 gabo exit 2 denies the call and names the failure" "$code:$out" "0:$(deny_for $'gabo plugin: gabo check --pre-command failed (exit 2); the call is blocked\nusage: gabo check --pre-command')"

run_hook "$p" "$json" FAKE_GABO_KILL=1
expect "S15 a killed gabo denies the call" "$code" 0
[[ "$out" == *"gabo check --pre-command failed (exit 137); the call is blocked"* ]] &&
  expect "S15 a killed gabo names the failure" ok ok ||
  expect "S15 a killed gabo names the failure" "$out" "*failed (exit 137)*"

# S15: gabo exit 1 with no output is still a deny, with a line that says so.
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT=""
expect "S15 a block without output still denies" "$code:$out" "0:$(deny_for "gabo plugin: gabo check --pre-command blocked the call without a message")"

# S15: gabo exit 0 with output is allowed silently.
run_hook "$p" "$json" FAKE_GABO_EXIT=0 FAKE_GABO_OUT="unexpected text"
expect "S15 exit 0 runs the call silently" "$code:$out:$err" "0::"

# S15: no gabo on PATH -> the call runs and one warning line is shown on stderr.
run_hook "$p" "$json" PATH="$work/empty"
expect "S15 no gabo on PATH runs the call with nothing on stdout" "$code:$out" "0:"
lines=$(printf '%s\n' "$err" | grep -c .)
expect "S15 no gabo on PATH prints one warning line" "$lines" 1
[[ "$err" == *warning* ]] && expect "S15 the no-gabo line is a warning" ok ok || expect "S15 the no-gabo line is a warning" "$err" "*warning*"

# Silent cases: checks option off, no .gabo/VERSION, empty stdin.
run_hook "$p" "$json" CLAUDE_PLUGIN_OPTION_CHECKS=false FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "checks option off is silent and skips gabo" "$code:$out:$err:$calls" "0:::"
nv=$(new_project noversion no-version)
run_hook "$nv" "$(cmd_json "$nv" 'git push origin main')" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "no .gabo/VERSION is silent and skips gabo" "$code:$out:$err:$calls" "0:::"
log="$work/gabo.log"; : >"$log"
out=$(cd "$p" && env -i PATH="$fakebin:/usr/bin:/bin" FAKE_GABO_LOG="$log" \
  FAKE_GABO_STDIN="$work/gabo.stdin" "$bash_bin" "$script" codex </dev/null 2>"$work/stderr"); code=$?
expect "empty stdin is silent" "$code:$out:$(cat "$work/stderr"):$(cat "$log")" "0:::"

# The relay passes any tool-call JSON to gabo unchanged: the tool name is not
# judged here (S3), gabo judges the call.
json=$(printf '{"hook_event_name":"PreToolUse","tool_name":"shell","tool_input":{"command":["ls"]}}')
run_hook "$p" "$json" FAKE_GABO_EXIT=0
expect "S3 the relay passes any shell-tool JSON unchanged" "$stdin_seen" "$json"

# S16: a relayed block writes nothing to the outcome file; the relay records
# no answer, because codex shows no prompt (probe item (d) no).
p2=$(new_project record)
json=$(cmd_json "$p2" 'git push origin main')
run_hook "$p2" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x" GABO_OUTCOME_FILE="$work/outcome.jsonl"
[[ ! -e "$work/outcome.jsonl" ]] && expect "S16 the relay writes no outcome file" absent absent || expect "S16 the relay writes no outcome file" present absent

# S3: the script holds no command name, path pattern or lock state; every rule
# lives in gabo. Comment lines are not checked.
bad=$(grep -nE '\bgit\b|\bpush\b|\bforce\b|\brm\b|\bbranch\b|\bworktree\b|\bmain\b|\bmaster\b|\bdev\b|\.env|secret|\bkeychain\b|docs/product|gabo\.yaml|\.gabo/|\.claude/|lock' "$script" | grep -v '^[0-9]*:[[:space:]]*#')
if [[ -z "$bad" ]]; then
  echo "ok    S3 gabo-pre-command.sh holds no command name, path pattern or lock state"
  passes=$((passes + 1))
else
  echo "FAIL  S3 gabo-pre-command.sh holds a rule"
  echo "$bad"
  failures=$((failures + 1))
fi

# S3: the relay and the shared helper hold no write.
for f in "$script" "$helper"; do
  name=$(basename "$f")
  if grep -nE '>[^&]|\btee\b|\bmkdir\b|\brm\b|\bmv\b|\bcp\b|\btouch\b' "$f" | grep -v '^[0-9]*:[[:space:]]*#' >/dev/null; then
    echo "FAIL  S3 $name holds a write"
    failures=$((failures + 1))
  else
    echo "ok    S3 $name holds no write"
    passes=$((passes + 1))
  fi
done

echo "passes=$passes failures=$failures"
[[ $failures -eq 0 ]]
