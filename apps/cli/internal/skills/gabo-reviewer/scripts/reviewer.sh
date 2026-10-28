#!/usr/bin/env bash
# gabo-reviewer entry point (single-user mode, the owner's own gh login):
#   reviewer.sh preflight            config, trust, base checkout and login checks only
#   reviewer.sh head <pr> <sha>      reviews that head (40-hex) if it is due
#   reviewer.sh due <pr> <sha>       exit 0 when that head is due, else 1
#   reviewer.sh list-due             prints "<pr> <sha>" for each due head
#   reviewer.sh once                 reviews every due head, one at a time
# Run it from a checkout of the repository (its gabo.yaml names the
# repository and the integration branch) or set REVIEWER_REPO. Every other
# repository setting comes from gabo.yaml in the reviewer's own clean checkout
# of the integration branch.
# A head is due unless its newest reviewer/review status from the bot login is
# final (success or failure, or error after REVIEWER_MAX_ATTEMPTS attempts)
# or a verdict comment by the bot login already signs it.
# `head` always ends with exactly one line:
#   VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>
# Never deletes branches, comments, or statuses; merges only through
# merge-head.sh (integration branch only, never main).
set -uo pipefail
source "$(dirname "$0")/common.sh"

HEAD_SCRIPT="$reviewer_dir/review-head.sh"
mode="${1:-}"
usage() {
  echo "usage: reviewer.sh preflight|once|list-due|due <pr> <sha>|head <pr> <40-hex sha>" >&2
  exit 2
}
[[ "$mode" =~ ^(once|preflight|list-due|due|head)$ ]] || usage
if [[ "$mode" == due || "$mode" == head ]]; then
  is_number "${2:-}" && [[ "${3:-}" =~ ^[0-9a-f]{40}$ ]] || usage
fi

preflight_failed() {
  echo "PREFLIGHT-FAILED $1"
  [[ "$mode" != head ]] || echo "VERDICT PR #$2 head $3 verdict=error merge=none"
  exit 2
}
[[ -z "${SUDO_USER:-}" ]] || preflight_failed "single-user mode runs as the calling user, never through sudo" "${2:-}" "${3:-}"
problem="$(validate_static)"
[[ -z "$problem" ]] || preflight_failed "$problem" "${2:-}" "${3:-}"
problem="$(repo_copy_problem)"
[[ -z "$problem" ]] || preflight_failed "$problem" "${2:-}" "${3:-}"
problem="$(trust_problem)"
[[ -z "$problem" ]] || preflight_failed "$problem" "${2:-}" "${3:-}"
for tool in jq flock git setsid; do
  command -v "$tool" >/dev/null || preflight_failed "$tool is not installed" "${2:-}" "${3:-}"
done
problem="$(sync_base)" || preflight_failed "${problem:-could not update the reviewer checkout of $BASE_BRANCH}" "${2:-}" "${3:-}"
load_base_config
problem="$(validate_config)"
[[ -z "$problem" ]] || preflight_failed "$problem" "${2:-}" "${3:-}"
problem="$(minimum_problem)"
[[ -z "$problem" ]] || preflight_failed "$problem" "${2:-}" "${3:-}"
resolve_bot_login || preflight_failed "gh cannot authenticate (gh auth login, or REVIEWER_GH_TOKEN)" "${2:-}" "${3:-}"
[[ "$GH_LOGIN" == "$BOT_LOGIN" ]] ||
  preflight_failed "the gh login is '$GH_LOGIN', not REVIEWER_BOT_LOGIN '$BOT_LOGIN'" "${2:-}" "${3:-}"
# The child scripts must use the same login without asking gh again.
export REVIEWER_BOT_LOGIN="$BOT_LOGIN" REVIEWER_REPO="$REPO" REVIEWER_BASE_BRANCH="$BASE_BRANCH"

# pr_comments <pr>: every issue comment of the PR as one JSON array.
pr_comments() {
  "$GH" api --paginate "repos/$REPO/issues/$1/comments" 2>/dev/null | jq -cs 'add // []'
}

due() {
  local state attempts
  state="$(latest_review_state "$2")"
  case "$state" in
    success | failure) return 1 ;;
    error)
      attempts="$(cat "$STATE/attempts/$2" 2>/dev/null || echo 0)"
      ((attempts < MAX_ATTEMPTS))
      ;;
    *) return 0 ;;
  esac
}

