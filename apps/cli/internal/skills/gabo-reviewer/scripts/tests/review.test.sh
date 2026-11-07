#!/usr/bin/env bash
# End-to-end tests of one review: reviewer.sh head -> review-head.sh ->
# merge-head.sh, offline with fake gh, claude, pi, codex, opencode, go and gabo, and a local bare
# repository standing in for GitHub.
# Run: bash tests/review.test.sh
set -uo pipefail
source "$(dirname "$0")/lib.sh"
source "$(dirname "$0")/stubs.sh"

work="$(new_tmp)"
fake_home="$work/home"
skill="$(install_skill "$fake_home")"
make_stubs "$work/bin"
# The owner's Pi login and model config, which a Pi review reuses.
mkdir -p "$fake_home/.pi/agent"
echo '{"openai": {"type": "oauth"}}' >"$fake_home/.pi/agent/auth.json"
echo '{"defaultProvider": "openai", "defaultModel": "gpt-5.5"}' >"$fake_home/.pi/agent/settings.json"
# The owner's Codex login and the isolation pass marker of the stub codex.
mkdir -p "$fake_home/.codex" "$fake_home/.config/gabo/reviewer-isolation"
echo '{"auth_mode": "chatgpt"}' >"$fake_home/.codex/auth.json"
echo "codex-cli 0.163.0" >"$fake_home/.config/gabo/reviewer-isolation/codex"

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
# fresh state dir and fixture, run from CALLER, with `engine $SCENARIO_ENGINE`
# when it is set; prints the VERDICT line.
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
      REVIEWER_GH="$work/bin/gh" REVIEWER_CLAUDE="$work/bin/claude" REVIEWER_PI="$work/bin/pi" REVIEWER_CODEX="$work/bin/codex" \
      REVIEWER_GABO_BIN="$work/bin/gabo" GABO_REVIEWER_STATE_DIR="$fx/state" REVIEWER_REMOTE_URL="$ORIGIN" \
      REVIEWER_GH_TOKEN=secret-gh CLAUDE_CODE_OAUTH_TOKEN=secret-claude \
      REVIEWER_NOW=2026-10-20T12:00:00Z REVIEWER_MERGE_DELAY=0 REVIEWER_MERGE_TRIES=2
    for kv in "$@"; do export "${kv?}"; done
    write_stub_context "$work/bin"
    bash "$skill/reviewer.sh" head 7 "$HEAD_SHA" ${SCENARIO_ENGINE+engine "$SCENARIO_ENGINE"} >"$fx/out.log" 2>&1
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
check "merge is pinned to the reviewed head" grep -qE "^gh pr merge 7 --repo example-org/example-repo --merge --match-head-commit $HEAD_SHA$" "$work/clean/calls.log"
expect_eq 3 "$(grep -c '^go-argc=' "$work/clean/calls.log")" "runs the three checks of review.checks"
check "checks are argv: go -C apps/cli build ./..." grep -qF "go-argc=4 args=go -C apps/cli build ./..." "$work/clean/calls.log"
check "checks run on the merge result" grep -qE "^go-argc=.* head=$MERGE_SHA$" "$work/clean/calls.log"
check "checks get no credential" bash -c "! grep '^check-env' '$work/clean/calls.log' | grep -q secret"
check "checks get the reviewer's own HOME" grep -qF "HOME=$work/clean/state/check-home CI=true" "$work/clean/calls.log"
check "gabo check is not run when review.checks does not list it" bash -c "! grep -q '^gabo-call check' '$work/clean/calls.log'"
check "review runs in the head checkout" grep -qE "^claude -p .* cwd=$work/clean/state/worktrees/pr-7$" "$work/clean/calls.log"
check "review gets only the Claude token" grep -qE "^cc-env GH_TOKEN=unset REVIEWER_GH_TOKEN=unset CLAUDE_CODE_OAUTH_TOKEN=secret-claude HOME=$work/clean/state/review-home$" "$work/clean/calls.log"
check "review is read-only" grep -qE '^claude -p .*--permission-mode dontAsk --tools Read,Grep,Glob,Bash' "$work/clean/calls.log"
check "prompt carries the PR diff" grep -q 'func main' "$work/clean/prompt_seen.txt"
check "merge-result worktree is removed" bash -c "! ls -d '$work/clean/state/worktrees/'merge-pr-* 2>/dev/null"
expect_eq 1 "$(grep -c 'gbai-reviewer-bot sha=' "$work/clean/all_posted_7.md")" "one verdict comment per head"

