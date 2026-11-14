#!/usr/bin/env bash
# Fixture tests for eval-run.sh (intent 0030 S4, S12): one accepted task runs at
# its starting commit in a scratch copy with the head's configuration applied,
# eval.runs_per_task times, and its outcome checks run as scripts. The engine is
# a fake command (GABO_EVAL_ENGINE_CMD); no real engine starts, and the tests
# assert that nothing is written outside the scratch copy and that gh is never
# called. Run: bash tests/eval-run.test.sh
set -uo pipefail
here="$(cd "$(dirname "$0")/.." && pwd -P)"
script="$here/eval-run.sh"
work="$(mktemp -d "${TMPDIR:-/tmp}/gabo-eval-run-test.XXXXXX")"
trap 'rm -rf "$work"' EXIT
export TMPDIR="$work/tmp"
mkdir -p "$TMPDIR"
failures=0
passes=0
ok() { echo "ok    $1"; passes=$((passes + 1)); }
bad() { echo "FAIL  $1"; failures=$((failures + 1)); }

gitc() {
  local d="$1"
  shift
  git -c user.name=fixture -c user.email=fixture@example.invalid -c commit.gpgsign=false -C "$d" "$@"
}

# Fixture repository: the starting commit holds the old rule and a source file;
# the head changes the rule (configuration) and the source file (not config).
repo="$work/repo"
mkdir -p "$repo/src"
gitc "$repo" init -q
printf 'rule: old\n' >"$repo/AGENTS.md"
printf 'a\n' >"$repo/src/app.txt"
gitc "$repo" add -A
gitc "$repo" commit -q -m "chore: starting point"
start="$(gitc "$repo" rev-parse HEAD)"
printf 'rule: new\n' >"$repo/AGENTS.md"
printf 'b\n' >"$repo/src/app.txt"
gitc "$repo" commit -q -am "docs: head changes the rule and a source file"
head="$(gitc "$repo" rev-parse HEAD)"
missing_commit="ffffffffffffffffffffffffffffffffffffffff"

# Fake coder engine: counts its runs, records its working directory, writes
# out.txt (done, or broken on the run named by FAKE_FAIL_RUN).
mkdir -p "$work/bin"
cat >"$work/bin/fake-engine" <<'EOF'
#!/usr/bin/env bash
n=$(( $(cat "$FAKE_COUNT" 2>/dev/null || echo 0) + 1 ))
echo "$n" >"$FAKE_COUNT"
cat >/dev/null
echo "run $n pwd=$(pwd)" >>"$FAKE_RUNS"
if [[ "$n" == "${FAKE_FAIL_RUN:-0}" ]]; then echo broken >out.txt; else echo done >out.txt; fi
exit 0
EOF
chmod +x "$work/bin/fake-engine"
fake="$work/bin/fake-engine"

# A gh shim that logs and fails: the runner must never call gh.
mkdir -p "$work/shim"
cat >"$work/shim/gh" <<EOF
#!/usr/bin/env bash
echo "gh \$*" >>"$work/gh.log"
exit 99
EOF
chmod +x "$work/shim/gh"

# task <starting-commit>: one accepted task with three outcome checks.
task() {
  jq -n --arg c "$1" '{
    id: "eval-001", source_issue: 821, source_pr: 849, source_wave: "I0030-fixture",
    starting_commit: $c, task_input: "Change the rule to new.", spec_refs: ["S4"],
    outcome_checks: ["grep -qx \"rule: new\" AGENTS.md", "grep -qx a src/app.txt", "grep -qx done out.txt"],
    acceptance: {accepted_by: "owner", accepted_on: "2026-11-14"}
  }'
}

cfg_coder="$work/coder.yaml"
printf 'runtimes:\n  coder: claude\neval:\n  min_tasks: 1\n' >"$cfg_coder"
cfg_two="$work/two.yaml"
printf 'runtimes:\n  coder: claude\neval:\n  min_tasks: 1\n  runs_per_task: 2\n' >"$cfg_two"
cfg_zero="$work/zero.yaml"
printf 'runtimes:\n  coder: claude\neval:\n  min_tasks: 1\n  runs_per_task: 0\n' >"$cfg_zero"
cfg_engine="$work/engine.yaml"
printf 'runtimes:\n  coder: gemini\neval:\n  min_tasks: 1\n' >"$cfg_engine"
cfg_noruntime="$work/noruntime.yaml"
printf 'eval:\n  min_tasks: 1\n' >"$cfg_noruntime"
cfg_min2="$work/min2.yaml"
printf 'runtimes:\n  coder: claude\n' >"$cfg_min2"

# run_eval <name> <evals-dir> <config> [fail-run] [head]: the runner with the
# fake engine and the gh shim first on PATH; prints its output, returns its code.
run_eval() {
  local name="$1" evals="$2" cfg="$3" fail="${4:-0}" h="${5:-$head}"
  FAKE_COUNT="$work/$name.count" FAKE_RUNS="$work/$name.runs" FAKE_FAIL_RUN="$fail" \
    GABO_EVAL_ENGINE_CMD="$fake" PATH="$work/shim:$PATH" \
    bash "$script" run "$evals" --repo "$repo" --head "$h" --config "$cfg" 2>&1
}

# has <label> <code> <want-code> <needle> <output>: exit code and a substring.
has() {
  local label="$1" code="$2" want="$3" needle="$4" out="$5"
  if [[ "$code" == "$want" ]] && grep -Fq -- "$needle" <<<"$out"; then
    ok "$label"
  else
    bad "$label (exit $code, want $want; want: $needle)"
    sed 's/^/      | /' <<<"$out"
  fi
}

