#!/usr/bin/env bash
# Tests for gabo-pre-command.sh, the plugin's PreToolUse Bash hook (intent
# 0028 S3, S10, S12, S15, S16; plan T10). A fake gabo stands on PATH and
# records its working directory, argv, the engine, task and outcome variables
# it sees and its stdin outside the project; every case runs in a scratch
# gabo project of its own under one mktemp -d.
# Run: bash plugins/claude-code/scripts/tests/gabo-pre-command.test.sh
set -uo pipefail

dir="$(cd "$(dirname "$0")/.." && pwd -P)"
script="$dir/gabo-pre-command.sh"
helper="$dir/gabo-hook-filter.sh"
bash_bin="$(command -v bash)"
failures=0
passes=0
work="$(mktemp -d "${TMPDIR:-/tmp}/gabo-plugin-pre-command-test.XXXXXX")"
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

# bash_json <project> <command as JSON text>: a Claude Code PreToolUse JSON
# for a Bash call; the command is already JSON-escaped.
bash_json() {
  printf '{"session_id":"s1","cwd":"%s","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"%s"}}' "$1" "$2"
}

# run_hook <project> <json> [env...]: feeds <json> to the hook and sets out,
# err, code, calls and stdin_seen.
run_hook() {
  local project="$1" json="$2"
  shift 2
  local log="$work/gabo.log" seen="$work/gabo.stdin" outcome="$work/outcome.jsonl"
  : >"$log"
  rm -f "$seen" "$outcome"
  local before after
  before=$(snapshot "$project")
  out=$(cd "$project" && env -i HOME="$work" PATH="$fakebin:/usr/bin:/bin" \
    CLAUDE_PROJECT_DIR="$project" FAKE_GABO_LOG="$log" FAKE_GABO_STDIN="$seen" "$@" \
    "$bash_bin" "$script" <<<"$json" 2>"$work/stderr")
  code=$?
  err=$(cat "$work/stderr")
  calls=$(cat "$log")
  stdin_seen=$(cat "$seen" 2>/dev/null)
  after=$(snapshot "$project")
  [[ "$before" == "$after" ]] || { echo "FAIL  hook changed $project"; failures=$((failures + 1)); }
}

json_for() { bash_json "$1" "$2"; }

# S3: every shell command reaches gabo check --pre-command, unchanged, in the
# project root, with GABO_ENGINE=claude on the call.
p=$(new_project shell)
json=$(json_for "$p" 'git push origin main')
run_hook "$p" "$json" FAKE_GABO_EXIT=0
expect "S3 an allowed shell call exits 0" "$code" 0
expect "S3 an allowed shell call prints nothing" "$out:$err" ":"
expect "S3 runs gabo check --pre-command once in the project root" "$calls" "$p check --pre-command engine=claude task=unset outcome=unset"
expect "S3 pipes the tool-call JSON unchanged" "$stdin_seen" "$json"

# S3: a command with JSON escapes and an agent id reaches gabo byte for byte.
json='{"session_id":"s1","hook_event_name":"PreToolUse","tool_name":"Bash","agent_id":"agent-7","tool_input":{"command":"echo \"hi\" \\ ok"}}'
run_hook "$p" "$json" FAKE_GABO_EXIT=0
expect "S3 pipes escaped text and the agent id unchanged" "$stdin_seen" "$json"

# S3: the engine is claude on every call, whatever the caller set.
json=$(json_for "$p" 'git status')
run_hook "$p" "$json" FAKE_GABO_EXIT=0 GABO_ENGINE=codex
expect "S3 GABO_ENGINE is claude even when the caller sets another" "$calls" "$p check --pre-command engine=claude task=unset outcome=unset"

# The cycle's GABO_TASK and GABO_OUTCOME_FILE pass through when it sets them.
run_hook "$p" "$json" FAKE_GABO_EXIT=0 GABO_TASK=I0028/T10 GABO_OUTCOME_FILE="$work/outcome.jsonl"
expect "S3 GABO_TASK and GABO_OUTCOME_FILE reach gabo" "$calls" "$p check --pre-command engine=claude task=I0028/T10 outcome=$work/outcome.jsonl"

# S15: gabo exit 1 -> hook exit 2, gabo's lines on stderr, nothing on stdout.
json=$(json_for "$p" 'git push --force origin feat/x')
line='✗ shell_guard: a force push is blocked (S4) next: use git push origin feat/x'
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="$line"
expect "S15 a blocked shell call exits 2" "$code" 2
expect "S15 a blocked shell call relays gabo's line on stderr" "$err" "$line"
expect "S15 a blocked shell call prints nothing on stdout" "$out" ""

# Two gabo lines both reach stderr.
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT=$'✗ a: one\n✗ b: two'
expect "S15 a blocked call relays every gabo line" "$code:$err" $'2:✗ a: one\n✗ b: two'

