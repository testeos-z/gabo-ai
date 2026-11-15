#!/usr/bin/env bash
# Tests for the cloud agent image scripts. They run on the host with stub
# `claude`, `engram` and `curl` commands; no Docker, network or credential.
# Usage: infra/cloud-agent/tests/run.sh
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
bin="$(cd "$here/../bin" && pwd)"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

failures=0
pass() { printf 'ok   %s\n' "$1"; }
fail() { printf 'FAIL %s\n' "$1"; failures=$((failures + 1)); }

# --- stubs -----------------------------------------------------------------

stubs="$scratch/stubs"
mkdir -p "$stubs"

# claude: keeps the registered MCP servers in $STUB_STATE/mcp.
cat >"$stubs/claude" <<'EOF'
#!/usr/bin/env bash
echo "claude $*" >>"$STUB_STATE/calls"
case "$1 $2" in
  "mcp get") grep -qx "$3" "$STUB_STATE/mcp" 2>/dev/null ;;
  "mcp add") echo engram >>"$STUB_STATE/mcp" ;;
  "mcp list") grep -qx engram "$STUB_STATE/mcp" 2>/dev/null && echo "engram: engram mcp --tools=agent - ✔ Connected"; true ;;
  *) exit 0 ;;
esac
EOF

# engram: version, cloud status and sync are driven by STUB_* variables.
cat >"$stubs/engram" <<'EOF'
#!/usr/bin/env bash
echo "engram $*" >>"$STUB_STATE/calls"
case "$1" in
  version) echo "engram ${STUB_ENGRAM_VERSION:-3.6.0}" ;;
  cloud) printf '%s\n' "${STUB_CLOUD_STATUS:-Cloud status: not configured (no effective server URL)}" ;;
  sync)
    case " $* " in
      *" --import "*) exit "${STUB_IMPORT_RC:-0}" ;;
      *) exit "${STUB_EXPORT_RC:-0}" ;;
    esac ;;
esac
EOF

# curl: copies $STUB_PAYLOAD to the -o target instead of downloading.
cat >"$stubs/curl" <<'EOF'
#!/usr/bin/env bash
out=""
while [ $# -gt 0 ]; do
  case "$1" in -o) out="$2"; shift 2 ;; *) shift ;; esac
