# gbai manual bootstrap (v0 framework for Claude Code, no compiler)

## Objective

Hand-write the gbai framework for Claude Code (agents, adapter skills, updated skills, config, reviewer skill) so gbai-team can be built with its own structure before the compiler exists.

## Problem and why

The compiler (`gbai generate`) is not built yet, but the owner wants to dogfood the framework now. These v0 files are what the compiler will later generate; plan tasks T009, T010, T024 and T040 formalize them.

## Scope

- Project config `gbai.yaml` and the owner's global config `~/.config/gbai/config.yaml` (DB_SCHEMA 3.1, 4.1; TRD 9.3).
- Agents in `.claude/agents/`: gbai-pm, gbai-dev-lead, gbai-cycle, gbai-coder, gbai-reviewer, gbai-plan-reviewer.
- Adapter skills in `.claude/skills/`: tasks (GitHub Issues via gh), repository_host (GitHub), memory (Engram), methodology (gentle-ai ODD + RDD).
- Update the embedded skills in `apps/cli/internal/skills/` (assistant, cycle, coder, pm, dev-lead, setup, reviewer notes) to the compiler-only model; drop cloud, Orca inbox and `gbai review` references.
- User-level `gbai-reviewer` skill in `~/.claude/skills/` with reviewer scripts adapted from vulcam `agents/pr-reviewer` (single-user mode, integration branch `dev`, Go checks).
- Integration branch `dev`.

Out of scope: notifications, direct sessions (Orca), agent_host (cloud), db-master and frontend agents, Go code changes.

## Constraints

- Owner decisions D18–D22 (docs/product/TRD.md). Auto mode never merges into `main`.
- Artifacts in English. No AI attribution in commits.

## Tasks

| id | task | route | status | commit |
|---|---|---|---|---|
| B1 | Config files: `gbai.yaml` and global config | delegated (2+ files, prep reading) | done | 48f4522, 41294ae (global file outside repo, uncommitted) |
| B2 | Adapter skills: tasks-github, repository-host-github, memory-engram, methodology-gentle-ai | delegated | done | 6f11894 |
| B3 | Update embedded skills to the compiler-only model and expose them to Claude Code | delegated | done | be9957c |
| B4 | Agents in `.claude/agents/` | delegated | done | 93096ec |
| B5 | User-level `gbai-reviewer` skill with adapted scripts and their shell tests | delegated | done | outside repo, no commit |
| B6 | Create and push integration branch `dev` | inline (git) | done | `git ls-remote --heads origin dev` → 7ad919b |

## Acceptance criteria

- Every agent and skill has valid frontmatter and references only live mechanisms (no removed commands).
- `gbai.yaml` validates against the documented keys; personal keys only in the global file.
- Reviewer scripts' shell tests pass; preflight runs in single-user mode.
- `go test ./...` in apps/cli still passes after skill edits (embedded skills test).

## Checks

Structural readback of agents/skills; `go test ./...` (apps/cli); reviewer shell tests.

## Progress

- 2026-10-26: feature document created on branch `feat/gbai-manual-bootstrap`.

- 2026-10-26: B6 done by parent: `dev` created on origin from main at 7ad919b.
- 2026-10-26: B1 `gbai.yaml` (project keys) and `~/.config/gbai/config.yaml` (personal keys); `go test ./...` in apps/cli: 435 passed.
- 2026-10-26: B2 four adapter skills; gh sub-issue and issue-dependency endpoints checked against docs.github.com REST docs (body ids are issue `id`, not `number`).
- 2026-10-26: B3 embedded skills rewritten; `skills_test.go` cycle rule strings updated (it required the removed `gbai worktree close`); `gbai-orca-inbox` reduced to a deprecation stub pending removal; skills exposed through relative symlinks in `.claude/skills/` (Claude Code docs: symlinked skill folders supported). `go test ./...`: 435 passed.
- 2026-10-26: B4 six agents; frontmatter parsed with PyYAML. `Agent(type)` lists are ignored inside subagents (Claude Code docs), so spawn limits are prompt rules; nesting depth assistant → cycle → coder → reviewer is 3, the documented default limit.

- 2026-10-26: B5 `~/.claude/skills/gbai-reviewer/` (SKILL.md, README.md, scripts, tests); shell tests: merge-head 16/0, preflight 28/0, review 36/0; real preflight `PREFLIGHT-OK repo=testeos-z/gbai-team base=dev login=leomerida15 mode=single-user engine=claude`.
- 2026-10-26: 41294ae sets `review.auto_merge: false` to match the reviewer default until the owner enables it.

## Next step

Owner review of open questions (alias and emails, auto-merge, stale user-level gbai skills, gbai-orca-inbox removal).
