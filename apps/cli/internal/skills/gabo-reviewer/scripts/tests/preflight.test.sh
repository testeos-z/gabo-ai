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
check "the reviewer keeps its own checkout of the integration branch" test -f "$work/state/base/gabo.yaml"
expect_eq "$DEV_SHA" "$(git -C "$work/state/base" rev-parse HEAD)" "its checkout is the integration branch tip"
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
echo "local edit" >>"$work/state2/base/AGENTS.md"
expect_match '^PREFLIGHT-FAILED .*checkout of dev .* is not clean' \
  "$(PRE_STATE="$work/state2" pre bash "$skill/reviewer.sh" preflight)" "a modified base checkout is refused"
git -C "$work/state2/base" checkout -q -- AGENTS.md
echo "stray" >"$work/state2/base/stray.txt"
expect_match '^PREFLIGHT-FAILED .*is not clean' \
  "$(PRE_STATE="$work/state2" pre bash "$skill/reviewer.sh" preflight)" "an untracked file in the base checkout is refused"
rm "$work/state2/base/stray.txt"
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
expect_match '^PREFLIGHT-FAILED gh cannot authenticate' \
  "$(pre REVIEWER_GH=/bin/false bash "$skill/reviewer.sh" preflight)" "no gh login is refused"
out="$(pre REVIEWER_BASE_BRANCH=main bash "$skill/reviewer.sh" head 7 "$sha")"
expect_match "^VERDICT PR #7 head $sha verdict=error merge=none$" "$out" "a refused head still ends with a VERDICT line"
expect_eq 2 "$(pre bash "$skill/reviewer.sh" head 7 abc123 >/dev/null 2>&1; echo $?)" "head needs a 40-hex sha"
expect_eq 2 "$(pre bash "$skill/reviewer.sh" head 'x;y' "$sha" >/dev/null 2>&1; echo $?)" "head needs a PR number"

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

finish
