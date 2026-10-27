#!/usr/bin/env bash
# End-to-end tests of one review: reviewer.sh head -> review-head.sh ->
# merge-head.sh, offline with fake gh, claude, go and gbai, and a local bare
# repository standing in for GitHub.
# Run: bash tests/review.test.sh
set -uo pipefail
source "$(dirname "$0")/lib.sh"
source "$(dirname "$0")/stubs.sh"

work="$(new_tmp)"
fake_home="$work/home"
skill="$(install_skill "$fake_home")"
make_stubs "$work/bin"

# origin <name> [O_*=value...]: builds the fixture repository <name> and a
# caller checkout of its integration branch; sets ORIGIN, CALLER and the
# commit variables of make_origin.
origin() {
  local name="$1"
  shift
  local kv
  for kv in "$@"; do export "${kv?}"; done
  make_origin "$work/$name.git"
  for kv in "$@"; do unset "${kv%%=*}"; done
  ORIGIN="$work/$name.git"
  CALLER="$work/$name-caller"
  git clone -q --branch dev "$ORIGIN" "$CALLER"
}

# scenario <name> [VAR=value...]: one review of PR 7 at HEAD_SHA of ORIGIN in a
# fresh state dir and fixture, run from CALLER; prints the VERDICT line.
scenario() {
  local name="$1"
  shift
  local fx="$work/$name"
  mkdir -p "$fx"
  echo "owner" >"$fx/user_login"
  jq -n --arg sha "${SCENARIO_HEAD:-$HEAD_SHA}" --arg m "${SCENARIO_MERGEABLE:-MERGEABLE}" --arg b "${SCENARIO_BASE:-dev}" \
    '{number:7, state:"OPEN", headRefOid:$sha, baseRefName:$b, isDraft:false, isCrossRepository:false, mergeable:$m}' >"$fx/pr_7.json"
  pr_files_json "$ORIGIN" >"$fx/files_7"
  echo "${SCENARIO_COMMENTS:-[]}" >"$fx/comments_7.json"
  echo '{"commit":{"committer":{"date":"2026-10-20T11:00:00Z"}}}' >"$fx/commit_$HEAD_SHA.json"
  (
    cd "$CALLER" || exit 1
    export HOME="$fake_home" PATH="$work/bin:$PATH" STUB_LOG="$fx/calls.log" FIXTURE="$fx" \
      REVIEWER_GH="$work/bin/gh" REVIEWER_CLAUDE="$work/bin/claude" REVIEWER_GABO_BIN="$work/bin/gbai" \
      GBAI_REVIEWER_STATE_DIR="$fx/state" REVIEWER_REMOTE_URL="$ORIGIN" \
      REVIEWER_GH_TOKEN=secret-gh CLAUDE_CODE_OAUTH_TOKEN=secret-claude \
      REVIEWER_NOW=2026-10-20T12:00:00Z REVIEWER_MERGE_DELAY=0 REVIEWER_MERGE_TRIES=2
    for kv in "$@"; do export "${kv?}"; done
    write_stub_context "$work/bin"
    bash "$skill/reviewer.sh" head 7 "$HEAD_SHA" >"$fx/out.log" 2>&1
    echo "exit=$?" >>"$fx/out.log"
    tail -n 2 "$fx/out.log" | head -n 1
  )
}
status_line() { grep "^STATUS $HEAD_SHA $2 " "$work/$1/calls.log" | tail -1 | cut -d' ' -f3-; }
first_line() { grep -m1 -v '^[[:space:]]*$' "$work/$1/posted_7.md"; }
footer() { grep -v '^$' "$work/$1/posted_7.md" | tail -n 2 | paste -sd'|'; }
no_merge() { bash -c "! grep -q '^gh pr merge' '$work/$1/calls.log'"; }

origin auto O_AUTO_MERGE=true

