#!/usr/bin/env bash
# Tests for the preflight (trust, location, login, base checkout, config from
# the integration branch, base version minimum) and the shared helpers.
# Run: bash tests/preflight.test.sh
set -uo pipefail
source "$(dirname "$0")/lib.sh"
source "$(dirname "$0")/stubs.sh"

work="$(new_tmp)"
fake_home="$work/home"
skill="$(install_skill "$fake_home")"
make_stubs "$work/bin"
fx="$work/fx"
mkdir -p "$fx"
echo "owner" >"$fx/user_login"
make_origin "$work/origin.git"
# The caller's checkout: a clone of the repository on its integration branch.
caller="$work/caller"
git clone -q --branch dev "$work/origin.git" "$caller"

# pre [VAR=value...] <cmd...>: runs from the caller checkout with the stubs.
pre() {
  (
    cd "$caller" || exit 1
    export HOME="$fake_home" STUB_LOG="$fx/calls.log" FIXTURE="$fx" REVIEWER_GH="$work/bin/gh" \
      GABO_REVIEWER_STATE_DIR="${PRE_STATE:-$work/state}" REVIEWER_REMOTE_URL="${PRE_ORIGIN:-$work/origin.git}" \
      REVIEWER_GABO_BIN="$work/bin/gabo" GABO_VERSION="${PRE_GABO_VERSION:-0.4.0}" PATH="$work/bin:$PATH"
    while [[ "${1:-}" == *=* ]]; do export "${1?}"; shift; done
    "$@"
  )
}
sha=0123456789abcdef0123456789abcdef01234567

echo "# preflight"
expect_match '^PREFLIGHT-OK repo=example-org/example-repo base=dev login=owner mode=single-user engine=claude placement=local minimum=none$' \
  "$(pre bash "$skill/reviewer.sh" preflight)" "installed skill passes; repo, base and placement come from gabo.yaml"
check "the reviewer keeps its own checkout of the integration branch" test -f "$work/state/base-example-org__example-repo/gabo.yaml"
expect_eq "$DEV_SHA" "$(git -C "$work/state/base-example-org__example-repo" rev-parse HEAD)" "its checkout is the integration branch tip"
expect_match '^PREFLIGHT-OK repo=other/repo ' "$(pre REVIEWER_REPO=other/repo bash "$skill/reviewer.sh" preflight)" \
  "REVIEWER_REPO overrides the caller's repo.name"
expect_match '^PREFLIGHT-FAILED no repository' \
  "$(cd "$work" && HOME="$fake_home" REVIEWER_GH="$work/bin/gh" GABO_REVIEWER_STATE_DIR="$work/state" bash "$skill/reviewer.sh" preflight)" \
  "outside a checkout with gabo.yaml and without REVIEWER_REPO it refuses"

