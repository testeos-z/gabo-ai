#!/usr/bin/env bash
# eval-run.sh runs the accepted agent eval tasks (intent 0030 S4, S12, D1). For
# each accepted task it runs eval.runs_per_task times the coder engine of
# runtimes.coder, in a mktemp scratch copy of the task's starting commit with
# the PR head's changed configuration applied; then it runs the task's outcome
# checks as scripts in that copy. A task passes only when every run passes.
#
#   eval-run.sh run <evals-dir> --head <ref> --config <gabo.yaml>
#                   [--repo <dir>] [--config-path <path>]...
#
# --head      the PR head whose changed configuration is applied (required).
# --repo      the repository that holds the starting commits (default: .).
# --config-path  a path of the head's diff that counts as configuration; other
#             changed paths are not applied (default: gabo.yaml .gabo .claude
#             AGENTS.md).
#
# The run writes nothing outside its scratch copies: no push, no PR, no comment,
# no tracker write, and no gh call. Each scratch path is printed; the parent
# removes it. GABO_EVAL_ENGINE_CMD replaces the engine with a shell command that
# reads the task input on stdin. It is for fixture tests only, and the report
# says so when it is set. The engine is bounded by GABO_EVAL_ENGINE_TIMEOUT
# seconds (default 600) when timeout(1) exists.
# Exit 0 when every task passes, 1 on a refusal or a failing task, 2 on bad usage.
set -uo pipefail
shopt -s nullglob

here="$(cd "$(dirname "$0")" && pwd -P)"
set_script="$here/eval-set.sh"

usage() {
  echo "usage: eval-run.sh run <evals-dir> --head <ref> --config <gabo.yaml> [--repo <dir>] [--config-path <path>]..." >&2
  exit 2
}

[[ "${1:-}" == run ]] || usage
shift
dir="${1:-}"
[[ -n "$dir" ]] || usage
shift
head_ref=""
config=""
repo="."
paths=()
while (($#)); do
  case "$1" in
    --head)
      head_ref="${2:-}"
      [[ -n "$head_ref" ]] || usage
      shift 2
      ;;
    --config)
      config="${2:-}"
      [[ -n "$config" ]] || usage
      shift 2
      ;;
    --repo)
      repo="${2:-}"
      [[ -n "$repo" ]] || usage
      shift 2
      ;;
    --config-path)
      [[ -n "${2:-}" ]] || usage
      paths+=("$2")
      shift 2
      ;;
    *) usage ;;
  esac
done
[[ -n "$head_ref" && -n "$config" ]] || usage
((${#paths[@]} > 0)) || paths=(gabo.yaml .gabo .claude AGENTS.md)

git -C "$repo" rev-parse --git-dir >/dev/null 2>&1 || {
  echo "refused --repo: $repo is not a git repository" >&2
  exit 1
}
head="$(git -C "$repo" rev-parse --verify -q "$head_ref^{commit}")" || {
  echo "refused --head: $head_ref is not a commit" >&2
  exit 1
}

# cfg_value <block> <key>: a value under a top-level block of gabo.yaml, in
# block style, with quotes removed. The same reading as eval-set.sh's cfg_int.
cfg_value() {
  awk -v blk="$1" -v key="$2" '
    /^[^[:space:]#]/ { in_b = ($0 ~ ("^" blk ":[[:space:]]*(#.*)?$")); next }
    in_b && $1 == key":" { print ($2 == "" ? "empty" : $2); exit }
  ' "$config" | tr -d "\"'"
}

# The counts come from eval-set.sh, which validates the config the same way the
# set check does (block style, every bound at least 1).
runs="$(bash "$set_script" config runs_per_task --config "$config")" || exit 1

engine="$(cfg_value runtimes coder)"
case "$engine" in
  "") echo "refused config: runtimes.coder is not set" >&2; exit 1 ;;
  claude | codex | pi | opencode) ;;
  *) echo "refused config: runtimes.coder must be claude, codex, pi or opencode" >&2; exit 1 ;;
esac

set_out="$(bash "$set_script" check "$dir" --config "$config" 2>&1)" || {
  printf '%s\n' "$set_out"
  exit 1
}
accepted_files=()
while IFS= read -r line; do
  [[ "$line" == "accepted "* ]] && accepted_files+=("$dir/${line#accepted }")
done <<<"$set_out"

scratches=()
limit="${GABO_EVAL_ENGINE_TIMEOUT:-600}"

# run_bounded <cmd...>: the command under timeout(1) when it exists.
run_bounded() {
  if command -v timeout >/dev/null 2>&1; then
    timeout "$limit" "$@"
  else
    "$@"
  fi
}

# engine_run <engine> <tree> <prompt> <log> <gh-config>: one coder run in the
# tree. The GitHub tokens (github.com and enterprise) and the SSH agent socket
# are removed from the environment and gh reads an empty config directory, so
# the engine cannot reach the owner's gh login through them. This is not a
# sandbox: the engine keeps the owner's HOME and filesystem access.
engine_run() {
  local engine="$1" tree="$2" prompt="$3" log="$4" gh_cfg="$5" text
  text="$(cat "$prompt")"
  (
    cd "$tree" || exit 1
    unset GH_TOKEN GITHUB_TOKEN GH_ENTERPRISE_TOKEN GITHUB_ENTERPRISE_TOKEN SSH_AUTH_SOCK
    export GH_CONFIG_DIR="$gh_cfg"
    if [[ -n "${GABO_EVAL_ENGINE_CMD:-}" ]]; then
      run_bounded bash -c "$GABO_EVAL_ENGINE_CMD"
    else
      case "$engine" in
        claude) run_bounded claude -p --permission-mode bypassPermissions "$text" ;;
        codex) run_bounded codex exec --ignore-user-config --ignore-rules -s workspace-write --ephemeral -C "$tree" - ;;
        pi) run_bounded pi --print "$text" ;;
        opencode) run_bounded opencode run --standalone "$text" ;;
      esac
    fi
  ) <"$prompt" >"$log" 2>&1
}

