# gbai CLI MVP

Locator: `odd/tasks/gbai-cli-mvp.md` · Engram mirror: `odd/gbai-cli-mvp/tasks` (project gabo-ai)
Branch: `feat/gbai-cli-mvp` (from `main` c0070ae)

## Objective

Build the first working version of `gbai`, the Go CLI + MCP server that configures local and cloud environments so autonomous coding agents (Claude Code or Pi, with gentle-ai) can follow a plan to completion, as defined in `docs/cloud-agent-flow.md` and the owner decisions recorded in Engram (project gabo-ai).

## Problem / why

Today the cloud agent flow is wired by hand over `railway ssh` (users, sudoers, tokens, review script, sessions). It is error-prone, undocumented in code and blocked the reviewer (sudo rule points to a missing script). The owner wants one tool that sets everything up with the least configuration.

## Scope (authorized 2026-10-20)

- Go module in `apps/cli`, single binary `gbai`, Turbo only orchestrates (`build`, `test`, `lint`, `check-types` delegate to Go).
- Hexagonal architecture as vertical slices: each command is a slice with its core (use case + domain), ports (interfaces owned by the core) and adapters (driving: cobra command / MCP tool; driven: exec, railway ssh, gh, orca, filesystem). Composition root in `cmd/gbai`.
- Slices: version, secrets, provision, repo, review (request + run), sessions, status, audit, doctor, init, plan, mcp.
- Skills shipped with the CLI (embedded): setup guide, planning agents, coder, reviewer, orca-inbox protocol, gbai-cycle.
- Local-only build and tests with fakes. **Out of scope: any mutation of the real server or GitHub** (needs separate owner authorization).

## Constraints

- Hard rules in `docs/cloud-agent-flow.md` (no polling, coder never merges/approves/sets statuses, verdict computed inside reviewer boundary, secrets only via stdin and never via argv/env/MCP, merges only to `dev` pinned to head SHA, cleanup of merged branches and finished worktrees only).
- Runtimes: Claude Code and Pi only, selectable per role (coder, reviewer, assistant); defaults coder=claude, reviewer=pi.
- Reviewer flow: post-PR hook/command → detached Orca terminal as `orca` → `sudo -n -u reviewer <review script>` → own clone, per-review git worktree at a local test merge (head pinned SHA + fresh `origin/dev`), conflict ⇒ blocking; checks + model as `reviewer-ci` without GitHub creds; script computes verdict, then posts as bot; prints one summary line; pane sends verdict to the coder's durable Orca mailbox; reviewer worktree removed after verdict.
- Communication between agents via Orca inside live panes, both Orca-facing sessions as `orca`.
- Test-first for behavior changes (`go test`), English artifacts, Conventional Commits, no AI attribution.
- ~400 changed lines per task is an advisory heuristic only.

## Tasks

Route per task is recorded in brackets.

- [ ] T1 Scaffold: Go module, cobra root + `version`, Turbo wiring, AGENTS.md stack update, test harness. [delegated: 2+ non-trivial files]
- [ ] T2 Shared kernel ports + adapters: command runner (exec), remote shell over `railway ssh` with stdin, config (`gbai.yaml`: server ids, users, runtimes per role, repo). [delegated]
- [ ] T3 `secrets set <name>` slice: no-echo read, refuse argv/env values, pipe via remote stdin into a 0600 file owned by the target OS user. [delegated]
- [ ] T4 `provision` slice: idempotent remote plan (sudo, users orca/reviewer/reviewer-ci, sudoers drop-in, dirs, review script install, gentle-ai `--persona neutral`, skills per user), `--dry-run` prints the plan. [delegated]
- [ ] T5 `review` slice: `review request` (launch detached Orca terminal with the sudo command, record coder mailbox) and `review run` (reviewer side: test merge worktree, checks, engine claude|pi read-only, verdict parse, gh post as bot, cleanup, summary line). [delegated]
- [ ] T6 `repo setup` slice: labels `mode:cloud`/`mode:local`, `dev` ruleset (bot approval + checks, no bypass) via gh. [delegated]
- [ ] T7 `sessions start|stop`, `status`, `audit`, `doctor` slices. [delegated]
- [ ] T8 `mcp` driving adapter (stdio) exposing the slices as tools (never secrets) + `init` slice (install embedded skills, register MCP for Claude/Pi) + embedded skills content. [delegated]
- [ ] T9 `plan` slice: scaffold the six product docs (PRD, TRD, implementation plan, DB schema + data, UX/UI, app flow) and docs update. [delegated]

## Acceptance criteria

- `bun run build|test|lint|check-types` (Turbo) pass; `go test ./...` green in `apps/cli`.
- `gbai --help` lists all commands; each slice has use-case tests with hand-written fakes of its ports.
- No secret value can enter argv, env, logs or MCP tool inputs (tested).
- Review run logic computes the verdict from local evidence only, pins merge to head SHA, and treats merge conflicts as blocking (tested with fakes).

## Checks

`cd apps/cli && go test ./... && go vet ./...`; `bun run test` at repo root.

## Delivery

Strategy `ask-on-risk`; forecast > 400 lines ⇒ chain strategy `feature-branch-chain` (pilot-cycle default). Push/PR are owner decisions.

## Progress

- 2026-10-20: feature document created; branch `feat/gbai-cli-mvp`. RDD effective mode: on (global).

## Next step

T1 scaffold.
