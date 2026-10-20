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
| `reviewer` (UID 2000) | `/data/reviewer` (0700) | Holds the **vulcambot** GitHub credentials (0600) and the reviewer env file (Claude OAuth token, settings). Reachable from `orca` only through one sudo rule that runs `reviewer.sh`. |
| `reviewer-ci` (UID 2001) | `/data/reviewer-ci` (0700) | Runs untrusted PR code (checks) and the review model, with a scrubbed environment and no GitHub credentials. |

Long-lived Claude Code sessions (named, in Orca terminals, user `orca`):

- **vulcam-coder** — coordinator. Computes eligible roadmap issues, launches up to 4 short coder sessions in parallel (one per issue, each in its own git worktree), and relays work to the reviewer.
- **vulcam-reviewer** — messenger. Receives "PR ready" messages, runs the reviewer script through sudo, and messages the verdict back.

They talk with Claude Code's **native cross-session messaging** (SendMessage/ListAgents), which only works between sessions of the same OS user — hence both run as `orca`.

## Flow of one issue

1. **vulcam-coder** picks an eligible issue: open, listed in the roadmap manifest, all `blockedBy` closed, not `needs-user`, not a decision issue, not claimed by another runner.
2. It claims the issue (label + marker comment), creates a worktree, and launches a **coder session**.
3. The coder session implements test-first, runs local checks, pushes, opens/updates the PR to `dev`, and ends as `HANDED`.
4. vulcam-coder messages **vulcam-reviewer**: `PR #<n> head <sha> ready for review`.
5. vulcam-reviewer runs `sudo -n -u reviewer .../reviewer.sh head <n> <sha>`. The script, as `reviewer`:
   - re-reads the PR from GitHub and skips it if the head moved;
   - checks out the head in its own worktree;
   - as `reviewer-ci`: runs the CI-equivalent checks (install, build, check-types, lint, affected tests, formatting);
   - runs the review engine (default **Claude Code**, read-only, `--setting-sources user --strict-mcp-config`, same review prompt as the old OpenCode GitHub Action; `opencode` and `gga` are selectable via `REVIEWER_ENGINE`);
   - posts, as **vulcambot**: commit statuses `reviewer/tests` and `reviewer/review`, and one comment whose first line is exactly `No blocking issues found.` or a blocking verdict with findings (failure logs in a collapsed `<details>`);
   - if clean and `REVIEWER_AUTO_MERGE=true`: approves and merges (`--merge --match-head-commit <sha>`).
6. vulcam-reviewer messages vulcam-coder: `PR #<n> head <sha> verdict=<clean|blocking|error> merge=<merged|needs-human-merge|skipped|...>`.
7. If blocking: vulcam-coder launches a fix session that reads vulcambot's findings on the PR, fixes, pushes, and goes back to step 4 (max 4 rounds, then `BLOCKED` + `needs-user`).
8. If merged: vulcam-coder recomputes eligibility (the merge may unblock other issues) and starts the next wave. With nothing eligible it sleeps until a message arrives.

## Identities and credentials

- **vulcambot**: GitHub machine account (email alias of the owner), collaborator with write on cappy-lab/vulcam. Classic token with `repo` scope (fine-grained tokens cannot reach org repos for non-member collaborators). Stored only for user `reviewer` via `gh auth login --with-token` from stdin.
- **Claude**: the owner's subscription. The coder sessions use the login of user `orca`; the reviewer uses a separate long-lived token from `claude setup-token` (`CLAUDE_CODE_OAUTH_TOKEN`), injected only into the review step, never into the checks that run PR code. Both share the subscription's usage limits.
- Secrets are always piped through stdin (`read -rsp ... T && printf '%s' "$T" | ...`), never passed as arguments, printed, or stored in Railway variables.

## Current state (2026-10-20)

- vulcam `dev` already contains the cloud-pilot plugin (`agents/cloud-pilot`) and the reviewer (`agents/pr-reviewer`, Claude engine default) — PRs #391, #392, #397, #398.
- Server: tools installed under `/data/home/.local` (Claude Code, OpenCode, Pi, gentle-ai, engram, codegraph); vulcam cloned at `/data/home/projects/vulcam` (`dev`); `reviewer`/`reviewer-ci` users, sudoers drop-in and reviewer Claude install created; **tokens not loaded yet**; an old continuous runner may still be idle in Orca terminal "cloud-pilot".
- `sudo` is not in the image: it is installed at runtime and must be reinstalled after every redeploy, together with the users (the image's `/etc` is rebuilt). The image comes from another project's repo (testeos-z/gbai-alejandria `infra/orca-server`), which must stay untouched.
- GitHub Actions on cappy-lab is blocked by billing (Free plan minutes exhausted); the reviewer replaces CI for merge decisions.

## What the gbai-team CLI must automate

1. `provision` — on the server, as root: install sudo, create/repair users, sudoers, reviewer install (idempotent; also run at every boot via the service start command).
2. `secrets set <github|claude>` — read a secret without echo and pipe it to the server (never in argv or env).
3. `sessions start` — start vulcam-coder and vulcam-reviewer in Orca terminals with their prompts/settings; `sessions stop`.
4. `status` — server health, sessions alive, claims, open PRs, last verdicts, pending `needs-user` issues.
5. `audit` — cross-check the Engram log and GitHub (merges only by the reviewer identity, verdict statuses on merged heads, no coder merges).
6. Pairing helpers for local Orca and a doctor command for the known gotchas below.

## Gotchas learned

- `railway ssh` lands as root with a bare PATH; run tools as `orca` via `setpriv` and unset a stray `GH_TOKEN` it injects.
- Railway service variables are visible to every process in the container — never put the reviewer's token there.
- `claude -p` loads a project's hooks and MCP servers unless started with `--setting-sources user --strict-mcp-config`; `--bare` ignores `CLAUDE_CODE_OAUTH_TOKEN`.
- Cross-session messaging is per OS user.
- GitHub Free private repos have 2,000 Actions minutes per month and a $0 default spending limit; failed check runs block client-side merge gates.
- Fine-grained PATs cannot access an org repo for an outside collaborator.
- The owner's local vulcam guard hook blocks commits on any branch named `main`/`dev`, even in other repos.