echo "# already reviewed"
fx="$work/clean"
out="$(cd "$CALLER" && HOME="$fake_home" PATH="$work/bin:$PATH" STUB_LOG="$fx/calls.log" FIXTURE="$fx" REVIEWER_GH="$work/bin/gh" \
  REVIEWER_GABO_BIN="$work/bin/gabo" REVIEWER_REMOTE_URL="$ORIGIN" GABO_REVIEWER_STATE_DIR="$fx/state" \
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
  "a PR that only edits .gabo/skills/ is not merged even with review.auto_merge: true"
check "no merge call for a catalog edit" no_merge skills_only
check "the needs-human comment names the catalog file" grep -qF '.gabo/skills/gabo-coder/SKILL.md' "$work/skills_only/posted_7.md"

echo "# configured reviewer.signing marker"
origin marker O_MARKER=team-bot
scenario marker >/dev/null
expect_eq "<!-- team-bot sha=$HEAD_SHA -->|🤖 gabo reviewer" "$(footer marker)" "the verdict comment carries the reviewer.signing marker"

echo "# gabo check listed in review.checks"
origin gabocheck O_CHECKS='go -C apps/cli test ./...
gabo check'
scenario gabocheck >/dev/null
check "gabo check runs with the configured binary" grep -qE '^gabo-call check GABO_UPGRADE_PR=unset GH_TOKEN=unset' "$work/gabocheck/calls.log"
check "the comment lists the check as configured" grep -qF '| `gabo check` | passed' "$work/gabocheck/posted_7.md"
origin gabocheck O_CHECKS='gabo check'
scenario gabocheck >/dev/null
check "gabo check (the current binary name) also runs the configured binary" grep -qE '^gabo-call check ' "$work/gabocheck/calls.log"

echo "# review.check_env"
origin checkenv O_CHECK_ENV='TEST_DB_URL'
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=skipped" \
  "$(scenario checkenv TEST_DB_URL=listed-value UNLISTED_VAR=unlisted-value)" "a listed, exported variable passes the checks"
check "a listed variable is visible to the checks" grep -qxF 'check-vars TEST_DB_URL=listed-value UNLISTED_VAR=unset' "$work/checkenv/calls.log"
check "an unlisted variable is not visible to the checks" bash -c "! grep '^check-vars' '$work/checkenv/calls.log' | grep -q unlisted-value"
check "the review step does not get listed variables" grep -qxF 'cc-vars TEST_DB_URL=unset UNLISTED_VAR=unset' "$work/checkenv/calls.log"
check "the value never reaches the comment" bash -c "! grep -q listed-value '$work/checkenv/all_posted_7.md'"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=blocking merge=none" "$(scenario checkenv_unset)" \
  "a listed, unset variable fails the checks"
expect_match '^reviewer/tests failure .*TEST_DB_URL' "$(status_line checkenv_unset reviewer/tests)" \
  "the reviewer/tests failure names the variable"
check "no check runs when a listed variable is unset" bash -c "! grep -q '^go-argc' '$work/checkenv_unset/calls.log'"
check "the comment names the unset variable" grep -qF 'TEST_DB_URL' "$work/checkenv_unset/posted_7.md"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=blocking merge=none" "$(scenario checkenv_empty TEST_DB_URL=)" \
  "a listed, empty variable fails the checks"
check "no check runs when a listed variable is empty" bash -c "! grep -q '^go-argc' '$work/checkenv_empty/calls.log'"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=blocking merge=none" \
  "$(scenario checkenv_failing TEST_DB_URL=listed-value FAIL_CHECK=vet)" "a failing check with a listed variable is blocking"
check "the failure output is posted" grep -qF 'connection refused' "$work/checkenv_failing/posted_7.md"
check "a listed value printed by a failing check never reaches the comment" \
  bash -c "! grep -q listed-value '$work/checkenv_failing/all_posted_7.md'"
check "the failure output names the redacted variable" grep -qF 'open [review.check_env:TEST_DB_URL]: connection refused' "$work/checkenv_failing/posted_7.md"

