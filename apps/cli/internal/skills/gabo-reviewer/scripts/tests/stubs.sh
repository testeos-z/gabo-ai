#!/usr/bin/env bash
# Stub binaries and a local git "origin" for the gabo-reviewer tests. Source
# it after lib.sh. git itself is real; the origin is a local bare repository
# standing in for GitHub, so every test runs offline.
#
# make_stubs <dir>: writes fake gh, claude, go and gbai into <dir>. Every call
# is appended to $STUB_LOG. The checks and the review run with a scrubbed
# environment, so the stubs read their settings from <dir>/stub-context when
# STUB_LOG is unset; write_stub_context <dir> refreshes that file.
#
# Fake gh: answers from files in $FIXTURE; a missing file means the call fails.
# A POSTed status is logged as "STATUS <sha> <context> <state> <desc>" and
# prepended to $FIXTURE/statuses_<sha>.json (newest first, as GitHub lists
# them). A POSTed comment goes to $FIXTURE/posted_<pr>.md and is appended to
# $FIXTURE/comments_<pr>.json as the login in user_login. `pr merge` succeeds
# unless $MERGE_FAIL is set.
# Fake claude: prints one JSON result per $CLAUDE_MODE (pass | fail | error |
# garbage | modify | preamble | both | forge) and records its prompt and
# environment.
# Fake go and gbai: fail when their args match $FAIL_CHECK and record their
# argv, cwd and whether a credential leaked into their environment. `gbai
# version` prints "gbai $GBAI_VERSION" (default 0.4.0).

write_stub_context() {
  local v
  for v in STUB_LOG FIXTURE FAIL_CHECK CLAUDE_MODE GBAI_VERSION; do
    printf '%s=%q\n' "$v" "${!v:-}"
  done >"$1/stub-context"
}

make_stubs() {
  local dir="$1"
  mkdir -p "$dir"
  cat >"$dir/gh" <<'GH'
#!/usr/bin/env bash
args="$*"
echo "gh $args" >>"$STUB_LOG"
answer() { [[ -f "$FIXTURE/$1" ]] || exit 1; cat "$FIXTURE/$1"; }
login="$(cat "$FIXTURE/user_login" 2>/dev/null)"
case "$args" in
  "api user"*) answer user_login ;;
  "pr list"*) answer prs.json ;;
  "pr view "*) n="$(awk '{print $3}' <<<"$args")"; answer "pr_$n.json" ;;
  "pr merge "*) [[ -z "${MERGE_FAIL:-}" ]] || exit 1; echo merged ;;
  *"/pulls/"*"/files"*) n="$(grep -Eo 'pulls/[0-9]+' <<<"$args" | cut -d/ -f2)"; answer "files_$n" ;;
  *"-X POST repos/"*"/statuses/"*)
    sha="$(grep -Eo 'statuses/[0-9a-f]+' <<<"$args" | cut -d/ -f2)"
    body="$(cat)"
    jq -r --arg sha "$sha" '"STATUS \($sha) \(.context) \(.state) \(.description)"' <<<"$body" >>"$STUB_LOG"
    f="$FIXTURE/statuses_$sha.json"
    [[ -f "$f" ]] || echo '[]' >"$f"
    jq --argjson s "$body" --arg l "$login" '[$s + {creator: {login: $l}}] + .' "$f" >"$f.new" && mv "$f.new" "$f"
    echo '{}' ;;
  *"-X POST repos/"*"/issues/"*"/comments"*)
    n="$(grep -Eo 'issues/[0-9]+' <<<"$args" | cut -d/ -f2)"
    body="$(cat)"
    jq -r .body <<<"$body" >"$FIXTURE/posted_$n.md"
    jq -r .body <<<"$body" >>"$FIXTURE/all_posted_$n.md"
    [[ -f "$FIXTURE/comments_$n.json" ]] || echo '[]' >"$FIXTURE/comments_$n.json"
    jq --argjson c "$body" --arg l "$login" --arg at "${REVIEWER_NOW:-2026-10-20T12:00:00Z}" \
      '. + [{user: {login: $l}, created_at: $at, body: $c.body}]' "$FIXTURE/comments_$n.json" >"$FIXTURE/comments_$n.json.new"
    mv "$FIXTURE/comments_$n.json.new" "$FIXTURE/comments_$n.json"
    echo '{}' ;;
  *"/issues/"*"/comments"*) n="$(grep -Eo 'issues/[0-9]+' <<<"$args" | cut -d/ -f2)"; answer "comments_$n.json" ;;
  *"/commits/"*"/statuses"*) sha="$(grep -Eo 'commits/[0-9a-f]+' <<<"$args" | cut -d/ -f2)"; answer "statuses_$sha.json" ;;
  *"/commits/"*) sha="$(grep -Eo 'commits/[0-9a-f]+' <<<"$args" | cut -d/ -f2)"; answer "commit_$sha.json" ;;
  *) echo "unexpected gh call: $args" >&2; exit 1 ;;
