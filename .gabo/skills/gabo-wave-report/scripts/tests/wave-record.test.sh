#!/usr/bin/env bash
# Tests for wave-record.sh (intent 0024 S1, S2): every run uses a fake HOME and
# XDG_STATE_HOME under one mktemp -d directory.

src="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
script="$src/wave-record.sh"
failures=0
passes=0
unset GABO_WAVE_ID XDG_STATE_HOME
work="$(mktemp -d "${TMPDIR:-/tmp}/gabo-wave-record-test.XXXXXX")"
export HOME="$work/home"
mkdir -p "$HOME"
export XDG_STATE_HOME="$work/state"
waves="$XDG_STATE_HOME/gabo/waves"

expect_eq() {
  if [[ "$2" == "$1" ]]; then
    echo "ok    $3"
    passes=$((passes + 1))
  else
    echo "FAIL  $3"
    echo "      want: $1"
    echo "      got:  $2"
    failures=$((failures + 1))
  fi
}

expect_match() {
  if grep -Eq -- "$1" <<<"$2"; then
    echo "ok    $3"
    passes=$((passes + 1))
  else
    echo "FAIL  $3"
    echo "      want match: $1"
    echo "      got: $2"
    failures=$((failures + 1))
  fi
}

run() { sh "$script" "$@" 2>&1; }

echo "# the script"
[[ -f "$script" ]] || { echo "FAIL  wave-record.sh is missing"; echo "1 failed"; exit 1; }
expect_eq "" "$(sh -n "$script" 2>&1)" "sh -n is clean"
if command -v dash >/dev/null 2>&1; then
  expect_eq "" "$(dash -n "$script" 2>&1)" "dash -n is clean"
fi

echo "# stamp"
RFC='[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z'
out="$(run --wave w1 stamp wave-started)"
expect_match "^wave-started $RFC\$" "$out" "stamp prints <event> <time>"
expect_eq "$out" "$(cat "$waves/w1/stamps")" "stamp appends the printed line"
out="$(run --wave w1 stamp session-started coder-a T1)"
expect_match "^session-started coder-a T1 $RFC\$" "$out" "session-started keeps its args"
expect_match "^pr-opened 7 $RFC\$" "$(run --wave w1 stamp pr-opened 7)" "pr-opened"
expect_match "^pr-merged 7 $RFC\$" "$(run --wave w1 stamp pr-merged 7)" "pr-merged"
expect_match "^wave-reported $RFC\$" "$(run --wave w1 stamp wave-reported)" "wave-reported"
expect_eq "5" "$(wc -l <"$waves/w1/stamps" | tr -d ' ')" "one line per stamp"
out="$(GABO_WAVE_ID=w2 sh "$script" stamp wave-started 2>&1)"
expect_match "^wave-started $RFC\$" "$out" "GABO_WAVE_ID gives the wave id"

expect_eq "$out" "$(cat "$waves/w2/stamps")" "GABO_WAVE_ID writes its own folder"

run --wave w1 stamp bogus-event >/dev/null
expect_eq "2" "$?" "an unknown event exits 2"
expect_eq "5" "$(wc -l <"$waves/w1/stamps" | tr -d ' ')" "an unknown event writes nothing"
sh "$script" stamp wave-started >/dev/null 2>&1
expect_eq "2" "$?" "no wave id exits 2"
expect_eq "wave-record: no wave id" "$(sh "$script" stamp wave-started 2>&1)" "no wave id message"

echo "# wave id guard"
before="$(find "$work" | sort)"
for bad in "a/b" ".." "../x" "a b"; do
  run --wave "$bad" stamp wave-started >/dev/null
  expect_eq "2" "$?" "id '$bad' is refused"
done
expect_eq "$before" "$(find "$work" | sort)" "a refused id writes nothing"

echo "# measures"
d="$waves/w3"
mkdir -p "$d"
cat >"$d/stamps" <<'EOS'
wave-started 2026-11-12T10:00:00Z
session-started coder-a T1 2026-11-12T10:01:00Z
session-started coder-b T2 2026-11-12T10:02:00Z
session-started coder-a T3 2026-11-12T10:03:00Z
pr-opened 11 2026-11-12T10:20:00Z
pr-opened 12 2026-11-12T10:25:00Z
pr-merged 11 2026-11-12T10:30:00Z
pr-merged 12 2026-11-12T10:40:00Z
wave-reported 2026-11-12T11:30:05Z
EOS
want="wall time: 5405 s
session-started coder-a: 2
session-started coder-b: 1
pr-opened: 2
pr-merged: 2"
m1="$(run --wave w3 measures)"
expect_eq "$want" "$m1" "two coders, two merged PRs"
expect_eq "$m1" "$(run --wave w3 measures)" "the same input twice gives the same output"
case "$m1" in *"~"*) expect_eq "no tilde" "tilde" "no ~ in output" ;; *) expect_eq "ok" "ok" "no ~ in output" ;; esac

d="$waves/w4"
mkdir -p "$d"
printf 'session-started coder-a T1 2026-11-12T10:01:00Z\nwave-reported 2026-11-12T11:00:00Z\n' >"$d/stamps"
expect_match "^wall time: unknown \(no wave-started stamp\)\$" "$(run --wave w4 measures | head -1)" "no wave-started reads unknown"

expect_match "^wall time: unknown \\(no wave-started stamp\\)\$" "$(run --wave w9 measures | head -1)" "a wave with no stamps reads unknown"

