#!/usr/bin/env bash
# eval-set.sh validates the agent eval task set under evals/ (intent 0030 S1,
# S2, S11). It reads task files and prints one verdict line per file; it runs
# no task, starts no engine and writes nothing (no push, PR or tracker write).
#
#   eval-set.sh check <evals-dir> [--config <gabo.yaml>]
#   eval-set.sh propose <evals-dir> <candidate.json> [--config <gabo.yaml>]
#   eval-set.sh config <min_tasks|max_tasks|runs_per_task|pass_threshold|turns_per_task_run|task_runs_per_eval_run> [--config <gabo.yaml>]
#
# eval.min_tasks (default 2), eval.max_tasks (default 10) and eval.runs_per_task
# (default 1) are read from the
# --config file, which must exist when it is given. The eval block must be in
# block style (eval: then indented keys). A config the script cannot read the
# way config.Load reads it is refused; it never falls back to the defaults.
# The caps eval.cap.turns_per_task_run and eval.cap.task_runs_per_eval_run
# (intent 0030 S8) have no default: unset (or null) prints nothing for them,
# and a value must be at least 1. The script writes no cap value.
# eval.pass_threshold (intent 0030 S7, S9) has no default either: unset (or null)
# prints nothing; a value must be a number from 0 to 1. The script sets none.
# Exit 0 when the set may run (or the proposal may be added for review), 1 on
# any refusal, 2 on bad usage.
set -uo pipefail
shopt -s nullglob

usage() {
  echo "usage: eval-set.sh check <evals-dir> [--config <gabo.yaml>]" >&2
  echo "       eval-set.sh propose <evals-dir> <candidate.json> [--config <gabo.yaml>]" >&2
  echo "       eval-set.sh config <min_tasks|max_tasks|runs_per_task|pass_threshold|turns_per_task_run|task_runs_per_eval_run> [--config <gabo.yaml>]" >&2
  exit 2
}

mode="${1:-}"
case "$mode" in check | propose | config) shift ;; *) usage ;; esac
dir=""
key=""
if [[ "$mode" == config ]]; then
  key="${1:-}"
  case "$key" in min_tasks | max_tasks | runs_per_task | pass_threshold | turns_per_task_run | task_runs_per_eval_run) ;; *) usage ;; esac
  shift
else
  dir="${1:-}"
  [[ -n "$dir" ]] || usage
  shift
fi
candidate=""
if [[ "$mode" == propose ]]; then
  candidate="${1:-}"
  [[ -n "$candidate" ]] || usage
  shift
fi
config=""
while (($#)); do
  case "$1" in
    --config)
      config="${2:-}"
      [[ -n "$config" ]] || usage
      shift 2
      ;;
    *) usage ;;
  esac
done

if [[ -n "$config" ]]; then
  [[ -f "$config" ]] || {
    echo "refused config: $config is not a file" >&2
    exit 1
  }
  if grep -Eq '^eval:[[:space:]]*[^[:space:]#]' "$config"; then
    echo "refused config: eval must be a block (eval: then indented keys), not an inline value" >&2
    exit 1
  fi
fi

