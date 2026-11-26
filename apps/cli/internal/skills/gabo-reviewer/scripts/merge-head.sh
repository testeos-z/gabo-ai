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
# human-only: review.human_only_paths plus the built-in .gabo/VERSION and
# .gabo/skills/, except on an intent pour (intent 0032 S5, D6: the head is an
# <NNNN-slug> intent branch with a folder on the integration branch, pouring
# into it): those paths were already approved on the task PRs, so holding
# again would make every intent merge manual. The eval hold still applies to
# a pour. No eval notice holds the head (intent 0030 S10): a head
# is held while a notice by the bot login names it (the marker opening the
# comment), and released only when a cleared marker by that login names it and
# no notice names it. A held or
# human-only PR gets a "needs human merge" comment and no merge (a person may
# still merge it). The merge is exactly
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

# eval_marker_heads <comments json> <notice|cleared>: the heads that the bot
# login names with an eval marker, one per line. Only the first line of a
# comment counts, the shape the eval runner posts: the bot's own comments carry
# text the PR controls (findings, check logs), so a marker further down never counts.
eval_marker_heads() {
  jq -r --arg l "$BOT_LOGIN" --arg k "$2" '
    [.[] | select(.user.login == $l) | ((.body // "") | split("\n")[0])
     | sub("\r$"; "") | gsub("^\\s+|\\s+$"; "")
     | select(test("^<!-- gabo-eval-" + $k + " sha=[0-9a-f]{40} -->$"))
     | capture("sha=(?<s>[0-9a-f]{40})").s] | unique[]' <<<"$1"
}

# eval_notice_hold <comments json> <head sha>: prints "eval notice" when a
# notice holds the head, nothing otherwise. No notice on the PR holds nothing;
# a notice naming the head holds it; a head without its own notice is released
# only by a cleared marker that names it.
eval_notice_hold() {
  local notices cleared
  notices="$(eval_marker_heads "$1" notice)" || return 1
  [[ -n "$notices" ]] || return 0
  grep -qxF "$2" <<<"$notices" && {
    echo "eval notice"
    return 0
  }
  cleared="$(eval_marker_heads "$1" cleared)" || return 1
  grep -qxF "$2" <<<"$cleared" || echo "eval notice"
  return 0
}

[[ "$AUTO_MERGE" == true ]] || skip "REVIEWER_AUTO_MERGE is not true; a human merges"
[[ -n "$BOT_LOGIN" ]] || resolve_bot_login || skip "gh cannot authenticate"

info="$("$GH" pr view "$pr" --repo "$REPO" --json number,state,baseRefName,headRefName,isDraft,isCrossRepository,headRefOid 2>/dev/null)" ||
  skip "could not read the PR"
[[ "$(jq -r .state <<<"$info")" == OPEN ]] || skip "the PR is not open"
pr_base="$(jq -r .baseRefName <<<"$info")"
[[ "$pr_base" != main && "$pr_base" != master ]] || skip "the PR targets $pr_base; the reviewer never merges into it"
[[ "$pr_base" == "$BASE_BRANCH" ]] || skip "the PR does not target $BASE_BRANCH"
msg="$(validate_base_branch)" || skip "$msg"
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
# Intent pour (intent 0032 S5, D6, amended by A1): the head is an
# <NNNN-slug> intent branch with a folder on the integration branch,
# pouring into it. The pour is proven only when every commit in the pour
# diff is provably approved: associated with a merged PR based on the
# intent branch (a task PR or a sync PR), as seen by the reviewer's own API
# queries, never by PR claims. Proven pours skip the hold for configured
# human-only paths (already approved on the task PRs); built-in paths
# always hold, and the eval hold below still applies.
pour=false
pour_proven=false
head_branch="$(jq -r .headRefName <<<"$info")"
if [[ "$pr_base" == "$INTEGRATION_BRANCH" && "$head_branch" =~ ^[0-9]{4}-[A-Za-z0-9._-]+$ ]] &&
  intent_dir_exists "$head_branch"; then
  pour=true
  pour_proven=true
  commits="$("$GH" api --paginate "repos/$REPO/pulls/$pr/commits" 2>/dev/null | jq -r '.[].sha')" ||
    pour_proven=false
  [[ -n "${commits:-}" ]] || pour_proven=false
  if [[ "$pour_proven" == true ]]; then
    for c in $commits; do
      pulls="$("$GH" api --paginate "repos/$REPO/commits/$c/pulls" 2>/dev/null)" ||
        { pour_proven=false; break; }
      jq -e --arg b "$head_branch" \
        '[.[] | select(.state == "closed" and .merged_at != null and .base.ref == $b)] | length > 0' \
        <<<"$pulls" >/dev/null || { pour_proven=false; break; }
    done
  fi
fi
protected=""
builtin=""
while IFS= read -r f; do
  is_builtin_human_only "$f" && builtin+="$f"$'\n'
  is_human_only "$f" && protected+="$f"$'\n'
done <<<"$files"
hold="$(eval_notice_hold "$comments" "$sha")" || skip "could not read the eval markers"
if [[ -n "$hold" || -n "$builtin" || ( -n "$protected" && "$pour_proven" == false ) ]]; then
  body="$(
    if [[ -n "$hold" ]]; then
      echo "Needs human merge: the review of \`$sha\` is clean, but an eval notice holds this head (intent 0030 S10), so gabo-reviewer does not merge it. A head is released only by a cleared marker from an eval run on that head at or above the threshold, and never while a notice names it."
      echo
    fi
    if [[ -n "$builtin" ]]; then
      echo "Needs human merge: the review of \`$sha\` is clean, but this PR changes built-in human-only paths, which always need a human merge, even on a proven intent pour (intent 0032 A1):"
      echo
      head -n 20 <<<"${builtin%$'\n'}" | sed 's/^/- `/; s/$/`/'
      echo
    fi
    if [[ -n "$protected" && "$pour_proven" == false ]]; then
      if [[ "$pour" == true ]]; then
        echo "Needs human merge: the review of \`$sha\` is clean, but this intent pour is not proven (at least one commit is not associated with a merged PR based on the intent branch), so the human-only paths still hold it (intent 0032 A1):"
      else
        echo "Needs human merge: the review of \`$sha\` is clean, but this PR changes human-only paths, so gabo-reviewer does not merge it:"
      fi
      echo
      head -n 20 <<<"${protected%$'\n'}" | sed 's/^/- `/; s/$/`/'
      echo
    fi
    echo "$NEEDS_HUMAN_MARKER"
    echo "$FOOTER"
  )"
  jq -n --arg b "$body" '{body: $b}' | "$GH" api -X POST "repos/$REPO/issues/$pr/comments" --input - >/dev/null ||
    fail "could not post the needs-human-merge comment"
  suffix=""
  if [[ -n "$hold" ]]; then
    suffix=" reason: eval notice"
    [[ -z "$protected" ]] || suffix+=" and human-only paths"
  fi
  log_event "NEEDS-HUMAN-MERGE" "#$pr $sha$suffix"
  exit 0
fi

"$GH" pr merge "$pr" --repo "$REPO" --merge --match-head-commit "$sha" >/dev/null 2>&1 ||
  fail "gh refused the merge (conflict, head moved, or branch rules)"
# The head worktree of a merged PR is not needed any more; a failure to remove
# it never changes the merge result (prune retries it).
remove_pr_worktree "$pr" || true
log_event "MERGED" "#$pr $sha"
