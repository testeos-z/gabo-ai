#!/usr/bin/env bash
# Tests for the reviewer state isolation by repository (intent 0026 S11): two
# repositories that share one state dir and the same PR number keep their own
# mirror, PR refs and base checkout; the legacy unqualified names stay in
# place; remove_run_worktree picks the mirror per worktree. Offline, on
# scratch state dirs only.
# Run: bash tests/repo-isolation.test.sh
set -uo pipefail
source "$(dirname "$0")/lib.sh"
source "$(dirname "$0")/stubs.sh"

work="$(new_tmp)"
fake_home="$work/home"
skill="$(install_skill "$fake_home")"
make_stubs "$work/bin"
state="$work/state"
g=(git -c user.name=test -c user.email=test@example.com -c commit.gpgsign=false -c init.defaultBranch=dev)

# origin <name> <O_PR>: the fixture repository <name> (its own head commit) and
# a caller checkout; sets <NAME>_ORIGIN, <NAME>_HEAD and <NAME>_CALLER.
origin() {
  local name="$1" up
  up="$(tr a-z A-Z <<<"$name")"
  O_PR="$2" make_origin "$work/$name.git"
  printf -v "${up}_ORIGIN" %s "$work/$name.git"
  printf -v "${up}_HEAD" %s "$HEAD_SHA"
  printf -v "${up}_DEV" %s "$DEV_SHA"
  git clone -q --branch dev "$work/$name.git" "$work/$name-caller"
}
origin a code
origin b skills

# review <name> <repo> <origin> <caller> <head>: one head review of PR 7 of
# <repo> in the shared state dir, with its own fixtures.
review() {
  local name="$1" repo="$2" org="$3" caller="$4" head="$5" fx="$work/fx-$1"
  mkdir -p "$fx"
  echo owner >"$fx/user_login"
  jq -n --arg sha "$head" '{number:7, state:"OPEN", headRefOid:$sha, baseRefName:"dev", isDraft:false, isCrossRepository:false, mergeable:"MERGEABLE"}' >"$fx/pr_7.json"
  git -C "$org" diff --name-only "$(git -C "$org" rev-parse refs/heads/dev~1 2>/dev/null || echo dev)" "$head" 2>/dev/null | jq -R . | jq -sc 'map({filename: .})' >"$fx/files_7"
  echo '[]' >"$fx/comments_7.json"
  echo '{"commit":{"committer":{"date":"2026-10-20T11:00:00Z"}}}' >"$fx/commit_$head.json"
  (
    cd "$caller" || exit 1
    export HOME="$fake_home" PATH="$work/bin:$PATH" STUB_LOG="$fx/calls.log" FIXTURE="$fx" \
      REVIEWER_GH="$work/bin/gh" REVIEWER_CLAUDE="$work/bin/claude" REVIEWER_GABO_BIN="$work/bin/gabo" \
      GABO_REVIEWER_STATE_DIR="$state" REVIEWER_REPO="$repo" REVIEWER_REMOTE_URL="$org" \
      REVIEWER_GH_TOKEN=secret-gh CLAUDE_CODE_OAUTH_TOKEN=secret-claude \
      REVIEWER_NOW=2026-10-20T12:00:00Z REVIEWER_MERGE_DELAY=0 REVIEWER_MERGE_TRIES=2
    write_stub_context "$work/bin"
    bash "$skill/reviewer.sh" head 7 "$head" >"$fx/out.log" 2>&1
    tail -n 1 "$fx/out.log"
  ) >/dev/null
  grep -c "^VERDICT PR #7 head $head " "$fx/out.log"
}