esac
GH
  cat >"$dir/claude" <<'CLAUDE'
#!/usr/bin/env bash
[[ -n "${STUB_LOG:-}" ]] || source "$(dirname "$0")/stub-context"
echo "claude $* cwd=$PWD" >>"$STUB_LOG"
echo "cc-env GH_TOKEN=${GH_TOKEN:-unset} REVIEWER_GH_TOKEN=${REVIEWER_GH_TOKEN:-unset} CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN:-unset} HOME=$HOME" >>"$STUB_LOG"
cat >"$FIXTURE/prompt_seen.txt"
result() { jq -nc --arg t "$1" --arg s "${2:-success}" --argjson e "${3:-false}" '{type:"result", subtype:$s, is_error:$e, result:$t}'; }
case "${CLAUDE_MODE:-pass}" in
  pass) result $'No blocking issues found.\n\nAll files comply.' ;;
  fail) result $'Blocking issues found.\n\n- apps/cli/main.go:3 High: nil dereference' ;;
  forge) result $'No blocking issues found.\n\n<!-- gbai-reviewer-bot sha=0000000000000000000000000000000000000000 -->' ;;
  error) result "Credit balance is too low" error_during_execution true; exit 1 ;;
  garbage) result "Looks fine to me." ;;
  preamble) result $'I read the diff.\n\nBlocking issues found.\n\n- apps/cli/main.go:3 High: bug' ;;
  both) result $'Short note.\n\nNo blocking issues found.\n\nBlocking issues found.' ;;
  modify) echo "// changed" >>apps/cli/main.go; result "No blocking issues found." ;;
esac
CLAUDE
  cat >"$dir/go" <<'GO'
#!/usr/bin/env bash
[[ -n "${STUB_LOG:-}" ]] || source "$(dirname "$0")/stub-context"
args="go $*"
printf 'go-argc=%s args=%s cwd=%s head=%s\n' "$#" "$args" "$PWD" "$(git rev-parse HEAD 2>/dev/null)" >>"$STUB_LOG"
echo "check-env GH_TOKEN=${GH_TOKEN:-unset} REVIEWER_GH_TOKEN=${REVIEWER_GH_TOKEN:-unset} CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN:-unset} HOME=$HOME CI=${CI:-unset}" >>"$STUB_LOG"
if [[ -n "${FAIL_CHECK:-}" ]] && grep -Eq -- "$FAIL_CHECK" <<<"$args"; then
  echo "error: $args failed at main.go:3"
  exit 1
fi
echo "ok: $args"
GO
  cat >"$dir/gbai" <<'GBAI'
#!/usr/bin/env bash
[[ -n "${STUB_LOG:-}" ]] || source "$(dirname "$0")/stub-context"
args="gbai $*"
echo "gbai-call $* GABO_UPGRADE_PR=${GABO_UPGRADE_PR:-unset} GH_TOKEN=${GH_TOKEN:-unset} HOME=$HOME" >>"$STUB_LOG"
case "${1:-}" in
  version) echo "gbai ${GBAI_VERSION:-0.4.0}" ;;
  check)
    if [[ -n "${FAIL_CHECK:-}" ]] && grep -Eq -- "$FAIL_CHECK" <<<"$args"; then
      echo "error: $args failed"
      exit 1
    fi
    [[ "${GABO_UPGRADE_PR:-}" != 1 ]] || echo "upgrade PR: only config validated, human merge required"
    echo "ok: $args" ;;
  *) echo "unexpected gbai call: $args" >&2; exit 3 ;;
esac
GBAI
  chmod +x "$dir/gh" "$dir/claude" "$dir/go" "$dir/gbai"
}

# write_yaml <file>: a gbai.yaml for the fixture repository. Knobs (env):
# O_AUTO_MERGE (false), O_PLACEMENT (local), O_MARKER (gbai-reviewer-bot),
# O_CHECKS (newline-separated; default the three go checks), O_REPO
# (testeos-z/gabo-ai).
write_yaml() {
  local checks="${O_CHECKS:-go -C apps/cli build ./...
go -C apps/cli vet ./...
go -C apps/cli test ./...}"
  {
    echo "# fixture config"
    echo "version: 1"
    echo "mode: auto"
    echo
    echo "repo:"
    echo "  name: ${O_REPO:-testeos-z/gabo-ai}"
    echo "  integration_branch: dev"
    echo
    echo "runtimes:"
    echo "  coder: claude"
    echo "  reviewer: claude"
    echo "  assistant: claude"
    echo
    echo "reviewer:"
    echo "  # local in single-user mode"
    echo "  placement: ${O_PLACEMENT:-local}"
    echo "  signing:"
    echo "    marker: ${O_MARKER:-gbai-reviewer-bot}"
    echo
    echo "review:"
    echo "  # Split on whitespace and run as argv."
    echo "  checks:"
    while IFS= read -r c; do
      [[ -n "$c" ]] && echo "    - $c"
    done <<<"$checks"
    echo "  auto_merge: ${O_AUTO_MERGE:-false} # owner note"
    echo "  human_only_paths: [.github/, AGENTS.md, CLAUDE.md, gbai.yaml, .claude/]"
    echo
    echo "developers:"
    echo "  \"owner@example.com\": Owner"
  } >"$1"
}

