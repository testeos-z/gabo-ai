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
# The coder's final answer is its report, which the runner judges (intent 0030 S5).
cat "${FAKE_REPORT:-$(dirname "$0")/report-full.txt}"
exit 0
EOF
chmod +x "$work/bin/fake-engine"
fake="$work/bin/fake-engine"

# Coder report fixtures: the 0027 report gate fields (S19) in their order, with
# the not-run and not-applicable forms, which count as present.
cat >"$work/bin/report-full.txt" <<'EOF'
odd: RED fixture report judged before the runner read it; GREEN the same fixture passes
spec S4: covered (the runner reads the report and the checks pass)
RDD: not run: consent pending
contract: written on the task issue
tests: runner judges report fields: pass
integration: not applicable: no CLI command changed
commits: 1234567 feat(eval): judge coder method evidence
verifier: not applicable: no behavior change
EOF
# The same report as bullets with backticks, the form a coder writes in its outcome.
sed -e 's/^/- `/' -e 's/$/`/' "$work/bin/report-full.txt" >"$work/bin/report-bullets.txt"
# Two linked S#: the second one covered by its own line.
{
  cat "$work/bin/report-full.txt"
  echo "spec S5: not covered: fixture for the second linked S# only"
} >"$work/bin/report-two.txt"

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
  local specs="${TASK_SPECS:-[\"S4\"]}"
  jq -n --arg c "$1" --argjson s "$specs" '{
    id: "eval-001", source_issue: 821, source_pr: 849, source_wave: "I0030-fixture",
    starting_commit: $c, task_input: "Change the rule to new.", spec_refs: $s,
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
  FAKE_COUNT="$work/$name.count" FAKE_RUNS="$work/$name.runs" FAKE_FAIL_RUN="$fail" FAKE_REPORT="${REPORT_FILE:-}" \
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

# --- S5: method evidence judged like the 0027 report gate (S19) ---------------

# A report missing one field reads fail and names the field: one fixture per
# field, in the gate's order (odd, spec, RDD, contract, tests, integration,
# commits, verifier). The field's line is removed from the complete report.
for pair in "odd:|odd" "spec S4:|spec S4" "RDD:|RDD" "contract:|contract" "tests:|tests" \
  "integration:|integration" "commits:|commits" "verifier:|verifier"; do
  IFS='|' read -r prefix label <<<"$pair"
  slug="${label// /-}"
  e="$work/missing-$slug/evals"
  mkdir -p "$e"
  task "$start" >"$e/eval-001.json"
  grep -v -F "$prefix" "$work/bin/report-full.txt" >"$work/report-missing-$slug.txt"
  out="$(REPORT_FILE="$work/report-missing-$slug.txt" run_eval "missing-$slug" "$e" "$cfg_coder")"
  code=$?
  has "S5 a report with no $label line reads fail and names the field" "$code" 1 \
    "task eval-001: fail (0 of 1 runs): $label line missing" "$out"
  has "S5 the run line carries the same reason as fail: $label line missing" "$code" 1 \
    "run 1: fail: $label line missing" "$out"
done

# A complete report passes; not run and not applicable forms count as present.
e="$work/evidence-full/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(run_eval evidence-full "$e" "$cfg_coder")"
code=$?
has "S5 a report with every field passes (not run and not applicable count as present)" "$code" 0 \
  "task eval-001: pass (1 of 1 runs)" "$out"

# The gate reads the same fields when the coder writes them as bullets with
# backticks, the form of its outcome.
e="$work/evidence-bullets/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(REPORT_FILE="$work/bin/report-bullets.txt" run_eval evidence-bullets "$e" "$cfg_coder")"
code=$?
has "S5 a report written as backticked bullets passes" "$code" 0 "task eval-001: pass (1 of 1 runs)" "$out"

