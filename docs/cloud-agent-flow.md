# Cloud agent flow

gbai-team is the CLI and harness that provisions and operates autonomous coding agents in the cloud. The first target is the **vulcam** repository (cappy-lab/vulcam). This document describes the agreed flow; the CLI must automate it end to end.

## Goals

- Agents work unattended: they pick roadmap issues, implement them, open PRs, and get them reviewed and merged into the integration branch (`dev`), wave after wave.
- Humans only steer: they decide product questions, exclude issues (label `needs-user`), and merge anything sensitive.
- Separation of duties: the agent that writes code never merges it; nobody approves (single-user mode posts a signed verdict comment instead).
- Everything is auditable on GitHub (statuses, review comments, merges) and in Engram (per-step log).

## Hard rules

1. **No polling loops.** Never wait for external state by repeating requests (sleep-and-check). React to events (messages between sessions). Only a one-shot reconcile at startup is allowed.
2. **The coder never merges, approves, or sets commit statuses.** Only `gbai review run` does, and it never approves (GitHub forbids self-approval): its verdict is the signed marker comment plus the `reviewer/tests` and `reviewer/review` statuses.
3. **The reviewer's verdict is computed by `gbai review run`**, from the PR head it merges in its own state clone. Messages are notifications only and can never decide a verdict.
4. **Secrets never live in service-wide environment variables** that the coding agent can read. Model credentials are written only by `gbai secrets set` (stdin, 0600); checks and the review engine run with GitHub credentials stripped from their environment.
5. Merges go to `dev` only, never `main`, pinned to the reviewed head SHA, never for human-only paths (`.claude/`, `.github/`, `.opencode/`, `AGENTS.md`, `CLAUDE.md`, `docs/policies/`, `tools/hook-tests/`, `agents/`).
6. Clean up, never lose work: merged branches and finished worktrees (coder and reviewer) are removed by the wrk prune step or the review cleanup; unmerged branches, data and history are never deleted. In cloud mode this is automatic: reviewer worktrees are removed right after the verdict is posted, and coder branches and worktrees only once GitHub confirms the PR is merged; local mode keeps the human approval of the prune step. Never force-push.

## Components

One server (Railway service **"vulcam cloud"**, project "cloud agents", workspace LatamEartCh C.A.) running the Orca headless image, with a persistent volume at `/data`. The owner watches everything from local Orca (the server is paired as the Orca environment "vulcam cloud"; each agent runs in a visible terminal).

OS user on the server (**single-user mode**, as in vulcam):

| User | Home | Purpose |
|---|---|---|
| `orca` | `/data/home` | Runs the Claude Code sessions, the coder's work **and the reviewer**, with the owner's own `gh` login. No sudo, no extra users. |

The reviewer keeps its own state in `$XDG_STATE_HOME/gbai/reviewer` (default `~/.local/state/gbai/reviewer`, or `review.state_dir`), mode 0700: `clone/` (its own clone, checked out on the integration branch), `worktrees/` (one per review) and `ci/` (check workspaces and a persistent HOME for the checks' caches). Model credentials live in `/data/home/secrets` (0700, files 0600).

**This is weaker than the former three-user model** (`reviewer` and `reviewer-ci` OS users behind sudo), and the owner accepted it as trust-based, like vulcam: the coder sessions run as the same OS user, so they can read the owner's `gh` login and the model credential files, and PR code run by the checks can write anywhere the owner can (including the owner's home and the reviewer state). What remains: checks and the engine get an environment built from scratch (only `PATH`, `HOME`, `USER`, `LOGNAME`, `GH_CONFIG_DIR` pointed at a fresh empty directory, plus the model credential for the engine), so no `GH_TOKEN`/`GITHUB_TOKEN` or gh login reaches them by default; the checks' and the engine's `PATH` drop entries inside the reviewer's `ci/` dir and relative entries; leftover processes are killed only when their cwd is inside `ci/` (a leftover that moved elsewhere survives, and the owner's own sessions are never touched). `gbai provision` removes the retired sudoers drop-in and wrappers; the old `reviewer`/`reviewer-ci` users and homes are left for the owner to delete (`userdel`) once their credentials are no longer needed.

Long-lived agent sessions (Orca terminals, user `orca`, started by `gbai sessions start`; runtime `claude` or `pi` per role from the gbai config, defaults coder=claude, reviewer=pi, assistant=claude):