echo "# location and trust"
inrepo="$caller/.gabo/skills/gabo-reviewer/scripts"
mkdir -p "$inrepo" && cp "$skill"/*.sh "$skill/config.env" "$inrepo/"
expect_match '^PREFLIGHT-FAILED .*inside the repository under review' \
  "$(pre bash "$inrepo/reviewer.sh" preflight)" "a copy inside the repository under review is refused"
rm -rf "$caller/.gabo/skills/gabo-reviewer"
other="$work/elsewhere"
mkdir -p "$other" && cp "$skill"/*.sh "$skill/config.env" "$other/"
expect_match '^PREFLIGHT-FAILED .*not from the user-level skill' "$(pre bash "$other/reviewer.sh" preflight)" \
  "a copy outside the user-level skill dir is refused"
alt="$fake_home/.claude/skills/gabo-reviewer"
mkdir -p "$fake_home/.claude/skills" && ln -s "$(dirname "$skill")" "$alt"
expect_match '^PREFLIGHT-OK ' "$(pre bash "$alt/scripts/reviewer.sh" preflight)" "the runtime link to the user-level copy is accepted"
chmod g+w "$skill/review-head.sh"
expect_match '^PREFLIGHT-FAILED .*review-head.sh is writable by group or others' \
  "$(pre bash "$skill/reviewer.sh" preflight)" "a group-writable script is refused"
chmod g-w "$skill/review-head.sh"
chmod o+w "$skill"
expect_match '^PREFLIGHT-FAILED .*writable by group or others' "$(pre bash "$skill/reviewer.sh" preflight)" "an other-writable scripts dir is refused"
chmod o-w "$skill"
expect_match '^PREFLIGHT-FAILED single-user mode runs as the calling user, never through sudo' \
  "$(pre SUDO_USER=root bash "$skill/reviewer.sh" preflight)" "sudo is refused"

echo "# clean checkout of the integration branch"
PRE_STATE="$work/state2" pre bash "$skill/reviewer.sh" preflight >/dev/null
echo "local edit" >>"$work/state2/base-example-org__example-repo/AGENTS.md"
expect_match '^PREFLIGHT-FAILED .*checkout of dev .* is not clean' \
  "$(PRE_STATE="$work/state2" pre bash "$skill/reviewer.sh" preflight)" "a modified base checkout is refused"
git -C "$work/state2/base-example-org__example-repo" checkout -q -- AGENTS.md
echo "stray" >"$work/state2/base-example-org__example-repo/stray.txt"
expect_match '^PREFLIGHT-FAILED .*is not clean' \
  "$(PRE_STATE="$work/state2" pre bash "$skill/reviewer.sh" preflight)" "an untracked file in the base checkout is refused"
rm "$work/state2/base-example-org__example-repo/stray.txt"
expect_match '^PREFLIGHT-OK ' "$(PRE_STATE="$work/state2" pre bash "$skill/reviewer.sh" preflight)" "a clean base checkout passes again"

echo "# login and config"
expect_match '^PREFLIGHT-FAILED the gh login is .owner., not REVIEWER_BOT_LOGIN .someone.' \
  "$(pre REVIEWER_BOT_LOGIN=someone bash "$skill/reviewer.sh" preflight)" "a configured login must match the gh login"
expect_match '^PREFLIGHT-FAILED REVIEWER_BASE_BRANCH must be the integration branch, never main' \
  "$(pre REVIEWER_BASE_BRANCH=main bash "$skill/reviewer.sh" preflight)" "main is never the base branch"
expect_match '^PREFLIGHT-FAILED REVIEWER_MODE=isolated belongs to the reviewer_placement adapter' \
  "$(pre REVIEWER_MODE=isolated bash "$skill/reviewer.sh" preflight)" "isolated mode is left to the placement adapter"
expect_match '^PREFLIGHT-FAILED review.checks is empty' \
  "$(pre REVIEWER_CHECKS=' ; ' bash "$skill/reviewer.sh" preflight)" "checks are required"
expect_match '^PREFLIGHT-FAILED review.check_env entry A=B must be an environment variable name' \
  "$(pre REVIEWER_CHECK_ENV='A=B' bash "$skill/reviewer.sh" preflight)" "a check_env value is refused"
for reserved in PATH HOME CI GOCACHE GOMODCACHE GOPATH GABO_UPGRADE_PR GH_TOKEN GITHUB_TOKEN CLAUDE_CODE_OAUTH_TOKEN REVIEWER_GH_TOKEN; do
  expect_match "^PREFLIGHT-FAILED review.check_env entry $reserved is reserved" \
    "$(pre REVIEWER_CHECK_ENV="DB_URL $reserved" bash "$skill/reviewer.sh" preflight)" "check_env refuses the reserved name $reserved"
done
expect_match '^PREFLIGHT-OK ' "$(pre REVIEWER_CHECK_ENV='DB_URL _X1' bash "$skill/reviewer.sh" preflight)" "valid check_env names pass"
expect_match '^PREFLIGHT-FAILED review.check_env entry PI_KEY is review.pi_credential_env' \
  "$(pre REVIEWER_PI_CREDENTIAL_ENV=PI_KEY REVIEWER_CHECK_ENV='DB_URL PI_KEY' bash "$skill/reviewer.sh" preflight)" \
  "check_env refuses the Pi credential name"
expect_match '^PREFLIGHT-FAILED gh cannot authenticate' \
  "$(pre REVIEWER_GH=/bin/false bash "$skill/reviewer.sh" preflight)" "no gh login is refused"
out="$(pre REVIEWER_BASE_BRANCH=main bash "$skill/reviewer.sh" head 7 "$sha")"
expect_match "^VERDICT PR #7 head $sha verdict=error merge=none$" "$out" "a refused head still ends with a VERDICT line"
expect_eq 2 "$(pre bash "$skill/reviewer.sh" head 7 abc123 >/dev/null 2>&1; echo $?)" "head needs a 40-hex sha"
expect_eq 2 "$(pre bash "$skill/reviewer.sh" head 'x;y' "$sha" >/dev/null 2>&1; echo $?)" "head needs a PR number"

echo "# reviewer engine"
O_REVIEWER=pi make_origin "$work/origin-pi.git"
pi_pre() { PRE_ORIGIN="$work/origin-pi.git" PRE_STATE="$work/state-pi" pre "$@"; }
expect_match '^PREFLIGHT-OK .* engine=pi ' "$(pi_pre REVIEWER_PI="$work/bin/pi" bash "$skill/reviewer.sh" preflight)" \
  "runtimes.reviewer: pi on the integration branch selects the Pi engine"
expect_match '^PREFLIGHT-OK .* engine=claude ' "$(pi_pre REVIEWER_ENGINE=claude bash "$skill/reviewer.sh" preflight)" \
  "REVIEWER_ENGINE in the environment overrides runtimes.reviewer"
expect_match '^PREFLIGHT-FAILED no pi binary \(REVIEWER_PI=/missing/pi\) for runtimes.reviewer pi$' \
  "$(pi_pre REVIEWER_PI=/missing/pi bash "$skill/reviewer.sh" preflight)" "a missing pi binary is refused"
expect_match '^PREFLIGHT-FAILED no claude binary \(REVIEWER_CLAUDE=/missing/claude\) for runtimes.reviewer claude$' \
  "$(pre REVIEWER_CLAUDE=/missing/claude bash "$skill/reviewer.sh" preflight)" "a missing claude binary is refused"
expect_match "^PREFLIGHT-FAILED REVIEWER_ENGINE must be claude, codex, pi or opencode \(runtimes.reviewer\), got 'gpt'$" \
  "$(pre REVIEWER_ENGINE=gpt bash "$skill/reviewer.sh" preflight)" "an unknown engine is refused"
expect_match '^PREFLIGHT-FAILED review.pi_credential_env must be an environment variable name' \
  "$(pi_pre REVIEWER_PI="$work/bin/pi" REVIEWER_PI_CREDENTIAL_ENV='a-b' bash "$skill/reviewer.sh" preflight)" "a bad credential env var name is refused"
expect_match '^PREFLIGHT-FAILED review.pi_credential_env may not name PATH' \
  "$(pi_pre REVIEWER_PI="$work/bin/pi" REVIEWER_PI_CREDENTIAL_ENV=PATH bash "$skill/reviewer.sh" preflight)" "a credential env var may not replace the review's own environment"

echo "# isolation pass marker (codex, opencode)"
marks="$fake_home/.config/gabo/reviewer-isolation"
rm -rf "$fake_home/.config/gabo"
expect_match '^PREFLIGHT-OK .* engine=claude ' "$(pre bash "$skill/reviewer.sh" preflight)" "claude needs no isolation pass marker"
expect_match '^PREFLIGHT-OK .* engine=pi ' "$(pi_pre REVIEWER_PI="$work/bin/pi" bash "$skill/reviewer.sh" preflight)" \
  "pi needs no isolation pass marker"
O_REVIEWER=codex make_origin "$work/origin-codex.git"
codex_pre() { PRE_ORIGIN="$work/origin-codex.git" PRE_STATE="$work/state-codex" pre REVIEWER_CODEX="$work/bin/codex" "$@"; }
out="$(codex_pre bash "$skill/reviewer.sh" preflight)"
expect_match "^PREFLIGHT-FAILED runtimes.reviewer codex has no isolation test pass for codex-cli 0.163.0: no marker $marks/codex; run the gabo-reviewer-config isolation test \(isolation-test.sh codex\)$" \
  "$out" "codex without a pass marker is refused, naming the missing isolation test"
mkdir -p "$marks"
echo "codex-cli 0.162.0" >"$marks/codex"
expect_match "^PREFLIGHT-FAILED runtimes.reviewer codex has no isolation test pass for codex-cli 0.163.0: $marks/codex records codex-cli 0.162.0; run the gabo-reviewer-config isolation test \(isolation-test.sh codex\)$" \
  "$(codex_pre bash "$skill/reviewer.sh" preflight)" "a marker for another codex version is refused"
echo "codex-cli 0.163.0" >"$marks/codex"
expect_match '^PREFLIGHT-OK .* engine=codex ' "$(codex_pre bash "$skill/reviewer.sh" preflight)" \
  "runtimes.reviewer: codex with a marker matching codex --version passes"
expect_match '^PREFLIGHT-FAILED .*isolation test' "$(codex_pre XDG_CONFIG_HOME="$work/xdg" bash "$skill/reviewer.sh" preflight)" \
  "the marker is read under XDG_CONFIG_HOME when it is set"
mkdir -p "$work/xdg/gabo/reviewer-isolation" && echo "codex-cli 0.163.0" >"$work/xdg/gabo/reviewer-isolation/codex"
expect_match '^PREFLIGHT-OK .* engine=codex ' "$(codex_pre XDG_CONFIG_HOME="$work/xdg" bash "$skill/reviewer.sh" preflight)" \
  "a matching marker under XDG_CONFIG_HOME passes"
expect_match '^PREFLIGHT-FAILED no codex binary \(REVIEWER_CODEX=gabo-test-no-codex\) for runtimes.reviewer codex$' \
  "$(codex_pre REVIEWER_CODEX=gabo-test-no-codex bash "$skill/reviewer.sh" preflight)" "codex missing from PATH is refused naming REVIEWER_CODEX"
# The version runs in a scrubbed environment: the stub reads it from its context file.
STUB_LOG="$fx/calls.log" CODEX_VERSION=0.170.0 write_stub_context "$work/bin"
expect_match '^PREFLIGHT-FAILED .*pass for codex-cli 0.170.0: .*isolation test \(isolation-test.sh codex\)$' \
  "$(pre REVIEWER_ENGINE=codex REVIEWER_CODEX="$work/bin/codex" bash "$skill/reviewer.sh" preflight)" \
  "REVIEWER_ENGINE=codex is accepted and an upgraded codex needs a new isolation test"
rm -f "$work/bin/stub-context"
oc_pre() { pre REVIEWER_ENGINE=opencode REVIEWER_OPENCODE="$work/bin/opencode" "$@"; }
expect_match "^PREFLIGHT-FAILED runtimes.reviewer opencode has no isolation test pass for 1.0.0: no marker $marks/opencode; run the gabo-reviewer-config isolation test \(isolation-test.sh opencode\)$" \
  "$(oc_pre bash "$skill/reviewer.sh" preflight)" "opencode without a pass marker is refused, naming the missing isolation test"
echo "0.9.0" >"$marks/opencode"
expect_match '^PREFLIGHT-FAILED .*opencode records 0.9.0; ' "$(oc_pre bash "$skill/reviewer.sh" preflight)" \
  "a marker for another opencode version is refused"
echo "1.0.0" >"$marks/opencode"
expect_match '^PREFLIGHT-OK .* engine=opencode ' "$(oc_pre bash "$skill/reviewer.sh" preflight)" \
  "opencode with a marker matching opencode --version passes"
expect_match '^PREFLIGHT-FAILED no opencode binary \(REVIEWER_OPENCODE=gabo-test-no-opencode\) for runtimes.reviewer opencode$' \
  "$(oc_pre REVIEWER_OPENCODE=gabo-test-no-opencode bash "$skill/reviewer.sh" preflight)" "opencode missing from PATH is refused naming REVIEWER_OPENCODE"
rm -rf "$fake_home/.config/gabo"

echo "# engine argument (runtimes.reviewer: session, intent 0013 S11, D5)"
# refused <label> <reason regex> [VAR=value...] <cmd...>: runs pre with a
# fresh call log; expects PREFLIGHT-FAILED <reason>, exit 2, and no engine
# run, no status or comment posted and no merge.
refusals=0
refused() {
  local label="$1" want="$2" log out code=0
  shift 2
  refusals=$((refusals + 1))
  log="$work/refused-$refusals.log"
  : >"$log"
  out="$(pre STUB_LOG="$log" "$@")" || code=$?
  expect_match "^PREFLIGHT-FAILED $want" "$out" "$label"
  expect_eq 2 "$code" "$label: exits 2"
  check "$label: no engine runs, nothing is posted or merged" \
    bash -c "! grep -qE '^(claude |pi |codex exec|opencode run|gh pr merge|gh api -X POST|STATUS )' '$log'"
}
O_REVIEWER=session O_ASSISTANT='[claude, pi]' make_origin "$work/origin-session.git"
sess() { PRE_ORIGIN="$work/origin-session.git" PRE_STATE="$work/state-session" pre REVIEWER_PI="$work/bin/pi" "$@"; }
s_refused() { PRE_ORIGIN="$work/origin-session.git" PRE_STATE="$work/state-session" refused "$@"; }
expect_match '^PREFLIGHT-OK .* engine=pi ' "$(sess bash "$skill/reviewer.sh" preflight engine pi)" \
  "session: engine pi listed in runtimes.assistant on the integration branch is accepted"
expect_match '^PREFLIGHT-OK .* engine=claude ' "$(sess bash "$skill/reviewer.sh" preflight engine claude)" \
  "session: engine claude listed in runtimes.assistant is accepted"
expect_match '^PREFLIGHT-OK .* engine=claude ' "$(pre REVIEWER_ENGINE=session bash "$skill/reviewer.sh" preflight engine claude)" \
  "REVIEWER_ENGINE=session stands in for runtimes.reviewer and takes a listed engine"
expect_match '^PREFLIGHT-OK .* engine=claude ' "$(pre bash "$skill/reviewer.sh" preflight)" \
  "a fixed runtimes.reviewer without an engine argument still passes"
# (a) a fixed runtimes.reviewer refuses the argument.
for m in preflight "head 7 $sha" "due 7 $sha" list-due once; do
  # shellcheck disable=SC2086
  refused "(a) runtimes.reviewer: claude refuses an engine argument ($m)" \
    "the engine argument is accepted only when runtimes.reviewer is session; runtimes.reviewer on dev is claude$" \
    bash "$skill/reviewer.sh" $m engine claude
done
out="$(pre bash "$skill/reviewer.sh" head 7 "$sha" engine pi)"
expect_match "^VERDICT PR #7 head $sha verdict=error merge=none$" "$out" "(a) a refused head still ends with a VERDICT line"
# (b) the engine must be listed in runtimes.assistant on the integration branch.
s_refused "(b) session refuses engine codex not listed in runtimes.assistant [claude, pi]" \
  "engine codex is not listed in runtimes.assistant on dev \(claude pi\)$" bash "$skill/reviewer.sh" preflight engine codex
s_refused "(b) session refuses engine opencode not listed in runtimes.assistant" \
  "engine opencode is not listed in runtimes.assistant on dev" bash "$skill/reviewer.sh" head 7 "$sha" engine opencode
# (c) session without the argument fails in every mode.
for m in preflight once list-due "due 7 $sha" "head 7 $sha"; do
  # shellcheck disable=SC2086
  s_refused "(c) session without an engine argument is refused ($m)" \
    "runtimes.reviewer on dev is session: pass the engine of the requesting wave as engine <claude\|codex\|pi\|opencode>$" \
    bash "$skill/reviewer.sh" $m
done
# (d) the value must be one of the four engines, a bare lowercase word,
# checked before anything else runs (no state dir, no fetch).
for v in gpt Claude 'pi;id' 'pi pi' ' pi' 'pi ' '$(touch '"$work"'/pwned)' '`touch '"$work"'/pwned`' ../pi pi/.. session '' $'pi\nclaude'; do
  PRE_ORIGIN="$work/origin-session.git" PRE_STATE="$work/state-d" refused "(d) engine value $(printf '%q' "$v") is refused" \
    "the engine argument must be " bash "$skill/reviewer.sh" preflight engine "$v"
done
PRE_ORIGIN="$work/origin-session.git" PRE_STATE="$work/state-d" refused "(d) engine with no value is refused" \
  "the engine argument needs exactly one value" bash "$skill/reviewer.sh" preflight engine
PRE_ORIGIN="$work/origin-session.git" PRE_STATE="$work/state-d" refused "(d) engine with two values is refused" \
  "the engine argument needs exactly one value" bash "$skill/reviewer.sh" head 7 "$sha" engine pi claude
check "(d) a bad value is never executed" bash -c "! test -e '$work/pwned'"
check "(d) a bad value is refused before any use (no state dir, no fetch)" bash -c "! test -e '$work/state-d'"
expect_match "^PREFLIGHT-FAILED the engine argument must be a bare lowercase word$" \
  "$(PRE_STATE="$work/state-d" pre bash "$skill/reviewer.sh" preflight engine '$(id)')" "(d) the refusal never echoes a value that is not a bare word"
expect_eq 2 "$(pre bash "$skill/reviewer.sh" preflight extra >/dev/null 2>&1; echo $?)" "any other extra argument is a usage error"
# (e) is end to end in review.test.sh: session only on the PR head.
# (f) REVIEWER_ENGINE set to a fixed engine closes the argument like (a), even
# when runtimes.reviewer on the integration branch is session.
s_refused "(f) REVIEWER_ENGINE=claude refuses an engine argument" \
  "the engine argument is accepted only when runtimes.reviewer is session; REVIEWER_ENGINE is claude$" \
  REVIEWER_ENGINE=claude bash "$skill/reviewer.sh" preflight engine pi
s_refused "(f) REVIEWER_ENGINE=pi refuses an engine argument (head)" \
  "the engine argument is accepted only when runtimes.reviewer is session; REVIEWER_ENGINE is pi$" \
  REVIEWER_ENGINE=pi bash "$skill/reviewer.sh" head 7 "$sha" engine pi
# (g) the isolation pass marker applies to the resolved engine.
O_REVIEWER=session O_ASSISTANT='[claude, codex, opencode]' make_origin "$work/origin-session-codex.git"
rm -rf "$fake_home/.config/gabo"
PRE_ORIGIN="$work/origin-session-codex.git" PRE_STATE="$work/state-session-codex" refused \
  "(g) session with listed engine codex and no isolation pass marker is refused, naming the isolation test" \
  "runtimes.reviewer session \(engine codex\) has no isolation test pass for codex-cli 0.163.0: no marker $marks/codex; run the gabo-reviewer-config isolation test \(isolation-test.sh codex\)$" \
  REVIEWER_CODEX="$work/bin/codex" bash "$skill/reviewer.sh" preflight engine codex
PRE_ORIGIN="$work/origin-session-codex.git" PRE_STATE="$work/state-session-codex" refused \
  "(g) session with listed engine opencode and no isolation pass marker is refused (head)" \
  "runtimes.reviewer session \(engine opencode\) has no isolation test pass .*isolation-test.sh opencode\)$" \
  REVIEWER_OPENCODE="$work/bin/opencode" bash "$skill/reviewer.sh" head 7 "$sha" engine opencode
mkdir -p "$marks" && echo "codex-cli 0.163.0" >"$marks/codex"
expect_match '^PREFLIGHT-OK .* engine=codex ' \
  "$(PRE_ORIGIN="$work/origin-session-codex.git" PRE_STATE="$work/state-session-codex" pre REVIEWER_CODEX="$work/bin/codex" bash "$skill/reviewer.sh" preflight engine codex)" \
  "(g) with a matching marker the listed codex engine passes"
rm -rf "$fake_home/.config/gabo"

echo "# placement"
O_PLACEMENT=github-actions make_origin "$work/origin-gha.git"
expect_match '^PREFLIGHT-FAILED reviewer.placement github-actions is planned for a later release; only local runs here' \
  "$(PRE_ORIGIN="$work/origin-gha.git" PRE_STATE="$work/state-gha" pre bash "$skill/reviewer.sh" preflight)" \
  "a placement other than local fails as planned for a later release"

echo "# base version minimum"
O_STAMP=0.4.0 make_origin "$work/origin-stamp.git"
stamp() { PRE_ORIGIN="$work/origin-stamp.git" PRE_STATE="$work/state-stamp" pre "$@"; }
expect_match '^PREFLIGHT-OK .* minimum=0.4.0$' "$(PRE_GABO_VERSION=0.4.0 stamp bash "$skill/reviewer.sh" preflight)" \
  "a binary equal to the base stamp passes"
expect_match '^PREFLIGHT-OK .* minimum=0.4.0$' "$(PRE_GABO_VERSION=0.10.1 stamp bash "$skill/reviewer.sh" preflight)" \
  "a newer binary passes (versions compare numerically)"
out="$(PRE_GABO_VERSION=0.3.9 stamp bash "$skill/reviewer.sh" preflight)"
expect_match '^PREFLIGHT-FAILED gabo 0.3.9 is older than the minimum 0.4.0 .*: go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@v0.4.0$' \
  "$out" "an older binary is refused with the TRD 9.4 upgrade command"
out="$(PRE_GABO_VERSION=dev stamp bash "$skill/reviewer.sh" preflight)"
expect_match '^PREFLIGHT-FAILED .*go install .*@v0.4.0$' "$out" "a binary without a release version is refused"
out="$(stamp REVIEWER_GABO_BIN="$work/missing/gabo" bash "$skill/reviewer.sh" preflight)"
expect_match '^PREFLIGHT-FAILED no gabo binary .*go install .*@v0.4.0$' "$out" "no binary with a base stamp is refused"
expect_match '^PREFLIGHT-OK .* minimum=none$' "$(pre REVIEWER_GABO_BIN="$work/missing/gabo" bash "$skill/reviewer.sh" preflight)" \
  "no stamp on the base means no minimum"
check "the reviewer never updates the binary" bash -c "! grep -E '^gabo-call (init|upgrade|install|update)' '$fx/calls.log'"

echo "# env file"
envf="$work/reviewer.env"
printf 'CLAUDE_CODE_OAUTH_TOKEN=tok\n' >"$envf"
chmod 644 "$envf"
expect_match '^PREFLIGHT-FAILED .*must be mode 600' "$(pre REVIEWER_ENV_FILE="$envf" bash "$skill/reviewer.sh" preflight 2>&1)" "a readable env file is refused"
chmod 600 "$envf"
printf 'REVIEWER_REPO=$(touch %s/pwned)/x\n' "$work" >"$envf"
pre REVIEWER_ENV_FILE="$envf" bash "$skill/reviewer.sh" preflight >/dev/null 2>&1
check "env files are read as data, never executed" bash -c "! test -e '$work/pwned'"

echo "# helpers"
helpers() { HOME="$fake_home" REVIEWER_BOT_LOGIN=owner bash -c "source '$skill/common.sh'; $1"; }
y="$caller/gabo.yaml"
expect_eq "example-org/example-repo" "$(helpers "yaml_get '$y' repo.name")" "yaml_get reads a nested scalar"
expect_eq "false" "$(helpers "yaml_get '$y' review.auto_merge")" "yaml_get drops a trailing comment"
expect_eq "go -C apps/cli build ./...|go -C apps/cli vet ./...|go -C apps/cli test ./..." \
  "$(helpers "yaml_get '$y' review.checks | paste -sd'|'")" "yaml_get reads a block list"
expect_eq ".github/|AGENTS.md|CLAUDE.md|gabo.yaml|.claude/" \
  "$(helpers "yaml_get '$y' review.human_only_paths | paste -sd'|'")" "yaml_get reads a flow list"
expect_eq "gbai-reviewer-bot" "$(helpers "yaml_get '$y' reviewer.signing.marker")" "yaml_get reads three levels"
expect_eq "" "$(helpers "yaml_get '$y' review.missing")" "a missing key is empty"
expect_eq "env-wins" "$(HOME="$fake_home" REVIEWER_REPO=env-wins bash -c "source '$skill/common.sh'; echo \"\$REPO\"")" "the environment sets the repo"
expect_eq "go -C apps/cli build ./...|go -C apps/cli vet ./..." \
  "$(HOME="$fake_home" REVIEWER_CHECKS='go -C apps/cli build ./...; go -C apps/cli vet ./...' bash -c "source '$skill/common.sh'; parse_checks | paste -sd'|'")" \
  "checks split on ';'"
hop() { HOME="$fake_home" REVIEWER_HUMAN_ONLY_PATHS="${HOP:-.github/ AGENTS.md CLAUDE.md gabo.yaml .claude/}" bash -c "source '$skill/common.sh'; $1"; }
for p in .github/workflows/x.yml .claude/agents/a.md AGENTS.md apps/cli/CLAUDE.md gabo.yaml .gabo/VERSION .gabo/skills/gabo-coder/SKILL.md; do
  check "human-only: $p" hop "is_human_only '$p'"
done
for p in .gabo/VERSION .gabo/skills/gabo-coder/SKILL.md apps/cli/internal/skills/gabo-reviewer/scripts/common.sh \
  skills/gabo-reviewer/SKILL.md; do
  check "always human-only, even with no configured paths: $p" \
    env HOME="$fake_home" REVIEWER_HUMAN_ONLY_PATHS=" " bash -c "source '$skill/common.sh'; is_human_only '$p'"
done
for p in apps/cli/main.go docs/product/PRD.md xAGENTS.md apps/VERSION .gabo/skillsx/a .gabo/VERSION.bak; do
  check "not human-only: $p" hop "! is_human_only '$p'"
done
vge() { HOME="$fake_home" bash -c "source '$skill/common.sh'; version_at_least '$1' '$2'"; }
check "0.4.0 >= 0.4.0" vge 0.4.0 0.4.0
check "0.10.0 >= 0.9.9" vge 0.10.0 0.9.9
check "1.0.0 >= 0.99.99" vge 1.0.0 0.99.99
check "not 0.3.9 >= 0.4.0" bash -c "$(declare -f vge); fake_home='$fake_home' skill='$skill'; ! vge 0.3.9 0.4.0"
check "not 0.4.0-rc.1 >= 0.4.0" bash -c "$(declare -f vge); fake_home='$fake_home' skill='$skill'; ! vge 0.4.0-rc.1 0.4.0"
check "not dev >= 0.4.0" bash -c "$(declare -f vge); fake_home='$fake_home' skill='$skill'; ! vge dev 0.4.0"
c() { jq -nc --arg l "$1" --arg at "$2" --arg b "$3" '{user:{login:$l}, created_at:$at, body:$b}'; }
m="<!-- gbai-reviewer-bot sha=$sha -->"
comments="$({
  c owner 2026-10-20T10:00:00Z $'No blocking issues found.\n'"$m"
  c owner 2026-10-20T13:00:00Z $'Blocking issues found.\n'"$m"
  c intruder 2026-10-20T14:00:00Z $'No blocking issues found.\n'"$m"
  c owner 2026-10-20T15:00:00Z $'No blocking issues found.\nno marker here'
} | jq -s .)"
expect_eq "blocking $sha" "$(helpers "latest_bot_verdict '$comments' 2026-10-20T11:00:00Z")" \
  "the newest marker comment by the bot login newer than the head wins"
expect_eq "" "$(helpers "latest_bot_verdict '$comments' 2026-10-20T16:00:00Z")" "comments older than the head never count"
expect_eq "" "$(HOME="$fake_home" REVIEWER_BOT_LOGIN=owner REVIEWER_MARKER=team-bot bash -c "source '$skill/common.sh'; latest_bot_verdict '$comments' 2026-10-20T11:00:00Z")" \
  "only the configured reviewer.signing marker counts"
check "a head with a verdict comment by the bot login is found" helpers "has_verdict_comment '$comments' $sha"
check "another head has no verdict comment" helpers "! has_verdict_comment '$comments' fedcba9876543210fedcba9876543210fedcba98"
check "another login's marker is not a verdict" \
  env HOME="$fake_home" REVIEWER_BOT_LOGIN=stranger bash -c "source '$skill/common.sh'; ! has_verdict_comment \"\$1\" $sha" _ "$comments"


echo "# engine set"
# A copy of the rows of engineSetRows in
# apps/cli/internal/engineconfig/engines_test.go (intent 0013 S2, D2):
# coder|reviewer|assistant|the set in role order. declared_agent_dirs
# mirrors engineconfig.Engines: each engine of the set gives its agent dir,
# session gives none. Every runtime skill, session's too, declares
# .<engine>/agents so a stray engine shows up.
engine_rows=(
  "session|opencode|[claude, pi]|opencode claude pi"
  "claude|session|[claude, pi]|claude pi"
  "session|session|[pi]|pi"
  "pi|pi|[codex, claude]|pi codex claude"
  "claude|claude|claude|claude"
  "pi|claude|claude|pi claude"
  "codex|claude|pi|codex claude pi"
  "codex|opencode||codex opencode"
)
# engine_base <dir> <coder> <reviewer> <assistant>: a reviewer checkout of
# the integration branch with that runtimes block and every runtime skill.
engine_base() {
  local e
  mkdir -p "$1"
  {
    echo "runtimes:"
    echo "  coder: $2"
    echo "  reviewer: $3"
    [[ -z "$4" ]] || echo "  assistant: $4"
  } >"$1/gabo.yaml"
  for e in claude codex opencode pi session; do
    mkdir -p "$1/.gabo/skills/gabo-runtime-$e"
    printf -- '---\nname: gabo-runtime-%s\nmetadata:\n  agent_dir: .%s/agents\n---\n' "$e" "$e" >"$1/.gabo/skills/gabo-runtime-$e/SKILL.md"
  done
}
agent_dirs() { HOME="$fake_home" bash -c "source '$skill/common.sh'; BASE_DIR='$1'; declared_agent_dirs | paste -sd' ' -"; }
i=0
for row in "${engine_rows[@]}"; do
  i=$((i + 1))
  IFS='|' read -r coder reviewer assistant set <<<"$row"
  engine_base "$work/engines-$i" "$coder" "$reviewer" "$assistant"
  want="$(for e in $set; do echo ".$e/agents/"; done | sort -u | paste -sd' ' -)"
  expect_eq "$want" "$(agent_dirs "$work/engines-$i")" "engine set of coder $coder, reviewer $reviewer, assistant ${assistant:-none}"
done
engine_base "$work/engines-block" session opencode ""
printf '  assistant:\n    - claude\n    - pi\n' >>"$work/engines-block/gabo.yaml"
expect_eq ".claude/agents/ .opencode/agents/ .pi/agents/" "$(agent_dirs "$work/engines-block")" "a block list of assistants"

finish