echo "# pi reviewer reusing the owner's Pi login (runtimes.reviewer: pi)"
origin pi O_REVIEWER=pi O_AUTO_MERGE=true O_PR=pi
v="$(scenario pi_clean)"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=merged" "$v" "a clean Pi review ends with the same VERDICT line and merges"
expect_eq "No blocking issues found." "$(first_line pi_clean)" "the Pi comment opens with the clean verdict"
expect_eq "<!-- gbai-reviewer-bot sha=$HEAD_SHA -->|🤖 gabo reviewer" "$(footer pi_clean)" "the Pi comment ends with the marker and the footer"
check "the Pi comment carries the findings" grep -qF 'All files comply.' "$work/pi_clean/posted_7.md"
check "the Pi comment names the engine" grep -qF 'then a Pi review' "$work/pi_clean/posted_7.md"
expect_match '^reviewer/review success pi: no blocking issues found' "$(status_line pi_clean reviewer/review)" "the review status names the engine"
check "claude never runs for a Pi reviewer" bash -c "! grep -q '^claude ' '$work/pi_clean/calls.log'"
check "the Pi review runs in the head checkout" grep -qE "^pi .* cwd=$work/pi_clean/state/worktrees/pr-7$" "$work/pi_clean/calls.log"
check "the Pi review is one-shot JSON without a saved session" grep -qE '^pi --mode json --no-session ' "$work/pi_clean/calls.log"
check "the Pi review is read-only: only read, grep, find and ls" grep -qE '^pi .* --tools read,grep,find,ls --exclude-tools bash,edit,write,mcp__\* ' "$work/pi_clean/calls.log"
check "the Pi review skips project files, extensions, skills, prompt templates, themes, context files and MCP" \
  grep -qE '^pi .*--no-approve --no-extensions --no-skills --no-prompt-templates --no-themes --no-context-files --no-mcp ' "$work/pi_clean/calls.log"
expect_eq "pi-project=ignored" "$(grep '^pi-project=' "$work/pi_clean/calls.log")" "the PR head's .pi settings, MCP servers and extension never load"
check "the Pi review reuses the owner's HOME and no token" grep -qxF \
  "pi-env GH_TOKEN=unset REVIEWER_GH_TOKEN=unset CLAUDE_CODE_OAUTH_TOKEN=unset PI_TEST_KEY=unset HOME=$fake_home PI_CODING_AGENT_DIR=unset" "$work/pi_clean/calls.log"
expect_eq "pi-files auth=yes settings=yes" "$(grep '^pi-files ' "$work/pi_clean/calls.log")" "the Pi review sees the owner's auth.json and settings.json"
check "the Pi prompt carries the PR diff" grep -q 'func main' "$work/pi_clean/prompt_seen.txt"
expect_eq 1 "$(grep -c 'gbai-reviewer-bot sha=' "$work/pi_clean/all_posted_7.md")" "one Pi verdict comment per head"
scenario pi_agent_dir PI_CODING_AGENT_DIR="$fake_home/.pi/agent" >/dev/null
check "a custom PI_CODING_AGENT_DIR of the owner reaches the review" \
  grep -qE "^pi-env .* HOME=$fake_home PI_CODING_AGENT_DIR=$fake_home/.pi/agent$" "$work/pi_agent_dir/calls.log"
check "without REVIEWER_PI_MODEL the owner's default Pi model is used" bash -c "! grep -qE '^pi .*--model' '$work/pi_clean/calls.log'"
scenario pi_model REVIEWER_PI_MODEL=openai-codex/gpt-5.3-codex >/dev/null
check "REVIEWER_PI_MODEL picks the Pi model" grep -qE '^pi .* --model openai-codex/gpt-5.3-codex -- ' "$work/pi_model/calls.log"
check "the comment names the Pi model" grep -qF 'then a Pi review (`openai-codex/gpt-5.3-codex`).' "$work/pi_model/posted_7.md"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=blocking merge=none" "$(scenario pi_fail PI_MODE=fail)" "blocking Pi findings are blocking"
check "the Pi findings are posted" grep -qF 'nil dereference' "$work/pi_fail/posted_7.md"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario pi_error PI_MODE=error)" "a Pi run that ends in error is an error"
expect_match 'pi exit 1' "$(status_line pi_error reviewer/review)" "the error status names the engine"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario pi_modify PI_MODE=modify REVIEWER_MAX_ATTEMPTS=1)" \
  "a Pi agent that edits the checkout is an error"

echo "# pi reviewer with review.pi_credential_env"
origin pikey O_REVIEWER=pi O_PI_ENV=PI_TEST_KEY
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=skipped" "$(scenario pi_key PI_TEST_KEY=secret-pi)" "a Pi review with a credential env var passes"
check "only the review gets the named credential, with an empty HOME of its own" grep -qxF \
  "pi-env GH_TOKEN=unset REVIEWER_GH_TOKEN=unset CLAUDE_CODE_OAUTH_TOKEN=unset PI_TEST_KEY=secret-pi HOME=$work/pi_key/state/review-home PI_CODING_AGENT_DIR=unset" "$work/pi_key/calls.log"