echo "# two repositories, one state dir, the same PR number"
expect_eq 1 "$(review a org-a/shared "$A_ORIGIN" "$work/a-caller" "$A_HEAD")" "repository A is reviewed (VERDICT line)"
expect_eq 1 "$(review b org-b/shared "$B_ORIGIN" "$work/b-caller" "$B_HEAD")" "repository B is reviewed (VERDICT line)"
ma="$state/org-a__shared.git"
mb="$state/org-b__shared.git"
check "A has its own qualified mirror" test -d "$ma"
check "B has its own qualified mirror" test -d "$mb"
check "no unqualified mirror repo.git is created" test ! -e "$state/repo.git"
check "no unqualified base checkout is created" test ! -e "$state/base"
check "A has its own base checkout" test -f "$state/base-org-a__shared/gabo.yaml"
check "B has its own base checkout" test -f "$state/base-org-b__shared/gabo.yaml"
check "the base checkouts have different git dirs" test \
  "$(realpath "$(git -C "$state/base-org-a__shared" rev-parse --git-common-dir)")" != "$(realpath "$(git -C "$state/base-org-b__shared" rev-parse --git-common-dir)")"
expect_eq "$A_HEAD" "$(git -C "$ma" rev-parse refs/remotes/pr/org-a__shared/7 2>/dev/null)" "A's mirror holds A's head of PR 7 under a qualified ref"
expect_eq "$B_HEAD" "$(git -C "$mb" rev-parse refs/remotes/pr/org-b__shared/7 2>/dev/null)" "B's mirror holds B's head of PR 7 under a qualified ref"
check "A's mirror does not hold B's head" bash -c "! git -C '$ma' cat-file -e '$B_HEAD^{commit}' 2>/dev/null"
check "no PR ref is keyed by the PR number alone" bash -c "! git -C '$ma' for-each-ref --format='%(refname)' | grep -E '^refs/remotes/pr/7(-merge)?$'"
check "the merge-result ref is qualified too" git -C "$ma" rev-parse -q --verify refs/remotes/pr/org-a__shared/7-merge
expect_eq "$A_ORIGIN" "$(git -C "$ma" remote get-url origin)" "A's mirror origin is still A's remote after B was reviewed"
expect_eq "$B_ORIGIN" "$(git -C "$mb" remote get-url origin)" "B's mirror origin is B's remote"
expect_eq "$A_DEV" "$(git -C "$state/base-org-a__shared" rev-parse HEAD)" "A's checkout is at A's integration tip"
expect_eq "$B_DEV" "$(git -C "$state/base-org-b__shared" rev-parse HEAD)" "B's checkout is at B's integration tip"
expect_eq 1 "$(find "$state/state/locks" -name 'mirror*.lock' | wc -l)" "the mirror lock stays one lock"
check "the mirror lock is mirror.lock" test -e "$state/state/locks/mirror.lock"

echo "# an existing unqualified mirror, refs and base stay in place (open question 9)"
legacy="$work/legacy"
mkdir -p "$legacy/worktrees"
"${g[@]}" init -q --bare "$legacy/repo.git"
tree="$(git -C "$legacy/repo.git" mktree </dev/null)"
lc="$("${g[@]}" -C "$legacy/repo.git" commit-tree "$tree" -m legacy)"
git -C "$legacy/repo.git" update-ref refs/heads/dev "$lc"
git -C "$legacy/repo.git" update-ref refs/remotes/pr/7 "$lc"
git -C "$legacy/repo.git" remote add origin "$work/legacy-origin.git"
git -C "$legacy/repo.git" worktree add -q --detach "$legacy/base" "$lc"
echo keep >"$legacy/base/legacy-marker"
git -C "$legacy/repo.git" worktree add -q --detach "$legacy/worktrees/pr-9" "$lc"
touch -d '10 days ago' "$legacy/worktrees/pr-9"
# a qualified head worktree of PR 7 made before the mirrors were qualified: it is
# registered in repo.git, which never receives the new head
git -C "$legacy/repo.git" worktree add -q --detach "$legacy/worktrees/org-a__shared__pr-7" "$lc"
state_saved="$state"
state="$legacy"
expect_eq 1 "$(review c org-a/shared "$A_ORIGIN" "$work/a-caller" "$A_HEAD")" "a review runs beside the unqualified names"
state="$state_saved"
check "the unqualified mirror keeps its ref" test "$(git -C "$legacy/repo.git" rev-parse refs/remotes/pr/7)" = "$lc"
check "the unqualified mirror keeps its origin" test "$(git -C "$legacy/repo.git" remote get-url origin)" = "$work/legacy-origin.git"
check "the unqualified base checkout is untouched" test -f "$legacy/base/legacy-marker"
check "the run created the qualified mirror beside them" test -d "$legacy/org-a__shared.git"
check "the old qualified head worktree is made again from the repository's mirror" test \
  "$(realpath "$(git -C "$legacy/worktrees/org-a__shared__pr-7" rev-parse --git-common-dir)")" = "$(realpath "$legacy/org-a__shared.git")"
