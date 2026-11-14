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
# out.txt (done, or broken on the run named by FAKE_FAIL_RUN). On the run named
# by FAKE_CAP_RUN it stops at a turn cap, as claude prints it, and exits 1.
# FAKE_STDIN keeps a copy of the prompt it received.
mkdir -p "$work/bin"
cat >"$work/bin/fake-engine" <<'EOF'
#!/usr/bin/env bash
n=$(( $(cat "$FAKE_COUNT" 2>/dev/null || echo 0) + 1 ))
echo "$n" >"$FAKE_COUNT"
cat >"${FAKE_STDIN:-/dev/null}"
echo "run $n pwd=$(pwd) enterprise=${GH_ENTERPRISE_TOKEN:-unset}/${GITHUB_ENTERPRISE_TOKEN:-unset} ssh=${SSH_AUTH_SOCK:-unset} token=${GH_TOKEN:-unset}/${GITHUB_TOKEN:-unset}" >>"$FAKE_RUNS"
if [[ "$n" == "${FAKE_CAP_RUN:-0}" ]]; then echo "Error: Reached max turns (2)"; exit 1; fi
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
cfg_runs_cap="$work/runs-cap.yaml"
printf 'runtimes:\n  coder: claude\neval:\n  min_tasks: 1\n  cap:\n    task_runs_per_eval_run: 1\n' >"$cfg_runs_cap"
cfg_turn_cap="$work/turn-cap.yaml"
printf 'runtimes:\n  coder: claude\neval:\n  min_tasks: 1\n  cap:\n    turns_per_task_run: 2\n' >"$cfg_turn_cap"
cfg_codex_cap="$work/codex-cap.yaml"
printf 'runtimes:\n  coder: codex\neval:\n  min_tasks: 1\n  cap:\n    turns_per_task_run: 3\n' >"$cfg_codex_cap"
cfg_cap_zero="$work/cap-zero.yaml"
printf 'runtimes:\n  coder: claude\neval:\n  min_tasks: 1\n  cap:\n    task_runs_per_eval_run: 0\n' >"$cfg_cap_zero"
cfg_cap_inline="$work/cap-inline.yaml"
printf 'runtimes:\n  coder: claude\neval:\n  min_tasks: 1\n  cap: {turns_per_task_run: 2}\n' >"$cfg_cap_inline"

