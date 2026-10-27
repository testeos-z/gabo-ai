#!/usr/bin/env bash
# Tests for merge-head.sh: the merge gate (integration branch only, pinned to
# the reviewed head, signed clean verdict, success statuses, no human-only
# paths). Run: bash tests/merge-head.test.sh
set -uo pipefail
source "$(dirname "$0")/lib.sh"
source "$(dirname "$0")/stubs.sh"

work="$(new_tmp)"
fake_home="$work/home"
skill="$(install_skill "$fake_home")"
make_stubs "$work/bin"
sha=0123456789abcdef0123456789abcdef01234567
other=fedcba9876543210fedcba9876543210fedcba98

# gate <name> [VAR=value...]: a fixture where every rule holds, then the
# overrides; prints the outcome line.
gate() {
  local name="$1"
  shift
  local fx="$work/$name"
  mkdir -p "$fx"
  echo owner >"$fx/user_login"
  jq -n --arg s "${G_HEAD:-$sha}" --arg b "${G_BASE:-dev}" --arg st "${G_STATE:-OPEN}" \
    '{number:7, state:$st, baseRefName:$b, isDraft:false, isCrossRepository:false, headRefOid:$s}' >"$fx/pr_7.json"
  jq -n --arg t "${G_TESTS:-success}" --arg l "${G_STATUS_LOGIN:-owner}" \
    '[{context:"reviewer/tests",state:$t,creator:{login:$l}},{context:"reviewer/review",state:"success",creator:{login:$l}}]' >"$fx/statuses_$sha.json"
  echo '{"commit":{"committer":{"date":"2026-10-20T11:00:00Z"}}}' >"$fx/commit_$sha.json"
  jq -n --arg b "${G_FIRST:-No blocking issues found.}"$'\n\n'"<!-- ${G_MARKER:-gbai-reviewer-bot} sha=${G_SIGNED:-$sha} -->"$'\n🤖 gabo reviewer' \
    --arg l "${G_COMMENT_LOGIN:-owner}" '[{user:{login:$l}, created_at:"2026-10-20T12:00:00Z", body:$b}]' >"$fx/comments_7.json"
  local files='[{"filename":"apps/cli/main.go"}]'
  echo "${G_FILES:-$files}" >"$fx/files_7"
  (
    export HOME="$fake_home" STUB_LOG="$fx/calls.log" FIXTURE="$fx" REVIEWER_GH="$work/bin/gh" \
      GBAI_REVIEWER_STATE_DIR="$fx/state" REVIEWER_AUTO_MERGE=true REVIEWER_NOW=2026-10-20T13:00:00Z \
      REVIEWER_REPO=testeos-z/gabo-ai REVIEWER_CHECKS='go test ./...' \
      REVIEWER_HUMAN_ONLY_PATHS='.claude/ .github/ gbai.yaml AGENTS.md CLAUDE.md'
    for kv in "$@"; do export "${kv?}"; done
    bash "$skill/merge-head.sh" 7 "$sha" 2>&1 | tail -n 1 | cut -d' ' -f2-
  )
}

expect_eq "MERGED #7 $sha" "$(gate ok)" "every rule holds: merges"
check "merge is pinned and plain" grep -qxF "gh pr merge 7 --repo testeos-z/gabo-ai --merge --match-head-commit $sha" "$work/ok/calls.log"
expect_match '^MERGE-SKIPPED .*REVIEWER_AUTO_MERGE' "$(gate off REVIEWER_AUTO_MERGE=false)" "auto merge off: skipped"
expect_match '^MERGE-SKIPPED .*targets main; the reviewer never merges into it' "$(G_BASE=main gate main)" "never merges into main"
check "no merge call for main" bash -c "! grep -q '^gh pr merge' '$work/main/calls.log'"
expect_match '^MERGE-SKIPPED .*does not target dev' "$(G_BASE=feature gate feature)" "other bases are skipped"
expect_match '^MERGE-SKIPPED .*head moved' "$(G_HEAD=$other gate moved)" "a moved head is skipped"
expect_match '^MERGE-SKIPPED .*not open' "$(G_STATE=MERGED gate closed)" "a closed PR is skipped"
expect_match '^MERGE-SKIPPED .* reviewer/tests is failure' "$(G_TESTS=failure gate tests)" "failing tests status is skipped"
expect_match '^MERGE-SKIPPED .* reviewer/tests is missing' "$(G_STATUS_LOGIN=intruder gate forged_status)" "statuses by another login never count"
expect_match '^MERGE-SKIPPED .*no reviewer verdict comment' "$(G_COMMENT_LOGIN=intruder gate forged_comment)" "a marker by another login never counts"
expect_match '^MERGE-SKIPPED .*not clean' "$(G_FIRST='Blocking issues found.' gate blocking)" "a blocking verdict is skipped"
expect_match '^MERGE-SKIPPED .*signs another head' "$(G_SIGNED=$other gate signed_other)" "a verdict for another head is skipped"
expect_eq "NEEDS-HUMAN-MERGE #7 $sha" "$(G_FILES='[{"filename":".claude/agents/x.md"}]' gate human)" "human-only paths need a human merge"
check "needs-human comment carries no verdict marker" bash -c "! grep -q 'gbai-reviewer-bot sha=' '$work/human/posted_7.md'"
expect_match '^MERGE-FAILED ' "$(gate refused MERGE_FAIL=1)" "a refused merge is MERGE-FAILED"

echo "# built-in human-only paths"
expect_eq "NEEDS-HUMAN-MERGE #7 $sha" "$(G_FILES='[{"filename":".gbai/skills/gabo-coder/SKILL.md"}]' gate catalog)" \
  "the catalog subtree .gbai/skills/ always needs a human merge"
check "no merge call for the catalog" bash -c "! grep -q '^gh pr merge' '$work/catalog/calls.log'"
expect_eq "NEEDS-HUMAN-MERGE #7 $sha" "$(G_FILES='[{"filename":".gbai/VERSION"}]' gate stamp REVIEWER_HUMAN_ONLY_PATHS=' ')" \
  "the stamp .gbai/VERSION needs a human merge even with no configured human-only paths"
check "no merge call for the stamp" bash -c "! grep -q '^gh pr merge' '$work/stamp/calls.log'"

echo "# reviewer.signing marker"
expect_eq "MERGED #7 $sha" "$(G_MARKER=team-bot gate team REVIEWER_MARKER=team-bot)" "the configured marker signs the verdict"
expect_match '^MERGE-SKIPPED .*no reviewer verdict comment' "$(gate default_marker REVIEWER_MARKER=team-bot)" \
  "a verdict with another marker never counts"

finish