expect_eq "$A_HEAD" "$(git -C "$legacy/worktrees/org-a__shared__pr-7" rev-parse HEAD)" "it is at the new head"
check "the legacy pr-9 worktree was removed by prune_legacy from repo.git" test ! -e "$legacy/worktrees/pr-9"
check "repo.git no longer lists pr-9" bash -c "! git -C '$legacy/repo.git' worktree list --porcelain | grep -q 'worktrees/pr-9'"

echo "# remove_run_worktree picks the mirror per worktree"
rm_state="$work/rm"
mkdir -p "$rm_state/worktrees" "$rm_state/state/locks"
for r in org-a__shared org-b__shared; do
  "${g[@]}" init -q --bare "$rm_state/$r.git"
  t="$(git -C "$rm_state/$r.git" mktree </dev/null)"
  c="$("${g[@]}" -C "$rm_state/$r.git" commit-tree "$t" -m c)"
  git -C "$rm_state/$r.git" update-ref refs/heads/dev "$c"
  git -C "$rm_state/$r.git" worktree add -q --detach "$rm_state/worktrees/${r}__pr-5" "$c"
  echo "$c" >"$rm_state/$r.commit"
done
git -C "$rm_state/org-a__shared.git" worktree add -q --detach "$rm_state/worktrees/merge-pr-5-abc.1p1" "$(cat "$rm_state/org-a__shared.commit")"
"${g[@]}" init -q --bare "$rm_state/repo.git"
t="$(git -C "$rm_state/repo.git" mktree </dev/null)"
c="$("${g[@]}" -C "$rm_state/repo.git" commit-tree "$t" -m c)"
git -C "$rm_state/repo.git" worktree add -q --detach "$rm_state/worktrees/pr-5" "$c"
git -C "$rm_state/repo.git" worktree add -q --detach "$rm_state/worktrees/org-a__shared__pr-6" "$c"
rm_wt() {
  (
    export HOME="$fake_home" GABO_REVIEWER_STATE_DIR="$rm_state" REVIEWER_REPO=org-a/shared
    cd "$work" || exit 1
    source "$skill/common.sh"
    remove_named_worktree "$1"
  ) >/dev/null 2>&1
}
check "a qualified worktree of A is removed from A's mirror" rm_wt org-a__shared__pr-5
check "A's worktree is gone" test ! -e "$rm_state/worktrees/org-a__shared__pr-5"
check "a qualified worktree of B is removed from B's mirror" rm_wt org-b__shared__pr-5
check "B's worktree is gone" test ! -e "$rm_state/worktrees/org-b__shared__pr-5"
check "the merge-pr-* worktree of the run is removed from the run's mirror" rm_wt merge-pr-5-abc.1p1
check "the merge-pr-* worktree is gone" test ! -e "$rm_state/worktrees/merge-pr-5-abc.1p1"
check "a legacy pr-5 worktree is removed from the unqualified repo.git" rm_wt pr-5
check "the legacy worktree is gone" test ! -e "$rm_state/worktrees/pr-5"
check "a qualified worktree that only repo.git lists is removed from repo.git" rm_wt org-a__shared__pr-6
check "that worktree is gone" test ! -e "$rm_state/worktrees/org-a__shared__pr-6"

finish