expect_eq "pi-files auth=no settings=no" "$(grep '^pi-files ' "$work/pi_key/calls.log")" "with a credential the owner's Pi files are not used"
check "checks never get the Pi credential" bash -c "! grep '^check-env' '$work/pi_key/calls.log' | grep -q secret-pi"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario pi_key_unset)" "a named credential env var that is empty is refused"
check "the refusal names the env var" grep -qF 'review.pi_credential_env names PI_TEST_KEY, which is empty' "$work/pi_key_unset/out.log"
check "nothing runs without the named credential" bash -c "! grep -qE '^(pi|go-argc)' '$work/pi_key_unset/calls.log'"

echo "# engine argument under runtimes.reviewer: session (intent 0013 S11, D5)"
origin session O_REVIEWER=session O_ASSISTANT='[claude, pi]' O_AUTO_MERGE=true O_PR=pi
v="$(SCENARIO_ENGINE=pi scenario session_pi)"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=merged" "$v" "session with engine pi listed in runtimes.assistant reviews and merges"
check "the session review runs on Pi" grep -qE '^pi --mode json --no-session ' "$work/session_pi/calls.log"
check "claude never runs for a session review on Pi" bash -c "! grep -q '^claude ' '$work/session_pi/calls.log'"
check "the comment says a Pi review" grep -qF 'then a Pi review' "$work/session_pi/posted_7.md"
expect_match '^reviewer/review success pi: no blocking issues found' "$(status_line session_pi reviewer/review)" \
  "the review status names the resolved engine"
v="$(scenario session_none)"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$v" "(c) session without an engine argument ends with an error VERDICT"
check "(c) the refusal names the missing engine argument" grep -qF 'PREFLIGHT-FAILED runtimes.reviewer on dev is session: pass the engine' "$work/session_none/out.log"
check "(c) no engine runs, nothing is posted or merged" \
  bash -c "! grep -qE '^(claude |pi |codex exec|opencode run|gh pr merge|gh api -X POST|STATUS )' '$work/session_none/calls.log'"
# (e) runtimes.reviewer: session only on the PR head (and in the caller's
# checkout of it): the reviewer reads the integration branch, where it is claude.
origin headonly O_PR=session
git -C "$CALLER" checkout -q "$HEAD_SHA"
v="$(SCENARIO_ENGINE=pi scenario session_headonly)"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$v" "(e) session only on the PR head: an engine argument ends with an error VERDICT"
check "(e) refused: runtimes.reviewer is read from the integration branch, never the head" \
  grep -qxF 'PREFLIGHT-FAILED the engine argument is accepted only when runtimes.reviewer is session; runtimes.reviewer on dev is claude' \
  "$work/session_headonly/out.log"
check "(e) no engine runs, nothing is posted or merged" \
  bash -c "! grep -qE '^(claude |pi |codex exec|opencode run|gh pr merge|gh api -X POST|STATUS )' '$work/session_headonly/calls.log'"
# The child scripts only ever get a resolved engine.
out="$(cd "$CALLER" && HOME="$fake_home" REVIEWER_GH="$work/bin/gh" GABO_REVIEWER_STATE_DIR="$work/session_child" \
  REVIEWER_REPO=example-org/example-repo REVIEWER_CHECKS='go test ./...' REVIEWER_ENGINE=session bash "$skill/review-head.sh" 7 "$HEAD_SHA" 2>&1)"
expect_match "^PREFLIGHT-FAILED REVIEWER_ENGINE must be claude, codex, pi or opencode \(runtimes.reviewer\), got 'session'$" "$out" \
  "review-head.sh refuses an unresolved session engine"

echo "# codex reviewer (runtimes.reviewer: codex)"
origin codex O_REVIEWER=codex O_AUTO_MERGE=true
v="$(scenario codex_clean)"
cx="$work/codex_clean"
cwt="$cx/state/worktrees/pr-7"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=merged" "$v" "a clean Codex review ends with the same VERDICT line and merges"
expect_eq "No blocking issues found." "$(first_line codex_clean)" "the Codex comment opens with the clean verdict"
expect_eq "<!-- gbai-reviewer-bot sha=$HEAD_SHA -->|🤖 gabo reviewer" "$(footer codex_clean)" "the Codex comment ends with the marker and the footer"
check "the final message is the review text" grep -qF 'All files comply.' "$cx/posted_7.md"
check "codex progress output is not the review text" bash -c "! grep -qF 'codex: reading the diff' '$cx/posted_7.md'"
check "the Codex comment names the engine" grep -qF 'then a Codex review' "$cx/posted_7.md"
expect_match '^reviewer/review success codex: no blocking issues found' "$(status_line codex_clean reviewer/review)" "the review status names the codex engine"
check "claude never runs for a Codex reviewer" bash -c "! grep -q '^claude ' '$cx/calls.log'"
check "the agent review runs codex exec in the head checkout" grep -qE "^codex exec .* cwd=$cwt$" "$cx/calls.log"
for flag in --ignore-user-config --ignore-rules '-s read-only' --ephemeral "-C $cwt"; do
  check "codex exec gets $flag" grep -qE "^codex exec (.* )?$flag( .*)? cwd=" "$cx/calls.log"
