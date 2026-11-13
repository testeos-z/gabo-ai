#!/bin/sh
# wave-record.sh [--wave <id>] stamp <event> <args...> | measures
#   | saves add <topic key> <observation id> | saves read <topic key> | saves unverified
#   | caps --engine <engine> [--paused]
# Records the steps of one wave and reports their measures (intent 0024).
# The wave id is --wave or GABO_WAVE_ID (not needed by caps). Files live only
# under ${XDG_STATE_HOME:-$HOME/.local/state}/gabo/waves/<id>/ (stamps, saves).
# caps reads wave.* of ./gabo.yaml read-only.
set -eu

if [ "${1:-}" = --wave ]; then
  [ $# -ge 2 ] || { echo "wave-record: no wave id" >&2; exit 2; }
  GABO_WAVE_ID=$2
  shift 2
fi
cmd=${1:-}
[ $# -eq 0 ] || shift
id=${GABO_WAVE_ID:-}
if [ "$cmd" != caps ]; then
  [ -n "$id" ] || { echo "wave-record: no wave id" >&2; exit 2; }
  case $id in
    *[!A-Za-z0-9._-]* | . | ..) echo "wave-record: invalid wave id" >&2; exit 2 ;;
  esac
fi
dir=${XDG_STATE_HOME:-$HOME/.local/state}/gabo/waves/$id
stamps=$dir/stamps
saves=$dir/saves
case $cmd in
  stamp)
    event=${1:-}
    [ $# -eq 0 ] || shift
    case $event:$# in
      wave-started:0 | wave-reported:0 | pr-opened:1 | pr-merged:1 | session-started:2) ;;
      *) echo "wave-record: usage: stamp wave-started | session-started <agent> <task id> | pr-opened <pr> | pr-merged <pr> | wave-reported" >&2; exit 2 ;;
    esac
    mkdir -p "$dir"
    line="$event${*:+ $*} $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "$line" >>"$stamps"
    echo "$line"
    ;;
  measures)
    [ -f "$stamps" ] || stamps=/dev/null
    awk '
      function epoch(t,  y, m, d, a) {
        y = substr(t, 1, 4) + 0; m = substr(t, 6, 2) + 0; d = substr(t, 9, 2) + 0
        a = int((14 - m) / 12); y -= a; m += 12 * a - 3
        d = d + int((153 * m + 2) / 5) + 365 * y + int(y / 4) - int(y / 100) + int(y / 400)
        return d * 86400 + substr(t, 12, 2) * 3600 + substr(t, 15, 2) * 60 + substr(t, 18, 2)
      }
      { t = epoch($NF) }
      $1 == "wave-started" && !s { s = 1; start = t }
      $1 == "wave-reported" { r = 1; end = t }
      $1 == "session-started" { n[$2]++ }
      $1 == "pr-opened" { o++ }
      $1 == "pr-merged" { m++ }
      END {
        if (!s) print "wall time: unknown (no wave-started stamp)"
        else if (!r) print "wall time: unknown (no wave-reported stamp)"
        else print "wall time: " end - start " s"
        for (a in n) print "session-started " a ": " n[a] | "sort"
        close("sort")
        print "pr-opened: " o + 0
        print "pr-merged: " m + 0
      }' "$stamps"
    ;;
  saves)
    action=${1:-}
    [ $# -eq 0 ] || shift
    case $action:$# in
      add:2 | read:1)
        case $1 in
          '' | *[[:space:]]*) echo "wave-record: invalid topic key" >&2; exit 2 ;;
        esac ;;
      unverified:0) ;;
      *) echo "wave-record: usage: saves add <topic key> <observation id> | read <topic key> | unverified" >&2; exit 2 ;;
    esac
    case $action in
      add)
        mkdir -p "$dir"
        echo "add $1 $2" >>"$saves" ;;
      read)
        if [ ! -f "$saves" ] || ! awk -v k="$1" '$1 == "add" && $2 == k { f = 1 } END { exit !f }' "$saves"; then
          echo "wave-record: topic key never added: $1" >&2
          exit 2
        fi
        echo "read $1" >>"$saves" ;;
      unverified)
        if [ -f "$saves" ]; then
          awk '
            $1 == "add" && !($2 in seen) { seen[$2] = 1; order[++n] = $2 }
            $1 == "read" { done[$2] = 1 }
            END {
              for (i = 1; i <= n; i++) if (!(order[i] in done)) { print order[i]; p = 1 }
              if (!p) print "none"
            }' "$saves"
        else
          echo none
        fi ;;
    esac
    ;;
  caps)
    engine=
    paused=
    while [ $# -gt 0 ]; do
      case $1 in
        --engine) [ $# -ge 2 ] || { echo "wave-record: caps: --engine needs a value" >&2; exit 2; }; engine=$2; shift 2 ;;
        --paused) paused=1; shift ;;
        *) echo "wave-record: usage: caps --engine <engine> [--paused]" >&2; exit 2 ;;
      esac
    done
    [ -n "$engine" ] || { echo "wave-record: usage: caps --engine <engine> [--paused]" >&2; exit 2; }
    case $engine in
      claude|opencode) turns="enforced by the engine" ;;
      *) turns=advisory ;;
    esac
    if [ -f gabo.yaml ]; then
      awk -v turns="$turns" '
        /^[^ \t#]/ { w = ($0 ~ /^wave:/) }
        w && /^[ \t]+max_coders:/ { c = 1 }
        w && /^[ \t]+max_fix_rounds:/ { f = 1 }
        w && /^[ \t]+max_coder_turns:/ { t = 1 }
        END {
          if (c) print "max_coders: enforced by the cycle"
          if (f) print "max_fix_rounds: advisory"
          if (t) print "max_coder_turns: " turns
        }' gabo.yaml
    fi
    echo "spend is not capped by gabo"
    [ -z "$paused" ] || echo "continue or stop"
    ;;
  *)
    echo "wave-record: usage: [--wave <id>] stamp <event> <args...> | measures | saves ... | caps --engine <engine> [--paused]" >&2
    exit 2
    ;;
esac
