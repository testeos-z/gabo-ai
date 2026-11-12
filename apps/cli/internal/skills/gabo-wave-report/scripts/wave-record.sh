#!/bin/sh
# wave-record.sh [--wave <id>] stamp <event> <args...> | measures
# Records the steps of one wave and reports their measures (intent 0024).
# The wave id is --wave or GABO_WAVE_ID. Files live only under
# ${XDG_STATE_HOME:-$HOME/.local/state}/gabo/waves/<id>/ (stamps).
set -eu

if [ "${1:-}" = --wave ]; then
  [ $# -ge 2 ] || { echo "wave-record: no wave id" >&2; exit 2; }
  GABO_WAVE_ID=$2
  shift 2
fi
id=${GABO_WAVE_ID:-}
[ -n "$id" ] || { echo "wave-record: no wave id" >&2; exit 2; }
case $id in
  *[!A-Za-z0-9._-]* | . | ..) echo "wave-record: invalid wave id" >&2; exit 2 ;;
esac
dir=${XDG_STATE_HOME:-$HOME/.local/state}/gabo/waves/$id
stamps=$dir/stamps
cmd=${1:-}
[ $# -eq 0 ] || shift

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
  *)
    echo "wave-record: usage: [--wave <id>] stamp <event> <args...> | measures" >&2
    exit 2
    ;;
esac