# cfg_int <key> <default>: eval.<key> from the config file. A key that is set
# but is not a whole number is refused; an absent key gives the default.
cfg_int() {
  local v=""
  if [[ -n "$config" ]]; then
    v="$(awk -v key="$1" '
      /^[^[:space:]#]/ { in_eval = ($0 ~ /^eval:[[:space:]]*(#.*)?$/); next }
      in_eval && $1 == key":" { print ($2 == "" ? "empty" : $2); exit }
    ' "$config")"
  fi
  if [[ -z "$v" ]]; then
    echo "$2"
  elif [[ "$v" =~ ^[0-9]+$ ]]; then
    echo "$v"
  else
    echo "refused config: eval.$1 is not a whole number" >&2
    return 1
  fi
}

min_raw="$(cfg_int min_tasks 2)" || exit 1
max_raw="$(cfg_int max_tasks 10)" || exit 1
runs_raw="$(cfg_int runs_per_task 1)" || exit 1

# cfg_cap <key>: eval.cap.<key> from the config file (intent 0030 S8). The caps
# have no default: an absent or null key prints nothing ("not set"). A set value
# must be a whole number of at least 1. The cap block is read by indentation
# under eval:, and an inline eval.cap value is refused.
cfg_cap() {
  local v=""
  if [[ -n "$config" ]]; then
    v="$(awk -v key="$1" '
      function indent(s) { match(s, /^ */); return RLENGTH }
      /^[^[:space:]#]/ { in_eval = ($0 ~ /^eval:[[:space:]]*(#.*)?$/); cap_ind = -1; cap_child = -1; next }
      !in_eval || /^[[:space:]]*(#.*)?$/ { next }
      {
        ind = indent($0)
        body = substr($0, ind + 1)
        if (cap_ind >= 0 && ind > cap_ind) {
          if (cap_child < 0) cap_child = ind
          if (ind == cap_child && $1 == key":") { print ($2 == "" ? "empty" : $2); exit }
          next
        }
        cap_ind = -1
        if (body ~ /^cap:[[:space:]]*(#.*)?$/) { cap_ind = ind; cap_child = -1; next }
        if (body ~ /^cap:[[:space:]]*(null|~)[[:space:]]*(#.*)?$/) next
        if (body ~ /^cap:/) { print "inline"; exit }
      }
    ' "$config" | tr -d "\"'")"
  fi
  case "$v" in
    inline)
      echo "refused config: eval.cap must be a block (eval.cap then indented keys), not an inline value" >&2
      return 1 ;;
    "" | null | "~") echo ""; return 0 ;;
    *[!0-9]* | empty)
      echo "refused config: eval.cap.$1 is not a whole number" >&2
      return 1 ;;
  esac
  if ((10#$v < 1)); then
    echo "refused config: eval.cap.$1 must be at least 1" >&2
    return 1
  fi
  echo "$((10#$v))"
}

cap_turns="$(cfg_cap turns_per_task_run)" || exit 1
cap_runs="$(cfg_cap task_runs_per_eval_run)" || exit 1

# cfg_threshold: eval.pass_threshold from the config file (intent 0030 S7, S9),
# block style like the counts. Unset or null prints nothing. A value must be a
# number from 0 to 1, as config.Load reads it; anything else is refused.
cfg_threshold() {
  local v=""
  if [[ -n "$config" ]]; then
    v="$(awk '
      /^[^[:space:]#]/ { in_eval = ($0 ~ /^eval:[[:space:]]*(#.*)?$/); next }
      in_eval && $1 == "pass_threshold:" { print ($2 == "" ? "empty" : $2); exit }
    ' "$config" | tr -d "\"'")"
  fi
  case "$v" in
    "" | null | "~") return 0 ;;
  esac
  if [[ "$v" == empty ]] || ! [[ "$v" =~ ^(0|1)(\.[0-9]+)?$|^\.[0-9]+$ ]] || ! awk -v v="$v" 'BEGIN { exit !(v + 0 <= 1) }'; then
    echo "refused config: eval.pass_threshold must be from 0 to 1" >&2
    return 1
  fi
  echo "$v"
}

threshold="$(cfg_threshold)" || exit 1
min=$((10#$min_raw))
max=$((10#$max_raw))
runs=$((10#$runs_raw))
((min >= 1)) || {
  echo "refused config: eval.min_tasks must be at least 1" >&2
  exit 1
}
((max >= 1)) || {
  echo "refused config: eval.max_tasks must be at least 1" >&2
  exit 1
}
((min <= max)) || {
  echo "refused config: eval.min_tasks is above eval.max_tasks" >&2
  exit 1
}
((runs >= 1)) || {
  echo "refused config: eval.runs_per_task must be at least 1" >&2
  exit 1
}

# config <key>: prints the validated count, for the runner (eval-run.sh), which
# reads the same keys the same way instead of parsing gabo.yaml itself.
if [[ "$mode" == config ]]; then
  case "$key" in
    min_tasks) echo "$min" ;;
    max_tasks) echo "$max" ;;
    runs_per_task) echo "$runs" ;;
    turns_per_task_run) [[ -z "$cap_turns" ]] || echo "$cap_turns" ;;
    task_runs_per_eval_run) [[ -z "$cap_runs" ]] || echo "$cap_runs" ;;
    pass_threshold) [[ -z "$threshold" ]] || echo "$threshold" ;;
  esac
  exit 0
fi

# A token or key pattern in the raw bytes, in any decoded string, or in any key
# of a task file refuses it (S11). The match is never printed.
secret_re='(^|[^A-Za-z0-9_])(gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|sk-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16}|xox[abprs]-[A-Za-z0-9-]{10,})|-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----'
id_re='^[a-z0-9][a-z0-9-]{0,62}$'
date_re='^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
num_re='^[1-9][0-9]*$'
commit_re='^[0-9a-f]{40}$'
spec_re='^S[1-9][0-9]*$'

# secret_in <file>: exit 0 when the file holds a token or key pattern. The
# decoded text is matched from a variable, not a pipe: grep -q would close the
# pipe early and fail the pipeline under pipefail.
secret_in() {
  local decoded
  grep -Eq -- "$secret_re" "$1" && return 0
  jq -e . "$1" >/dev/null 2>&1 || return 1
  decoded="$(jq -r '.. | strings' "$1"; jq -r '[paths | map(tostring) | join(".")] | .[]' "$1")"
  grep -Eq -- "$secret_re" <<<"$decoded"
}

# readable <file>: prints the refusal reason for a file that is not a JSON
# task object, a token holder, or nothing.
readable() {
  local f="$1"
  if secret_in "$f"; then
    echo "token or key pattern"
  elif ! jq -e . "$f" >/dev/null 2>&1; then
    echo "not valid JSON"
  elif ! jq -e 'type == "object"' "$f" >/dev/null 2>&1; then
    echo "not a JSON object"
  fi
}

# problem <file>: prints the first problem with the required fields and their
# formats, or nothing when the task is well formed.
problem() {
  local f="$1" k v
  for k in id source_issue source_pr source_wave starting_commit task_input spec_refs outcome_checks; do
    v="$(jq -c --arg k "$k" '.[$k] | select(. != null and . != "" and . != [])' "$f")"
    [[ -n "$v" ]] || {
      echo "missing $k"
      return
    }
  done
  jq -e --arg re "$id_re" '.id | type == "string" and test($re)' "$f" >/dev/null || {
    echo "id must be lowercase letters, digits and hyphens, for example eval-001"
    return
  }
  [[ "$(jq -r '.source_issue | tostring' "$f")" =~ $num_re ]] || {
    echo "source_issue is not a positive number"
    return
  }
  [[ "$(jq -r '.source_pr | tostring' "$f")" =~ $num_re ]] || {
    echo "source_pr is not a positive number"
    return
  }
  jq -e '.source_wave | type == "string" and length > 0' "$f" >/dev/null || {
    echo "source_wave must be a non-empty string"
    return
  }
  [[ "$(jq -r '.starting_commit' "$f")" =~ $commit_re ]] || {
    echo "starting_commit is not a 40-hex commit"
    return
  }
  jq -e --arg re "$spec_re" '.spec_refs | type == "array" and all(.[]; type == "string" and test($re))' "$f" >/dev/null || {
    echo "spec_refs must be S<n> ids"
    return
  }
  jq -e '.outcome_checks | type == "array" and all(.[]; type == "string" and length > 0)' "$f" >/dev/null || {
    echo "outcome_checks must be a list of check commands"
    return
  }
  if [[ "$(jq -r '.acceptance != null' "$f")" == true ]]; then
    jq -e '.acceptance | type == "object"' "$f" >/dev/null || {
      echo "acceptance must be an object"
      return
    }
    v="$(jq -c '.acceptance.accepted_by | select(. != null and . != "")' "$f")"
    [[ -n "$v" ]] || {
      echo "missing acceptance.accepted_by"
      return
    }
    jq -e '.acceptance.accepted_by | type == "string"' "$f" >/dev/null || {
      echo "acceptance.accepted_by must be a string"
      return
    }
    v="$(jq -r '.acceptance.accepted_on // empty' "$f")"
    [[ -n "$v" ]] || {
      echo "missing acceptance.accepted_on"
      return
    }
    jq -e --arg re "$date_re" '.acceptance.accepted_on | type == "string" and test($re)' "$f" >/dev/null || {
      echo "acceptance.accepted_on must be YYYY-MM-DD"
      return
    }
  fi
}

if [[ "$mode" == check ]]; then
  accepted=0
  total=0
  refused=0
  seen=$'\n'
  for f in "$dir"/*.json; do
    total=$((total + 1))
    name="$(basename "$f")"
    reason="$(readable "$f")"
    [[ -n "$reason" ]] || reason="$(problem "$f")"
    if [[ -n "$reason" ]]; then
      echo "refused $name: $reason"
      refused=$((refused + 1))
      continue
    fi
    id="$(jq -r '.id' "$f")"
    if [[ "$seen" == *$'\n'"$id"$'\n'* ]]; then
      echo "refused $name: duplicate id $id"
      refused=$((refused + 1))
      continue
    fi
    seen+="$id"$'\n'
    if [[ "$(jq -r '.acceptance == null' "$f")" == true ]]; then
      echo "not accepted $name"
      continue
    fi
    echo "accepted $name"
    accepted=$((accepted + 1))
  done
  status=0
  if ((accepted < min)); then
    echo "set below minimum: $accepted of $min"
    status=1
  fi
  if ((total > max)); then
    echo "set above maximum: $total of $max"
    status=1
  fi
  ((refused == 0)) || status=1
  ((status == 0)) && echo "set: $accepted accepted of $total tasks, minimum $min, maximum $max"
  exit "$status"
fi

# propose: the candidate joins the set for review, so it needs no acceptance
# record yet; the set may hold at most eval.max_tasks task files.
[[ -f "$candidate" ]] || {
  echo "refused $(basename "$candidate"): not a file"
  exit 1
}
base="$(basename "$candidate")"
existing=0
for f in "$dir"/*.json; do
  [[ "$(basename "$f")" == "$base" ]] && continue
  existing=$((existing + 1))
done
if ((existing >= max)); then
  echo "maximum reached: $existing of $max tasks (eval.max_tasks); the proposal is not added"
  exit 1
fi
reason="$(readable "$candidate")"
[[ -n "$reason" ]] || reason="$(problem "$candidate")"
if [[ -n "$reason" ]]; then
  echo "refused $base: $reason"
  exit 1
fi
cid="$(jq -r '.id' "$candidate")"
for f in "$dir"/*.json; do
  [[ "$(basename "$f")" == "$base" ]] && continue
  if [[ "$(jq -r '.id' "$f" 2>/dev/null)" == "$cid" ]]; then
    echo "refused $base: duplicate id $cid"
    exit 1
  fi
done
echo "proposal ok: $base"
exit 0
