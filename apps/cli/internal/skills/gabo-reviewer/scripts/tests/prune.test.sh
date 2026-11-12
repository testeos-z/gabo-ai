#!/usr/bin/env bash
# Tests for the reviewer's worktree cleanup (intent 0024 S11): prune_worktrees
# in common.sh, `reviewer.sh prune`, the prune at the start of `head`, and the
# removal of pr-<n> after a merge. Offline, on scratch state dirs only.
# Run: bash tests/prune.test.sh
set -uo pipefail
source "$(dirname "$0")/lib.sh"
source "$(dirname "$0")/stubs.sh"

work="$(new_tmp)"
fake_home="$work/home"
skill="$(install_skill "$fake_home")"
make_stubs "$work/bin"
g=(git -c user.name=test -c user.email=test@example.com -c commit.gpgsign=false -c init.defaultBranch=dev)

# state_dir <name>: a reviewer state dir with a mirror holding one commit;
# prints its path.
state_dir() {
  local d="$work/$1"
  mkdir -p "$d/state/locks" "$d/worktrees" "$d/fx"
  "${g[@]}" init -q --bare "$d/repo.git"
  local tree commit
  tree="$(git -C "$d/repo.git" mktree </dev/null)"
  commit="$("${g[@]}" -C "$d/repo.git" commit-tree "$tree" -m c)"
  git -C "$d/repo.git" update-ref refs/heads/dev "$commit"
  echo "$commit" >"$d/commit"
  echo owner >"$d/fx/user_login"
  echo "$d"
}
# add_wt <dir> <n> <state>: worktree pr-<n> in the mirror; the PR is <state>
# ("none": unreadable).
add_wt() {
  git -C "$1/repo.git" worktree add -q --detach "$1/worktrees/pr-$2" "$(cat "$1/commit")"
  [[ "$3" == none ]] || jq -n --arg s "$3" '{state:$s}' >"$1/fx/pr_$2.json"
}
# run_prune <dir> [current pr] [VAR=value...]: runs prune_worktrees from the
# installed common.sh; prints its output.
run_prune() {
  local d="$1" cur="${2:-}"
  shift 2 2>/dev/null || shift $#
  (
    export HOME="$fake_home" STUB_LOG="$d/calls.log" FIXTURE="$d/fx" REVIEWER_GH="$work/bin/gh" \
      GABO_REVIEWER_STATE_DIR="$d" REVIEWER_REPO=example-org/example-repo
    for kv in "$@"; do export "${kv?}"; done
    cd "$work" || exit 1
    source "$skill/common.sh"
    prune_worktrees "$cur"
    echo "rc=$?"
  ) 2>&1
}
exists() { [[ -e "$1/worktrees/pr-$2" ]]; }
gone() { [[ ! -e "$1/worktrees/pr-$2" ]]; }

