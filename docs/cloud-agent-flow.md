# Cloud agent flow

gbai-team is the CLI and harness that provisions and operates autonomous coding agents in the cloud. The first target is the **vulcam** repository (cappy-lab/vulcam). This document describes the agreed flow; the CLI must automate it end to end.

## Goals

- Agents work unattended: they pick roadmap issues, implement them, open PRs, and get them reviewed and merged into the integration branch (`dev`), wave after wave.
- Humans only steer: they decide product questions, exclude issues (label `needs-user`), and merge anything sensitive.
- Separation of duties: the agent that writes code never approves or merges it.
- Everything is auditable on GitHub (statuses, review comments, merges) and in Engram (per-step log).

## Hard rules

1. **No polling loops.** Never wait for external state by repeating requests (sleep-and-check). React to events (messages between sessions). Only a one-shot reconcile at startup is allowed.
2. **The coder never merges, approves, or sets commit statuses.** Only the reviewer identity does.
3. **The reviewer's verdict is computed inside the reviewer boundary**, from the PR head it checks out. Messages are notifications only and can never decide a verdict.
4. **Secrets never live in service-wide environment variables** that the coding agent can read. The reviewer's GitHub token lives in a 0600 file owned by a separate OS user.
5. Merges go to `dev` only, never `main`, pinned to the reviewed head SHA, never for human-only paths (`.claude/`, `.github/`, `.opencode/`, `AGENTS.md`, `CLAUDE.md`, `docs/policies/`, `tools/hook-tests/`, `agents/`).
6. Clean up, never lose work: merged branches and finished worktrees (coder and reviewer) are removed by the wrk prune step or the review cleanup; unmerged branches, data and history are never deleted. In cloud mode this is automatic: reviewer worktrees are removed right after the verdict is posted, and coder branches and worktrees only once GitHub confirms the PR is merged; local mode keeps the human approval of the prune step. Never force-push.

## Components

One server (Railway service **"vulcam cloud"**, project "cloud agents", workspace LatamEartCh C.A.) running the Orca headless image, with a persistent volume at `/data`. The owner watches everything from local Orca (the server is paired as the Orca environment "vulcam cloud"; each agent runs in a visible terminal).

OS users on the server:

| User | Home | Purpose |
|---|---|---|
| `orca` | `/data/home` | Runs both Claude Code sessions and the coder's work. Cannot read the reviewer's token. |
| `reviewer` (UID 2000) | `/data/reviewer` (0700) | Holds the **vulcambot** GitHub credentials (0600) and the model credential files. Reachable from `orca` only through one sudo rule that runs the root-owned `/usr/local/libexec/gbai/review-run`. |
| `reviewer-ci` (UID 2001) | `/data/reviewer-ci` (0700) | Runs untrusted PR code (checks) and the review model, with a scrubbed environment and no GitHub credentials. Reachable from `reviewer` only through `/usr/local/libexec/gbai/review-ci`. |

Long-lived agent sessions (Orca terminals, user `orca`, started by `gbai sessions start`; runtime `claude` or `pi` per role from `gbai.yaml`, defaults coder=claude, reviewer=pi, assistant=claude):

- **gbai-coder** — coordinator. Computes eligible roadmap issues, launches up to 4 short coder sessions in parallel (one per issue, each in its own git worktree) following the `gbai-cycle` and `gbai-coder` skills.
- **gbai-reviewer** — reviewer-side session; the review itself runs in detached terminals opened by `gbai review request`.
- **gbai-assistant** — planning orchestrator (`gbai-assistant` skill) that keeps the six product docs current.

Sessions talk through **Orca durable mailboxes** (`run:<id>` / `dispatch:<id>`, see the `gbai-orca-inbox` skill), never through terminal handles. Messages are notifications only.

## Flow of one issue

1. **gbai-coder** picks an eligible issue: open, listed in the roadmap manifest, all `blockedBy` closed, not `needs-user`, not a decision issue, not claimed by another runner.
2. It claims the issue (label + marker comment), creates a worktree, and launches a **coder session**.
3. The coder session implements test-first, runs local checks, pushes, opens/updates the PR to `dev` with label `mode:cloud` or `mode:local` (from `GBAI_MODE`).
4. After **every push** the coder runs `gbai review request --pr <n> --sha <head> --notify <own mailbox>`. There is no webhook: this one command opens a detached Orca terminal and returns immediately.
5. That terminal runs `sudo -n -u reviewer /usr/local/libexec/gbai/review-run head <n> <sha>`, a root-owned wrapper that execs `gbai review run` with a clean environment. As `reviewer`, gbai:
   - re-reads the PR from GitHub and skips it if the head moved;
   - in its own clone, adds a per-review worktree and makes a **local test merge** of the pinned head into a fresh `origin/dev`; a conflict is a blocking verdict;
   - streams the merged tree (`git archive`) into a fresh `reviewer-ci` workspace through `sudo -n -u reviewer-ci /usr/local/libexec/gbai/review-ci`;
   - as `reviewer-ci`, runs the configured checks (`review.checks`, argv only, no shell), then the read-only review engine (`claude` or `pi`, `runtimes.reviewer`) with the model credential piped over stdin;
   - computes the verdict itself from that local evidence and posts, as **vulcambot**, the statuses `reviewer/tests` and `reviewer/review` and one comment whose first line is exactly `No blocking issues found.` or `Blocking issues found.`;
   - if clean, `mode:cloud`, `review.auto_merge` and no human-only path: approves and merges pinned to the SHA;
   - removes the worktree and the workspace, and prints one summary line.
