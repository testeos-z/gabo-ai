#!/usr/bin/env bash
# Demo of the end-to-end mini wave (T034) for the parts the e2e test cannot
# fake: real agents in Orca, a real tracker and memory, a real review.
#
# What this script does itself, all local and in a fresh scratch dir:
#   1. builds gabo at a release version,
#   2. creates a sample repository with gabo init, a gabo.yaml and every agent,
#   3. runs gabo check on it,
#   4. prints the steps the owner runs with real agents, and the checklist.
# It never pushes, never creates issues, PRs or branches on a remote and never
# writes to a real memory: the printed steps are for the owner, on a
# throwaway repository the owner owns.
#
# Usage: bash apps/cli/e2e/demo-mini-wave.sh <owner>/<throwaway-repo>
set -euo pipefail

repo="${1:-}"
if [[ ! "$repo" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
  echo "usage: $0 <owner>/<throwaway-repo>" >&2
  exit 2
fi
if [[ "$repo" == */gabo-ai || "$repo" == */gbai-team ]]; then
  echo "refusing $repo: run the demo against a throwaway repository, never gabo-ai itself" >&2
  exit 2
fi

cli="$(cd "$(dirname "$0")/.." && pwd -P)"
version="${GABO_DEMO_VERSION:-0.9.0}"
scratch="$(mktemp -d "${TMPDIR:-/tmp}/gabo-demo.XXXXXX")"
sample="$scratch/sample"
mkdir -p "$scratch/bin" "$sample"

go -C "$cli" build -ldflags "-X main.buildVersion=$version" -o "$scratch/bin/gabo" ./cmd/gabo
gabo="$scratch/bin/gabo"

git -C "$sample" init -q -b dev
# gabo init installs the user-level reviewer skill; keep it in the scratch
# home unless the owner wants the real one (GABO_DEMO_REAL_HOME=1).
if [[ "${GABO_DEMO_REAL_HOME:-}" != 1 ]]; then
  export XDG_DATA_HOME="$scratch/share"
fi
(cd "$sample" && "$gabo" init >/dev/null)
cat >"$sample/gabo.yaml" <<YAML
version: 1
mode: auto
repo:
  name: $repo
  integration_branch: dev
runtimes:
  coder: claude
  reviewer: claude
  assistant: claude
slots:
  tasks:
    adapter: .gabo/skills/gabo-contract-tasks/examples/tasks-gh.md
  repository_host:
    adapter: .gabo/skills/gabo-contract-repository-host/examples/host-github.md
  memory:
    adapter: .gabo/skills/gabo-contract-memory/examples/memory-engram.md
  methodology:
    adapter: .gabo/skills/gabo-contract-methodology/examples/methodology-tdd.md
  reviewer_placement:
    adapter: .gabo/skills/gabo-contract-reviewer-placement/examples/reviewer-local.md
reviewer:
  placement: local
  signing:
    marker: gbai-reviewer-bot
review:
  checks:
    - gabo check
  auto_merge: true
  human_only_paths: [.github/, AGENTS.md, CLAUDE.md, gabo.yaml, .claude/]
parallelism: 1
worktree:
  keep: []
YAML
mkdir -p "$sample/.gabo/runtimes/gabo-runtime-claude" "$sample/.claude/agents"
printf -- '---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n---\n' \
  >"$sample/.gabo/runtimes/gabo-runtime-claude/SKILL.md"
for id in $(sed -n 's/^  - id: //p; s/^- id: //p' "$sample/.gabo/skills/gabo-agent-templates/agents.yaml"); do
  printf -- '---\nname: %s\ngabo-version: %s\n---\n' "$id" "$version" >"$sample/.claude/agents/$id.md"
done
# The links gabo-runtime-claude makes and gabo check requires: each spec
# skill under .claude/skills, and the user-level reviewer link (kept in the
# scratch dir unless GABO_DEMO_REAL_HOME=1, where the real one must exist).
mkdir -p "$sample/.claude/skills"
for name in $(sed -n 's/^    skills: \[\(.*\)\]$/\1/p' "$sample/.gabo/skills/gabo-agent-templates/agents.yaml" | tr ',' '\n' | tr -d ' ' | sort -u); do
  ln -s "../../.gabo/skills/$name" "$sample/.claude/skills/$name"
done
check_env=()
if [[ "${GABO_DEMO_REAL_HOME:-}" != 1 ]]; then
  mkdir -p "$scratch/claude/skills"
  ln -s "$XDG_DATA_HOME/gabo/skills/gabo-reviewer" "$scratch/claude/skills/gabo-reviewer"
  check_env=("CLAUDE_CONFIG_DIR=$scratch/claude")
fi
echo "# sample for the gabo mini wave demo" >"$sample/README.md"
git -C "$sample" add -A
git -C "$sample" -c user.name=demo -c user.email=demo@example.com -c commit.gpgsign=false commit -qm "chore: sample repository"
(cd "$sample" && env "${check_env[@]}" XDG_CONFIG_HOME="$scratch/config" "$gabo" check)

cat <<STEPS

Sample repository ready: $sample
gabo binary:             $gabo
Scratch dir (delete it when done): $scratch

The agent placeholders above only satisfy gabo check. For a real run, ask
your agent to rebuild them with the runtime skill (gabo-setup rebuild step).

Steps with real agents (you run them; this script ran none of them):
  1. Create the throwaway repository $repo, push the sample's dev branch.
  2. Create two task issues there: T001 and T002, T002 blocked by T001.
  3. Open an Orca session on the sample, start the assistant and say:
     "run wave 1".
  4. When T001 is DONE and T002's coder has started, stop the session.
  5. Plant a worktree the wave did not create:
     git -C $sample worktree add ../stray -b stray origin/dev
  6. Open a new session and say: "resume wave 1".

Checklist:
  [ ] each PR head got exactly one gabo reviewer comment and one VERDICT
  [ ] both issues are closed with "DONE via PR #<n>"
  [ ] T001's coder was not launched again on resume
  [ ] gabo/ledger/wave-1 lists every worktree, branch and coder handle,
      all removed; no ledger worktree is left on disk
  [ ] the stray worktree is listed under Leftovers and still exists
  [ ] the wave report (last entry of gabo/trace/wave-1) has every section
      of gabo-wave-report, Interactions 0 in auto mode
  [ ] no question reached you; decisions are under gabo/decision/wave-1/
STEPS