echo "# prune_worktrees"
d="$(state_dir rules)"
add_wt "$d" 1 MERGED
add_wt "$d" 2 CLOSED
add_wt "$d" 3 OPEN
add_wt "$d" 4 OPEN
touch -d '10 days ago' "$d/worktrees/pr-4"
add_wt "$d" 5 OPEN
touch -d '10 days ago' "$d/worktrees/pr-5"
add_wt "$d" 6 MERGED
add_wt "$d" 7 none
add_wt "$d" 8 CLOSED
add_wt "$d" 9 MERGED
mkdir "$d/worktrees/pr-10" "$d/outside"
jq -n '{state:"CLOSED"}' >"$d/fx/pr_10.json"
echo keep >"$d/outside/file"
ln -s "$d/outside" "$d/worktrees/pr-11"
jq -n '{state:"CLOSED"}' >"$d/fx/pr_11.json"
mkdir "$d/worktrees/pr-x"
git -C "$d/repo.git" worktree add -q --detach "$d/worktrees/merge-pr-3-abc.1p1" "$(cat "$d/commit")"
# pr-6 is the current PR and not open; pr-5 is the current PR, open and old; pr-8 is locked.
( exec 9>"$d/state/locks/pr-8.lock"; flock 9; sleep 30 ) &
holder=$!
sleep 0.3
out="$(run_prune "$d" 5)"
kill "$holder" 2>/dev/null
wait "$holder" 2>/dev/null
check "a merged PR's worktree is removed" gone "$d" 1
check "a closed PR's worktree is removed" gone "$d" 2
check "an open fresh worktree stays" exists "$d" 3
check "an open worktree unused for more than 7 days is removed" gone "$d" 4
check "the current PR's old open worktree stays" exists "$d" 5
check "the current PR, not open, is removed" gone "$d" 6
check "an unreadable PR state never removes a fresh worktree" exists "$d" 7
check "a worktree of a run in progress (lock held) stays" exists "$d" 8
check "a merged PR after the locked one is still processed" gone "$d" 9
check "a directory that is not a worktree of the mirror stays" exists "$d" 10
expect_match 'WARN refusing to remove .*pr-10: not a worktree of the mirror' "$(cat "$d/state/reviewer.log")" "the foreign directory is refused with a WARN"
check "a symlink stays and its target is untouched" bash -c "[[ -L '$d/worktrees/pr-11' && -f '$d/outside/file' ]]"
expect_match 'WARN refusing to remove .*pr-11' "$(cat "$d/state/reviewer.log")" "the symlink is refused with a WARN"
check "a non-numeric pr-* name is ignored" exists "$d" x
check "merge-pr-* worktrees are not touched by prune" test -d "$d/worktrees/merge-pr-3-abc.1p1"
expect_match 'WARN prune: could not read the state of PR #7' "$(cat "$d/state/reviewer.log")" "an unreadable state is logged"
expect_match 'rc=0$' "$out" "prune ends 0 although some paths were refused"

echo "# REVIEWER_KEEP_DAYS"
d="$(state_dir keep)"
add_wt "$d" 1 OPEN
touch -d '3 days ago' "$d/worktrees/pr-1"
run_prune "$d" "" >/dev/null
check "default keeps a 3 day old open worktree" exists "$d" 1
run_prune "$d" "" REVIEWER_KEEP_DAYS=2 >/dev/null
check "the environment overrides the days" gone "$d" 1
check "config.env ships REVIEWER_KEEP_DAYS=7" grep -qx 'REVIEWER_KEEP_DAYS=7' "$skill/config.env"

echo "# the lock is held while prune decides and removes"
d="$(state_dir race)"
add_wt "$d" 1 CLOSED
cat >"$work/bin/gh-race" <<RACE
#!/usr/bin/env bash
# a review that starts while prune reads the PR state: it must not get the lock
if (exec 5>"$d/state/locks/pr-1.lock"; flock -n 5); then echo free >"$d/probe"; else echo busy >"$d/probe"; fi
exec "$work/bin/gh" "\$@"
RACE
chmod +x "$work/bin/gh-race"
run_prune "$d" "" REVIEWER_GH="$work/bin/gh-race" >/dev/null
expect_eq "busy" "$(cat "$d/probe" 2>/dev/null)" "pr-<n>.lock is held while the PR state is read"
check "the race fixture's worktree is still pruned" gone "$d" 1

echo "# 349 worktrees of closed PRs"
d="$(state_dir many)"
for n in $(seq 1 349); do
  mkdir -p "$d/worktrees"
  git -C "$d/repo.git" worktree add -q --detach "$d/worktrees/pr-$n" "$(cat "$d/commit")"
  echo '{"state":"CLOSED"}' >"$d/fx/pr_$n.json"
done
add_wt "$d" 400 OPEN
run_prune "$d" "" >/dev/null
expect_eq "pr-400" "$(ls "$d/worktrees" | paste -sd' ')" "only the worktree of the open PR is left"

