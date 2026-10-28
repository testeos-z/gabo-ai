#!/usr/bin/env bash
# Shared settings and helpers for the gabo-reviewer scripts. Source it; do not
# run it. Follows the vulcam pr-reviewer pattern in single-user mode (TRD D20).
# External binaries are injectable so tests run offline with stubs:
#   REVIEWER_GH (default gh), REVIEWER_CLAUDE (default claude),
#   REVIEWER_GABO_BIN (default gabo).
#
# Settings, highest first: the environment, the private env file, the
# skill's config.env, gabo.yaml on the integration branch (read from the
# reviewer's own clean checkout of it, never from a PR head), built-in
# defaults. reviewer.sh exports the resolved values to its child scripts.

reviewer_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

# load_kv_file <file>: reads KEY=value lines (never executed as shell) and
# exports each REVIEWER_*/token key that is not already set in the
# environment, so the environment always wins.
load_kv_file() {
  local line key value
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" =~ ^[[:space:]]*(#|$) ]] && continue
    [[ "$line" =~ ^([A-Z][A-Z0-9_]*)=(.*)$ ]] || continue
    key="${BASH_REMATCH[1]}"
    value="${BASH_REMATCH[2]}"
    [[ "$key" =~ ^(REVIEWER_[A-Z0-9_]+|CLAUDE_CODE_OAUTH_TOKEN)$ ]] || continue
    [[ -n "${!key+set}" ]] && continue
    printf -v "$key" '%s' "$value"
    export "${key?}"
  done <"$1"
}

