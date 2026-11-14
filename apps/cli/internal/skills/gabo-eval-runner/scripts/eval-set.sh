#!/usr/bin/env bash
# eval-set.sh validates the agent eval task set under evals/ (intent 0030 S1,
# S2, S11). It reads task files and prints one verdict line per file; it runs
# no task, starts no engine and writes nothing (no push, PR or tracker write).
#
#   eval-set.sh check <evals-dir> [--config <gabo.yaml>]
#   eval-set.sh propose <evals-dir> <candidate.json> [--config <gabo.yaml>]
#
# eval.min_tasks (default 2) and eval.max_tasks (default 10) are read from the
# --config file; without it, or without the key, the defaults apply.
# Exit 0 when the set may run (or the proposal may be added for review), 1 on
# any refusal, 2 on bad usage.
set -uo pipefail
shopt -s nullglob

usage() {
  echo "usage: eval-set.sh check <evals-dir> [--config <gabo.yaml>]" >&2
  echo "       eval-set.sh propose <evals-dir> <candidate.json> [--config <gabo.yaml>]" >&2
  exit 2
}

mode="${1:-}"
case "$mode" in check | propose) shift ;; *) usage ;; esac
dir="${1:-}"
[[ -n "$dir" ]] || usage
shift
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

# cfg_int <key> <default>: eval.<key> from the config file. A key that is set
# but is not a whole number is refused; an absent file or key gives the default.
cfg_int() {
  local v=""
  if [[ -n "$config" && -f "$config" ]]; then
    v="$(awk -v key="$1" '
      /^[^[:space:]#]/ { in_eval = ($0 ~ /^eval:[[:space:]]*(#.*)?$/); next }
      in_eval && $1 == key":" { print $2; exit }
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

min="$(cfg_int min_tasks 2)" || exit 1
max="$(cfg_int max_tasks 10)" || exit 1

# A token or key pattern anywhere in the file refuses it (S11). The match is
# never printed.
secret_re='(^|[^A-Za-z0-9_])(gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|sk-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16}|xox[abprs]-[A-Za-z0-9-]{10,})|-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----'
id_re='^[1-9][0-9]*$'
commit_re='^[0-9a-f]{40}$'
spec_re='^S[1-9][0-9]*$'

# problem <file>: prints the first problem with the required fields, or
# nothing when the task is well formed.
problem() {
  local f="$1" k v
  for k in id source_issue source_pr starting_commit task_input spec_refs outcome_checks; do
    v="$(jq -c --arg k "$k" '.[$k] | select(. != null and . != "" and . != [])' "$f")"
    [[ -n "$v" ]] || {
      echo "missing $k"
      return
    }
  done
  [[ "$(jq -r '.source_issue | tostring' "$f")" =~ $id_re ]] || {
    echo "source_issue is not a positive number"
    return
  }
  [[ "$(jq -r '.source_pr | tostring' "$f")" =~ $id_re ]] || {
    echo "source_pr is not a positive number"
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
    for k in accepted_by accepted_on; do
      v="$(jq -r --arg k "$k" '.acceptance[$k] // empty' "$f")"
      [[ -n "$v" ]] || {
        echo "missing acceptance.$k"
        return
      }
    done
  fi
}

# readable <file>: prints the refusal reason for a file that is not a JSON
# task object, a token holder, or nothing.
readable() {
  local f="$1"
  if grep -Eq -- "$secret_re" "$f"; then
    echo "token or key pattern"
  elif ! jq -e . "$f" >/dev/null 2>&1; then
    echo "not valid JSON"
  elif ! jq -e 'type == "object"' "$f" >/dev/null 2>&1; then
    echo "not a JSON object"
  fi
}

if [[ "$mode" == check ]]; then
  accepted=0
  total=0
  refused=0
  for f in "$dir"/*.json; do
    total=$((total + 1))
    name="$(basename "$f")"
    reason="$(readable "$f")"
    if [[ -n "$reason" ]]; then
      echo "refused $name: $reason"
      refused=$((refused + 1))
      continue
    fi
    if [[ "$(jq -r '.acceptance == null' "$f")" == true ]]; then
      echo "not accepted $name"
      continue
    fi
    reason="$(problem "$f")"
    if [[ -n "$reason" ]]; then
      echo "refused $name: $reason"
      refused=$((refused + 1))
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
if [[ ! -f "$candidate" ]]; then
  echo "refused $base: not a file"
  exit 1
fi
reason="$(readable "$candidate")"
[[ -n "$reason" ]] || reason="$(problem "$candidate")"
if [[ -n "$reason" ]]; then
  echo "refused $base: $reason"
  exit 1
fi
echo "proposal ok: $base"
exit 0