echo "# prune_caches (S12)"
mkdir -p "$work/gobin" "$work/gobin2"
cat >"$work/gobin/go" <<'GO'
#!/usr/bin/env bash
# stub go: `go clean -cache` empties $GOCACHE; GO_FAIL=1 fails it
echo "go $*" >>"${STUB_LOG:-/dev/null}"
[[ "${GO_FAIL:-}" != 1 ]] || { echo "go: boom" >&2; exit 1; }
[[ "$1 $2" == "clean -cache" ]] && rm -rf "${GOCACHE:?}"/* && exit 0
exit 2
GO
chmod +x "$work/gobin/go"
# fill <dir> <kib>
fill() { mkdir -p "$1"; head -c "$(($2 * 1024))" /dev/zero >"$1/blob"; }
total_kib() { du -sk "$1/cache/go-build" "$1/cache/go-mod" "$1/check-home" 2>/dev/null | awk '{s+=$1} END {print s+0}'; }
# run_caches <dir> [VAR=value...]: runs prune_caches from the installed common.sh
run_caches() {
  local d="$1"
  shift
  (
    export HOME="$fake_home" STUB_LOG="$d/calls.log" FIXTURE="$d/fx" REVIEWER_GH="$work/bin/gh" \
      GABO_REVIEWER_STATE_DIR="$d" REVIEWER_REPO=example-org/example-repo PATH="$work/gobin:$PATH"
    for kv in "$@"; do export "${kv?}"; done
    cd "$work" || exit 1
    source "$skill/common.sh"
    prune_caches
    echo "rc=$?"
  ) 2>&1
}
d="$(state_dir caches)"
fill "$d/cache/go-build" 3000
fill "$d/cache/go-mod" 500
fill "$d/check-home/old" 1500
fill "$d/check-home/new" 100
touch -d '10 days ago' "$d/check-home/old"
out="$(run_caches "$d" REVIEWER_CACHE_MAX=2M)"
expect_eq "1" "$(grep -c '^pruned: ' <<<"$out")" "exactly one pruned line is printed over the limit"
expect_match 'pruned: go-build cache, check-home older than 7 days \(' "$out" "the line names what was pruned and the size freed"
check "the total is under the limit after prune" bash -c "(( $(total_kib "$d") < 2048 ))"
check "check-home contents newer than the keep days stay" test -e "$d/check-home/new"
check "check-home contents older than the keep days go" test ! -e "$d/check-home/old"
check "go clean -cache ran" grep -q 'go clean -cache' "$d/calls.log"
check "go-mod is not touched" test -e "$d/cache/go-mod/blob"
expect_match 'rc=0$' "$out" "prune_caches ends 0"

d="$(state_dir caches_under)"
fill "$d/cache/go-build" 300
fill "$d/check-home/old" 100
touch -d '10 days ago' "$d/check-home/old"
out="$(run_caches "$d" REVIEWER_CACHE_MAX=2M)"
check "under the limit nothing is removed" test -e "$d/cache/go-build/blob" -a -e "$d/check-home/old"
check "under the limit no line is printed" bash -c "! grep -q '^pruned:' <<<'$out'"
check "under the limit go is not run" bash -c "! grep -qs 'go clean' '$d/calls.log'"
check "config.env ships REVIEWER_CACHE_MAX=2G" grep -qx 'REVIEWER_CACHE_MAX=2G' "$skill/config.env"

d="$(state_dir caches_fail)"
fill "$d/cache/go-build" 3000
out="$(run_caches "$d" REVIEWER_CACHE_MAX=1M GO_FAIL=1)"
expect_match 'WARN prune: go clean -cache failed' "$(cat "$d/state/reviewer.log")" "a failing go clean is logged"
expect_match 'rc=0$' "$out" "a failing go clean does not fail prune"

d="$(state_dir caches_lock)"
fill "$d/cache/go-build" 3000
( exec 9>"$d/state/locks/pr-5.lock"; flock 9; sleep 30 ) &
holder=$!
sleep 0.3
run_caches "$d" REVIEWER_CACHE_MAX=1M >/dev/null
kill "$holder" 2>/dev/null
wait "$holder" 2>/dev/null
check "a review holding a lock blocks the trim" test -e "$d/cache/go-build/blob"
expect_match 'WARN prune: .*in progress' "$(cat "$d/state/reviewer.log")" "the skipped trim is logged"

d="$(state_dir caches_held)"
fill "$d/cache/go-build" 3000
touch "$d/state/locks/pr-9.lock"
cat >"$work/gobin2/go" <<PROBE
#!/usr/bin/env bash
if (exec 5>"$d/state/locks/pr-9.lock"; flock -n 5); then echo free >"$d/probe"; else echo busy >"$d/probe"; fi
rm -rf "\${GOCACHE:?}"/*
PROBE
chmod +x "$work/gobin2/go"
run_caches "$d" REVIEWER_CACHE_MAX=1M PATH="$work/gobin2:$PATH" >/dev/null
expect_eq "busy" "$(cat "$d/probe" 2>/dev/null)" "the pr locks are held while the cache is removed"

echo "# the guard accepts only direct children named pr-* or merge-pr-*"
d="$(state_dir guard)"
out="$(
  export HOME="$fake_home" GABO_REVIEWER_STATE_DIR="$d" REVIEWER_REPO=example-org/example-repo
  cd "$work" || exit 1
  source "$skill/common.sh"
  remove_run_worktree "$d/worktrees/other-1"; echo "other=$?"
  remove_run_worktree "$d/worktrees/pr-1/sub"; echo "sub=$?"
  remove_run_worktree "$work/pr-1"; echo "outside=$?"
)"
expect_match 'other=1' "$out" "another name is refused"
expect_match 'sub=1' "$out" "a nested path is refused"
expect_match 'outside=1' "$out" "a path outside the worktrees dir is refused"

echo "# removal after the merge (merge-head.sh)"
merge_case() {
  local name="$1" fx d
  shift
  d="$(state_dir "$name")"
  fx="$d/fx"
  add_wt "$d" 7 OPEN
  jq -n '{number:7, state:"OPEN", baseRefName:"dev", isDraft:false, isCrossRepository:false, headRefOid:"0123456789abcdef0123456789abcdef01234567"}' >"$fx/pr_7.json"
  local sha=0123456789abcdef0123456789abcdef01234567
  jq -n '[{context:"reviewer/tests",state:"success",creator:{login:"owner"}},{context:"reviewer/review",state:"success",creator:{login:"owner"}}]' >"$fx/statuses_$sha.json"
  echo '{"commit":{"committer":{"date":"2026-10-20T11:00:00Z"}}}' >"$fx/commit_$sha.json"
  jq -n --arg b $'No blocking issues found.\n\n<!-- gbai-reviewer-bot sha='"$sha"$' -->\n🤖 gabo reviewer' \
    '[{user:{login:"owner"}, created_at:"2026-10-20T12:00:00Z", body:$b}]' >"$fx/comments_7.json"
  echo '[{"filename":"apps/cli/main.go"}]' >"$fx/files_7"
  (
    export HOME="$fake_home" STUB_LOG="$fx/calls.log" FIXTURE="$fx" REVIEWER_GH="$work/bin/gh" \
      GABO_REVIEWER_STATE_DIR="$d" REVIEWER_AUTO_MERGE=true REVIEWER_NOW=2026-10-20T13:00:00Z \
      REVIEWER_REPO=example-org/example-repo REVIEWER_CHECKS='go test ./...'
    for kv in "$@"; do export "${kv?}"; done
    cd "$work" || exit 1
    bash "$skill/merge-head.sh" 7 "$sha" >"$d/out.log" 2>&1
  )
  echo "$d"
}
d="$(merge_case merged)"
expect_match ' MERGED #7 ' "$(tail -n 1 "$d/out.log")" "the merge succeeds and its line stays last"
check "the pr-<n> worktree is removed after a successful merge" gone "$d" 7
d="$(merge_case notmerged MERGE_FAIL=1)"
expect_match ' MERGE-FAILED ' "$(tail -n 1 "$d/out.log")" "a refused merge is MERGE-FAILED"
check "the pr-<n> worktree stays when the merge fails" exists "$d" 7
d="$(merge_case skipped REVIEWER_AUTO_MERGE=false)"
check "the pr-<n> worktree stays when the merge is skipped" exists "$d" 7

echo "# reviewer.sh prune and head"
origin_dir="$work/origin.git"
make_origin "$origin_dir"
caller="$work/caller"
git clone -q --branch dev "$origin_dir" "$caller"
reviewer() {
  local name="$1" mode="$2"
  shift 2
  local d="$work/$name"
  mkdir -p "$d/fx"
  echo owner >"$d/fx/user_login"
  jq -n --arg sha "$HEAD_SHA" '{number:7, state:"OPEN", headRefOid:$sha, baseRefName:"dev", isDraft:false, isCrossRepository:false, mergeable:"MERGEABLE"}' >"$d/fx/pr_7.json"
  pr_files_json "$origin_dir" >"$d/fx/files_7"
  echo '[]' >"$d/fx/comments_7.json"
  echo '{"commit":{"committer":{"date":"2026-10-20T11:00:00Z"}}}' >"$d/fx/commit_$HEAD_SHA.json"
  (
    cd "$caller" || exit 1
    export HOME="$fake_home" PATH="$work/bin:$PATH" STUB_LOG="$d/calls.log" FIXTURE="$d/fx" \
      REVIEWER_GH="$work/bin/gh" REVIEWER_CLAUDE="$work/bin/claude" REVIEWER_GABO_BIN="$work/bin/gabo" \
      GABO_REVIEWER_STATE_DIR="$d" REVIEWER_REMOTE_URL="$origin_dir" REVIEWER_GH_TOKEN=secret-gh \
      CLAUDE_CODE_OAUTH_TOKEN=secret-claude REVIEWER_NOW=2026-10-20T12:00:00Z REVIEWER_MERGE_DELAY=0 \
      REVIEWER_MERGE_TRIES=2
    for kv in "$@"; do export "${kv?}"; done
    write_stub_context "$work/bin"
    if [[ "$mode" == head ]]; then
      bash "$skill/reviewer.sh" head 7 "$HEAD_SHA" >"$d/out.log" 2>&1
    else
      bash "$skill/reviewer.sh" prune >"$d/out.log" 2>&1
    fi
    echo "exit=$?" >>"$d/out.log"
  )
  echo "$d"
}
seed() {
  # a mirror with the stale worktrees of PRs 1 (closed) and 2 (open)
  local d="$work/$1"
  mkdir -p "$d/worktrees" "$d/fx"
  git init -q --bare "$d/repo.git"
  git -C "$d/repo.git" remote add origin "$origin_dir"
  git -C "$d/repo.git" fetch -q --no-tags origin "+refs/heads/dev:refs/remotes/origin/dev"
  for n in 1 2; do
    git -C "$d/repo.git" worktree add -q --detach "$d/worktrees/pr-$n" "$DEV_SHA"
  done
  echo '{"state":"CLOSED"}' >"$d/fx/pr_1.json"
  echo '{"state":"OPEN"}' >"$d/fx/pr_2.json"
}
seed prune_cmd
fill "$work/prune_cmd/cache/go-build" 3000
mkdir -p "$work/gobin3" && cp "$work/gobin/go" "$work/gobin3/go"
d="$(reviewer prune_cmd prune REVIEWER_CACHE_MAX=1M PATH="$work/gobin3:$PATH")"
expect_match '^pruned: ' "$(cat "$d/out.log")" "reviewer.sh prune prints the pruned line"
check "reviewer.sh prune exits 0" grep -qx 'exit=0' "$d/out.log"
check "reviewer.sh prune removes the closed PR's worktree" gone "$d" 1
check "reviewer.sh prune keeps the open PR's worktree" exists "$d" 2
seed head_run
d="$(reviewer head_run head)"
expect_match "VERDICT PR #7 head $HEAD_SHA verdict=" "$(tail -n 2 "$d/out.log" | head -n 1)" "head still ends with its VERDICT line"
check "head prunes the closed PR's worktree at its start" gone "$d" 1
check "head keeps the open PR's worktree" exists "$d" 2

finish