# yaml_get <file> <dotted.key>: prints a scalar, or one line per list item
# (block "- x" or flow "[x, y]"), from the small YAML subset gabo.yaml uses.
# Comments and surrounding quotes are dropped; a missing key prints nothing.
read -r -d '' _yaml_awk <<'AWK'
function trim(s) { gsub(/^[ \t]+|[ \t]+$/, "", s); return s }
function unq(s) {
  if (s ~ /^".*"$/ || s ~ /^\047.*\047$/) s = substr(s, 2, length(s) - 2)
  return s
}
function nocomment(s,   q, i) {
  q = substr(s, 1, 1)
  if (q == "\"" || q == "\047") {
    i = index(substr(s, 2), q)
    if (i) return substr(s, 1, i + 1)
    return s
  }
  sub(/[ \t]+#.*$/, "", s)
  return s
}
/^[ \t]*(#|$)/ { next }
{
  match($0, /^ */); ind = RLENGTH; line = substr($0, ind + 1)
  if (inlist) {
    if (line ~ /^-([ \t]|$)/ && ind >= listind) {
      v = unq(trim(nocomment(trim(substr(line, 2)))))
      if (v != "") print v
      next
    }
    inlist = 0
  }
  if (line ~ /^-/) next
  if (!match(line, /^("[^"]*"|[^ \t:#"][^:#]*):([ \t]|$)/)) next
  k = substr(line, 1, RLENGTH); sub(/:[ \t]*$/, "", k); k = unq(trim(k))
  rest = trim(nocomment(trim(substr(line, RLENGTH + 1))))
  while (n > 0 && inds[n] >= ind) n--
  n++; keys[n] = k; inds[n] = ind
  p = keys[1]; for (i = 2; i <= n; i++) p = p "." keys[i]
  if (p != want) next
  if (rest == "") { inlist = 1; listind = ind; next }
  if (rest ~ /^\[.*\]$/) {
    m = split(substr(rest, 2, length(rest) - 2), a, ",")
    for (i = 1; i <= m; i++) { v = unq(trim(a[i])); if (v != "") print v }
    next
  }
  print unq(rest)
}
AWK
yaml_get() {
  [[ -r "$1" ]] || return 0
  awk -v want="$2" "$_yaml_awk" "$1"
}

# frontmatter_get <file> <dotted.key>: yaml_get on the "---" frontmatter.
frontmatter_get() {
  [[ -r "$1" ]] || return 0
  yaml_get <(awk 'NR == 1 && $0 != "---" { exit } NR > 1 && $0 == "---" { exit } NR > 1 { print }' "$1") "$2"
}

# caller_yaml <key>: a key of gabo.yaml at the top of the caller's checkout.
# Used only to find the repository and its integration branch; every other
# repository setting comes from the reviewer's own checkout of that branch.
caller_yaml() {
  local top
  top="$(git rev-parse --show-toplevel 2>/dev/null)" || return 0
  yaml_get "$top/gabo.yaml" "$1"
}

# Secrets (CLAUDE_CODE_OAUTH_TOKEN, REVIEWER_GH_TOKEN) live only in a private
# env file outside the skill; refused unless it is mode 600 and ours.
REVIEWER_ENV_FILE="${REVIEWER_ENV_FILE:-${XDG_CONFIG_HOME:-$HOME/.config}/gabo/reviewer.env}"
if [[ -f "$REVIEWER_ENV_FILE" ]]; then
  if [[ "$(stat -c '%a %u' "$REVIEWER_ENV_FILE")" != "600 $(id -u)" ]]; then
    echo "PREFLIGHT-FAILED $REVIEWER_ENV_FILE must be mode 600 and owned by $(id -un)" >&2
    exit 2
  fi
  load_kv_file "$REVIEWER_ENV_FILE"
fi
# Non-secret defaults shipped with the skill.
[[ -f "$reviewer_dir/config.env" ]] && load_kv_file "$reviewer_dir/config.env"

GH="${REVIEWER_GH:-gh}"
CLAUDE="${REVIEWER_CLAUDE:-claude}"
GABO_BIN="${REVIEWER_GABO_BIN:-gabo}"
GABO_INSTALL="${REVIEWER_GABO_INSTALL:-github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo}"
REPO="${REVIEWER_REPO:-$(caller_yaml repo.name | head -n 1)}"
BASE_BRANCH="${REVIEWER_BASE_BRANCH:-$(caller_yaml repo.integration_branch | head -n 1)}"
BASE_BRANCH="${BASE_BRANCH:-dev}"
MODE="${REVIEWER_MODE:-single-user}"
ENGINE="${REVIEWER_ENGINE:-claude}"
BOT_LOGIN="${REVIEWER_BOT_LOGIN:-}"
WORKDIR="${GABO_REVIEWER_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/gabo/reviewer}"
STATE="$WORKDIR/state"
BASE_DIR="$WORKDIR/base"
MAX_ATTEMPTS="${REVIEWER_MAX_ATTEMPTS:-2}"
CHECK_TIMEOUT="${REVIEWER_CHECK_TIMEOUT:-1800}"
REVIEW_TIMEOUT="${REVIEWER_REVIEW_TIMEOUT:-900}"
HEAD_TIMEOUT="${REVIEWER_HEAD_TIMEOUT:-7200}"
CLAUDE_MODEL="${REVIEWER_CLAUDE_MODEL:-}"
CLAUDE_MAX_TURNS="${REVIEWER_CLAUDE_MAX_TURNS:-40}"
DIFF_MAX_BYTES="${REVIEWER_DIFF_MAX_BYTES:-200000}"
MERGE_TRIES="${REVIEWER_MERGE_TRIES:-5}"
MERGE_DELAY="${REVIEWER_MERGE_DELAY:-3}"
REMOTE_URL="${REVIEWER_REMOTE_URL:-https://github.com/$REPO.git}"
# The trusted install locations: the engine-neutral user-level copy (TRD 9.4)
# and the runtime's skills dir, which links or holds it.
TRUSTED_DIRS=("${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts"
  "$HOME/.claude/skills/gabo-reviewer/scripts")
# Paths a human always merges, whatever review.human_only_paths says: the
# catalog version stamp and the catalog copy (TRD 9.4, issue #79).
BUILTIN_STAMP=".gabo/VERSION"
BUILTIN_CATALOG=".gabo/skills/"
# The reviewer's own source (any skills/gabo-reviewer/ dir, like the embedded
# copy in gabo-ai) is human-only too: a PR may not weaken its own gate.
BUILTIN_REVIEWER="skills/gabo-reviewer/"
# The Claude Code token is kept in a shell variable, out of the environment:
# only the review step gets it, inside its scrubbed `env -i`.
CLAUDE_TOKEN="${CLAUDE_CODE_OAUTH_TOKEN:-}"
unset CLAUDE_CODE_OAUTH_TOKEN
# Prefer the reviewer-specific token so a generic GH_TOKEN of another account
# is never used by accident.
[[ -z "${REVIEWER_GH_TOKEN:-}" ]] || export GH_TOKEN="$REVIEWER_GH_TOKEN"

TESTS_CONTEXT="reviewer/tests"
REVIEW_CONTEXT="reviewer/review"
VERDICT_CLEAN="No blocking issues found."
VERDICT_BLOCKING="Blocking issues found."
FOOTER="🤖 gabo reviewer"
UPGRADE_REPORT="upgrade PR: only config validated, human merge required"

# apply_repo_settings: the repository-level settings from REVIEWER_* (set by
# the environment or by load_base_config).
apply_repo_settings() {
  CHECKS_SPEC="${REVIEWER_CHECKS:-}"
  AUTO_MERGE="${REVIEWER_AUTO_MERGE:-false}"
  HUMAN_ONLY_PATHS="${REVIEWER_HUMAN_ONLY_PATHS-.claude/ .github/ gabo.yaml AGENTS.md CLAUDE.md}"
  PLACEMENT="${REVIEWER_PLACEMENT:-local}"
  AGENT_DIRS="${REVIEWER_AGENT_DIRS:-}"
  # Names (never values) of the variables passed from the reviewer's own
  # environment into the checks (review.check_env).
  CHECK_ENV="${REVIEWER_CHECK_ENV:-}"
  MINIMUM="${REVIEWER_MINIMUM:-none}"
  # The reviewer posts with the owner's own login, which the owner and the
  # coders also use, so a verdict comment counts only with this marker on a
  # line of its own AND from BOT_LOGIN. Trust-based in single-user mode.
  MARKER="${REVIEWER_MARKER:-gbai-reviewer-bot}"
  MARKER_PREFIX="<!-- $MARKER"
  NEEDS_HUMAN_MARKER="<!-- $MARKER:needs-human-merge -->"
}
apply_repo_settings

# verdict_marker <sha>: the hidden marker that signs one reviewed head.
verdict_marker() { printf '%s sha=%s -->' "$MARKER_PREFIX" "$1"; }

# bot_footer <sha>: the block that ends every verdict comment; the visible
# footer is the last line.
bot_footer() { printf '\n%s\n%s\n' "$(verdict_marker "$1")" "$FOOTER"; }

# The jq definition of the head a comment signs: the sha of its last line
# that is exactly a verdict marker, or null.
_marker_sha_jq='def marker_sha: [.body | split("\n")[] | sub("\r$"; "")
  | select(startswith($p) and endswith(" -->")) | ltrimstr($p) | rtrimstr(" -->")
  | select(test("^[0-9a-f]{40}$"))] | last;'

# latest_bot_verdict <comments json> <head commit date>: from a REST list of
# issue comments, the newest one by BOT_LOGIN that carries a verdict marker
# on a line of its own and was created after the head commit. Prints
# "<clean|blocking|other> <sha from its marker>", or nothing.
latest_bot_verdict() {
  jq -r --arg l "$BOT_LOGIN" --arg d "$2" --arg clean "$VERDICT_CLEAN" --arg blocking "$VERDICT_BLOCKING" \
    --arg p "$MARKER_PREFIX sha=" "$_marker_sha_jq"'
    [.[] | select(.user.login == $l and .created_at > $d) | select(marker_sha != null)]
    | sort_by(.created_at) | last // empty
    | (.body | split("\n") | map(gsub("^\\s+|\\s+$"; "")) | map(select(. != "")) | first // "") as $first
    | "\(if $first == $clean then "clean" elif $first == $blocking then "blocking" else "other" end) \(marker_sha)"' <<<"$1"
}

# has_verdict_comment <comments json> <sha>: true when a comment by BOT_LOGIN
# already signs <sha> with the marker (one verdict per head, R26).
has_verdict_comment() {
  [[ "$(jq -r --arg l "$BOT_LOGIN" --arg s "$2" --arg p "$MARKER_PREFIX sha=" "$_marker_sha_jq"'
    [.[] | select(.user.login == $l) | select(marker_sha == $s)] | length' <<<"$1" 2>/dev/null)" =~ ^[1-9] ]]
}

now_utc() { echo "${REVIEWER_NOW:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"; }
is_number() { [[ "$1" =~ ^[1-9][0-9]*$ ]]; }

# log_event <event> <detail>: one line to stdout and to the state log.
log_event() {
  mkdir -p "$STATE"
  local line
  line="$(now_utc) $1 $(tr '\r\n\t' '   ' <<<"$2" | sed -E 's/[[:space:]]+/ /g; s/ $//')"
  echo "$line"
  echo "$line" >>"$STATE/reviewer.log"
}

# parse_checks: one check per line on stdout, from CHECKS_SPEC (';'-separated).
parse_checks() {
  local item
  IFS=';' read -ra _items <<<"$CHECKS_SPEC"
  for item in "${_items[@]}"; do
    item="$(sed -E 's/^[[:space:]]+|[[:space:]]+$//g' <<<"$item")"
    [[ -n "$item" ]] && printf '%s\n' "$item"
  done
}

# is_gabo_check <argv0> <argv1>: true for a "gabo check" entry of
# review.checks, which runs with GABO_BIN.
is_gabo_check() { [[ "$1" == gabo && "$2" == check ]]; }

# is_human_only <path>: true when a human must merge a change to <path>.
is_human_only() {
  local p="$1" entry
  [[ "$p" == "$BUILTIN_STAMP" || "$p" == "$BUILTIN_CATALOG"* ]] && return 0
  [[ "$p" == "$BUILTIN_REVIEWER"* || "$p" == */"$BUILTIN_REVIEWER"* ]] && return 0
  for entry in $HUMAN_ONLY_PATHS; do
    if [[ "$entry" == */ ]]; then
      [[ "$p" == "$entry"* ]] && return 0
    else
      [[ "$p" == "$entry" || "$p" == */"$entry" ]] && return 0
    fi
  done
  return 1
}

# version_at_least <have> <minimum>: true when the binary version <have>
# (x.y.z, optional leading v; a pre-release is older than its release) is at
# least <minimum> (x.y.z). Anything else, like "dev", is not.
version_at_least() {
  local min="$2" core pre
  [[ "$min" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || return 1
  [[ "$1" =~ ^v?([0-9]+\.[0-9]+\.[0-9]+)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]] || return 1
  core="${BASH_REMATCH[1]}"
  pre="${BASH_REMATCH[2]}"
  if [[ "$core" == "$min" ]]; then
    [[ -z "$pre" ]]
    return
  fi
  [[ "$(printf '%s\n%s\n' "$core" "$min" | sort -V | tail -n 1)" == "$core" ]]
}

# repo_copy_problem: prints why these scripts are a copy inside the
# repository under review (a PR could have written them), or nothing.
repo_copy_problem() {
  local top caller_top
  top="$(git -C "$reviewer_dir" rev-parse --show-toplevel 2>/dev/null)" || return 0
  caller_top="$(git rev-parse --show-toplevel 2>/dev/null)"
  if [[ -f "$top/gabo.yaml" || "$top" == "$caller_top" ]] ||
    git -C "$top" remote -v 2>/dev/null | grep -qiF "/$REPO"; then
    echo "the reviewer scripts at $reviewer_dir are inside the repository under review ($top); run the user-level copy"
  fi
}

# trust_problem: prints why the scripts may not be trusted, or nothing. A PR
# cannot change files under the user-level skill dirs, so the code that
# reviews stays out of reach of the code under review, as long as it runs from
# there and nobody else can write it.
trust_problem() {
  local d real ok=false f
  for d in "${TRUSTED_DIRS[@]}"; do
    real="$(realpath -e -- "$d" 2>/dev/null)" || continue
    [[ "$reviewer_dir" == "$real" ]] && ok=true
  done
  [[ "$ok" == true ]] ||
    { echo "the reviewer scripts run from $reviewer_dir, not from the user-level skill ${TRUSTED_DIRS[0]}"; return; }
  for f in "$reviewer_dir/.." "$reviewer_dir" "$reviewer_dir"/*.sh "$reviewer_dir/config.env"; do
    [[ -e "$f" ]] || continue
    [[ "$(stat -c '%u' "$f")" == "$(id -u)" ]] || { echo "$f is not owned by $(id -un)"; return; }
    [[ -z "$(find "$f" -maxdepth 0 -perm /022 2>/dev/null)" ]] || { echo "$f is writable by group or others"; return; }
  done
}

# validate_static: the settings that do not depend on the repository.
validate_static() {
  local n
  [[ "$REPO" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] ||
    { echo "no repository: set REVIEWER_REPO or run from a checkout whose gabo.yaml sets repo.name (got '${REPO}')"; return; }
  [[ "$BASE_BRANCH" =~ ^[A-Za-z0-9._/-]+$ ]] || { echo "REVIEWER_BASE_BRANCH has unexpected characters"; return; }
  [[ "$BASE_BRANCH" != main && "$BASE_BRANCH" != master ]] ||
    { echo "REVIEWER_BASE_BRANCH must be the integration branch, never $BASE_BRANCH"; return; }
  [[ "$MODE" != isolated ]] ||
    { echo "REVIEWER_MODE=isolated belongs to the reviewer_placement adapter; these scripts run single-user mode only"; return; }
  [[ "$MODE" == single-user ]] || { echo "REVIEWER_MODE must be single-user"; return; }
  [[ "$ENGINE" == claude ]] || { echo "REVIEWER_ENGINE must be claude"; return; }
  [[ -z "$BOT_LOGIN" || "$BOT_LOGIN" =~ ^[A-Za-z0-9-]+(\[bot\])?$ ]] || { echo "REVIEWER_BOT_LOGIN must be a GitHub login"; return; }
  [[ "$GABO_BIN" =~ ^[A-Za-z0-9._/-]+$ ]] || { echo "REVIEWER_GABO_BIN has unexpected characters"; return; }
  [[ "$GABO_INSTALL" =~ ^[A-Za-z0-9._/-]+$ ]] || { echo "REVIEWER_GABO_INSTALL has unexpected characters"; return; }
  for n in MAX_ATTEMPTS CHECK_TIMEOUT REVIEW_TIMEOUT HEAD_TIMEOUT CLAUDE_MAX_TURNS MERGE_TRIES DIFF_MAX_BYTES; do
    is_number "${!n}" || { echo "REVIEWER_$n must be a positive integer (got '${!n}')"; return; }
  done
  [[ "$MERGE_DELAY" =~ ^[0-9]+$ ]] || { echo "REVIEWER_MERGE_DELAY must be a whole number of seconds"; return; }
  [[ "$CLAUDE_MODEL" =~ ^[][A-Za-z0-9._:@-]*$ ]] || echo "REVIEWER_CLAUDE_MODEL has unexpected characters"
}

# check_env_reserved <name>: true for a name review.check_env may not list:
# the variables the reviewer sets for the checks itself and its credentials.
check_env_reserved() {
  case "$1" in
    PATH | HOME | CI | GOCACHE | GOMODCACHE | GOPATH | GABO_UPGRADE_PR | GH_* | GITHUB_* | CLAUDE_* | REVIEWER_*) return 0 ;;
  esac
  return 1
}

# validate_config: prints the first problem, or nothing.
validate_config() {
  local problem entry
  problem="$(validate_static)"
  [[ -z "$problem" ]] || { echo "$problem"; return; }
  [[ "$PLACEMENT" == local ]] ||
    { echo "reviewer.placement $PLACEMENT is planned for a later release; only local runs here"; return; }
  [[ "$MARKER" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]] || { echo "reviewer.signing.marker must be one word (got '$MARKER')"; return; }
  [[ "$AUTO_MERGE" =~ ^(true|false)$ ]] || { echo "review.auto_merge must be true or false"; return; }
  [[ -n "$(parse_checks)" ]] || { echo "review.checks is empty"; return; }
  for entry in $CHECK_ENV; do
    [[ "$entry" =~ ^[A-Z_][A-Z0-9_]*$ ]] ||
      { echo "review.check_env entry $entry must be an environment variable name"; return; }
    ! check_env_reserved "$entry" || { echo "review.check_env entry $entry is reserved for the reviewer"; return; }
  done
  for entry in $AGENT_DIRS; do
    [[ "$entry" =~ ^[A-Za-z0-9._-][A-Za-z0-9._/-]*/$ && ! "$entry" =~ (^|/)\.\.(/|$) ]] ||
      { echo "agent directory $entry must stay inside the repository"; return; }
  done
}

# sync_base: fetches the integration branch into the reviewer's own mirror and
# moves its own checkout ($BASE_DIR) to the tip. Prints the problem and fails
# when that checkout is not a clean checkout of the integration branch.
sync_base() {
  local mirror="$WORKDIR/repo.git" base_ref="refs/remotes/origin/$BASE_BRANCH"
  mkdir -p "$STATE/locks" || { echo "cannot create $STATE"; return 1; }
  (
    flock 7
    if [[ ! -d "$mirror" ]]; then
      git init -q --bare "$mirror" && git -C "$mirror" remote add origin "$REMOTE_URL" ||
        { echo "cannot create the reviewer's mirror $mirror"; exit 1; }
    fi
    git -C "$mirror" remote set-url origin "$REMOTE_URL" 2>/dev/null
    git -C "$mirror" fetch -q --no-tags origin "+refs/heads/$BASE_BRANCH:$base_ref" >/dev/null 2>&1 ||
      { echo "could not fetch $BASE_BRANCH from $REMOTE_URL"; exit 1; }
    tip="$(git -C "$mirror" rev-parse "$base_ref")"
    if [[ -e "$BASE_DIR" ]]; then
      [[ "$(realpath -e -- "$(git -C "$BASE_DIR" rev-parse --git-common-dir 2>/dev/null)" 2>/dev/null)" == "$(realpath -e -- "$mirror")" ]] ||
        { echo "$BASE_DIR is not the reviewer's checkout of $BASE_BRANCH"; exit 1; }
      dirty="$(git -C "$BASE_DIR" status --porcelain --untracked-files=all 2>&1)" ||
        { echo "cannot read the reviewer's checkout of $BASE_BRANCH at $BASE_DIR"; exit 1; }
      [[ -z "$dirty" ]] ||
        { echo "the reviewer's checkout of $BASE_BRANCH at $BASE_DIR is not clean: $(head -n 3 <<<"$dirty" | paste -sd, -)"; exit 1; }
      git -C "$BASE_DIR" checkout -q --detach "$tip" >/dev/null 2>&1 ||
        { echo "could not move the reviewer's checkout of $BASE_BRANCH to $tip"; exit 1; }
    else
      git -C "$mirror" worktree prune >/dev/null 2>&1
      git -C "$mirror" worktree add -q --detach "$BASE_DIR" "$tip" >/dev/null 2>&1 ||
        { echo "could not check out $BASE_BRANCH at $BASE_DIR"; exit 1; }
    fi
    [[ "$(git -C "$BASE_DIR" rev-parse HEAD 2>/dev/null)" == "$tip" ]] ||
      { echo "the reviewer's checkout at $BASE_DIR is not at the $BASE_BRANCH tip"; exit 1; }
  ) 7>"$STATE/locks/mirror.lock"
}

# declared_agent_dirs: the agent directories that the runtime skills of
# runtimes.{coder, reviewer, assistant} declare (metadata.agent_dir), from
# the catalog copy on the integration branch, one per line with a final "/".
declared_agent_dirs() {
  local y="$BASE_DIR/gabo.yaml" rt f d
  for rt in $({ yaml_get "$y" runtimes.coder; yaml_get "$y" runtimes.reviewer; yaml_get "$y" runtimes.assistant; } | sort -u); do
    for f in "$BASE_DIR"/.gabo/{skills,runtimes}/gabo-runtime-"$rt"/SKILL.md; do
      d="$(frontmatter_get "$f" metadata.agent_dir | head -n 1)"
      [[ -n "$d" ]] && printf '%s/\n' "${d%/}"
    done
  done | sort -u
}

# load_base_config: the repository settings from gabo.yaml in the reviewer's
# checkout of the integration branch, for every REVIEWER_* key the
# environment does not set; exported so the child scripts see the same.
load_base_config() {
  local y="$BASE_DIR/gabo.yaml" stamp
  if [[ -z "${REVIEWER_CHECKS+set}" ]]; then
    REVIEWER_CHECKS="$(yaml_get "$y" review.checks | paste -sd';' -)"
  fi
  [[ -n "${REVIEWER_AUTO_MERGE+set}" ]] || REVIEWER_AUTO_MERGE="$(yaml_get "$y" review.auto_merge | head -n 1)"
  if [[ -z "${REVIEWER_HUMAN_ONLY_PATHS+set}" ]] && grep -qE '^[[:space:]]+human_only_paths:' "$y" 2>/dev/null; then
    REVIEWER_HUMAN_ONLY_PATHS="$(yaml_get "$y" review.human_only_paths | paste -sd' ' -)"
  fi
  [[ -n "${REVIEWER_MARKER+set}" ]] || REVIEWER_MARKER="$(yaml_get "$y" reviewer.signing.marker | head -n 1)"
  [[ -n "${REVIEWER_PLACEMENT+set}" ]] || REVIEWER_PLACEMENT="$(yaml_get "$y" reviewer.placement | head -n 1)"
  [[ -n "${REVIEWER_AGENT_DIRS+set}" ]] || REVIEWER_AGENT_DIRS="$(declared_agent_dirs | paste -sd' ' -)"
  [[ -n "${REVIEWER_CHECK_ENV+set}" ]] || REVIEWER_CHECK_ENV="$(yaml_get "$y" review.check_env | paste -sd' ' -)"
  # The base stamp is a minimum; the PR head's stamp never counts.
  REVIEWER_MINIMUM=none
  if [[ -f "$BASE_DIR/$BUILTIN_STAMP" ]]; then
    stamp="$(cat "$BASE_DIR/$BUILTIN_STAMP")"
    REVIEWER_MINIMUM="${stamp:-invalid}"
  fi
  export REVIEWER_CHECKS REVIEWER_AUTO_MERGE REVIEWER_MARKER REVIEWER_PLACEMENT REVIEWER_AGENT_DIRS REVIEWER_MINIMUM \
    REVIEWER_CHECK_ENV
  [[ -z "${REVIEWER_HUMAN_ONLY_PATHS+set}" ]] || export REVIEWER_HUMAN_ONLY_PATHS
  [[ -n "$REVIEWER_AUTO_MERGE" ]] || unset REVIEWER_AUTO_MERGE
  [[ -n "$REVIEWER_MARKER" ]] || unset REVIEWER_MARKER
  [[ -n "$REVIEWER_PLACEMENT" ]] || unset REVIEWER_PLACEMENT
  apply_repo_settings
}

# gabo_version: the version the gabo binary reports ("gabo 1.2.3" -> 1.2.3).
gabo_version() { timeout 60 "$GABO_BIN" version 2>/dev/null | awk 'NR == 1 { print $NF }'; }

# minimum_problem: prints why the gabo binary does not meet the integration
# branch's stamp, with the upgrade command, or nothing. The scripts never
# update the binary themselves.
minimum_problem() {
  local name have
  [[ "$MINIMUM" == none ]] && return 0
  [[ "$MINIMUM" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] ||
    { echo "$BASE_BRANCH:$BUILTIN_STAMP is not a version (x.y.z)"; return; }
  name="$(basename "$GABO_BIN")"
  if ! command -v "$GABO_BIN" >/dev/null 2>&1; then
    echo "no $name binary (REVIEWER_GABO_BIN=$GABO_BIN) but $BASE_BRANCH:$BUILTIN_STAMP needs at least $MINIMUM; install it with: go install $GABO_INSTALL@v$MINIMUM"
    return
  fi
  have="$(gabo_version)"
  version_at_least "$have" "$MINIMUM" ||
    echo "$name ${have:-unknown} is older than the minimum $MINIMUM in $BASE_BRANCH:$BUILTIN_STAMP; upgrade it with: go install $GABO_INSTALL@v$MINIMUM"
}

# resolve_bot_login: sets BOT_LOGIN to the active gh login when it is empty.
# Fails when gh cannot authenticate.
resolve_bot_login() {
  local login
  login="$("$GH" api user --jq .login 2>/dev/null)" || return 1
  [[ -n "$login" ]] || return 1
  if [[ -z "$BOT_LOGIN" ]]; then
    BOT_LOGIN="$login"
  fi
  GH_LOGIN="$login"
}

# latest_review_state <sha>: the newest reviewer/review state set by the bot
# login (statuses from any other account are ignored), or empty.
latest_review_state() {
  "$GH" api --paginate "repos/$REPO/commits/$1/statuses" 2>/dev/null |
    jq -rs --arg c "$REVIEW_CONTEXT" --arg l "$BOT_LOGIN" \
      '[add // [] | .[] | select(.context == $c and .creator.login == $l)] | first | .state // ""'
}

# post_status <sha> <context> <state> <description> (cut to 140 chars).
post_status() {
  jq -n --arg s "$3" --arg c "$2" --arg d "${4:0:140}" '{state:$s, context:$c, description:$d}' |
    "$GH" api -X POST "repos/$REPO/statuses/$1" --input - >/dev/null
}

# run_bounded <seconds> <log> <cmd...>: runs the command in its own session
# (process group), output to <log>, stdin from RUN_STDIN or /dev/null; on
# timeout the group gets TERM then KILL. Returns the exit code, 124 on timeout.
run_bounded() {
  local limit="$1" log="$2" pid code=0 waited=0
  shift 2
  setsid "$@" >"$log" 2>&1 <"${RUN_STDIN:-/dev/null}" &
  pid=$!
  while kill -0 "$pid" 2>/dev/null; do
    if ((waited >= limit * 10)); then
      kill -TERM -- "-$pid" 2>/dev/null
      sleep 5
      kill -KILL -- "-$pid" 2>/dev/null
      wait "$pid" 2>/dev/null
      echo "gabo-reviewer: timed out after ${limit}s" >>"$log"
      return 124
    fi
    sleep 0.1
    waited=$((waited + 1))
  done
  wait "$pid" || code=$?
  kill -TERM -- "-$pid" 2>/dev/null || true
  return "$code"
}

# remove_run_worktree <path>: removes the one merge-result worktree a run
# created. Refuses anything that is not a direct child of $WORKDIR/worktrees
# named merge-pr-*, a symlink, or not registered in the reviewer's own mirror.
remove_run_worktree() {
  local p="${1:-}" base="$WORKDIR/worktrees" mirror_dir="$WORKDIR/repo.git" real_p line listed=false
  [[ -n "$p" ]] || return 0
  if [[ "$p" != "$base"/merge-pr-* || "${p#"$base"/}" == */* || -L "$p" ]]; then
    log_event "WARN" "refusing to remove $p: not a merge-pr-* worktree directly under $base" >/dev/null
    return 1
  fi
  if [[ -e "$p" ]]; then
    real_p="$(realpath -e -- "$p" 2>/dev/null)" || real_p=""
    while IFS= read -r line; do
      [[ "$line" == "worktree "* ]] || continue
      if [[ -n "$real_p" && "$(realpath -e -- "${line#worktree }" 2>/dev/null)" == "$real_p" ]]; then
        listed=true
        break
      fi
    done < <(git -C "$mirror_dir" worktree list --porcelain 2>/dev/null)
    if [[ "$listed" != true ]]; then
      log_event "WARN" "refusing to remove $p: not a worktree of the mirror $mirror_dir" >/dev/null
      return 1
    fi
    git -C "$mirror_dir" worktree remove --force "$p" >/dev/null 2>&1 ||
      { log_event "WARN" "could not remove $p (remove it by hand)" >/dev/null; return 1; }
  fi
  git -C "$mirror_dir" worktree prune >/dev/null 2>&1 || true
}

strip_ansi() { sed -E $'s/\e\\[[0-9;]*[A-Za-z]//g; s/\r$//'; }
fence_safe() { sed "s/\`\`\`/'''/g"; }

# tail_chars <max chars> <file>: the last lines of <file> that fit in <max>.
tail_chars() {
  local max="$1" file="$2" out
  out="$(tail -n 200 "$file" 2>/dev/null | strip_ansi | fence_safe)"
  if ((${#out} > max)); then
    out="[... truncated ...]
${out: -$max}"
  fi
  printf '%s\n' "$out"
}