echo "# caps"
cproj="$work/capsproj"
mkdir -p "$cproj"
caps() { (cd "$cproj" && sh "$script" caps "$@" 2>&1); }
printf 'wave:\n  max_fix_rounds: 2\n' >"$cproj/gabo.yaml"
for e in claude codex opencode pi other; do
  expect_eq "max_fix_rounds: advisory
spend is not capped by gabo" "$(caps --engine $e)" "max_fix_rounds is advisory on $e"
done
printf 'wave:\n  max_coders: 2\n' >"$cproj/gabo.yaml"
expect_eq "max_coders: enforced by the cycle
spend is not capped by gabo" "$(caps --engine claude)" "max_coders is enforced by the cycle"
printf 'review:\n  x: 1\nwave:\n  max_coders: 2 # c\n  max_coder_turns: 30\n  max_fix_rounds: 1\nother:\n  max_coders: 9\n' >"$cproj/gabo.yaml"
want="max_coders: enforced by the cycle
max_fix_rounds: advisory
max_coder_turns: enforced by the engine
spend is not capped by gabo"
expect_eq "$want" "$(caps --engine claude)" "all keys on claude, other sections ignored"
expect_match "^max_coder_turns: advisory\$" "$(caps --engine codex | grep turns)" "max_coder_turns is advisory on codex"
expect_match "^max_coder_turns: enforced by the engine\$" "$(caps --engine opencode | grep turns)" "max_coder_turns is enforced by the engine on opencode (steps)"
for e in pi other; do
  expect_match "^max_coder_turns: advisory\$" "$(caps --engine $e | grep turns)" "max_coder_turns is advisory on $e"
done
expect_eq "continue or stop" "$(caps --engine claude --paused | tail -1)" "--paused ends with continue or stop"
expect_eq "spend is not capped by gabo" "$(caps --engine claude --paused | tail -2 | head -1)" "spend line precedes the paused line"
printf 'name: x\n' >"$cproj/gabo.yaml"
expect_eq "spend is not capped by gabo" "$(caps --engine claude)" "no wave key prints only the spend line"
rm "$cproj/gabo.yaml"
expect_eq "spend is not capped by gabo" "$(caps --engine claude)" "no gabo.yaml prints only the spend line"
caps >/dev/null
expect_eq "2" "$?" "caps without --engine exits 2"
printf 'wave:\n  max_coders: 2\n' >"$cproj/gabo.yaml"
h1="$(cksum <"$cproj/gabo.yaml")"
caps --engine claude --paused >/dev/null
expect_eq "$h1" "$(cksum <"$cproj/gabo.yaml")" "caps leaves gabo.yaml unchanged"
expect_eq "gabo.yaml" "$(ls "$cproj")" "caps writes nothing in the project dir"

echo "# saves"
expect_eq "none" "$(run --wave s1 saves unverified)" "no saves prints none"
run --wave s1 saves add k/one 11 >/dev/null
run --wave s1 saves add k/two 12 >/dev/null
run --wave s1 saves add k/three 13 >/dev/null
run --wave s1 saves read k/one >/dev/null
run --wave s1 saves read k/three >/dev/null
expect_eq "k/two" "$(run --wave s1 saves unverified)" "three adds and two reads print the unread key"
run --wave s1 saves add k/two 14 >/dev/null
expect_eq "k/two" "$(run --wave s1 saves unverified)" "a key added twice counts once"
run --wave s1 saves read k/two >/dev/null
expect_eq "none" "$(run --wave s1 saves unverified)" "every key read prints none"
run --wave s1 saves add k/one 15 >/dev/null
expect_eq "none" "$(run --wave s1 saves unverified)" "a re-add of a read key stays read"
out="$(run --wave s1 saves read k/never)"
rc=$?
expect_eq "2" "$rc" "reading a key never added exits 2"
expect_match "never added" "$out" "reading a key never added says so"
run --wave s1 saves add k/onlykey >/dev/null
expect_eq "2" "$?" "add without an observation id exits 2"
run --wave s1 saves bogus >/dev/null
expect_eq "2" "$?" "an unknown saves action exits 2"
run --wave s1 saves add "a b" 1 >/dev/null
expect_eq "2" "$?" "a key with a space is refused"
expect_eq "none" "$(run --wave s1 saves unverified)" "a refused add writes nothing"
expect_eq "" "$(find "$waves/s1" -type f -not -name saves)" "saves writes only its file in the wave folder"

echo "# read-only on gabo.yaml and no stray files"
proj="$work/proj"
mkdir -p "$proj"
printf 'wave:\n  max_coders: 2\n' >"$proj/gabo.yaml"
h1="$(cksum <"$proj/gabo.yaml")"
(cd "$proj" && sh "$script" --wave w5 stamp wave-started >/dev/null && sh "$script" --wave w5 measures >/dev/null)
expect_eq "$h1" "$(cksum <"$proj/gabo.yaml")" "gabo.yaml is unchanged"
expect_eq "gabo.yaml" "$(ls "$proj")" "nothing new in the project dir"
expect_eq "" "$(find "$work" -type f -not -path "$waves/*" -not -path "$proj/*" -not -path "$cproj/*")" "no file outside the wave folders"

echo
echo "$passes passed, $failures failed"
if ((failures)); then
  echo "kept $work for inspection"
  exit 1
fi
rm -rf -- "$work"
echo "all tests passed"