# Each linked S# needs its own spec line.
e="$work/evidence-two/evals"
mkdir -p "$e"
TASK_SPECS='["S4","S5"]' task "$start" >"$e/eval-001.json"
out="$(REPORT_FILE="$work/bin/report-two.txt" run_eval evidence-two "$e" "$cfg_coder")"
code=$?
has "S5 two linked S# with a covered or not-covered line each pass" "$code" 0 "task eval-001: pass (1 of 1 runs)" "$out"
out="$(run_eval evidence-two-missing "$e" "$cfg_coder")"
code=$?
has "S5 a linked S# with no spec line reads fail and names it" "$code" 1 \
  "task eval-001: fail (0 of 1 runs): spec S5 line missing" "$out"

# A failing outcome check with complete evidence reads fail with the observed result.
e="$work/evidence-check/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(run_eval evidence-check "$e" "$cfg_coder" 1)"
code=$?
has "S5 a failing check with complete evidence reads fail: check, with its exit code" "$code" 1 \
  "run 1: fail: check grep -qx done out.txt failed (exit 1)" "$out"

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

# S8 a failed run before a turn cut still makes the run exit 1 (the failure is
# not hidden by the pause).
cfg_fail_cut="$work/fail-cut.yaml"
printf 'runtimes:\n  coder: claude\neval:\n  min_tasks: 1\n  runs_per_task: 2\n  cap:\n    turns_per_task_run: 2\n' >"$cfg_fail_cut"
e="$work/caps-failcut/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(FAKE_CAP_RUN=2 run_eval caps-failcut "$e" "$cfg_fail_cut" 1)"
code=$?
has "S8 a cut task with an earlier failed run reads not run and names the failure" "$code" 1 \
  "task eval-001: not run: cap eval.cap.turns_per_task_run (a run before the cap failed)" "$out"

# S8 a cap block indented wider than two spaces still pauses the run.
cfg_wide_cap="$work/wide-cap.yaml"
printf 'runtimes:\n  coder: claude\neval:\n    min_tasks: 1\n    cap:\n        task_runs_per_eval_run: 1\n' >"$cfg_wide_cap"
e="$work/caps-wide/evals"
three_tasks "$e"
out="$(run_eval caps-wide "$e" "$cfg_wide_cap")"
code=$?
has "S8 a four-space cap block is read and pauses the run" "$code" 3 \
  "task eval-002: not run: cap eval.cap.task_runs_per_eval_run" "$out"

# S8 only claude's own error line is a turn-cap stop: codex printing the same
# words is a failing run, not a cut one.
e="$work/caps-echo/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(FAKE_CAP_RUN=1 run_eval caps-echo "$e" "$cfg_codex_cap")"
code=$?
has "S8 a non-claude engine that prints the phrase is a failing run, not a turn cut" "$code" 1 \
  "task eval-001: fail (0 of 1 runs)" "$out"

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
if grep -q "status: \|not run" <<<"$out"; then
  bad "S8 with no cap set the run is not paused and its summary line is unchanged"
else
  has "S8 with no cap set the summary line is unchanged" "$code" 0 "tasks: 1, failing: 0" "$out"
fi
has "S8 the report names the intent 0020 rule" "$code" 0 "intent 0020" "$out"
has "S8 the report says gabo does not cap spend" "$code" 0 "spend is not capped by gabo" "$out"

# --- S6, S7: the run report and the baseline comparison ----------------------

# line_of <output> <prefix>: the first output line that starts with the prefix.
line_of() { awk -v p="$2" 'index($0, p) == 1 { print; exit }' <<<"$1"; }

# baseline_file <evals-dir> <commit> <tasks-json>: the baseline the owner accepted,
# at evals/baseline/baseline.json, outside the task set (evals/*.json).
baseline_file() {
  mkdir -p "$1/baseline"
  jq -n --arg c "$2" --argjson t "$3" '{commit: $c, run_id: "eval-fixture", accepted_by: "owner", accepted_on: "2026-11-14", tasks: $t}' >"$1/baseline/baseline.json"
}

# S7 one passing task: every field of the run report is present.
e="$work/report/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
out="$(run_eval report "$e" "$cfg_coder")"
code=$?
rid="$(line_of "$out" "run id: " | sed 's/^run id: //')"
has "S7 the report names the head commit it measured" "$code" 0 "eval-run: head $head" "$out"
if [[ "$(line_of "$out" "changed configuration: ")" == "changed configuration: AGENTS.md" ]]; then
  ok "S7 the report names the changed configuration files it measured, and no source path"