done
check "the prompt goes on stdin (codex exec ... -)" grep -qE "^codex exec .* - cwd=$cwt$" "$cx/calls.log"
check "the Codex prompt carries the PR diff" grep -q 'func main' "$cx/prompt_seen.txt"
check "the Codex review gets no GitHub or Claude token, a reviewer-owned HOME and CODEX_HOME" grep -qxF \
  "codex-env GH_TOKEN=unset REVIEWER_GH_TOKEN=unset CLAUDE_CODE_OAUTH_TOKEN=unset HOME=$cx/state/review-home CODEX_HOME=$cx/state/codex-home" "$cx/calls.log"
expect_eq "codex-vars CODEX_HOME,HOME,NO_COLOR,PATH,TERM" "$(grep '^codex-vars ' "$cx/calls.log")" "the Codex review runs in an env -i environment"
expect_eq "codex-files auth=yes" "$(grep '^codex-files ' "$cx/calls.log")" "the reviewer-owned CODEX_HOME holds the owner's Codex login"
check "the login is linked, never copied" test -L "$cx/state/codex-home/auth.json"
expect_eq "" "$(git -C "$cwt" status --porcelain)" "git status of the checkout is unchanged"
gates="$(grep -nE '^go-argc=' "$cx/calls.log" | tail -n 1 | cut -d: -f1)"
review="$(grep -nE '^codex exec ' "$cx/calls.log" | head -n 1 | cut -d: -f1)"
check "the deterministic gates run before the agent review" test "${gates:-999999}" -lt "${review:-0}"
expect_eq 3 "$(grep -c '^go-argc=' "$cx/calls.log")" "the Codex review still runs the three checks"
expect_eq 1 "$(grep -c 'gbai-reviewer-bot sha=' "$cx/all_posted_7.md")" "one Codex verdict comment per head"
check "without REVIEWER_CODEX_MODEL the owner's default Codex model is used" bash -c "! grep -qE '^codex exec .* -m ' '$cx/calls.log'"
scenario codex_model REVIEWER_CODEX_MODEL=gpt-5.3-codex >/dev/null
check "REVIEWER_CODEX_MODEL picks the Codex model" grep -qE '^codex exec .* -m gpt-5.3-codex - cwd=' "$work/codex_model/calls.log"
check "the comment names the Codex model" grep -qF 'then a Codex review (`gpt-5.3-codex`).' "$work/codex_model/posted_7.md"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=blocking merge=none" "$(scenario codex_fail CODEX_MODE=fail)" "blocking Codex findings are blocking"
check "the Codex findings are posted" grep -qF 'nil dereference' "$work/codex_fail/posted_7.md"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario codex_error CODEX_MODE=error)" "a Codex run that fails is an error"
expect_match 'codex exit 1' "$(status_line codex_error reviewer/review)" "the error status names the codex engine"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario codex_silent CODEX_MODE=silent)" \
  "a Codex run without a final message is an error"
expect_match 'no verdict line in the codex output' "$(status_line codex_silent reviewer/review)" "the missing final message is reported"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario codex_modify CODEX_MODE=modify REVIEWER_MAX_ATTEMPTS=1)" \
  "a Codex agent that edits the checkout is an error"
check "the edited checkout is reported" grep -qF 'the review agent modified the checkout' "$work/codex_modify/posted_7.md"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario codex_missing REVIEWER_CODEX=gabo-test-no-codex)" \
  "codex missing from PATH is an error"
check "the refusal names REVIEWER_CODEX" grep -qF 'no codex binary (REVIEWER_CODEX=gabo-test-no-codex)' "$work/codex_missing/out.log"
check "nothing runs without codex" bash -c "! grep -qE '^(codex|go-argc)' '$work/codex_missing/calls.log'"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario codex_unmarked XDG_CONFIG_HOME="$work/no-marker")" \
  "a Codex reviewer without an isolation pass marker never reviews"