- **gbai-coder** — coordinator. Computes eligible roadmap issues, launches up to 4 short coder sessions in parallel (one per issue, each in its own git worktree) following the `gbai-cycle` and `gbai-coder` skills.
- **gbai-reviewer** — reviewer-side session; the review itself runs in detached terminals opened by `gbai review request`.
- **gbai-assistant** — planning orchestrator (`gbai-assistant` skill) that keeps the six product docs current.

Sessions talk through **Orca durable mailboxes** (`run:<id>` / `dispatch:<id>`, see the `gbai-orca-inbox` skill), never through terminal handles. Messages are notifications only.

## Flow of one issue

1. **gbai-coder** picks an eligible issue: open, listed in the roadmap manifest, all `blockedBy` closed, not `needs-user`, not a decision issue, not claimed by another runner.
2. It claims the issue (label + marker comment), creates a worktree, and launches a **coder session**.
3. The coder session implements test-first, runs local checks, pushes, opens/updates the PR to `dev` with label `mode:cloud` or `mode:local` (from `GBAI_MODE`).
4. After **every push** the coder runs `gbai review request --pr <n> --sha <head> --notify <own mailbox>`. There is no webhook: this one command opens a detached Orca terminal and returns immediately.
5. That terminal runs `gbai --config /etc/gbai/gbai.yaml review run head <n> <sha>` as `orca` (no sudo; `gbai review request --keep-worktree` forwards `--keep-worktree`). gbai:
   - takes the review lock in its state dir and **refuses to run** (non-zero exit, nothing read or posted) unless its own clone is missing (a fresh state) or a clean checkout of the integration branch: on the branch, no tracked, untracked or ignored changes;
   - re-reads the PR from GitHub and skips it if the head moved;
   - resets its clean clone to the fetched `origin/dev` (a rewritten branch is followed, not refused), adds a per-review worktree and makes a **local test merge** of the pinned head into a fresh `origin/dev`; a conflict is a blocking verdict;
   - copies the merged tree (`git archive`) into a fresh workspace under its `ci/` dir;
   - runs the configured checks there (`review.checks`, argv only, no shell), then the read-only review engine (`claude` or `pi`, `runtimes.reviewer`) with the model credential piped over stdin and a fresh empty HOME, both with an environment built from scratch (no GitHub credentials, empty `GH_CONFIG_DIR`); leftover processes whose cwd is inside `ci/` are killed (TERM, one bounded wait, KILL) before each check and before the engine;
   - computes the verdict itself from that local evidence and posts, with the owner's `gh` login, the statuses `reviewer/tests` and `reviewer/review` and one comment whose first line is exactly `No blocking issues found.` or `Blocking issues found.`, ending with a visible footer and a hidden marker that signs the head;
   - if clean, `mode:cloud`, `review.auto_merge`, no human-only path and its newest marker comment signs the head: merges pinned to the SHA (it never approves);
   - removes the worktree and the workspace, and prints one summary line.
6. The terminal pipes that line to `gbai review notify`, which sends it to the coder's mailbox: `PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>`.
7. If blocking: gbai-coder launches a fix session that reads the reviewer's marker comment findings on the PR, fixes, pushes, and goes back to step 4 (max 4 rounds, then `BLOCKED` + `needs-user`).
8. If merged: gbai-coder recomputes eligibility (the merge may unblock other issues) and starts the next wave. With nothing eligible it ends its turn until a message arrives.

## Identities and credentials

- **GitHub**: the owner's own account, through `gh` logged in as `orca` (browser login, or `gbai secrets set reviewer-github` piped over stdin). The reviewer posts, sets statuses and merges with it; `bot.login` in `gbai.yaml` is the owner's login and defaults to the login `gh` reports (`gbai review run`, `audit` and `status` all use it; when set it must match `gh`). The former **vulcambot** machine account is no longer used.
- **Claude**: the owner's subscription. The coder sessions use the login of user `orca`; the reviewer uses a separate long-lived token from `claude setup-token` (`CLAUDE_CODE_OAUTH_TOKEN`, stored by `gbai secrets set` under `/data/home/secrets`), injected only into the review engine, never into the checks that run PR code. Both share the subscription's usage limits.
- Secrets are always piped through stdin (`read -rsp ... T && printf '%s' "$T" | ...`), never passed as arguments, printed, or stored in Railway variables.