list_due() {
  local prs n sha draft cross
  prs="$("$GH" pr list --repo "$REPO" --base "$BASE_BRANCH" --state open \
    --json number,headRefOid,isDraft,isCrossRepository --limit 200 2>/dev/null)" || return 1
  while read -r n sha draft cross; do
    [[ "$draft" == false && "$cross" == false && "$sha" =~ ^[0-9a-f]{40}$ ]] && is_number "$n" || continue
    due "$n" "$sha" && echo "$n $sha"
  done < <(jq -r '.[] | "\(.number) \(.headRefOid) \(.isDraft) \(.isCrossRepository)"' <<<"$prs")
}

# review_head <pr> <sha>: one head, bounded by REVIEWER_HEAD_TIMEOUT; prints
# the outcome lines and the VERDICT line. PR output in the log never starts
# with a timestamp followed by an outcome event, because review-head.sh writes
# check output to separate files only.
review_head() {
  mkdir -p "$STATE/logs"
  log_event "DISPATCH" "#$1 $2"
  local log="$STATE/logs/head-$1.log" code=0 verdict merge outcome
  # The Claude token reaches the child only; review-head.sh moves it out of the
  # environment again and hands it to the review step alone.
  CLAUDE_CODE_OAUTH_TOKEN="$CLAUDE_TOKEN" run_bounded "$HEAD_TIMEOUT" "$log" bash "$HEAD_SCRIPT" "$1" "$2" || code=$?
  outcome="$(cat "$log")"
  has() { grep -qE "^[^ ]+ $1 #$2( |$)" <<<"$outcome"; }
  grep -E '^[^ ]+ (CONFLICT|UPGRADE-PR|REVIEWED|MERGED|MERGE-SKIPPED|NEEDS-HUMAN-MERGE|MERGE-FAILED|SKIP|RETRY) #' <<<"$outcome" ||
    grep -E '^BUSY #' <<<"$outcome" || tail -n 1 "$log"
  verdict=error
  if has RETRY "$1"; then
    verdict=error
  elif has CONFLICT "$1"; then
    verdict=blocking
  elif grep -qE "^[^ ]+ REVIEWED #$1 $2 tests=success review=success$" <<<"$outcome"; then
    verdict=clean
  elif grep -qE "^[^ ]+ REVIEWED #$1 .* review=error" <<<"$outcome"; then
    verdict=error
  elif has REVIEWED "$1"; then
    verdict=blocking
  elif has SKIP "$1" || grep -qE "^BUSY #$1" <<<"$outcome"; then
    verdict=skipped
  fi
  merge=none
  if has MERGED "$1"; then
    merge=merged
  elif has NEEDS-HUMAN-MERGE "$1"; then
    merge=needs-human-merge
  elif has MERGE-SKIPPED "$1" || has MERGE-FAILED "$1"; then
    merge=skipped
  fi
  echo "VERDICT PR #$1 head $2 verdict=$verdict merge=$merge"
  return "$code"
}

case "$mode" in
  preflight)
    echo "PREFLIGHT-OK repo=$REPO base=$BASE_BRANCH login=$BOT_LOGIN mode=$MODE engine=$ENGINE placement=$PLACEMENT minimum=$MINIMUM"
    exit 0
    ;;
  due) due "$2" "$3"; exit ;;
  head)
    if ! due "$2" "$3"; then
      echo "SKIP #$2 $3 already has a final reviewer/review status"
      echo "VERDICT PR #$2 head $3 verdict=skipped merge=none"
      exit 0
    fi
    if ! comments="$(pr_comments "$2")"; then
      echo "RETRY #$2 $3 could not read the PR comments"
      echo "VERDICT PR #$2 head $3 verdict=error merge=none"
      exit 1
    fi
    if has_verdict_comment "$comments" "$3"; then
      echo "SKIP #$2 $3 already has a verdict comment"
      echo "VERDICT PR #$2 head $3 verdict=skipped merge=none"
      exit 0
    fi
    review_head "$2" "$3"
    exit
    ;;
  list-due)
    list_due || { echo "LIST-FAILED could not list the open PRs" >&2; exit 1; }
    exit 0
    ;;
  once)
    listing="$(list_due)" || { echo "LIST-FAILED could not list the open PRs" >&2; exit 1; }
    while read -r n sha; do
      [[ -n "${n:-}" ]] || continue
      review_head "$n" "$sha"
    done <<<"$listing"
    ;;
esac