# snapshot: the fixture repository's git state, for the no-write assertion.
snapshot() {
  {
    find "$repo/.git" -type f -exec cksum {} + | sort
    gitc "$repo" status --porcelain
    gitc "$repo" for-each-ref
    gitc "$repo" stash list
  }
}

# --- S4: one accepted task, one run, passes ---------------------------------

e="$work/one/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
before="$(snapshot)"
out="$(run_eval one "$e" "$cfg_coder")"
code=$?
has "S4 an accepted task runs once and passes when all its checks pass" "$code" 0 \
  "task eval-001: pass (1 of 1 runs)" "$out"
has "S4 the report names the engine of runtimes.coder" "$code" 0 "engine: claude" "$out"
has "S4 the report names runs per task: 1 by default" "$code" 0 "runs per task: 1" "$out"
has "S4 the three outcome checks are each run and reported as pass" "$code" 0 \
  "check: grep -qx a src/app.txt: pass (exit 0)" "$out"
if [[ "$(cat "$work/one.count" 2>/dev/null)" == 1 ]]; then
  ok "S4 the engine runs exactly once for one run per task"
else
  bad "S4 the engine runs exactly once for one run per task (count $(cat "$work/one.count" 2>/dev/null))"
fi
if grep -q "pwd=$work/tmp/" "$work/one.runs" 2>/dev/null && ! grep -q "pwd=$repo" "$work/one.runs" 2>/dev/null; then
  ok "S4 the engine runs in a mktemp scratch copy, never in the repository"
else
  bad "S4 the engine runs in a mktemp scratch copy, never in the repository"
fi

# --- S4: nothing is written outside the scratch copy ------------------------

after="$(snapshot)"
if [[ "$before" == "$after" ]]; then
  ok "S4 the repository's git state is unchanged by a run (no push, no ref, no index write)"
else
  bad "S4 the repository's git state is unchanged by a run"
fi
if [[ ! -e "$work/gh.log" ]]; then
  ok "S4 gh is never called (no PR, comment or tracker write)"
else
  bad "S4 gh is never called: $(cat "$work/gh.log")"
fi
scratch="$(sed -n 's/.*(scratch \([^)]*\)).*/\1/p' <<<"$out" | head -1)"
if [[ -n "$scratch" && -f "$scratch/tree/AGENTS.md" ]]; then
  ok "S4 the scratch path is printed and kept for the parent to remove"
else
  bad "S4 the scratch path is printed and kept for the parent to remove (got: ${scratch:-none})"
fi

# --- S4: runs per task, a failing run reads fail and both are kept -----------

e="$work/two/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(run_eval two "$e" "$cfg_two" 2)"
code=$?
has "S4 two runs with one failing read fail (1 of 2 runs passed)" "$code" 1 \
  "task eval-001: fail (1 of 2 runs)" "$out"
has "S4 both run results are kept: run 1 passes" "$code" 1 "run 1: pass" "$out"
has "S4 both run results are kept: run 2 fails" "$code" 1 "run 2: fail" "$out"
has "S4 the report names runs per task: 2 from eval.runs_per_task" "$code" 1 "runs per task: 2" "$out"
if [[ "$(cat "$work/two.count" 2>/dev/null)" == 2 ]]; then
  ok "S4 the engine runs eval.runs_per_task times"
else
  bad "S4 the engine runs eval.runs_per_task times (count $(cat "$work/two.count" 2>/dev/null))"
fi

# --- S4: a check the runner did not execute never reads pass -----------------

e="$work/notrun/evals"
mkdir -p "$e"
task "$missing_commit" >"$e/eval-001.json"
out="$(run_eval notrun "$e" "$cfg_coder")"
code=$?
has "S4 a task whose starting commit is not in the repository fails" "$code" 1 "task eval-001: fail" "$out"
if grep -q ": pass (exit" <<<"$out"; then
  bad "S4 a check that did not run never reads pass: $out"
else
  ok "S4 a check that did not run never reads pass"
fi
has "S4 the not-run checks are reported as not run" "$code" 1 "not run" "$out"

# --- refusals: no engine runs -------------------------------------------------

e="$work/refuse/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"

out="$(run_eval refuse-zero "$e" "$cfg_zero")"
code=$?
has "S4 eval.runs_per_task of 0 is refused before any run" "$code" 1 \
  "refused config: eval.runs_per_task must be at least 1" "$out"

out="$(run_eval refuse-engine "$e" "$cfg_engine")"
code=$?
has "S4 an unknown runtimes.coder engine is refused before any run" "$code" 1 \
  "refused config: runtimes.coder must be claude, codex, pi or opencode" "$out"

out="$(run_eval refuse-noruntime "$e" "$cfg_noruntime")"
code=$?
has "S4 a config with no runtimes.coder is refused before any run" "$code" 1 \
  "refused config: runtimes.coder is not set" "$out"

out="$(run_eval refuse-min "$e" "$cfg_min2")"
code=$?
has "S4 a set below eval.min_tasks is refused before any run" "$code" 1 \
  "set below minimum: 1 of 2" "$out"

out="$(run_eval refuse-head "$e" "$cfg_coder" 0 nope)"
code=$?
has "S4 a head that is not a commit is refused" "$code" 1 "refused --head: nope is not a commit" "$out"

for n in refuse-zero refuse-engine refuse-noruntime refuse-min refuse-head; do
  if [[ ! -e "$work/$n.count" ]]; then
    ok "S4 $n starts no engine"
  else
    bad "S4 $n starts no engine (count file exists)"
  fi
done

echo "passes: $passes failures: $failures"
[[ "$failures" == 0 ]]