## Current state (2026-10-20)

- vulcam `dev` already contains the cloud-pilot plugin (`agents/cloud-pilot`) and the reviewer (`agents/pr-reviewer`, Claude engine default) — PRs #391, #392, #397, #398.
- Server: tools installed under `/data/home/.local` (Claude Code, OpenCode, Pi, gentle-ai, engram, codegraph); vulcam cloned at `/data/home/projects/vulcam` (`dev`); the `reviewer`/`reviewer-ci` users and sudoers drop-in of the retired three-user model still exist until `gbai provision` (drop-in, wrappers) and the owner (`userdel`) remove them; **tokens not loaded yet**; an old continuous runner may still be idle in Orca terminal "cloud-pilot".
- Single-user mode needs neither `sudo` nor extra users, so a redeploy (which rebuilds the image's `/etc`) only needs `gbai provision` for the binary, config and skills. The image comes from another project's repo (testeos-z/gbai-alejandria `infra/orca-server`), which must stay untouched.
- GitHub Actions on cappy-lab is blocked by billing (Free plan minutes exhausted); the reviewer replaces CI for merge decisions.

## What the gbai CLI automates

| Need | Command (CLI / MCP tool) |
|---|---|
| Prepare the server as root, idempotently: packages, `orca` user check, config, binary, removal of the retired reviewer sudoers/wrappers, skills for `orca` | `gbai provision [--dry-run]` / `provision_plan`, `provision_apply` |
| Store a secret without echo, piped over stdin only | `gbai secrets set <name>` (human terminal only; MCP `secret_instructions` returns the command) |
| Labels and the integration ruleset | `gbai repo setup [--dry-run]` / `repo_setup_plan`, `repo_setup_apply` |
| Start/stop the sessions | `gbai sessions start|stop [--role] [--mode]` / `sessions_start`, `sessions_stop` |
| Request a review after a push; reviewer side | `gbai review request [--keep-worktree]` / `review_request`; `gbai review run`, `review ci`, `review notify` |
| Status: server, sessions, open PRs, verdicts, needs-user | `gbai status` / `status` (a failed owner login lookup only turns the verdicts unknown) |
| Close an approved coder worktree (under `<main checkout>-worktrees/`, not the main one or on `worktree.keep`, clean, pushed, no ignored secrets, worked on, no open PR, merged into the integration branch or by a merged PR holding its HEAD); never `--force`; deletes the merged local branch with `git branch -d` | `gbai worktree close <path>` (one line: `CLOSED <path> branch=<b> reason=<in-base\|merged-pr>` or `REFUSED <path> <reason>`) |
| Audit merges (owner login only, verdict status and signed marker comment on merged heads, no coder merges) | `gbai audit` / `audit` |
| Known gotchas | `gbai doctor` / `doctor` |
| Which config files are read (system `/etc/gbai/gbai.yaml` → user `$XDG_CONFIG_HOME/gbai/config.yaml`, else `~/.config/gbai/config.yaml` → project `gbai.yaml` at the git top-level; each overrides only the keys it sets; `--config <file>` reads exactly one file) and the resolved values, credential-looking values redacted | `gbai config` |
| Local agent setup: skills and MCP registration | `gbai init [--runtime] [--scope] [--dry-run]` |
| Product docs for the planning agents | `gbai plan new|status` |
| MCP server for the local agent | `gbai mcp` |

The Engram cross-check of `audit` and pairing helpers for local Orca are not automated yet. See `docs/architecture.md` for the code layout.

## Gotchas learned

- `railway ssh` lands as root with a bare PATH; run tools as `orca` via `setpriv` and unset a stray `GH_TOKEN` it injects.
- Railway service variables are visible to every process in the container — never put a GitHub or model token there.
- `claude -p` loads a project's hooks and MCP servers unless started with `--setting-sources user --strict-mcp-config`; `--bare` ignores `CLAUDE_CODE_OAUTH_TOKEN`.
- Cross-session messaging is per OS user.
- GitHub Free private repos have 2,000 Actions minutes per month and a $0 default spending limit; failed check runs block client-side merge gates.
- Fine-grained PATs cannot access an org repo for an outside collaborator.
- The owner's local vulcam guard hook blocks commits on any branch named `main`/`dev`, even in other repos.