check "the refusal names the missing isolation test" grep -qF 'isolation-test.sh codex' "$work/codex_unmarked/out.log"
check "nothing runs without the isolation pass" bash -c "! grep -qE '^(codex|go-argc)' '$work/codex_unmarked/calls.log'"

echo "# opencode reviewer (runtimes.reviewer: opencode)"
origin opencode O_REVIEWER=opencode O_AUTO_MERGE=true O_PR=opencode
echo "1.0.0" >"$fake_home/.config/gabo/reviewer-isolation/opencode"
# The owner's OpenCode login, linked into the reviewer-owned home.
mkdir -p "$fake_home/.local/share/opencode"
echo '{"openai": {"type": "oauth"}}' >"$fake_home/.local/share/opencode/auth.json"
oc() { scenario "$1" REVIEWER_OPENCODE="$work/bin/opencode" "${@:2}"; }
v="$(oc opencode_clean)"
ox="$work/opencode_clean"
owt="$ox/state/worktrees/pr-7"
oxh="$ox/state/review-home/opencode/pr-7"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=merged" "$v" \
  "a clean OpenCode review ends with the same VERDICT line and merges"
expect_eq "No blocking issues found." "$(first_line opencode_clean)" "the OpenCode comment opens with the clean verdict"
expect_eq "<!-- gbai-reviewer-bot sha=$HEAD_SHA -->|🤖 gabo reviewer" "$(footer opencode_clean)" "the OpenCode comment ends with the marker and the footer"
check "the final assistant text is the review" grep -qF 'All files comply.' "$ox/posted_7.md"
check "earlier assistant text is not the review" bash -c "! grep -qF 'Reading the diff.' '$ox/posted_7.md'"
check "the OpenCode comment names the engine" grep -qF 'then an OpenCode review' "$ox/posted_7.md"
expect_match '^reviewer/review success opencode: no blocking issues found' "$(status_line opencode_clean reviewer/review)" \
  "the review status names the opencode engine"
check "claude never runs for an OpenCode reviewer" bash -c "! grep -q '^claude ' '$ox/calls.log'"
run_line="$(grep -E '^opencode run ' "$ox/calls.log" | head -n 1)"
expect_match '^opencode run --standalone --format json --agent gabo-review --file [^ ]+/prompt\.md ' "$run_line" \
  "the agent review runs opencode run --standalone --format json with the prompt and diff by --file"
check "the run is outside the checkout" bash -c "! grep -qE ' cwd=$owt(/|$)' <<<\"\$1\"" _ "$run_line"
check "the attached file is outside the checkout" bash -c "! grep -qE -- '--file $owt/' <<<\"\$1\"" _ "$run_line"
expect_eq "opencode-project=none" "$(grep '^opencode-project=' "$ox/calls.log")" \
  "the PR's opencode.json, .opencode/agents/ and .opencode/plugins/x.js are not passed to the run"
check "the OpenCode prompt carries the PR diff" grep -q 'func main' "$ox/prompt_seen.txt"
check "the OpenCode prompt names the checkout to read" grep -qF "$owt" "$ox/prompt_seen.txt"
check "the OpenCode prompt names git -C <checkout>" grep -qF "git -C $owt diff" "$ox/prompt_seen.txt"
expect_eq "" "$(cat "$ox/stdin_seen.txt")" "nothing is passed on stdin"
check "the OpenCode review gets no GitHub or Claude token and reviewer-owned HOME and XDG dirs" grep -qxF \
  "opencode-env GH_TOKEN=unset REVIEWER_GH_TOKEN=unset CLAUDE_CODE_OAUTH_TOKEN=unset HOME=$ox/state/review-home XDG_CONFIG_HOME=$oxh/config XDG_DATA_HOME=$oxh/data XDG_STATE_HOME=$oxh/state XDG_CACHE_HOME=$oxh/cache" \
  "$ox/calls.log"
expect_eq "opencode-vars HOME,NO_COLOR,PATH,TERM,XDG_CACHE_HOME,XDG_CONFIG_HOME,XDG_DATA_HOME,XDG_STATE_HOME" \
  "$(grep '^opencode-vars ' "$ox/calls.log")" "the OpenCode review runs in an env -i environment"