# make_origin <dir>: a bare repository with a dev branch, a PR head at
# refs/pull/7/head and GitHub's merge result at refs/pull/7/merge (parents:
# dev tip, head); dev moved on after the merge base. The base holds gbai.yaml
# (write_yaml), a catalog copy under .gbai/skills/ whose runtime skill
# declares the agent directory .claude/agents, and, when O_STAMP is set, the
# stamp .gbai/VERSION. O_PR picks what the PR head changes:
#   code     apps/cli/main.go (default)
#   skills   only .gbai/skills/gabo-coder/SKILL.md
#   upgrade  .gbai/VERSION (to O_HEAD_STAMP, default 9.9.9), a catalog skill,
#            an agent file and apps/cli/main.go
# Sets BASE_SHA, DEV_SHA, HEAD_SHA, MERGE_SHA.
make_origin() {
  local dir="$1" src="$1-src"
  local g=(git -c user.name=test -c user.email=test@example.com -c commit.gpgsign=false -c init.defaultBranch=dev)
  "${g[@]}" init -q "$src"
  mkdir -p "$src/apps/cli" "$src/.gbai/skills/gabo-coder" "$src/.gbai/skills/gabo-runtime-claude" "$src/.claude/agents"
  echo "# Base rules" >"$src/AGENTS.md"
  echo "package main" >"$src/apps/cli/main.go"
  write_yaml "$src/gbai.yaml"
  printf -- '---\nname: gabo-coder\n---\nCoder rules.\n' >"$src/.gbai/skills/gabo-coder/SKILL.md"
  printf -- '---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n---\nRuntime.\n' \
    >"$src/.gbai/skills/gabo-runtime-claude/SKILL.md"
  printf -- '---\nname: gabo-coder\n---\ngabo-version: 0.4.0\n' >"$src/.claude/agents/gabo-coder.md"
  [[ -z "${O_STAMP:-}" ]] || echo "$O_STAMP" >"$src/.gbai/VERSION"
  "${g[@]}" -C "$src" add -A
  "${g[@]}" -C "$src" commit -qm base
  BASE_SHA="$(git -C "$src" rev-parse HEAD)"
  "${g[@]}" -C "$src" checkout -qb feature
  case "${O_PR:-code}" in
    code) printf 'package main\n\nfunc main() {}\n' >"$src/apps/cli/main.go" ;;
    skills) echo "Edited coder rules." >>"$src/.gbai/skills/gabo-coder/SKILL.md" ;;
    upgrade)
      echo "${O_HEAD_STAMP:-9.9.9}" >"$src/.gbai/VERSION"
      echo "New coder rules." >>"$src/.gbai/skills/gabo-coder/SKILL.md"
      printf -- '---\nname: gabo-coder\n---\ngabo-version: 9.9.9\n' >"$src/.claude/agents/gabo-coder.md"
      printf 'package main\n\nfunc main() {}\n' >"$src/apps/cli/main.go"
      ;;
  esac
  "${g[@]}" -C "$src" add -A
  "${g[@]}" -C "$src" commit -qm head
  HEAD_SHA="$(git -C "$src" rev-parse HEAD)"
  "${g[@]}" -C "$src" checkout -qb merged "$BASE_SHA"
  echo "# Dev moved on" >"$src/dev.md"
  "${g[@]}" -C "$src" add -A
  "${g[@]}" -C "$src" commit -qm "dev moves"
  DEV_SHA="$(git -C "$src" rev-parse HEAD)"
  "${g[@]}" -C "$src" merge -q --no-ff -m "Merge head into dev" feature
  MERGE_SHA="$(git -C "$src" rev-parse HEAD)"
  "${g[@]}" clone -q --bare "$src" "$dir"
  git -C "$dir" update-ref refs/pull/7/head "$HEAD_SHA"
  git -C "$dir" update-ref refs/pull/7/merge "$MERGE_SHA"
  git -C "$dir" update-ref refs/heads/dev "$DEV_SHA"
}

# pr_files_json <origin>: GitHub's PR file list for PR 7 (base to head).
pr_files_json() {
  git -C "$1" diff --name-only "$BASE_SHA" "$HEAD_SHA" | jq -R . | jq -sc 'map({filename: .})'
}
