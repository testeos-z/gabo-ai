#!/usr/bin/env bash
# Tests for gabo-pre-write.sh, the plugin's PreToolUse Write|Edit hook
# (intent 0001 S7, S12, D8; plan T8). A fake gabo stands on PATH and records
# its working directory, argv and stdin outside the project; every case runs
# in a scratch gabo project of its own under one mktemp -d.
# Run: bash plugins/claude-code/scripts/tests/gabo-pre-write.test.sh
set -uo pipefail

dir="$(cd "$(dirname "$0")/.." && pwd -P)"
script="$dir/gabo-pre-write.sh"
helper="$dir/gabo-hook-filter.sh"
bash_bin="$(command -v bash)"
failures=0
passes=0
work="$(mktemp -d "${TMPDIR:-/tmp}/gabo-plugin-pre-write-test.XXXXXX")"
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

# Fake gabo: appends "<cwd> <argv>" to $FAKE_GABO_LOG, copies its stdin to
# $FAKE_GABO_STDIN, prints $FAKE_GABO_OUT to stdout (as the real gabo check
# --pre-write does) and exits $FAKE_GABO_EXIT.
fakebin="$work/bin"
mkdir -p "$fakebin" "$work/empty"
cat >"$fakebin/gabo" <<'GABO'
#!/usr/bin/env bash
echo "$PWD $*" >>"$FAKE_GABO_LOG"
cat >"$FAKE_GABO_STDIN"
[[ -n "${FAKE_GABO_OUT:-}" ]] && printf '%s\n' "$FAKE_GABO_OUT"
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

# write_json <project> <file_path>: a Claude Code PreToolUse JSON for a Write.
write_json() {
  printf '{"session_id":"s1","cwd":"%s","hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"%s","content":"say \\"file_path\\": \\"gabo.yaml\\"\\n"}}' "$1" "$2"
}