6. The terminal pipes that line to `gbai review notify`, which sends it to the coder's mailbox: `PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>`.
7. If blocking: gbai-coder launches a fix session that reads vulcambot's findings on the PR, fixes, pushes, and goes back to step 4 (max 4 rounds, then `BLOCKED` + `needs-user`).
8. If merged: gbai-coder recomputes eligibility (the merge may unblock other issues) and starts the next wave. With nothing eligible it ends its turn until a message arrives.

## Identities and credentials

- **vulcambot**: GitHub machine account (email alias of the owner), collaborator with write on cappy-lab/vulcam. Classic token with `repo` scope (fine-grained tokens cannot reach org repos for non-member collaborators). Stored only for user `reviewer` via `gh auth login --with-token` from stdin.
- **Claude**: the owner's subscription. The coder sessions use the login of user `orca`; the reviewer uses a separate long-lived token from `claude setup-token` (`CLAUDE_CODE_OAUTH_TOKEN`), injected only into the review step, never into the checks that run PR code. Both share the subscription's usage limits.
- Secrets are always piped through stdin (`read -rsp ... T && printf '%s' "$T" | ...`), never passed as arguments, printed, or stored in Railway variables.

## Current state (2026-10-20)

- vulcam `dev` already contains the cloud-pilot plugin (`agents/cloud-pilot`) and the reviewer (`agents/pr-reviewer`, Claude engine default) — PRs #391, #392, #397, #398.
- Server: tools installed under `/data/home/.local` (Claude Code, OpenCode, Pi, gentle-ai, engram, codegraph); vulcam cloned at `/data/home/projects/vulcam` (`dev`); `reviewer`/`reviewer-ci` users, sudoers drop-in and reviewer Claude install created; **tokens not loaded yet**; an old continuous runner may still be idle in Orca terminal "cloud-pilot".
- `sudo` is not in the image: it is installed at runtime and must be reinstalled after every redeploy, together with the users (the image's `/etc` is rebuilt). The image comes from another project's repo (testeos-z/gbai-alejandria `infra/orca-server`), which must stay untouched.
- GitHub Actions on cappy-lab is blocked by billing (Free plan minutes exhausted); the reviewer replaces CI for merge decisions.

## What the gbai CLI automates

| Need | Command (CLI / MCP tool) |
|---|---|
| Prepare the server as root, idempotently: packages, users, config, binary, review wrappers, sudoers, skills for `orca` | `gbai provision [--dry-run]` / `provision_plan`, `provision_apply` |
| Store a secret without echo, piped over stdin only | `gbai secrets set <name>` (human terminal only; MCP `secret_instructions` returns the command) |
| Labels and the integration ruleset | `gbai repo setup [--dry-run]` / `repo_setup_plan`, `repo_setup_apply` |
| Start/stop the sessions | `gbai sessions start|stop [--role] [--mode]` / `sessions_start`, `sessions_stop` |
| Request a review after a push; reviewer side | `gbai review request` / `review_request`; `gbai review run`, `review ci`, `review notify` |
| Status: server, sessions, open PRs, verdicts, needs-user | `gbai status` / `status` |
| Audit merges (bot only, verdict status on merged heads, no coder merges) | `gbai audit` / `audit` |
| Known gotchas | `gbai doctor` / `doctor` |
| Local agent setup: skills and MCP registration | `gbai init [--runtime] [--scope] [--dry-run]` |
| Product docs for the planning agents | `gbai plan new|status` |
| MCP server for the local agent | `gbai mcp` |

The Engram cross-check of `audit` and pairing helpers for local Orca are not automated yet. See `docs/architecture.md` for the code layout.

## Gotchas learned

- `railway ssh` lands as root with a bare PATH; run tools as `orca` via `setpriv` and unset a stray `GH_TOKEN` it injects.
- Railway service variables are visible to every process in the container — never put the reviewer's token there.
- `claude -p` loads a project's hooks and MCP servers unless started with `--setting-sources user --strict-mcp-config`; `--bare` ignores `CLAUDE_CODE_OAUTH_TOKEN`.
- Cross-session messaging is per OS user.
- GitHub Free private repos have 2,000 Actions minutes per month and a $0 default spending limit; failed check runs block client-side merge gates.
- Fine-grained PATs cannot access an org repo for an outside collaborator.
- The owner's local vulcam guard hook blocks commits on any branch named `main`/`dev`, even in other repos.
