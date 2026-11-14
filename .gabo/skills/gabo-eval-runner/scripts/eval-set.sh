#!/usr/bin/env bash
# eval-set.sh validates the agent eval task set under evals/ (intent 0030 S1,
# S2, S11). It reads task files and prints one verdict line per file; it runs
# no task, starts no engine and writes nothing (no push, PR or tracker write).
#
#   eval-set.sh check <evals-dir> [--config <gabo.yaml>]
#   eval-set.sh propose <evals-dir> <candidate.json> [--config <gabo.yaml>]
#
# eval.min_tasks (default 2) and eval.max_tasks (default 10) are read from the
# --config file, which must exist when it is given. The eval block must be in
# block style (eval: then indented keys). A config the script cannot read the
# way config.Load reads it is refused; it never falls back to the defaults.
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
min=$((10#$min_raw))
max=$((10#$max_raw))
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
    for k in accepted_by accepted_on; do
      v="$(jq -r --arg k "$k" '.acceptance[$k] // empty' "$f")"
      [[ -n "$v" ]] || {
        echo "missing acceptance.$k"
        return
      }
    done
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