# run_one <task-file> <id> <starting-commit> <run-no> <task-input>: one run in a
# fresh scratch copy. Prints the run line and one line per outcome check, and
# returns 0 only when the run passes. A check that did not run is `not run`.
run_one() {
  local file="$1" id="$2" start="$3" n="$4" input="$5"
  local scratch tree reason="" ecode=0 checks_ok=1 verdict why="" tag="" c rc detail=""
  local -a checks
  mapfile -t checks < <(jq -r '.outcome_checks[]' "$file")
  scratch="$(mktemp -d "${TMPDIR:-/tmp}/gabo-eval-run.XXXXXX")"
  scratches+=("$scratch")
  tree="$scratch/tree"
  mkdir -p "$tree" "$scratch/gh-config"

  if ! git -C "$repo" cat-file -e "$start^{commit}" 2>/dev/null; then
    reason="starting commit not in repository"
  elif ! git -C "$repo" archive "$start" | tar -x -C "$tree"; then
    reason="starting commit could not be copied"
  elif ! git -C "$repo" diff --no-color --no-ext-diff --no-textconv --binary "$start" "$head" -- "${paths[@]}" >"$scratch/head-config.patch"; then
    reason="head configuration could not be read"
  elif [[ -s "$scratch/head-config.patch" ]] && ! (cd "$tree" && GIT_CEILING_DIRECTORIES="$scratch" git apply --whitespace=nowarn "$scratch/head-config.patch") 2>"$scratch/apply.log"; then
    reason="head configuration does not apply"
  fi

  if [[ -z "$reason" ]]; then
    printf '%s\n' "$input" >"$scratch/prompt.txt"
    engine_run "$engine" "$tree" "$scratch/prompt.txt" "$scratch/engine.log" "$scratch/gh-config" || ecode=$?
    for c in "${checks[@]}"; do
      rc=0
      (cd "$tree" && run_bounded bash -c "$c") >>"$scratch/checks.log" 2>&1 || rc=$?
      if ((rc == 0)); then
        detail+="    check: $c: pass (exit 0)"$'\n'
      else
        detail+="    check: $c: fail (exit $rc)"$'\n'
        checks_ok=0
      fi
    done
  else
    for c in "${checks[@]}"; do
      detail+="    check: $c: not run ($reason)"$'\n'
    done
  fi

  verdict=pass
  if [[ -n "$reason" ]]; then
    verdict=fail
    why="$reason"
  elif ((ecode != 0)); then
    verdict=fail
    why="engine exit $ecode"
  elif ((checks_ok == 0)); then
    verdict=fail
    why="a check failed"
  fi
  [[ -z "$why" ]] || tag=" ($why)"
  printf '%s\n' "$verdict" >"$scratch/result"
  printf '  run %s: %s%s (scratch %s)\n' "$n" "$verdict" "$tag" "$scratch"
  printf '%s' "$detail"
  [[ "$verdict" == pass ]]
}

engine_label="$engine"
[[ -z "${GABO_EVAL_ENGINE_CMD:-}" ]] || engine_label="$engine (fake command: GABO_EVAL_ENGINE_CMD)"
echo "eval-run: head $head"
echo "engine: $engine_label"
echo "runs per task: $runs"

failed_tasks=0
total_tasks=0
for f in "${accepted_files[@]}"; do
  total_tasks=$((total_tasks + 1))
  id="$(jq -r '.id' "$f")"
  start="$(jq -r '.starting_commit' "$f")"
  input="$(jq -r '.task_input' "$f")"
  echo "task $id: starting $start"
  passed=0
  for ((n = 1; n <= runs; n++)); do
    if run_one "$f" "$id" "$start" "$n" "$input"; then
      passed=$((passed + 1))
    fi
  done
  if ((passed == runs)); then
    echo "task $id: pass ($passed of $runs runs)"
  else
    echo "task $id: fail ($passed of $runs runs)"
    failed_tasks=$((failed_tasks + 1))
  fi
done

echo "tasks: $total_tasks, failing: $failed_tasks"
echo "scratch copies, kept for the parent to remove:"
for s in "${scratches[@]}"; do
  echo "  $s"
done
((failed_tasks == 0)) || exit 1
exit 0