done
cp "$STUB_PAYLOAD" "$out"
EOF
chmod +x "$stubs"/*

new_case() {
  STUB_STATE="$(mktemp -d "$scratch/case.XXXX")"
  export STUB_STATE
  : >"$STUB_STATE/calls"
  mkdir -p "$STUB_STATE/home"
  unset STUB_ENGRAM_VERSION STUB_CLOUD_STATUS STUB_EXPORT_RC STUB_IMPORT_RC STUB_PAYLOAD
}

run() { env HOME="$STUB_STATE/home" PATH="$stubs:$bin:/usr/bin:/bin" "$@"; }

# --- fetch-verified ---------------------------------------------------------

new_case
printf 'payload' >"$STUB_STATE/payload"
export STUB_PAYLOAD="$STUB_STATE/payload"
good="$(sha256sum "$STUB_STATE/payload" | cut -d' ' -f1)"
if run fetch-verified https://example.invalid/a "$good" "$STUB_STATE/a" >/dev/null 2>&1 &&
   [ "$(cat "$STUB_STATE/a")" = payload ]; then
  pass "fetch-verified keeps a download whose SHA-256 matches"
else
  fail "fetch-verified keeps a download whose SHA-256 matches"
fi

bad="0000000000000000000000000000000000000000000000000000000000000000"
if ! run fetch-verified https://example.invalid/b "$bad" "$STUB_STATE/b" >/dev/null 2>&1 &&
   [ ! -e "$STUB_STATE/b" ]; then
  pass "fetch-verified fails and removes a download whose SHA-256 differs"
else
  fail "fetch-verified fails and removes a download whose SHA-256 differs"
fi

if ! run fetch-verified https://example.invalid/c "" "$STUB_STATE/c" >/dev/null 2>&1 &&
   [ ! -e "$STUB_STATE/c" ]; then
  pass "fetch-verified refuses an empty checksum"
else
  fail "fetch-verified refuses an empty checksum"
fi

# --- gabo-engram-sync -------------------------------------------------------

new_case
if out="$(run gabo-engram-sync 2>&1)" && ! grep -q 'engram sync' "$STUB_STATE/calls" &&
   grep -q 'not configured' <<<"$out"; then
  pass "gabo-engram-sync skips when Engram Cloud is not configured"
else
  fail "gabo-engram-sync skips when Engram Cloud is not configured"
fi

new_case
export STUB_CLOUD_STATUS="Cloud status: configured (target=cloud)"
if run gabo-engram-sync >/dev/null 2>&1 &&
   [ "$(grep -c '^engram sync' "$STUB_STATE/calls")" = 2 ] &&
   grep -qx 'engram sync --cloud --project gabo-ai' "$STUB_STATE/calls" &&
   grep -qx 'engram sync --cloud --import --project gabo-ai' "$STUB_STATE/calls"; then
  pass "gabo-engram-sync exports then imports project gabo-ai when configured"
else
  fail "gabo-engram-sync exports then imports project gabo-ai when configured"
fi

new_case
export STUB_CLOUD_STATUS="Cloud status: configured (target=cloud)" STUB_EXPORT_RC=1
if ! run gabo-engram-sync >/dev/null 2>&1 && ! grep -q -- '--import' "$STUB_STATE/calls"; then
  pass "gabo-engram-sync fails without importing when the export fails"
else
  fail "gabo-engram-sync fails without importing when the export fails"
fi

# --- gabo-make-seed --------------------------------------------------------

gitc() { git -c user.name=t -c user.email=t@example.invalid "$@"; }

# "GitHub": main is the default branch, dev is one commit ahead of main.
upstream="$scratch/upstream.git"
work="$scratch/work"
git init -q -b main "$work"
gitc -C "$work" commit -q --allow-empty -m init
gitc -C "$work" checkout -q -b dev
gitc -C "$work" commit -q --allow-empty -m dev-work
git clone -q --bare "$work" "$upstream"
git -C "$upstream" symbolic-ref HEAD refs/heads/main

# A fresh clone of the default branch: dev is only a remote-tracking branch.
checkout="$scratch/checkout"
git clone -q "$upstream" "$checkout"
git -C "$checkout" config gabo.test.private-setting must-not-be-copied
gitc -C "$checkout" branch experiment
gitc -C "$checkout" tag v0-local

seed="$scratch/seed.git"
if run gabo-make-seed "$checkout/.git" "$seed" https://github.com/testeos-z/gabo-ai.git >/dev/null 2>&1 &&
   [ "$(git -C "$seed" symbolic-ref HEAD)" = refs/heads/dev ] &&
   [ "$(git -C "$seed" rev-parse dev)" = "$(git -C "$upstream" rev-parse dev)" ] &&
   [ "$(git -C "$seed" remote get-url origin)" = https://github.com/testeos-z/gabo-ai.git ] &&
   ! git -C "$seed" config --local --get gabo.test.private-setting >/dev/null; then
  pass "gabo-make-seed keeps dev from a default-branch clone and not the source git config"
else
  fail "gabo-make-seed keeps dev from a default-branch clone and not the source git config"
fi

if [ "$(git -C "$seed" for-each-ref --format='%(refname)' | sort | tr '\n' ' ')" = "refs/heads/dev refs/heads/main " ]; then
  pass "gabo-make-seed holds only dev and main, no other local branch or tag"
else
  fail "gabo-make-seed holds only dev and main, no other local branch or tag"
fi

# A seed without dev whose HEAD is main (the source only knew main).
nodev_src="$scratch/nodev"
git clone -q --single-branch --branch main "$upstream" "$nodev_src"
nodev_seed="$scratch/nodev-seed.git"
run gabo-make-seed "$nodev_src/.git" "$nodev_seed" https://github.com/testeos-z/gabo-ai.git >/dev/null 2>&1 || true
if ! git -C "$nodev_seed" rev-parse --verify --quiet refs/heads/dev >/dev/null &&
   git -C "$nodev_seed" rev-parse --verify --quiet refs/heads/main >/dev/null; then
  pass "gabo-make-seed builds a seed without dev when the source has none"
else
  fail "gabo-make-seed builds a seed without dev when the source has none"
fi

# --- gabo-agent-start -------------------------------------------------------

start_env() {
  run env GABO_AGENT_USER="$(id -un)" GABO_REPO_SEED="$seed" \
    GABO_REPO_URL=https://github.com/testeos-z/gabo-ai.git \
    GABO_ENGRAM_SYNC_ON_START=0 "$@"
}

new_case
if out="$(start_env gabo-agent-start echo started 2>&1)" && [ "$(tail -n1 <<<"$out")" = started ] &&
   grep -qx 'claude mcp add --transport stdio --scope user engram -- '"$stubs"'/engram mcp --tools=agent' "$STUB_STATE/calls"; then
  pass "gabo-agent-start registers engram as a user-scope MCP server and runs the command"
else
  fail "gabo-agent-start registers engram as a user-scope MCP server and runs the command"
fi

repo="$STUB_STATE/home/gabo-ai"
if [ "$(git -C "$repo" remote get-url origin 2>/dev/null)" = https://github.com/testeos-z/gabo-ai.git ] &&
   [ "$(git -C "$repo" branch --show-current)" = dev ]; then
  pass "gabo-agent-start checks out the repository on dev from the image seed"
else
  fail "gabo-agent-start checks out the repository on dev from the image seed"
fi

: >"$STUB_STATE/calls"
touch "$repo/local-work" 2>/dev/null || true
if start_env gabo-agent-start true >/dev/null 2>&1 &&
   ! grep -q 'mcp add' "$STUB_STATE/calls" && [ -e "$repo/local-work" ]; then
  pass "gabo-agent-start keeps an existing registration and checkout"
else
  fail "gabo-agent-start keeps an existing registration and checkout"
fi

new_case
if start_env GABO_REPO_SEED="$nodev_seed" GABO_REPO_URL="$upstream" gabo-agent-start true >/dev/null 2>&1 &&
   [ "$(git -C "$STUB_STATE/home/gabo-ai" branch --show-current)" = dev ] &&
   [ "$(git -C "$STUB_STATE/home/gabo-ai" rev-parse HEAD)" = "$(git -C "$upstream" rev-parse dev)" ]; then
  pass "gabo-agent-start fetches dev from origin when the seed lacks it"
else
  fail "gabo-agent-start fetches dev from origin when the seed lacks it"
fi

new_case
if out="$(start_env GABO_REPO_SEED="$nodev_seed" GABO_REPO_URL="$scratch/missing.git" gabo-agent-start true 2>&1)" &&
   grep -q 'WARNING: neither the image seed nor .* gave branch dev' <<<"$out"; then
  pass "gabo-agent-start warns when neither the seed nor origin has dev"
else
  fail "gabo-agent-start warns when neither the seed nor origin has dev"
fi

# A git that treats the seed as owned by another user (CVE-2024-32004 check).
otherowner="$scratch/otherowner"
mkdir -p "$otherowner"
{
  echo '#!/usr/bin/env bash'
  echo "case \" \$* \" in *\"$seed\"*) export GIT_TEST_ASSUME_DIFFERENT_OWNER=1 ;; esac"
  echo 'exec /usr/bin/git "$@"'
} >"$otherowner/git"
chmod +x "$otherowner/git"

new_case
if out="$(env HOME="$STUB_STATE/home" PATH="$otherowner:$stubs:$bin:/usr/bin:/bin" \
     GABO_AGENT_USER="$(id -un)" GABO_REPO_SEED="$seed" GABO_REPO_URL="$scratch/missing.git" \
     GABO_ENGRAM_SYNC_ON_START=0 gabo-agent-start echo started 2>&1)" &&
   [ "$(tail -n1 <<<"$out")" = started ] &&
   [ "$(git -C "$STUB_STATE/home/gabo-ai" branch --show-current)" = dev ]; then
  pass "gabo-agent-start checks out dev from a seed owned by another user"
else
  fail "gabo-agent-start checks out dev from a seed owned by another user"
fi

# --- gabo-agent-check -------------------------------------------------------

check_env() {
  run env GABO_ENGRAM_VERSION=3.6.0 GABO_REQUIRED_TOOLS="engram claude git" "$@"
}

new_case
start_env gabo-agent-start true >/dev/null 2>&1
if check_env gabo-agent-check >/dev/null 2>&1; then
  pass "gabo-agent-check passes on a started agent"
else
  fail "gabo-agent-check passes on a started agent"
fi

export STUB_ENGRAM_VERSION=3.5.0
if ! out="$(check_env gabo-agent-check 2>&1)" && grep -q 'FAIL engram version' <<<"$out"; then
  pass "gabo-agent-check fails when engram is not the pinned version"
else
  fail "gabo-agent-check fails when engram is not the pinned version"
fi

new_case
if ! out="$(check_env gabo-agent-check 2>&1)" && grep -q 'FAIL engram MCP server' <<<"$out" &&
   grep -q 'FAIL repository' <<<"$out"; then
  pass "gabo-agent-check fails before the agent was started"
else
  fail "gabo-agent-check fails before the agent was started"
fi

new_case
start_env gabo-agent-start true >/dev/null 2>&1
git -C "$STUB_STATE/home/gabo-ai" checkout -q -b feature
if ! out="$(check_env gabo-agent-check 2>&1)" && grep -q 'FAIL repository: .* is not on dev' <<<"$out"; then
  pass "gabo-agent-check fails when the checkout is not on dev"
else
  fail "gabo-agent-check fails when the checkout is not on dev"
fi

# The image is bun-only: the default tool check requires bun and not npm.
new_case
out="$(run env -u GABO_REQUIRED_TOOLS GABO_ENGRAM_VERSION=3.6.0 gabo-agent-check 2>&1 || true)"
if grep -q 'PATH: bun' <<<"$out" && ! grep -qw 'npm' <<<"$out"; then
  pass "gabo-agent-check requires bun and not npm by default"
else
  fail "gabo-agent-check requires bun and not npm by default"
fi

if [ "$failures" -gt 0 ]; then
  printf '%d test(s) failed\n' "$failures"
  exit 1
fi
echo "all tests passed"
