#!/usr/bin/env bash
# Tests for isolation-test.sh, the one-time reviewer isolation test of the
# codex and opencode engines (intent 0001 S10, plan T5). Fake codex and
# opencode stand on PATH; every run uses a fake HOME, XDG_CONFIG_HOME and
# TMPDIR of its own, so no real marker is ever written.
# Run: bash tests/isolation.test.sh
set -uo pipefail

script="$(cd "$(dirname "$0")/.." && pwd -P)/isolation-test.sh"
failures=0
passes=0
unset XDG_CONFIG_HOME XDG_DATA_HOME XDG_STATE_HOME XDG_CACHE_HOME CODEX_HOME
work="$(mktemp -d "${TMPDIR:-/tmp}/gabo-reviewer-config-test.XXXXXX")"

check() {
  local label="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    echo "ok    $label"
    passes=$((passes + 1))
  else
    echo "FAIL  $label"
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

expect_no_match() {
  if grep -Eq -- "$1" <<<"$2"; then
    echo "FAIL  $3"
    echo "      unwanted match: $1"
    echo "      got: $2"
    failures=$((failures + 1))
  else
    echo "ok    $3"
    passes=$((passes + 1))
  fi
}

# make_stubs <dir>: fake codex and opencode. Each call appends its argv to
# $STUB_LOG and writes its cwd and environment to $STUB_LOG.env.<cli>. The
# engine runs with a scrubbed environment, so the stubs read STUB_LOG and the
# modes from <dir>/stub-context, which run() rewrites for every case.
# CODEX_MODE / OPENCODE_MODE: ok (ignores the planted files), agents (obeys
# the planted AGENTS.md), config (applies the planted .codex/config.toml),
# plugin (loads the planted .opencode/ plugin), error (exits 1).
make_stubs() {
  mkdir -p "$1"
  cat >"$1/codex" <<'CODEX'
#!/usr/bin/env bash
source "$(dirname "$0")/stub-context"
echo "codex $*" >>"$STUB_LOG"
[[ "${1:-}" != --version ]] || { echo "codex-cli 9.9.9"; exit 0; }
{ echo "cwd=$PWD"; env; } >"$STUB_LOG.env.codex"
prompt="${*: -1}"
case "${CODEX_MODE:-ok}" in
  ok) grep -Eo 'GABO-ISOLATION-OK-[A-Za-z0-9]+' <<<"$prompt" | head -1 ;;
  agents) cat AGENTS.md ;;
  config) sh -c "$(sed -n 's/^notify = \["sh", "-c", "\(.*\)"\]$/\1/p' .codex/config.toml)" ;;
  error) echo "codex: not logged in" >&2; exit 1 ;;
esac
CODEX
  cat >"$1/opencode" <<'OPENCODE'
#!/usr/bin/env bash
source "$(dirname "$0")/stub-context"
echo "opencode $*" >>"$STUB_LOG"
[[ "${1:-}" != --version ]] || { echo "v9.9.9"; exit 0; }
{ echo "cwd=$PWD"; env; } >"$STUB_LOG.env.opencode"
prompt="${*: -1}"
case "${OPENCODE_MODE:-ok}" in
  ok) grep -Eo 'GABO-ISOLATION-OK-[A-Za-z0-9]+' <<<"$prompt" | head -1 ;;
  plugin)
    for f in .opencode/plugin/*.js .opencode/plugins/*.js; do
      [[ -f "$f" ]] || continue
      c="$(sed -n 's/.*writeFileSync("\([^"]*\)".*/\1/p' "$f")"
      [[ -z "$c" ]] || echo loaded >"$c"
    done
    grep -Eo 'GABO-ISOLATION-OK-[A-Za-z0-9]+' <<<"$prompt" | head -1 ;;
  error) echo "opencode: no provider" >&2; exit 1 ;;
esac
OPENCODE
  chmod 755 "$1/codex" "$1/opencode"
}

bin="$work/bin"
make_stubs "$bin"
owner="$work/owner"
# The owner's project: runtimes.reviewer must never change.
project="$work/project"
mkdir -p "$owner" "$project"
printf 'runtimes:\n  reviewer: claude\n' >"$project/gabo.yaml"
yaml_before="$(cat "$project/gabo.yaml")"

# run <case> [VAR=value...] <engine>: runs the script from the owner's project
# with the stubs, a fake HOME and its own TMPDIR; prints its output and exit code.
run() {
  local c="$1"
  shift
  rm -rf "$owner" "$work/tmp-$c"
  mkdir -p "$owner" "$work/tmp-$c"
  : >"$work/calls-$c.log"
  (
    cd "$project" || exit 1
    export HOME="$owner" TMPDIR="$work/tmp-$c" STUB_LOG="$work/calls-$c.log" PATH="$bin:$PATH"
    while [[ "${1:-}" == *=* ]]; do export "${1?}"; shift; done
    for v in STUB_LOG CODEX_MODE OPENCODE_MODE; do printf '%s=%q\n' "$v" "${!v:-}"; done >"$bin/stub-context"
    bash "$script" "$@" 2>&1
    echo "exit=$?"
  )
}
marker() { echo "$owner/.config/gabo/reviewer-isolation/$1"; }

echo "# codex passes"
out="$(run codex-pass codex)"
log="$(cat "$work/calls-codex-pass.log")"
expect_match 'exit=0' "$out" "an isolated codex passes"
expect_match 'reviewer engine `codex`: isolation test passed' "$out" "the pass names the engine"
check "the pass writes the marker under \$HOME/.config/gabo/reviewer-isolation/codex" test -f "$(marker codex)"
expect_match '^codex-cli 9\.9\.9$' "$(cat "$(marker codex)" 2>/dev/null)" "the marker holds the engine version"
expect_match '^codex exec --ignore-user-config --ignore-rules -s read-only --ephemeral ' "$log" \
  "codex runs with --ignore-user-config --ignore-rules -s read-only --ephemeral"
