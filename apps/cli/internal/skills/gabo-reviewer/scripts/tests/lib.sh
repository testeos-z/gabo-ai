#!/usr/bin/env bash
# Shared helpers for the gabo-reviewer tests. Source it; do not run it.
# Every test installs a copy of the scripts into a fake HOME, at the
# user-level location the trust check expects
# (${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer/scripts).

src="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
failures=0
passes=0
export REVIEWER_ENV_FILE="/nonexistent/gabo-reviewer-test-env"
unset GH_TOKEN REVIEWER_GH_TOKEN CLAUDE_CODE_OAUTH_TOKEN GABO_REVIEWER_STATE_DIR XDG_DATA_HOME XDG_STATE_HOME XDG_CONFIG_HOME CODEX_HOME CODEX_API_KEY OPENAI_API_KEY SUDO_USER \
  GABO_UPGRADE_PR
# Repository-level settings come from the fixture's gabo.yaml, never from the
# environment of whoever runs the tests.
for _v in $(compgen -v REVIEWER_); do
  [[ "$_v" == REVIEWER_ENV_FILE ]] || unset "$_v"
done

new_tmp() { mktemp -d "${TMPDIR:-/tmp}/gabo-reviewer-test.XXXXXX"; }

# install_skill <home>: copies the scripts to
# <home>/.local/share/gabo/skills/gabo-reviewer/scripts with safe modes and
# prints that directory.
install_skill() {
  local d="$1/.local/share/gabo/skills/gabo-reviewer/scripts"
  mkdir -p "$d"
  chmod 755 "$1" "$1/.local" "$1/.local/share" "$1/.local/share/gabo" "$1/.local/share/gabo/skills" \
    "$1/.local/share/gabo/skills/gabo-reviewer" "$d"
  cp "$src"/*.sh "$src/config.env" "$d/"
  chmod 755 "$d"/*.sh
  chmod 644 "$d/config.env"
  (cd "$d" && pwd -P)
}

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

# finish: prints the totals; removes this test's own $work dir when every
# test passed (kept for inspection after a failure).
finish() {
  echo
  echo "$passes passed, $failures failed"
  if ((failures)); then
    echo "kept ${work:-} for inspection"
    exit 1
  fi
  [[ "${work:-}" != "${TMPDIR:-/tmp}"/gabo-reviewer-test.* ]] || rm -rf -- "$work"
  echo "all tests passed"
}