# run_hook <project> <json> [env...]: feeds <json> to the hook and sets out,
# err, code, calls and stdin_seen.
run_hook() {
  local project="$1" json="$2"
  shift 2
  local log="$work/gabo.log" seen="$work/gabo.stdin"
  : >"$log"
  rm -f "$seen"
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

plan_rel='docs/product/intents/0009-x/plan.md'
fail_line="✗ $plan_rel: spec.md is not accepted (Acceptance table has no filled row)"

# S12: gabo exit 1 -> hook exit 2, gabo's lines on stderr; the tool-call JSON
# reaches gabo check --pre-write unchanged, in the project root.
p=$(new_project blocked)
json=$(write_json "$p" "$p/$plan_rel")
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="$fail_line"
expect "S12 blocked write exits 2" "$code" 2
expect "S12 blocked write relays gabo's line on stderr" "$err" "$fail_line"
expect "S12 blocked write prints nothing on stdout" "$out" ""
expect "S12 runs gabo check --pre-write once in the project root" "$calls" "$p check --pre-write"
expect "S12 pipes the tool-call JSON unchanged" "$stdin_seen" "$json"
[[ -e "$p/$plan_rel" ]] && expect "S12 blocked plan.md does not exist" exists absent || expect "S12 blocked plan.md does not exist" absent absent

# Two gabo lines both reach stderr.
run_hook "$p" "$json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT=$'✗ a: one\n✗ b: two'
expect "S12 relays every gabo line" "$code:$err" $'2:✗ a: one\n✗ b: two'

# S12: gabo exit 0 -> exit 0, silent.
p=$(new_project accepted)
json=$(write_json "$p" "$p/$plan_rel")
run_hook "$p" "$json" FAKE_GABO_EXIT=0 FAKE_GABO_OUT=""
expect "S12 accepted write exits 0 silent" "$code:$out:$err" "0::"
expect "S12 accepted write asked gabo" "$calls" "$p check --pre-write"
expect "S12 accepted write piped the JSON unchanged" "$stdin_seen" "$json"

# An Edit call and a relative gabo path reach gabo too.
edit_json=$(printf '{"hook_event_name":"PreToolUse","tool_name":"Edit","tool_input":{"file_path":"%s","old_string":"a","new_string":"b"}}' "$p/gabo.yaml")
run_hook "$p" "$edit_json" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "an Edit of a gabo file asks gabo" "$code:$calls" "2:$p check --pre-write"
expect "an Edit JSON is piped unchanged" "$stdin_seen" "$edit_json"
run_hook "$p" "$(write_json "$p" "$plan_rel")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "a relative gabo path asks gabo" "$code:$calls" "2:$p check --pre-write"

# S7: gabo exit 2 -> exit 0 with exactly one warning line.
run_hook "$p" "$json" FAKE_GABO_EXIT=2 FAKE_GABO_OUT="usage: gabo check --pre-write"
expect "S7 gabo exit 2 exits 0" "$code" 0
expect "S7 gabo exit 2 prints nothing on stdout" "$out" ""
lines=$(printf '%s\n' "$err" | grep -c .)
expect "S7 gabo exit 2 prints one warning line" "$lines" 1
[[ "$err" == *warning* ]] && expect "S7 warning line says warning" ok ok || expect "S7 warning line says warning" "$err" "*warning*"

# S7: the four silent cases, each exit 0 with no output and no gabo call.
p=$(new_project quiet)
run_hook "$p" "$(write_json "$p" "$p/src/main.go")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S7 non-gabo path is silent and skips gabo" "$code:$out:$err:$calls" "0:::"
run_hook "$p" "$(write_json "$p" "$work/other/$plan_rel")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S7 gabo-like path outside the project is silent" "$code:$out:$err:$calls" "0:::"

nv=$(new_project noversion no-version)
run_hook "$nv" "$(write_json "$nv" "$nv/$plan_rel")" FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S7 no .gabo/VERSION is silent and skips gabo" "$code:$out:$err:$calls" "0:::"

run_hook "$p" "$(write_json "$p" "$p/$plan_rel")" PATH="$work/empty"
expect "S7 no gabo on PATH is silent" "$code:$out:$err:$calls" "0:::"

run_hook "$p" "$(write_json "$p" "$p/$plan_rel")" CLAUDE_PLUGIN_OPTION_CHECKS=false FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "S7 checks option off is silent and skips gabo" "$code:$out:$err:$calls" "0:::"
run_hook "$p" "$(write_json "$p" "$p/$plan_rel")" CLAUDE_PLUGIN_OPTION_CHECKS=true FAKE_GABO_EXIT=1 FAKE_GABO_OUT="✗ x"
expect "checks option on asks gabo" "$code:$calls" "2:$p check --pre-write"

# Empty stdin never blocks.
log="$work/gabo.log"; : >"$log"
out=$(cd "$p" && env -i PATH="$fakebin:/usr/bin:/bin" CLAUDE_PROJECT_DIR="$p" FAKE_GABO_LOG="$log" \
  FAKE_GABO_STDIN="$work/gabo.stdin" "$bash_bin" "$script" </dev/null 2>"$work/stderr"); code=$?
expect "empty stdin is silent" "$code:$out:$(cat "$work/stderr"):$(cat "$log")" "0:::"

# S12: the script and its shared filter hold no write and no acceptance or
# plan.md rule; every rule lives in gabo.
for f in "$script" "$helper"; do
  name=$(basename "$f")
  if [[ ! -f "$f" ]] || grep -niE '>[^&]|\btee\b|\bmkdir\b|\brm\b|\bmv\b|\bcp\b|\btouch\b|accept|plan\.md|spec\.md|intents' "$f" | grep -v '^[0-9]*:[[:space:]]*#' >/dev/null; then
    echo "FAIL  S12 $name holds a write or a rule"
    [[ -f "$f" ]] && grep -niE '>[^&]|\btee\b|\bmkdir\b|\brm\b|\bmv\b|\bcp\b|\btouch\b|accept|plan\.md|spec\.md|intents' "$f" | grep -v '^[0-9]*:[[:space:]]*#'
    failures=$((failures + 1))
  else
    echo "ok    S12 $name holds no write and no rule"
    passes=$((passes + 1))
  fi
done

echo "passes=$passes failures=$failures"
[[ $failures -eq 0 ]]