# run_eval <name> <evals-dir> <config> [fail-run] [head]: the runner with the
# fake engine and the gh shim first on PATH; prints its output, returns its code.
run_eval() {
  local name="$1" evals="$2" cfg="$3" fail="${4:-0}" h="${5:-$head}"
  FAKE_COUNT="$work/$name.count" FAKE_RUNS="$work/$name.runs" FAKE_FAIL_RUN="$fail" \
    GH_ENTERPRISE_TOKEN=ghe-secret GITHUB_ENTERPRISE_TOKEN=ghe-secret SSH_AUTH_SOCK=/tmp/agent.sock \
    GH_TOKEN=gh-secret GITHUB_TOKEN=gh-secret \
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

# --- isolation: the engine sees no GitHub or agent credentials ---------------

if grep -q "enterprise=unset/unset ssh=unset token=unset/unset" "$work/one.runs" 2>/dev/null; then
  ok "S4 the engine does not see GitHub tokens, enterprise tokens or the SSH agent"
else
  bad "S4 the engine does not see GitHub tokens, enterprise tokens or the SSH agent: $(cat "$work/one.runs" 2>/dev/null)"
fi

# --- installed catalog: scripts without the exec bit still run ---------------

# The CLI writes catalog files as 0644. The runner calls its sibling eval-set.sh
# through bash, so a copy with no exec bit runs the same way.
noexec="$work/noexec"
mkdir -p "$noexec/scripts"
cp "$here/eval-run.sh" "$here/eval-set.sh" "$noexec/scripts/"
chmod 644 "$noexec/scripts/eval-run.sh" "$noexec/scripts/eval-set.sh"
e="$work/noexec/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(FAKE_COUNT="$work/noexec.count" FAKE_RUNS="$work/noexec.runs" FAKE_FAIL_RUN=0 \
  GABO_EVAL_ENGINE_CMD="$fake" PATH="$work/shim:$PATH" \
  bash "$noexec/scripts/eval-run.sh" run "$e" --repo "$repo" --head "$head" --config "$cfg_coder" 2>&1)"
code=$?
has "S4 the runner works from a catalog copy with no exec bit (bash runs its sibling)" "$code" 0 \
  "task eval-001: pass (1 of 1 runs)" "$out"

# --- a parent repository does not swallow the head configuration -------------

# TMPDIR inside a git repository: git apply must still apply to the scratch tree.
outer="$work/outer"
mkdir -p "$outer"
gitc "$outer" init -q
mkdir -p "$outer/tmp"
e="$work/outer-case/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(TMPDIR="$outer/tmp" FAKE_COUNT="$work/outer.count" FAKE_RUNS="$work/outer.runs" FAKE_FAIL_RUN=0 \
  GABO_EVAL_ENGINE_CMD="$fake" PATH="$work/shim:$PATH" \
  bash "$script" run "$e" --repo "$repo" --head "$head" --config "$cfg_coder" 2>&1)"
code=$?
has "S4 a TMPDIR inside a git repository still gets the head configuration applied" "$code" 0 \
  "task eval-001: pass (1 of 1 runs)" "$out"

# --- S8: caps pause the run; a cut task is never fail or pass ----------------

# three_tasks <evals-dir>: three accepted tasks, eval-001 to eval-003.
three_tasks() {
  mkdir -p "$1"
  local id
  for id in eval-001 eval-002 eval-003; do
    task "$start" | jq --arg id "$id" '.id = $id' >"$1/$id.json"
  done
}

# S8 task_runs_per_eval_run of 1: the second task is the first not started.
e="$work/caps-runs/evals"
three_tasks "$e"
out="$(run_eval caps-runs "$e" "$cfg_runs_cap")"
code=$?
has "S8 the first task runs and passes before the cap fires" "$code" 3 \
  "task eval-001: pass (1 of 1 runs)" "$out"
has "S8 the second task reads not run: cap <name>" "$code" 3 \
  "task eval-002: not run: cap eval.cap.task_runs_per_eval_run" "$out"
has "S8 the third task reads not run: cap <name>" "$code" 3 \
  "task eval-003: not run: cap eval.cap.task_runs_per_eval_run" "$out"
has "S8 the report is marked paused and names the cap" "$code" 3 \
  "status: paused by cap eval.cap.task_runs_per_eval_run" "$out"
has "S8 the pause asks the owner to continue or stop" "$code" 3 "continue or stop?" "$out"
has "S8 the report names the cap value" "$code" 3 "cap: eval.cap.task_runs_per_eval_run 1" "$out"
if grep -Eq "task eval-00[23]: (pass|fail)" <<<"$out" || grep -q "pass rate" <<<"$out"; then
  bad "S8 a task stopped by a cap never reads fail or pass, and the set gets no pass rate: $out"
else
  ok "S8 a task stopped by a cap never reads fail or pass, and the set gets no pass rate"
fi
if grep -q "task eval-002: starting" <<<"$out"; then
  bad "S8 a task the cap stops before its first run is not reported as starting"
else
  ok "S8 a task the cap stops before its first run is not reported as starting"
fi
if [[ "$(cat "$work/caps-runs.count" 2>/dev/null)" == 1 ]]; then
  ok "S8 the engine runs once: the cap stops the next task run before it starts"
else
  bad "S8 the engine runs once (count $(cat "$work/caps-runs.count" 2>/dev/null))"
fi

# S8 turns_per_task_run of 2: the engine stops at its turn cap on the second run.
e="$work/caps-turns/evals"
three_tasks "$e"
out="$(FAKE_CAP_RUN=2 run_eval caps-turns "$e" "$cfg_turn_cap")"
code=$?
has "S8 a task before the turn cap passes" "$code" 3 "task eval-001: pass (1 of 1 runs)" "$out"
has "S8 the task the engine stopped at its turn cap reads not run: cap <name>" "$code" 3 \
  "task eval-002: not run: cap eval.cap.turns_per_task_run" "$out"
has "S8 the task after it reads not run: cap <name>, the run pauses" "$code" 3 \
  "task eval-003: not run: cap eval.cap.turns_per_task_run" "$out"
has "S8 the report is marked paused by the turn cap" "$code" 3 \
  "status: paused by cap eval.cap.turns_per_task_run" "$out"
if grep -Eq "task eval-00[23]: (pass|fail)" <<<"$out"; then
  bad "S8 a turn-capped task never reads fail or pass: $out"
else
  ok "S8 a turn-capped task never reads fail or pass"
fi
if [[ "$(cat "$work/caps-turns.count" 2>/dev/null)" == 2 ]]; then
  ok "S8 the engine runs twice: the turn cap pauses before the third task starts"
else
  bad "S8 the engine runs twice (count $(cat "$work/caps-turns.count" 2>/dev/null))"
fi

# S8 claude gets its turn cap as --max-turns (its own print-mode flag); a fake
# claude on PATH records its arguments and stops at the cap.
mkdir -p "$work/claude-bin"
cat >"$work/claude-bin/claude" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>"$work/claude.args"
cat >/dev/null
echo "Error: Reached max turns (2)"
exit 1
EOF
chmod +x "$work/claude-bin/claude"
e="$work/caps-claude/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(env -u GABO_EVAL_ENGINE_CMD PATH="$work/claude-bin:$work/shim:$PATH" \
  bash "$script" run "$e" --repo "$repo" --head "$head" --config "$cfg_turn_cap" 2>&1)"
code=$?
if grep -q -- "--max-turns 2" "$work/claude.args" 2>/dev/null; then
  ok "S8 claude is started with --max-turns 2 for eval.cap.turns_per_task_run"
else
  bad "S8 claude is started with --max-turns 2 (args: $(cat "$work/claude.args" 2>/dev/null))"
fi
has "S8 a claude run stopped at its turn cap reads not run: cap, not fail" "$code" 3 \
  "task eval-001: not run: cap eval.cap.turns_per_task_run" "$out"
has "S8 the report says the turn cap is enforced by the engine for claude" "$code" 3 \
  "enforced by the engine" "$out"

# S8 advisory engines: the launch input carries the limit (intent 0020 D2).
e="$work/caps-codex/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(FAKE_STDIN="$work/caps-codex.prompt" run_eval caps-codex "$e" "$cfg_codex_cap")"
code=$?
has "S8 codex's launch input carries the turn limit" "$code" 0 \
  "Turn limit for this run: 3 turns." "$(cat "$work/caps-codex.prompt" 2>/dev/null)"
has "S8 the report says the turn cap is advisory for codex" "$code" 0 "advisory" "$out"

# S8 no cap set: the report says so, and the run is unchanged.
e="$work/caps-none/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(run_eval caps-none "$e" "$cfg_coder")"
code=$?
has "S8 with no cap set the report reads cap: none set" "$code" 0 "cap: none set" "$out"
has "S8 with no cap set the run is complete, not paused" "$code" 0 "status: complete" "$out"
has "S8 the report names the intent 0020 rule" "$code" 0 "intent 0020" "$out"
has "S8 the report says gabo does not cap spend" "$code" 0 "spend is not capped by gabo" "$out"

# --- refusals: no engine runs -------------------------------------------------

e="$work/refuse/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"

out="$(run_eval refuse-cap-zero "$e" "$cfg_cap_zero")"
code=$?
has "S8 eval.cap.task_runs_per_eval_run of 0 is refused before any run" "$code" 1 \
  "refused config: eval.cap.task_runs_per_eval_run must be at least 1" "$out"

out="$(run_eval refuse-cap-inline "$e" "$cfg_cap_inline")"
code=$?
has "S8 an inline eval.cap value is refused before any run" "$code" 1 \
  "refused config: eval.cap must be a block" "$out"

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

for n in refuse-zero refuse-engine refuse-noruntime refuse-min refuse-head refuse-cap-zero refuse-cap-inline; do
  if [[ ! -e "$work/$n.count" ]]; then
    ok "S4 $n starts no engine"
  else
    bad "S4 $n starts no engine (count file exists)"
  fi
done

echo "passes: $passes failures: $failures"
[[ "$failures" == 0 ]]
