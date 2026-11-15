#!/usr/bin/env bash
# gabo-reviewer entry point (single-user mode, the owner's own gh login):
#   reviewer.sh preflight            config, trust, base checkout and login checks only
#   reviewer.sh head <pr> <sha>      reviews that head (40-hex) if it is due
#   reviewer.sh due <pr> <sha>       exit 0 when that head is due, else 1
#   reviewer.sh list-due             prints "<pr> <sha>" for each due head
#   reviewer.sh prune                removes the <owner>__<repo>__pr-<n> worktrees of PRs that are not open and
#                                    the stale ones (REVIEWER_KEEP_DAYS) and keeps the caches under
#                                    REVIEWER_CACHE_MAX; also runs at the start of head
#   reviewer.sh once                 reviews every due head, one at a time
# Every mode takes a trailing `engine <claude|codex|pi|opencode>`, required
# when runtimes.reviewer (or REVIEWER_ENGINE) is `session` and refused
# otherwise; under session the engine must be listed in runtimes.assistant on
# the integration branch (intent 0013 S11, D5).
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
  echo "usage: reviewer.sh preflight|prune|once|list-due|due <pr> <sha>|head <pr> <40-hex sha> [engine <engine>]" >&2
  exit 2
}
[[ "$mode" =~ ^(once|preflight|list-due|due|head|prune)$ ]] || usage
nargs=1
if [[ "$mode" == due || "$mode" == head ]]; then
  is_number "${2:-}" && [[ "${3:-}" =~ ^[0-9a-f]{40}$ ]] || usage
  nargs=3
fi
# The optional trailing `engine <engine>` (intent 0013 S11, D5).
extra=("${@:nargs+1}")
((${#extra[@]} == 0)) || [[ "${extra[0]}" == engine ]] || usage

preflight_failed() {
  echo "PREFLIGHT-FAILED $1"
  [[ "$mode" != head ]] || echo "VERDICT PR #$2 head $3 verdict=error merge=none"
  exit 2
}
# The engine argument is checked before anything else uses it.
ARG_ENGINE_SET=false
ARG_ENGINE=""
if ((${#extra[@]})); then
  ((${#extra[@]} == 2)) || preflight_failed "the engine argument needs exactly one value: engine <claude|codex|pi|opencode>" "${2:-}" "${3:-}"
  problem="$(engine_arg_shape_problem "${extra[1]}")"
  [[ -z "$problem" ]] || preflight_failed "$problem" "${2:-}" "${3:-}"
  ARG_ENGINE_SET=true
  ARG_ENGINE="${extra[1]}"
fi
[[ -z "${SUDO_USER:-}" ]] || preflight_failed "single-user mode runs as the calling user, never through sudo" "${2:-}" "${3:-}"
problem="$(validate_static unresolved)"
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
# Resolves runtimes.reviewer session to the engine argument; refuses the
# argument under a fixed engine and session without it.
problem="$(engine_arg_problem "$ARG_ENGINE_SET" "$ARG_ENGINE")"
[[ -z "$problem" ]] || preflight_failed "$problem" "${2:-}" "${3:-}"
if [[ "$ENGINE" == session ]]; then
  SESSION_ENGINE=true
  ENGINE="$ARG_ENGINE"
  # The child scripts review on the resolved engine.
  REVIEWER_ENGINE="$ARG_ENGINE"
  export REVIEWER_ENGINE
fi
problem="$(validate_config)"
[[ -z "$problem" ]] || preflight_failed "$problem" "${2:-}" "${3:-}"
problem="$(minimum_problem)"
[[ -z "$problem" ]] || preflight_failed "$problem" "${2:-}" "${3:-}"
problem="$(engine_problem)"
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
  # The whole run holds cache.lock shared: prune_caches takes it exclusive, so
  # it never deletes a cache under a running check and a review that starts
  # during a prune waits for it instead of being skipped.
  (
    mkdir -p "$STATE/locks"
    exec 5>"$STATE/locks/cache.lock"
    # A bounded wait: a stuck prune never hangs the head. The run itself does
    # not inherit the descriptor (5>&-), so PR code and the review cannot drop
    # the lock or keep it.
    if ! flock -s -w "$HEAD_TIMEOUT" 5; then
      echo "$(now_utc) RETRY #$1 $2 the cache lock stayed held; run the review again" >"$log"
      exit 1
    fi
    CLAUDE_CODE_OAUTH_TOKEN="$CLAUDE_TOKEN" run_bounded "$HEAD_TIMEOUT" "$log" bash "$HEAD_SCRIPT" "$1" "$2" 5>&-
  ) || code=$?
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

# Best effort: a prune problem is logged and never changes a review or its verdict.
[[ "$mode" != head && "$mode" != prune ]] || prune_worktrees "${2:-}" >/dev/null 2>&1 || true
# Same for the caches: prints one `pruned:` line only when it trimmed something.
[[ "$mode" != head && "$mode" != prune ]] || prune_caches 2>/dev/null || true

case "$mode" in
  prune) exit 0 ;;
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