else
  bad "S7 the report names the changed configuration files it measured (got: $(line_of "$out" "changed configuration: "))"
fi
has "S7 the report names the engine" "$code" 0 "engine: claude" "$out"
has "S7 the report names the runs per task" "$code" 0 "runs per task: 1" "$out"
has "S7 the report gives one line per task" "$code" 0 "task eval-001: pass (1 of 1 runs)" "$out"
has "S7 the report gives the pass rate of the set" "$code" 0 "pass rate: 1.00 (1 of 1 tasks)" "$out"
has "S7 with no threshold set the report reads threshold: not set" "$code" 0 "threshold: not set" "$out"
has "S7 with no baseline the report reads baseline: none" "$code" 0 "baseline: none" "$out"
has "S7 with no baseline nothing is compared, and the report says so" "$code" 0 \
  "moved from pass to fail: not compared (baseline: none)" "$out"
if grep -Eq '^run id: eval-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{6}$' <<<"$out"; then
  ok "S7 the report names its run id"
else
  bad "S7 the report names its run id (got: $rid)"
fi
has "S7 the report names the memory topic gabo/eval/<head sha>/<run id> (PD3)" "$code" 0 \
  "memory topic: gabo/eval/$head/$rid" "$out"
if grep -qi notice <<<"$out"; then
  bad "S6 with no baseline the run posts no notice and says nothing of one"
else
  ok "S6 with no baseline the run posts no notice and says nothing of one"
fi
if [[ -e "$e/baseline" ]]; then
  bad "S6 the runner creates no baseline file"
else
  ok "S6 the runner creates no baseline file"
fi

# S7 the threshold the owner set is in effect; a value outside 0 to 1 is refused.
cfg_thr="$work/threshold.yaml"
printf 'runtimes:\n  coder: claude\neval:\n  min_tasks: 1\n  pass_threshold: 0.8\n' >"$cfg_thr"
out="$(run_eval thr "$e" "$cfg_thr")"
code=$?
has "S7 the threshold in effect is named in the report" "$code" 0 "threshold: 0.8" "$out"
cfg_thr_high="$work/threshold-high.yaml"
printf 'runtimes:\n  coder: claude\neval:\n  min_tasks: 1\n  pass_threshold: 1.5\n' >"$cfg_thr_high"
out="$(run_eval thr-high "$e" "$cfg_thr_high")"
code=$?
has "S7 a threshold above one is refused before any engine runs" "$code" 1 \
  "refused config: eval.pass_threshold must be from 0 to 1" "$out"
if [[ -e "$work/thr-high.count" ]]; then
  bad "S7 a refused threshold starts no engine (count file exists)"
else
  ok "S7 a refused threshold starts no engine"
fi

# S6 a baseline is named with its commit; a task that passed on it and fails now
# is listed under the tasks that moved from pass to fail, with the failed check.
e="$work/base-moved/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
baseline_file "$e" "$start" '{"eval-001": "pass"}'
out="$(run_eval base-moved "$e" "$cfg_coder" 1)"
code=$?
has "S6 an accepted baseline is named with its path and its commit" "$code" 1 \
  "baseline: $e/baseline/baseline.json at commit $start" "$out"
has "S7 a task that passed on the baseline and fails now is listed with the failed check and its exit" "$code" 1 \
  "moved from pass to fail: eval-001: check grep -qx done out.txt failed (exit 1)" "$out"
if [[ "$(cat "$work/base-moved.count" 2>/dev/null)" == 1 ]]; then
  ok "S6 the baseline comparison reads the file and starts the one run it needs"
else
  bad "S6 the baseline comparison starts one run (count $(cat "$work/base-moved.count" 2>/dev/null))"
fi

e="$work/base-same/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
baseline_file "$e" "$start" '{"eval-001": "pass"}'
out="$(run_eval base-same "$e" "$cfg_coder")"
code=$?
has "S7 a task that still passes on the baseline is not listed as moved" "$code" 0 \
  "moved from pass to fail: none" "$out"