# S10 and S12: exit 3 (ask) is treated as 1 on claude, whose probe item (d)
# is no: the call is blocked and the message says no prompt is shown.
run_hook "$p" "$json" FAKE_GABO_EXIT=3 FAKE_GABO_OUT="✗ shell_guard: a force push is blocked (the state was ask)"
expect "S10 exit 3 exits 2 on claude" "$code" 2
[[ "$err" == *"✗ shell_guard: a force push is blocked"* ]] && expect "S10 exit 3 relays gabo's line" ok ok || expect "S10 exit 3 relays gabo's line" "$err" "*gabo line*"
[[ "$err" == *"Claude Code shows no prompt for an ask"* ]] && expect "S10 exit 3 says no prompt is shown" ok ok || expect "S10 exit 3 says no prompt is shown" "$err" "*no prompt*"

# S15: gabo exit 2 (usage or error) or killed -> hook exit 2, failure named.
run_hook "$p" "$json" FAKE_GABO_EXIT=2 FAKE_GABO_OUT="usage: gabo check --pre-command"
expect "S15 gabo exit 2 blocks the call" "$code" 2
[[ "$err" == *"gabo check --pre-command failed (exit 2)"* ]] && expect "S15 gabo exit 2 names the failure" ok ok || expect "S15 gabo exit 2 names the failure" "$err" "*failed (exit 2)*"
[[ "$err" == *"usage: gabo check --pre-command"* ]] && expect "S15 gabo exit 2 relays its output" ok ok || expect "S15 gabo exit 2 relays its output" "$err" "*usage*"

run_hook "$p" "$json" FAKE_GABO_KILL=1
expect "S15 a killed gabo blocks the call" "$code" 2
[[ "$err" == *"gabo check --pre-command failed (exit 137)"* ]] && expect "S15 a killed gabo names the failure" ok ok || expect "S15 a killed gabo names the failure" "$err" "*failed (exit 137)*"

# S15: gabo exit 1 with no output is still a block, with a line that says so.
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT=""
expect "S15 a block without output still exits 2" "$code" 2
[[ "$err" == *"blocked the call without a message"* ]] && expect "S15 a block without output says so" ok ok || expect "S15 a block without output says so" "$err" "*without a message*"

# S15: gabo exit 0 with output is allowed silently; the relay shows nothing.
run_hook "$p" "$json" FAKE_GABO_EXIT=0 FAKE_GABO_OUT="unexpected text"
expect "S15 exit 0 runs the call silently" "$code:$err" "0:"

# S15: no gabo on PATH -> the call runs and one warning line is shown.
run_hook "$p" "$json" PATH="$work/empty"
expect "S15 no gabo on PATH exits 0" "$code" 0
lines=$(printf '%s\n' "$err" | grep -c .)
expect "S15 no gabo on PATH prints one warning line" "$lines" 1
[[ "$err" == *warning* ]] && expect "S15 the no-gabo line is a warning" ok ok || expect "S15 the no-gabo line is a warning" "$err" "*warning*"

# Silent cases: checks option off, no .gabo/VERSION, empty stdin.
run_hook "$p" "$json" CLAUDE_PLUGIN_OPTION_CHECKS=false FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "checks option off is silent and skips gabo" "$code:$out:$err:$calls" "0:::"
nv=$(new_project noversion no-version)
run_hook "$nv" "$(json_for "$nv" 'git push origin main')" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "no .gabo/VERSION is silent and skips gabo" "$code:$out:$err:$calls" "0:::"
log="$work/gabo.log"; : >"$log"
out=$(cd "$p" && env -i PATH="$fakebin:/usr/bin:/bin" CLAUDE_PROJECT_DIR="$p" FAKE_GABO_LOG="$log" \
  FAKE_GABO_STDIN="$work/gabo.stdin" "$bash_bin" "$script" </dev/null 2>"$work/stderr"); code=$?
expect "empty stdin is silent" "$code:$out:$(cat "$work/stderr"):$(cat "$log")" "0:::"

# The Bash call's tool name is never checked by the relay: the hook judges what
# gabo judges (S3). A non-Bash payload still goes to gabo unchanged.
json=$(printf '{"hook_event_name":"PreToolUse","tool_name":"PowerShell","tool_input":{"command":"Get-ChildItem"}}')
run_hook "$p" "$json" FAKE_GABO_EXIT=0
expect "S3 the relay passes any shell-tool JSON unchanged" "$stdin_seen" "$json"

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

# S16: a relayed block writes nothing to the outcome file; the relay records
# no answer, because claude shows no prompt (probe item (d) no).
p2=$(new_project record)
json=$(json_for "$p2" 'git push origin main')
run_hook "$p2" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x" GABO_OUTCOME_FILE="$work/outcome.jsonl"
[[ ! -e "$work/outcome.jsonl" ]] && expect "S16 the relay writes no outcome file" absent absent || expect "S16 the relay writes no outcome file" present absent

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