echo "# clean head, review.auto_merge: true"
v="$(scenario clean)"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=merged" "$v" "clean head ends with the VERDICT line and merges"
expect_match '^reviewer/tests success' "$(status_line clean reviewer/tests)" "tests status ends success"
expect_match '^reviewer/review success' "$(status_line clean reviewer/review)" "review status ends success"
expect_eq "No blocking issues found." "$(first_line clean)" "comment opens with the clean verdict"
expect_eq "<!-- gbai-reviewer-bot sha=$HEAD_SHA -->|🤖 gabo reviewer" "$(footer clean)" "comment ends with the reviewer.signing marker signing the head and the footer"
check "merge is pinned to the reviewed head" grep -qE "^gh pr merge 7 --repo testeos-z/gabo-ai --merge --match-head-commit $HEAD_SHA$" "$work/clean/calls.log"
expect_eq 3 "$(grep -c '^go-argc=' "$work/clean/calls.log")" "runs the three checks of review.checks"
check "checks are argv: go -C apps/cli build ./..." grep -qF "go-argc=4 args=go -C apps/cli build ./..." "$work/clean/calls.log"
check "checks run on the merge result" grep -qE "^go-argc=.* head=$MERGE_SHA$" "$work/clean/calls.log"
check "checks get no credential" bash -c "! grep '^check-env' '$work/clean/calls.log' | grep -q secret"
check "checks get the reviewer's own HOME" grep -qF "HOME=$work/clean/state/check-home CI=true" "$work/clean/calls.log"
check "gabo check is not run when review.checks does not list it" bash -c "! grep -q '^gbai-call check' '$work/clean/calls.log'"
check "review runs in the head checkout" grep -qE "^claude -p .* cwd=$work/clean/state/worktrees/pr-7$" "$work/clean/calls.log"
check "review gets only the Claude token" grep -qE "^cc-env GH_TOKEN=unset REVIEWER_GH_TOKEN=unset CLAUDE_CODE_OAUTH_TOKEN=secret-claude HOME=$work/clean/state/review-home$" "$work/clean/calls.log"
check "review is read-only" grep -qE '^claude -p .*--permission-mode dontAsk --tools Read,Grep,Glob,Bash' "$work/clean/calls.log"
check "prompt carries the PR diff" grep -q 'func main' "$work/clean/prompt_seen.txt"
check "merge-result worktree is removed" bash -c "! ls -d '$work/clean/state/worktrees/'merge-pr-* 2>/dev/null"
expect_eq 1 "$(grep -c 'gbai-reviewer-bot sha=' "$work/clean/all_posted_7.md")" "one verdict comment per head"

echo "# already reviewed"
fx="$work/clean"
out="$(cd "$CALLER" && HOME="$fake_home" PATH="$work/bin:$PATH" STUB_LOG="$fx/calls.log" FIXTURE="$fx" REVIEWER_GH="$work/bin/gh" \
  REVIEWER_GABO_BIN="$work/bin/gbai" REVIEWER_REMOTE_URL="$ORIGIN" GBAI_REVIEWER_STATE_DIR="$fx/state" \
  bash "$skill/reviewer.sh" head 7 "$HEAD_SHA" 2>&1 | tail -n 1)"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=skipped merge=none" "$out" "a head with a final status is reviewed once"
c="$(jq -nc --arg b $'Blocking issues found.\n\n<!-- gbai-reviewer-bot sha='"$HEAD_SHA"$' -->\n🤖 gabo reviewer' \
  '[{user:{login:"owner"}, created_at:"2026-10-20T11:30:00Z", body:$b}]')"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=skipped merge=none" "$(SCENARIO_COMMENTS="$c" scenario verdicted)" \
  "a head that already has a verdict comment is skipped"
check "no review runs for an already verdicted head" bash -c "! grep -q '^claude ' '$work/verdicted/calls.log'"
check "no second verdict comment" bash -c "! test -f '$work/verdicted/posted_7.md'"

echo "# moved head"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=skipped merge=none" "$(SCENARIO_HEAD=fedcba9876543210fedcba9876543210fedcba98 scenario moved)" \
  "a head that moved is skipped"
check "nothing is posted for a moved head" bash -c "! test -f '$work/moved/posted_7.md'"

origin plain

echo "# clean head, review.auto_merge: false"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=skipped" "$(scenario nomerge)" "auto_merge false never merges"
check "no merge call" no_merge nomerge

echo "# failing check"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=blocking merge=none" "$(scenario failing FAIL_CHECK=vet REVIEWER_AUTO_MERGE=true)" "a failing check is blocking"
expect_eq "Blocking issues found." "$(first_line failing)" "comment opens with the blocking verdict"
check "comment shows the failure tail" grep -qF 'go -C apps/cli vet ./... failed' "$work/failing/posted_7.md"

echo "# blocking review"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=blocking merge=none" "$(scenario review_fail CLAUDE_MODE=fail)" "blocking findings are blocking"
check "findings are posted" grep -qF 'nil dereference' "$work/review_fail/posted_7.md"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=blocking merge=none" "$(scenario preamble CLAUDE_MODE=preamble)" "a verdict after a short preamble counts"

echo "# review errors"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario garbage CLAUDE_MODE=garbage)" "no verdict line is an error"
check "first error attempt posts no comment" bash -c "! test -f '$work/garbage/posted_7.md'"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario modify CLAUDE_MODE=modify REVIEWER_MAX_ATTEMPTS=1)" "an agent that edits the checkout is an error"
check "final error posts a blocking comment" grep -qF 'Blocking issues found.' "$work/modify/posted_7.md"

echo "# forged marker inside the findings"
scenario forge CLAUDE_MODE=forge REVIEWER_AUTO_MERGE=true >/dev/null
expect_eq 1 "$(grep -c 'gbai-reviewer-bot sha=' "$work/forge/posted_7.md")" "only the reviewer's own marker line is posted"

