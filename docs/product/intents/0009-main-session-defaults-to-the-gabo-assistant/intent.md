# Intent 0009 Main session defaults to the gabo assistant

| field | value |
|---|---|
| intent id | 0009-main-session-defaults-to-the-gabo-assistant |
| status | Accepted |
| owner | gabo-pm |
| product card | #171 Runtime integrations |
| date | 2026-11-06 |

## Problem

When the owner opens a session in a gabo project, the engine's main session does not act as the gabo assistant. Nothing tells it to: no runtime skill (`gabo-runtime-claude`, `gabo-runtime-codex`, `gabo-runtime-opencode`, `gabo-runtime-pi`, `gabo-runtime-builder`) writes anything into the engine's main-session instructions file, and `gabo check` does not look for it. The assistant skill is not even reachable by name: it is never linked into an engine's skill directory, because the runtime skills link only the skills some agent file lists and the assistant has no agent file. So the owner gets a generic coding agent, which may write code or run long work in the foreground instead of delegating it to the gabo subagents, and the owner has to ask for the assistant by hand every session. The vulcam project session reported it on the owner's behalf, and the gabo-reviewer findings on cappy-lab/vulcam PR #638 add two more problems: a plain "you are the assistant" rule in a project instructions file would reach every session in the repository, coder and reviewer sessions included, against their roles; and a block written by hand has no owner, so it goes stale when `runtimes.assistant` changes. The owner, verbatim: "2. arreglaremos para que por defecto hable con el asistente si no indica que quiere hablar con otro."

## Who is affected

Every owner of a gabo project, with the engine named in `runtimes.assistant` (`claude`, `codex`, `pi` or `opencode`), at the start of every session.

## Desired outcome

- In a gabo project, the main session of the engine in `runtimes.assistant` acts as the gabo assistant from the first message, unless the owner says they want to talk to another agent.
- The assistant loads its skill and follows its Session Start before the first reply, stays the owner's single point of contact, never blocks on long work, and delegates writing and long work to the gabo subagents.
- Coder, reviewer and other role sessions keep their own role; only the owner's interactive main session becomes the assistant.
- The default follows `gabo.yaml`: it is written, refreshed and removed by gabo when `runtimes.assistant` changes, never left stale.
- `gabo check` reports when that default is missing or does not match `gabo.yaml`.
- The change ships in gabo release 0.2.8 (tag `apps/cli/v0.2.8`). Owner, verbatim: "3. subiremos la version a 0.2.8".

## Constraints

- gabo writes only project-level files and only what it needs, keeping every other byte of a file it touches (intent 0004 S8, intent 0006 constraints); never the user's global engine instructions (`~/.claude/CLAUDE.md`, `~/.codex/AGENTS.md`, the Pi agent home, `~/.config/opencode/AGENTS.md`).
- `CLAUDE.md` and `AGENTS.md` are `review.human_only_paths` in `gabo.yaml`: in this repository a change to them needs the owner's merge.
- Runtime knowledge stays in the runtime skills (TRD D24); `gabo check` stays read-only (PRD R33).
- The owner may still talk to any agent directly; the assistant never blocks that (gabo-assistant Spokesperson, J9).
- The assistant runs only in the owner's main session (`gabo-assistant` Activation Contract); sessions started with a role prompt or headless keep their role.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Which file is each engine's main-session instructions file, and which one wins when several exist? | answered in spec (evidence base) |
| 2 | Can each engine give the main session its role directly, without reaching role sessions? | answered in spec (D1): Claude Code and OpenCode yes (project default agent); Codex and Pi no (scoped block). |

## Links

| kind | reference |
|---|---|
| product card | [#171 Runtime integrations](https://github.com/testeos-z/gabo-ai/issues/171) |
| intent issue | [#305 I0009: Main session defaults to the gabo assistant](https://github.com/testeos-z/gabo-ai/issues/305) |
| PRD | [PRD](../../PRD.md) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) |
| review findings | gabo-reviewer findings on cappy-lab/vulcam PR #638 (assistant skill not linked; project instruction files reach every session; no owner for the block) |
| related intent | 0008-stricter-gabo-check (the stricter check set this block check joins), 0006-engine-integration-at-init |