e="$work/base-failed/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
baseline_file "$e" "$start" '{"eval-001": "fail"}'
out="$(run_eval base-failed "$e" "$cfg_coder" 1)"
code=$?
has "S7 a task that failed on the baseline is not moved from pass to fail" "$code" 1 \
  "moved from pass to fail: none" "$out"

# S6 a malformed baseline is refused before any engine runs.
e="$work/base-commit/evals"
mkdir -p "$e/baseline"
task "$start" >"$e/eval-001.json"
jq -n '{commit: "nothex", tasks: {}}' >"$e/baseline/baseline.json"
out="$(run_eval base-commit "$e" "$cfg_coder")"
code=$?
has "S6 a baseline whose commit is not 40 hex is refused" "$code" 1 \
  "refused baseline: commit is not a 40-hex commit" "$out"
e="$work/base-tasks/evals"
mkdir -p "$e/baseline"
task "$start" >"$e/eval-001.json"
jq -n --arg c "$start" '{commit: $c, tasks: []}' >"$e/baseline/baseline.json"
out="$(run_eval base-tasks "$e" "$cfg_coder")"
code=$?
has "S6 a baseline whose tasks are not an object is refused" "$code" 1 \
  "refused baseline: tasks must be an object" "$out"
if [[ -e "$work/base-commit.count" || -e "$work/base-tasks.count" ]]; then
  bad "S6 a refused baseline starts no engine"
else
  ok "S6 a refused baseline starts no engine"
fi

# S6 the runner never writes the baseline: its bytes are the same after a run.
e="$work/base-readonly/evals"
mkdir -p "$e"
task "$start" >"$e/eval-001.json"
baseline_file "$e" "$start" '{"eval-001": "pass"}'
before="$(cksum <"$e/baseline/baseline.json")"
out="$(run_eval base-readonly "$e" "$cfg_coder" 1)"
after="$(cksum <"$e/baseline/baseline.json")"
if [[ "$before" == "$after" ]]; then
  ok "S6 the runner never writes the baseline file, even when a run fails against it"
else
  bad "S6 the runner never writes the baseline file"
fi
if [[ "$(find "$e" -newer "$e/baseline/baseline.json" -type f | wc -l)" == 0 ]]; then
  ok "S6 nothing in the evals folder is written after the baseline"
else
  bad "S6 nothing in the evals folder is written after the baseline"
fi

# S7 the pass rate of the set counts the tasks that pass (a task passes when all its runs pass).
e="$work/rate/evals"
three_tasks "$e"
out="$(run_eval rate-fail "$e" "$cfg_coder" 2)"
code=$?
has "S7 the pass rate counts the tasks that pass: two of three" "$code" 1 "pass rate: 0.67 (2 of 3 tasks)" "$out"
has "S7 the failing task keeps its failed check and its exit code" "$code" 1 \
  "task eval-002: fail (0 of 1 runs): check grep -qx done out.txt failed (exit 1)" "$out"

# S7 two saved reports are read back and their pass rates compared (the fixture
# read-back; the engram round trip is the assistant's, outside this worktree).
out1="$(run_eval rate-all "$e" "$cfg_coder")"
out2="$(run_eval rate-fail-2 "$e" "$cfg_coder" 2)"
r1="$(awk 'index($0, "pass rate: ") == 1 { print $3; exit }' <<<"$out1")"
r2="$(awk 'index($0, "pass rate: ") == 1 { print $3; exit }' <<<"$out2")"
rid1="$(line_of "$out1" "run id: " | sed 's/^run id: //')"
rid2="$(line_of "$out2" "run id: " | sed 's/^run id: //')"
if [[ "$r1" == 1.00 && "$r2" == 0.67 && "$rid1" != "$rid2" ]]; then
  ok "S7 two reports read back: pass rates 1.00 and 0.67 compare, under two run ids"
else
  bad "S7 two reports read back (rates '$r1' and '$r2', run ids '$rid1' and '$rid2')"
fi

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
