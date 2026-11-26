#!/usr/bin/env bash
# Tests for validate_base_branch (intent 0032 S6): the caller's base must be
# the integration branch or an intent folder existing under
# docs/product/intents/ on it; anything else is skipped without verdict or
# merge. Run: bash tests/base-branch.test.sh
set -uo pipefail
source "$(dirname "$0")/lib.sh"
source "$(dirname "$0")/stubs.sh"

work="$(new_tmp)"
fake_home="$work/home"
skill="$(install_skill "$fake_home")"
make_stubs "$work/bin"
sha=0123456789abcdef0123456789abcdef01234567

echo "# validate_base_branch unit"
# probe <name> [VAR=value...]: sources common.sh with the stub gh and runs
# the check; prints "rc=<n> <message>".
probe() {
  local name="$1"
  shift
  local fx="$work/$name"
  mkdir -p "$fx"
  (
    export HOME="$fake_home" STUB_LOG="$fx/calls.log" FIXTURE="$fx" REVIEWER_GH="$work/bin/gh" \
      REVIEWER_REPO=example-org/example-repo REVIEWER_BASE_BRANCH=dev
    for kv in "$@"; do export "${kv?}"; done
    # shellcheck disable=SC1091
    source "$skill/common.sh"
    if msg="$(validate_base_branch)"; then echo "rc=0 $msg"; else echo "rc=1 $msg"; fi
  )
}

expect_eq "rc=0 " "$(probe unit_dev)" "the integration branch is accepted without an API call"
check "dev: no folder lookup" bash -c "! grep -q 'contents/docs/product/intents' '$work/unit_dev/calls.log'"
mkdir -p "$work/unit_intent"
echo '[]' >"$work/unit_intent/intent_dir_0000-probe.json"
expect_eq "rc=0 " "$(probe unit_intent REVIEWER_BASE_BRANCH=0000-probe)" "an existing intent folder is accepted"
expect_match '^rc=1 .*does not exist on dev' "$(probe unit_unknown REVIEWER_BASE_BRANCH=0099-unknown)" \
  "an unknown folder is refused without verdict or merge"
expect_match '^rc=1 .*(never merges into it|not the integration branch)' "$(probe unit_main REVIEWER_BASE_BRANCH=main)" \
  "main is refused"
# A transient folder-check failure refuses with its own message, never as a
# missing folder (reviewer finding on PR 913).
expect_match '^rc=1 .*could not be verified' "$(probe unit_transient REVIEWER_BASE_BRANCH=0000-probe G_API_FAIL_CONTENTS=1)" \
  "an unverifiable intent folder is refused without verdict or merge"

echo "# merge-head integration"
# gate <name> [VAR=value...]: a fixture where every rule holds, then the
# overrides; prints the outcome line.
gate() {
  local name="$1"
  shift
  local fx="$work/$name"
  mkdir -p "$fx"
  echo owner >"$fx/user_login"
  jq -n --arg s "$sha" --arg b "${G_BASE:-dev}" \
    '{number:7, state:"OPEN", baseRefName:$b, isDraft:false, isCrossRepository:false, headRefOid:$s}' >"$fx/pr_7.json"
  jq -n --arg l owner \
    '[{context:"reviewer/tests",state:"success",creator:{login:$l}},{context:"reviewer/review",state:"success",creator:{login:$l}}]' >"$fx/statuses_$sha.json"
  echo '{"commit":{"committer":{"date":"2026-10-20T11:00:00Z"}}}' >"$fx/commit_$sha.json"
  jq -n --arg b 'No blocking issues found.'$'\n\n<!-- gbai-reviewer-bot sha='"$sha"' -->'$'\n🤖 gabo reviewer' \
    --arg l owner '[{user:{login:$l}, created_at:"2026-10-20T12:00:00Z", body:$b}]' >"$fx/comments_7.json"
  echo '[{"filename":"apps/cli/main.go"}]' >"$fx/files_7"
  (
    export HOME="$fake_home" STUB_LOG="$fx/calls.log" FIXTURE="$fx" REVIEWER_GH="$work/bin/gh" \
      GABO_REVIEWER_STATE_DIR="$fx/state" REVIEWER_AUTO_MERGE=true REVIEWER_NOW=2026-10-20T13:00:00Z \
      REVIEWER_REPO=example-org/example-repo REVIEWER_CHECKS='go test ./...' \
      REVIEWER_HUMAN_ONLY_PATHS='.claude/ .github/ gabo.yaml AGENTS.md CLAUDE.md'
    for kv in "$@"; do export "${kv?}"; done
    cd "$fx" || exit 1
    bash "$skill/merge-head.sh" 7 "$sha" >"$fx/out.log" 2>&1
    tail -n 1 "$fx/out.log" | cut -d' ' -f2-
  )
}

expect_match '^MERGE-SKIPPED .*does not exist on dev' \
  "$(G_BASE=0099-unknown gate m_unknown REVIEWER_BASE_BRANCH=0099-unknown)" \
  "merge into an unknown base is skipped"
check "unknown base: no merge call" bash -c "! grep -q '^gh pr merge' '$work/m_unknown/calls.log'"
mkdir -p "$work/m_intent"
echo '[]' >"$work/m_intent/intent_dir_0000-probe.json"
expect_eq "MERGED #7 $sha" \
  "$(G_BASE=0000-probe gate m_intent REVIEWER_BASE_BRANCH=0000-probe)" \
  "merge into an existing intent branch proceeds"

echo "# review-head integration"
# review <name> [VAR=value...]: runs review-head.sh up to the base gate.
review() {
  local name="$1"
  shift
  local fx="$work/$name"
  mkdir -p "$fx"
  echo owner >"$fx/user_login"
  jq -n --arg s "$sha" --arg b "${G_BASE:-dev}" \
    '{number:7, state:"OPEN", baseRefName:$b, isDraft:false, isCrossRepository:false, headRefOid:$s}' >"$fx/pr_7.json"
  (
    export HOME="$fake_home" STUB_LOG="$fx/calls.log" FIXTURE="$fx" REVIEWER_GH="$work/bin/gh" \
      PATH="$work/bin:$PATH" GABO_REVIEWER_STATE_DIR="$fx/state" REVIEWER_REPO=example-org/example-repo \
      REVIEWER_CHECKS='go test ./...' REVIEWER_ENGINE=claude REVIEWER_AUTO_MERGE=true
    for kv in "$@"; do export "${kv?}"; done
    cd "$fx" || exit 1
    bash "$skill/review-head.sh" 7 "$sha" >"$fx/out.log" 2>&1
    tail -n 1 "$fx/out.log" | cut -d' ' -f2-
  )
}

expect_match '^SKIP .*does not exist on dev' \
  "$(G_BASE=0099-unknown review r_unknown REVIEWER_BASE_BRANCH=0099-unknown)" \
  "review of an unknown base is skipped without verdict"
check "unknown base: no verdict comment posted" bash -c "[[ ! -f '$work/r_unknown/posted_7.md' ]]"

finish