expect_eq "opencode-files auth=yes" "$(grep '^opencode-files ' "$ox/calls.log")" "the reviewer-owned home holds the owner's OpenCode login"
check "the login is linked, never copied" test -L "$oxh/data/opencode/auth.json"
cfg="$ox/opencode_config.json"
for scope in '.permission' '.agent["gabo-review"].permission'; do
  expect_eq "deny deny deny" "$(jq -r "$scope | \"\(.edit) \(.write) \(.patch)\"" "$cfg" 2>/dev/null)" \
    "the reviewer-owned config denies edit, write and patch ($scope)"
  for key in shell bash; do
    rules="$(jq -r "$scope.$key | to_entries | map(\"\(.key)=\(.value)\") | sort | join(\",\")" "$cfg" 2>/dev/null)"
    allowed=""
    for g in git "git -C $owt"; do
      for c in diff log show status; do allowed+=",$g $c *=allow,$g $c=allow"; done
    done
    expect_eq "*=deny$(tr ',' '\n' <<<"${allowed#,}" | LC_ALL=C sort | sed 's/^/,/' | tr -d '\n')" \
      "$(tr ',' '\n' <<<"$rules" | grep -v '=deny$' | sed '1i *=deny' | paste -sd, -)" \
      "$key is limited to git diff, git log, git show and git status, also as git -C <checkout> ($scope)"
    for g in git "git -C $owt"; do
      for c in diff log show status; do
        for o in --output --ext-diff --textconv; do
          check "$key denies $g $c $o ($scope)" grep -qxF "$g $c *$o*=deny" <<<"$(tr ',' '\n' <<<"$rules")"
        done
      done
      check "$key denies $g diff --no-index ($scope)" grep -qxF "$g diff *--no-index*=deny" <<<"$(tr ',' '\n' <<<"$rules")"
    done
    expect_eq true "$(jq "$scope.$key | [to_entries[1:][] | .value] | to_entries
      | ([.[] | select(.value == \"allow\") | .key] | max) < ([.[] | select(.value == \"deny\") | .key] | min)" "$cfg")" \
      "the $key denies come after the allows ($scope)"
  done
  expect_eq "*=deny,$owt/**=allow" \
    "$(jq -r "$scope.external_directory | to_entries | map(\"\(.key)=\(.value)\") | sort | join(\",\")" "$cfg" 2>/dev/null)" \
    "only the checkout is readable outside the run directory ($scope)"
done
expect_eq primary "$(jq -r '.agent["gabo-review"].mode' "$cfg" 2>/dev/null)" "the config holds one read-only primary agent"
expect_eq '["gabo-review"]' "$(jq -c '.agent | keys' "$cfg" 2>/dev/null)" "the config holds exactly one agent"
expect_eq "" "$(git -C "$owt" status --porcelain)" "git status of the checkout is unchanged"
gates="$(grep -nE '^go-argc=' "$ox/calls.log" | tail -n 1 | cut -d: -f1)"
review="$(grep -nE '^opencode run ' "$ox/calls.log" | head -n 1 | cut -d: -f1)"
check "the deterministic gates run before the agent review" test "${gates:-999999}" -lt "${review:-0}"
expect_eq 3 "$(grep -c '^go-argc=' "$ox/calls.log")" "the OpenCode review still runs the three checks"
expect_eq 1 "$(grep -c 'gbai-reviewer-bot sha=' "$ox/all_posted_7.md")" "one OpenCode verdict comment per head"
# The owner's opencode.json and .opencode/ in a parent of the state dir (the
# default state dir sits in the owner's home) stay out: the run directory is
# a git repository of its own, where OpenCode's upward config search stops.
op="$work/opencode_parent_config"
mkdir -p "$op/.opencode/agents"
echo '{"permission": {"bash": "allow"}}' >"$op/opencode.json"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=merged" "$(oc opencode_parent_config)" \
  "an OpenCode review with OpenCode config above the state dir still runs"
expect_eq "opencode-project=none" "$(grep '^opencode-project=' "$op/calls.log")" \
  "opencode.json and .opencode/ above the state dir are not loaded"
check "the run directory is the root of a git repository with a commit" \
  git -C "$op/state/opencode-run/pr-7" rev-parse --verify -q HEAD
expect_eq "$op/state/opencode-run/pr-7" "$(git -C "$op/state/opencode-run/pr-7" rev-parse --show-toplevel)" \
  "the git root of the run is the run directory"
check "without REVIEWER_OPENCODE_MODEL the owner's default model is used" bash -c "! grep -qE '^opencode run .* -m ' '$ox/calls.log'"
oc opencode_model REVIEWER_OPENCODE_MODEL=openai/gpt-5.5 >/dev/null
check "REVIEWER_OPENCODE_MODEL picks the OpenCode model" grep -qE '^opencode run .* -m openai/gpt-5.5 ' "$work/opencode_model/calls.log"
check "the comment names the OpenCode model" grep -qF 'then an OpenCode review (`openai/gpt-5.5`).' "$work/opencode_model/posted_7.md"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=blocking merge=none" "$(oc opencode_fail OPENCODE_MODE=fail)" "blocking OpenCode findings are blocking"
check "the OpenCode findings are posted" grep -qF 'nil dereference' "$work/opencode_fail/posted_7.md"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(oc opencode_error OPENCODE_MODE=error)" "an OpenCode run that fails is an error"
expect_match 'opencode exit 1' "$(status_line opencode_error reviewer/review)" "the error status names the opencode engine"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(oc opencode_silent OPENCODE_MODE=silent)" \
  "an OpenCode run without assistant text is an error"
expect_match 'no verdict line in the opencode output' "$(status_line opencode_silent reviewer/review)" "the missing review text is reported"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(oc opencode_error0 OPENCODE_MODE=error0)" \
  "an OpenCode error event with exit 0 is an error"
expect_match 'the opencode run reported an error' "$(status_line opencode_error0 reviewer/review)" "the error event is reported"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(oc opencode_crash OPENCODE_MODE=crash)" \
  "an OpenCode exit 1 without an error event is an error, even with a verdict line"
expect_match 'opencode exit 1' "$(status_line opencode_crash reviewer/review)" "the exit code is reported"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario opencode_missing REVIEWER_OPENCODE=gabo-test-no-opencode)" \
  "opencode missing from PATH is an error"
check "the refusal names REVIEWER_OPENCODE" grep -qF 'no opencode binary (REVIEWER_OPENCODE=gabo-test-no-opencode)' "$work/opencode_missing/out.log"
check "nothing runs without opencode" bash -c "! grep -qE '^(opencode|go-argc)' '$work/opencode_missing/calls.log'"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(oc opencode_unmarked XDG_CONFIG_HOME="$work/no-marker")" \
  "an OpenCode reviewer without an isolation pass marker never reviews"
check "the refusal names the missing isolation test" grep -qF 'isolation-test.sh opencode' "$work/opencode_unmarked/out.log"
check "nothing runs without the isolation pass" bash -c "! grep -qE '^(opencode run|go-argc)' '$work/opencode_unmarked/calls.log'"
rm -f "$fake_home/.config/gabo/reviewer-isolation/opencode"

echo "# base version minimum"
origin stamped O_STAMP=0.4.0
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario old_cli GABO_VERSION=0.3.0)" "an older binary than the base stamp ends in error"
check "the refusal names the upgrade command" grep -qF 'go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@v0.4.0' "$work/old_cli/out.log"
check "nothing runs with an older binary" bash -c "! grep -qE '^(claude|go-argc)' '$work/old_cli/calls.log'"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=skipped" "$(scenario new_cli GABO_VERSION=0.5.0)" "a newer binary reviews"

echo "# upgrade PR"
origin upgrade O_STAMP=0.4.0 O_PR=upgrade O_HEAD_STAMP=9.9.9 O_AUTO_MERGE=true O_CHECKS='go -C apps/cli test ./...
gabo check'
v="$(scenario upgrade GABO_VERSION=0.4.1)"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=clean merge=needs-human-merge" "$v" \
  "an upgrade PR is reviewed with the CLI it has (the head stamp is never the minimum) and never merged"
check "no merge call for an upgrade PR" no_merge upgrade
check "the CLI runs with GABO_UPGRADE_PR=1" grep -qE '^gabo-call check GABO_UPGRADE_PR=1 ' "$work/upgrade/calls.log"
check "the comment reports the upgrade PR" grep -qxF 'upgrade PR: only config validated, human merge required' "$work/upgrade/all_posted_7.md"
check "the output reports the upgrade PR" grep -qF 'upgrade PR: only config validated, human merge required' "$work/upgrade/out.log"
up="$(sed -n '/^### Upgrade PR/,/^### Review/p' "$work/upgrade/all_posted_7.md")"
expect_match '`apps/cli/main.go`' "$up" "the report lists changed paths outside the catalog and agent directories"
expect_match '`.gabo/VERSION`' "$up" "the report lists the stamp"
check "the report leaves out the catalog subtree" bash -c "! grep -q '^- \`.gabo/skills/' <<<\"\$1\"" _ "$up"
check "the report leaves out the declared agent directory" bash -c "! grep -q '^- \`.claude/agents/' <<<\"\$1\"" _ "$up"
expect_eq "VERDICT PR #7 head $HEAD_SHA verdict=error merge=none" "$(scenario upgrade_old GABO_VERSION=0.3.0)" \
  "an upgrade PR still needs the base minimum"

finish