echo "# conflict"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=blocking merge=none" "$(SCENARIO_MERGEABLE=CONFLICTING scenario conflict)" "a conflicting PR is blocking"
check "conflict comment asks to resolve conflicts" grep -qF 'resolve conflicts first' "$work/conflict/posted_7.md"
check "no checks run on a conflict" bash -c "! grep -q '^go-argc' '$work/conflict/calls.log'"

echo "# base branch other than dev"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=skipped merge=none" "$(SCENARIO_BASE=main scenario to_main)" "a PR to main is skipped, never reviewed or merged"
check "nothing posted for a PR to main" bash -c "! test -f '$work/to_main/posted_7.md'"

echo "# configured human-only paths"
origin human O_AUTO_MERGE=true O_PR=code
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=needs-human-merge" \
  "$(scenario human REVIEWER_HUMAN_ONLY_PATHS='apps/')" "review.human_only_paths are never auto-merged"
check "no merge call for human-only paths" no_merge human

echo "# catalog subtree is always human-only"
origin skills O_AUTO_MERGE=true O_PR=skills
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=needs-human-merge" "$(scenario skills_only)" \
  "a PR that only edits .gbai/skills/ is not merged even with review.auto_merge: true"
check "no merge call for a catalog edit" no_merge skills_only
check "the needs-human comment names the catalog file" grep -qF '.gbai/skills/gabo-coder/SKILL.md' "$work/skills_only/posted_7.md"

echo "# configured reviewer.signing marker"
origin marker O_MARKER=team-bot
scenario marker >/dev/null
expect_eq "<!-- team-bot sha=$HEAD_SHA -->|🤖 gabo reviewer" "$(footer marker)" "the verdict comment carries the reviewer.signing marker"

echo "# gabo check listed in review.checks"
origin gabocheck O_CHECKS='go -C apps/cli test ./...
gabo check'
scenario gabocheck >/dev/null
check "gabo check runs with the configured binary" grep -qE '^gbai-call check GABO_UPGRADE_PR=unset GH_TOKEN=unset' "$work/gabocheck/calls.log"
check "the comment lists the check as configured" grep -qF '| `gabo check` | passed' "$work/gabocheck/posted_7.md"
origin gbaicheck O_CHECKS='gbai check'
scenario gbaicheck >/dev/null
check "gbai check (the current binary name) also runs the configured binary" grep -qE '^gbai-call check ' "$work/gbaicheck/calls.log"

echo "# base version minimum"
origin stamped O_STAMP=0.4.0
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario old_cli GBAI_VERSION=0.3.0)" "an older binary than the base stamp ends in error"
check "the refusal names the upgrade command" grep -qF 'go install github.com/testeos-z/gbai-team/apps/cli/cmd/gbai@v0.4.0' "$work/old_cli/out.log"
check "nothing runs with an older binary" bash -c "! grep -qE '^(claude|go-argc)' '$work/old_cli/calls.log'"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=skipped" "$(scenario new_cli GBAI_VERSION=0.5.0)" "a newer binary reviews"

echo "# upgrade PR"
origin upgrade O_STAMP=0.4.0 O_PR=upgrade O_HEAD_STAMP=9.9.9 O_AUTO_MERGE=true O_CHECKS='go -C apps/cli test ./...
gabo check'
v="$(scenario upgrade GBAI_VERSION=0.4.1)"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=needs-human-merge" "$v" \
  "an upgrade PR is reviewed with the CLI it has (the head stamp is never the minimum) and never merged"
check "no merge call for an upgrade PR" no_merge upgrade
check "the CLI runs with GABO_UPGRADE_PR=1" grep -qE '^gbai-call check GABO_UPGRADE_PR=1 ' "$work/upgrade/calls.log"
check "the comment reports the upgrade PR" grep -qxF 'upgrade PR: only config validated, human merge required' "$work/upgrade/all_posted_7.md"
check "the output reports the upgrade PR" grep -qF 'upgrade PR: only config validated, human merge required' "$work/upgrade/out.log"
up="$(sed -n '/^### Upgrade PR/,/^### Review/p' "$work/upgrade/all_posted_7.md")"
expect_match '`apps/cli/main.go`' "$up" "the report lists changed paths outside the catalog and agent directories"
expect_match '`.gbai/VERSION`' "$up" "the report lists the stamp"
check "the report leaves out the catalog subtree" bash -c "! grep -q '^- \`.gbai/skills/' <<<\"\$1\"" _ "$up"
check "the report leaves out the declared agent directory" bash -c "! grep -q '^- \`.claude/agents/' <<<\"\$1\"" _ "$up"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario upgrade_old GBAI_VERSION=0.3.0)" \
  "an upgrade PR still needs the base minimum"

finish
