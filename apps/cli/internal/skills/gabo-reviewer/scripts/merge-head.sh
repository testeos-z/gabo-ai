#!/usr/bin/env bash
# The reviewer's merge gate:
#   merge-head.sh <pr> <reviewed 40-hex sha>
# The reviewer posts with the owner's account, which also authors the PRs, and
# GitHub forbids approving your own PR, so it never approves: the verdict
# comment is the approval. It merges only when REVIEWER_AUTO_MERGE=true, the
# PR is open, targets the integration branch (never main), is not a draft,
# comes from this repository, its head is still the reviewed SHA, the newest
# reviewer/tests and reviewer/review statuses by the bot login on that SHA are
# success, the newest marker comment by the bot login newer than the head
# commit opens with the clean verdict and signs that SHA, and no changed path
# is human-only: review.human_only_paths plus the built-in .gabo/VERSION and
# .gabo/skills/ (then it comments "needs human merge" and merges nothing).
# The merge is exactly
#   gh pr merge <pr> --repo <repo> --merge --match-head-commit <sha>
# (never --admin, --auto, --squash, --rebase, or branch deletion).
# Prints one line: MERGED, MERGE-SKIPPED <reason>, NEEDS-HUMAN-MERGE (exit 0)
# or MERGE-FAILED <reason> (exit 1).
set -uo pipefail
source "$(dirname "$0")/common.sh"

pr="${1:-}"
sha="${2:-}"
if ! is_number "$pr" || [[ ! "$sha" =~ ^[0-9a-f]{40}$ ]]; then
  echo "usage: merge-head.sh <pr number> <40-char head sha>" >&2
  exit 2
fi
problem="$(validate_config)"
if [[ -n "$problem" ]]; then
  echo "PREFLIGHT-FAILED $problem" >&2
  exit 2
fi
skip() {
  log_event "MERGE-SKIPPED" "#$pr $sha $1"
  exit 0
}
fail() {
  log_event "MERGE-FAILED" "#$pr $sha $1"
  exit 1
}

[[ "$AUTO_MERGE" == true ]] || skip "REVIEWER_AUTO_MERGE is not true; a human merges"
[[ -n "$BOT_LOGIN" ]] || resolve_bot_login || skip "gh cannot authenticate"

info="$("$GH" pr view "$pr" --repo "$REPO" --json number,state,baseRefName,isDraft,isCrossRepository,headRefOid 2>/dev/null)" ||
  skip "could not read the PR"
[[ "$(jq -r .state <<<"$info")" == OPEN ]] || skip "the PR is not open"
pr_base="$(jq -r .baseRefName <<<"$info")"
[[ "$pr_base" != main && "$pr_base" != master ]] || skip "the PR targets $pr_base; the reviewer never merges into it"
[[ "$pr_base" == "$BASE_BRANCH" ]] || skip "the PR does not target $BASE_BRANCH"
[[ "$(jq -r .isDraft <<<"$info")" == false ]] || skip "the PR is a draft"
[[ "$(jq -r .isCrossRepository <<<"$info")" == false ]] || skip "the PR comes from a fork"
[[ "$(jq -r .headRefOid <<<"$info")" == "$sha" ]] || skip "head moved since the review"

statuses="$("$GH" api --paginate "repos/$REPO/commits/$sha/statuses" 2>/dev/null | jq -cs 'add // []')" ||
  skip "could not read the commit statuses"
for context in "$TESTS_CONTEXT" "$REVIEW_CONTEXT"; do
  state="$(jq -r --arg c "$context" --arg l "$BOT_LOGIN" \
    '[.[] | select(.context == $c and .creator.login == $l)] | first | .state // "missing"' <<<"$statuses")"
  [[ "$state" == success ]] || skip "$context is $state"
done

head_date="$("$GH" api "repos/$REPO/commits/$sha" 2>/dev/null | jq -r '.commit.committer.date // empty')"
[[ -n "$head_date" ]] || skip "could not read the head commit"
comments="$("$GH" api --paginate "repos/$REPO/issues/$pr/comments" 2>/dev/null | jq -cs 'add // []')" ||
  skip "could not read the PR comments"
verdict="$(latest_bot_verdict "$comments" "$head_date")" || skip "could not read the PR comments"
[[ -n "$verdict" ]] || skip "no reviewer verdict comment (marker, $BOT_LOGIN) newer than the head commit"
[[ "${verdict% *}" == clean ]] || skip "the newest reviewer verdict comment is not clean (${verdict% *})"
[[ "${verdict#* }" == "$sha" ]] || skip "the newest reviewer verdict comment signs another head (${verdict#* })"

files="$("$GH" api --paginate "repos/$REPO/pulls/$pr/files" 2>/dev/null | jq -r '.[].filename')" ||
  skip "could not read the PR files"
[[ -n "$files" ]] || skip "could not read the PR files (empty list)"
protected=""
while IFS= read -r f; do
  is_human_only "$f" && protected+="$f"$'\n'
done <<<"$files"
if [[ -n "$protected" ]]; then
  body="$(
    echo "Needs human merge: the review of \`$sha\` is clean, but this PR changes human-only paths, so gabo-reviewer does not merge it:"
    echo
    head -n 20 <<<"${protected%$'\n'}" | sed 's/^/- `/; s/$/`/'
    echo
    echo "$NEEDS_HUMAN_MARKER"
    echo "$FOOTER"
  )"
  jq -n --arg b "$body" '{body: $b}' | "$GH" api -X POST "repos/$REPO/issues/$pr/comments" --input - >/dev/null ||
    fail "could not post the needs-human-merge comment"
  log_event "NEEDS-HUMAN-MERGE" "#$pr $sha"
  exit 0
fi

"$GH" pr merge "$pr" --repo "$REPO" --merge --match-head-commit "$sha" >/dev/null 2>&1 ||
  fail "gh refused the merge (conflict, head moved, or branch rules)"
# The head worktree of a merged PR is not needed any more; a failure to remove
# it never changes the merge result (prune retries it).
remove_pr_worktree "$pr" || true
log_event "MERGED" "#$pr $sha"