envf="$work/calls-codex-pass.log.env.codex"
codex_home="$(sed -n 's/^CODEX_HOME=//p' "$envf")"
expect_match "^$work/tmp-codex-pass/" "$codex_home" "codex gets a CODEX_HOME of its own in the scratch dir"
check "its CODEX_HOME holds no config.toml" test ! -e "$codex_home/config.toml"
cwd="$(sed -n 's/^cwd=//p' "$envf")"
expect_match "^$work/tmp-codex-pass/" "$cwd" "codex runs inside a scratch PR clone under mktemp -d"
check "the scratch PR clone is removed afterwards" test -z "$(ls -A "$work/tmp-codex-pass")"
expect_match "^$yaml_before\$" "$(cat "$project/gabo.yaml")" "runtimes.reviewer is unchanged"

echo "# codex fails"
for mode in agents config; do
  out="$(run "codex-$mode" CODEX_MODE="$mode" codex)"
  expect_match 'exit=1' "$out" "codex obeying the planted $mode exits 1"
  check "no marker when codex obeys the planted $mode" test ! -e "$(marker codex)"
  expect_match "^$yaml_before\$" "$(cat "$project/gabo.yaml")" "runtimes.reviewer is unchanged ($mode)"
done
out="$(run codex-agents CODEX_MODE=agents codex)"
expect_match 'reviewer engine `codex`: isolation test failed: planted AGENTS.md changed the reviewer.s behaviour' "$out" \
  "the fail names the engine and the AGENTS.md check"
expect_match 'runtimes.reviewer unchanged' "$out" "the fail says runtimes.reviewer is unchanged"
out="$(run codex-config CODEX_MODE=config codex)"
expect_match 'reviewer engine `codex`: isolation test failed: planted .codex/ config ran or changed the reviewer.s config' "$out" \
  "the fail names the engine and the .codex/ config check"
out="$(run codex-error CODEX_MODE=error codex)"
expect_match 'exit=1' "$out" "a codex run that errors fails"
expect_match 'reviewer engine `codex`: isolation test failed: codex run exited 1' "$out" "the fail names the failed run"
check "no marker when the codex run errors" test ! -e "$(marker codex)"

echo "# opencode passes"
out="$(run opencode-pass opencode)"
log="$(cat "$work/calls-opencode-pass.log")"
expect_match 'exit=0' "$out" "an isolated opencode passes"
check "the pass writes the marker" test -f "$(marker opencode)"
expect_match '^v9\.9\.9$' "$(cat "$(marker opencode)" 2>/dev/null)" "the marker holds the engine version"
expect_match '^opencode run --standalone ' "$log" "opencode runs as opencode run --standalone"
envf="$work/calls-opencode-pass.log.env.opencode"
for v in HOME XDG_CONFIG_HOME XDG_DATA_HOME XDG_STATE_HOME XDG_CACHE_HOME; do
  expect_match "^$work/tmp-opencode-pass/" "$(sed -n "s/^$v=//p" "$envf")" "opencode gets its own $v"
done
expect_match "^cwd=$work/tmp-opencode-pass/" "$(grep '^cwd=' "$envf")" "opencode runs inside the scratch PR clone"

echo "# opencode fails"
out="$(run opencode-plugin OPENCODE_MODE=plugin opencode)"
expect_match 'exit=1' "$out" "opencode loading the planted plugin exits 1"
expect_match 'reviewer engine `opencode`: isolation test failed: planted .opencode/ plugin executed' "$out" \
  "the fail names the engine and the plugin check"
check "no marker when the plugin executed" test ! -e "$(marker opencode)"
expect_match "^$yaml_before\$" "$(cat "$project/gabo.yaml")" "runtimes.reviewer is unchanged (plugin)"

echo "# marker location follows XDG_CONFIG_HOME"
out="$(run xdg XDG_CONFIG_HOME="$work/tmp-xdg-cfg" codex)"
check "the marker goes to \$XDG_CONFIG_HOME/gabo/reviewer-isolation/codex" test -f "$work/tmp-xdg-cfg/gabo/reviewer-isolation/codex"
rm -rf "$work/tmp-xdg-cfg"

echo "# never for claude or pi"
for e in claude pi; do
  out="$(run "$e" "$e")"
  expect_match 'exit=0' "$out" "$e exits 0"
  expect_match "reviewer engine \`$e\`: no isolation test needed" "$out" "$e needs no isolation test"
  expect_no_match '.' "$(cat "$work/calls-$e.log")" "no engine CLI runs for $e"
  check "no marker for $e" test ! -e "$owner/.config/gabo/reviewer-isolation"
done

echo "# usage and missing CLI"
out="$(run usage)"
expect_match 'exit=2' "$out" "no engine is a usage error"
out="$(run bogus bogus)"
expect_match 'exit=2' "$out" "an unknown engine is a usage error"
out="$(run missing PATH=/usr/bin:/bin codex)"
expect_match 'exit=1' "$out" "a missing CLI fails"
expect_match 'reviewer engine `codex`: CLI `codex` not found on `PATH`' "$out" "the fail names the engine and its CLI"
check "no marker when the CLI is missing" test ! -e "$(marker codex)"

echo
echo "$passes passed, $failures failed"
if ((failures)); then
  echo "kept $work for inspection"
  exit 1
fi
rm -rf -- "$work"
echo "all tests passed"
