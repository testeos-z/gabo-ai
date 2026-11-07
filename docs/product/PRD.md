# Product Requirements (PRD)

| field | value |
|---|---|
| status | Draft |
| owner | gabo-pm |
| version | 0.11 |
| date | 2026-11-03 |
| scope | The gabo-ai framework itself |
| linked docs | [TRD](TRD.md), [MIGRATIONS](MIGRATIONS.md), intents [`docs/product/intents/`](intents/README.md), diagrams `docs/product/diagrams/`, previous version [PRD 0.10](archive/PRD-0.10.md) |

This is the conceptual product document: what gabo-ai offers and what its user gets. It holds no acceptance criteria and no task links. Each change is an intent in [`docs/product/intents/`](intents/README.md), and its `spec.md` holds the requirements of that change with their acceptance criteria. The requirement ids R1–R39 and the section numbers are stable, because skills, tests and issues cite them (for example PRD 8.1, 10.1 and 10.2). The full acceptance criteria, the stories column and the dated amendment history of earlier versions are kept in [PRD 0.10](archive/PRD-0.10.md).

## Amendment 2026-11-03: per-change model (feature `per-change-sdlc`)

Approved (owner 2026-11-03, feature `per-change-sdlc`, applied directly by the owner's decision). gabo-ai's own documents now follow the model gabo ships: no upfront global plan, one intent per change, one wave per intent. Each changed requirement carries the note **Amended (owner 2026-11-03, per-change-sdlc)**.

| # | change | where |
|---|---|---|
| C1 | Planning is per change: intent, then spec, then (after a human accepts the spec) plan and tasks. There is no global implementation plan and no user-story set; the tracker is the global view. | R2, R20, 3, 7 |
| C2 | A wave is one intent. `wave.after` (`next` or `wait`, unset means `wait`) decides whether the assistant chains the next intent or waits for the owner. | R3, R4, 3 |
| C3 | Directiva adjusts an intent's `plan.md` and its tasks without re-planning; there is no global plan to redo. | R7, 11 |
| C4 | Plan-change scopes are one task, one intent, or several intents or the PRD/TRD conventions. | R19, R27, 10.2 |
| C5 | `gabo check` also validates the intent folders, including the human acceptance of a spec before its plan. | R33 |
| C6 | The PRD is conceptual: the requirements table has no acceptance criteria, stories or verification columns. | 8 |

## 1. Product in one sentence

gabo-ai is a framework that organizes a team of AI agents (direction, development and review) around an assistant that is your default point of contact, so that each change moves from intent to merged code wave after wave, and questions reach whoever can resolve them, using your methodology, your tracker and your infrastructure.

## 2. Problem

In AI-agent development the human is both the strongest and the weakest point of the flow. Strongest, because only they validate code and plans and turn ideas into product. Weakest, because they get exhausted, and today they also act as the glue between agents: they plan with one, launch coders with another, review results and answer every question.

When the human is tired or absent, work stops. Coder questions get lost or are resolved by guessing, the plan is not adjusted between waves, and each team has to adapt to the methodology, tracker and infrastructure a tool imposes. Planning everything upfront makes it worse: every iteration collides with a plan written before anyone knew the details.

gabo-ai protects the human's time and energy for what only they can do: validate, plan and decide.

### Evidence

Observed in the owner's own project "vulcam", an agent-built product where the owner coordinated coder and reviewer agents, between 2026-10-20 and 2026-10-24 (approved by the owner on 2026-10-25):

- **Resources left behind filled the disk.** On 2026-10-22 the disk filled: about 6 GB of loose worktrees, about 6 GB of reviewer end-to-end copies and 126 GB in podman, including 121 stopped containers. The owner had to clean up by hand.
- **Worktrees could not be closed cleanly.** The worktree for issue #280 had to be force-removed by the owner.
- **Tracker updates were silently skipped.** Board moves (Backlog → Ready) were skipped without notice.
- **The human was the relay.** Coordination between the planning session, the coder sessions and the reviewer was relayed by the owner.

## 3. Users

The first release covers both users.

1. **Solo developer working with AI agents.** Today they are the glue between agents. They want to describe a change, let the agent team build it, and receive only the decisions that matter.
2. **Tech lead of a small team mixing people and agents.** The team already has its tracker (GitHub issues, Plane, Trello), methodology and infrastructure. They want to add agents without changing how the team works.

### Roles

| role | needs |
|---|---|
| Owner (the session owner, identified by git user and alias) | Make product decisions, accept each spec, validate, answer the questions the team could not resolve |
| Assistant (default spokesperson; not part of directiva) | Start waves, receive reports, route questions, answer the owner |
| Directiva (pm, dev-lead, db-master and frontend agents) | Write the PRD, the TRD and each intent's documents, create and adjust tasks, research answers |
| Development (gabo-cycle, coders as native subagents, an independent reviewer) | Implement, review and report task outcomes |

### Journeys

The global flows live in the [app flow](APP_FLOW.md) and `docs/product/diagrams/`.

1. **Configure the team, once.** Setup guides the owner through the stack, the slots and the agents; `gabo check` confirms the result.
2. **One change at a time.** Amended (owner 2026-11-03, per-change-sdlc). The owner states a change; directiva writes its intent and spec; a human accepts the spec; directiva writes its plan and creates its tasks in the tracker with dependencies. Specs are accepted one at a time; there is no batch of accepted intents.
3. **One wave per intent.** Amended (owner 2026-11-03, per-change-sdlc). The assistant starts the wave of one accepted intent → the cycle launches coders on its unblocked tasks → coders open PRs → the reviewer returns its verdict → coders report → the cycle updates the tracker and sends the wave report to the assistant.
4. **Between waves.** The assistant asks directiva for adjustments or researched answers; directiva adjusts the intent's plan and tasks, not a re-plan. Then `wave.after` decides: `next` starts the next accepted intent without the owner, `wait` stops until the owner asks.
5. **End.** The assistant answers the owner.

## 4. Goals, non-goals and later

**Goals**

1. **Protect the human's energy.** The human receives product decisions, spec acceptances, validations and questions the team could not resolve, not coordination work.
2. **Work never stalls.** Waves advance without the human launching, watching or relaying anything.
3. **Questions surface.** Every question reaches whoever can resolve it (directiva, assistant or human). In auto mode each decision is recorded with its reason.
4. **Plan per change, not upfront.** Amended (owner 2026-11-03, per-change-sdlc). Each change is planned when it comes and adjusted without redoing it; no waterfall plan collides with later iterations.
5. **Adapts to the team.** Methodology, tracker, infrastructure, stack and reviewer are chosen by the user; the assistant helps configure them.

**Non-goals (permanent)**

- Imposing a development methodology or a stack.
- Replacing the tracker or being one.
- Replacing the human in product decisions, spec acceptance or final validation.
- Being an IDE or a coding agent.
- Providing its own infrastructure.
- Providing its own TODO or dashboard tool (tracker views and the assistant cover it).

**Later** — see [section 12](#12-later).

## 5. Appetite

The first release is scoped to one supported tracker (GitHub issues), the local reviewer, the Claude Code, Pi and OpenCode runtimes (OpenCode amended (owner 2026-11-03, [intent 0004](intents/0004-opencode-runtime/spec.md) S12)), and manual and auto modes. There is no fixed wave budget: the release takes as many waves as its scope needs. Claude Code, Pi and OpenCode are shipped and tested as runtime skills (R34; OpenCode per intent 0004); the user's agent can add another engine through the runtime builder (R35) without a gabo-ai release.

## 6. Assumptions and dependencies

- **Agent runtimes.** Agents run on Claude Code, Pi and OpenCode (OpenCode amended (owner 2026-11-03, [intent 0004](intents/0004-opencode-runtime/spec.md) S12)). Pi core has no subagents, so a Pi target needs a subagent package; gentle-pi is recommended and setup offers to install it. Any engine is usable only when it supports subagents and skills, or an equivalent. Amended (owner 2026-11-06, [intent 0013](intents/0013-several-engines-in-parallel-in-one-project/spec.md) S1, S8): several engines can work in one project at the same time, each in its own terminal with its own assistant, coders and reviewer on its own subscription; waves of different sessions never take the same intent.
- **Compatibility tiers (amended owner 2026-11-04).** Owner, verbatim: "si cambie el PRD ahora buscaremos dos tipos de compatibilidad - full con los motores soportados y flexible con cualquier otro que solo seran skills y configuraciones". **Full** tier: the supported engines `claude`, `codex`, `pi` and `opencode`. gabo ships their plugin or package and hooks, `gabo init` sets up their project-level integration (R32) and `gabo check` reads the engine files of the full-tier engines that have project keys, `claude`, `codex` and `pi` (R33). OpenCode is full tier through its runtime skill and stage-gate plugin (`gabo-runtime-opencode`), which `gabo init` reports, with no engine-key check. Amended (owner 2026-11-06, intent 0008): Codex, OpenCode and Pi share `.agents/skills` as their skill directory and Claude Code keeps `.claude/skills`; for every full-tier engine `gabo check` also applies the per-agent skill, declared-name, version, reviewer isolation and leftover-link rules (R33) through what the engine's runtime skill declares, and a full-tier runtime skill missing that declaration fails. **Flexible** tier: any other engine that meets the rule above. It gets only the gabo skills and configuration: no plugin and no hooks. `gabo init` writes nothing engine-specific for it and reports it as flexible tier, and `gabo check` stays engine-neutral for it. Amended (owner 2026-11-06, intent 0008): a flexible-tier runtime skill that declares no agent skill list gets one `gabo check` warning naming the engine and is never failed by the per-agent rules. Today `gabo.yaml` accepts only the four full-tier names in `runtimes.*`, so the flexible tier becomes usable once [intent 0005](intents/0005-any-runtime-engine/intent.md) (#176) lets `gabo.yaml` accept a flexible engine name.
- **Runtime builder access.** The runtime builder needs web access or a documentation MCP; without them it asks the user for documentation links (R35).
- **Direct sessions.** Direct sessions and the single assistant terminal depend on Orca in the first release (R14, R18).
- **Tracker.** The team's tracker is reachable through an adapter; GitHub issues in the first release (R11). It holds the product cards, their intents' tasks and the dependencies between them.
- **Repository host.** A git host (GitHub or GitLab) for PRs, comments and merges (R9, R10).
- **Persistent memory.** A memory provider (for example Engram) records auto-mode decisions (R6). gabo-ai recommends that it be shared by every agent of the project except the reviewer (R37).
- **Recommended stack.** gabo-ai recommends, but does not require, the gentle-ai stack: ODD in the methodology slot, RDD as coder-side review, Engram in the memory slot. Setup offers it first and installs it only if the user accepts.
- **Notifications.** A chat tool (for example Slack) or a request on the task may be used to notify the owner (R16).

### Memory backends researched

Researched 2026-10-27 (issue #107). "Shares" means two developers or a cloud agent on another machine read the same memory. Credentials and access are the user's own backend setup, outside gabo.

| backend | how agents connect | shares across machines | shares across developers | notes |
|---|---|---|---|---|
| Engram + Engram Cloud (recommended) | `engram mcp` as an MCP server; autosync to a self-hosted Engram Cloud server | yes, through the cloud server | yes, per enrolled project | Self-hostable; cloud sync is opt-in and project-scoped. Managed principals need an explicit per-project grant. |
| claude-mem | Claude Code plugin with a local SQLite store | only with paid Cloud Sync (per user) | only with the paid team plan | Local install alone does not meet R37 |
| Mem0 (hosted MCP) | Remote MCP with browser sign-in or an API key | yes | through a shared Mem0 organization and project | Hosted, not self-hosted |
| OpenMemory (Mem0, local) | Local MCP server for one machine | no | no | Does not meet R37 |

Sources for each row are listed in [PRD 0.10](archive/PRD-0.10.md#plan-change-2026-10-27-issue-107-shared-project-memory).

## 7. How work is organized

Amended (owner 2026-11-03, per-change-sdlc). This section replaces the former user-story index; the stories are archived in [`archive/stories/`](archive/stories/README.md).

- **Product card → intents.** A product card in the tracker can hold several intents, because one product change can need several technical changes.
- **Intent → spec → plan → tasks.** An intent states the problem. Its spec states the requirements, acceptance criteria, UX and flow, and any data change, and a human accepts it. Only then is its plan written, with one task per reviewable PR. Each task belongs to exactly one intent; its dependencies may cross intents.
- **One wave per intent.** A wave runs the unblocked tasks of one accepted intent. Chaining waves is optional (`wave.after`).
- **Global documents.** This PRD (what the product offers), the [TRD](TRD.md) (global conventions), the optional `DESIGN.md` (global design norms; gabo-ai has none) and [MIGRATIONS](MIGRATIONS.md) (the step-by-step log of data and config changes). There is no upfront database schema and no global plan.
- **Methodology stays open.** The spec is per change; the methodology slot (ODD or another) still governs how a coder builds a task.

## 8. What the product offers

Conceptual only. Type: F functional, Q quality, C constraint. The ids are stable so an intent or a TRD decision can cite them; the acceptance criteria of each change live in its intent's `spec.md`, and those of the requirements as first built are in [PRD 0.10](archive/PRD-0.10.md#8-requirements).

| id | type | what the product offers | what the user gets | priority |
|---|---|---|---|---|
| R1 | F | The assistant is the default point of contact of every session, and any directiva or development agent can still be addressed directly. | One place to talk to the whole team, without being blocked from any agent. | must |
| R2 | F | Amended (owner 2026-11-03, per-change-sdlc): directiva turns each change into an intent, a spec and, once a human accepts the spec, a plan whose tasks it creates in the tracker with blocked-by / unblocks dependencies. There is no global plan written upfront. | Work is planned when a change comes, every task traces to its intent, and there is no waterfall plan to redo. | must |
| R3 | F | Amended (owner 2026-11-03, per-change-sdlc): tasks are handled in three blocks: creation (directiva, per intent), read (the cycle, at the start of that intent's wave) and modification (the cycle updates status, directiva adjusts content). A task changed after a wave is read again at the next wave of its intent. | Task state stays consistent across waves without anyone re-entering it. | must |
| R4 | F | Amended (owner 2026-11-03, per-change-sdlc): the assistant starts each wave and receives its report, with no relay by the human. A wave runs one accepted intent. When it closes, `wave.after` decides: `next` starts the next accepted intent with unblocked tasks, `wait` (the default when unset) stops until the owner asks. | A single wave can run unattended, for example in the cloud while the owner is away, and the owner chooses whether waves chain. | must |
| R5 | F | Every question a coder raises reaches the assistant and the wave report, and is closed only with an answer and who gave it. | No question is lost or settled by guessing. | must |
| R6 | F | Manual and auto modes: in manual mode questions reach the owner through the assistant; in auto mode the assistant decides and records each decision with its reason in persistent memory. | The owner chooses how involved to be and can audit every decision taken without them. | must |
| R7 | F | Amended (owner 2026-11-03, per-change-sdlc): directiva adjusts an intent's `plan.md` and its tasks between waves or mid-wave without re-planning (existing tasks keep their identity), and returns researched answers to development questions. There is no global plan. | The owner changes course without losing work or restarting planning. | must |
| R8 | F | The coder uses the methodology declared in the methodology slot, linked as `gabo-methodology`; switching it is one config line, a rebuild of the affected agents and a passing `gabo check`. | The team keeps its own way of building software. | must |
| R9 | F | Every PR is reviewed in two layers: an optional coder-side review per methodology, and a mandatory reviewer-side merge gate (tests, build, lint and an independent agent review) run by deterministic scripts installed at user level, in a clean environment, never merging into main. | Only validated code merges, and PR code cannot change the scripts that review it. | must |
| R10 | F | After each coder report the tracker shows the task status, and every PR is linked to its task. | The tracker always reflects the real state of the work. | must |
| R11 | F | Pluggable trackers: GitHub issues in the first release; another tracker is a provider skill plus a plugin or MCP, without changing the agents' skills. | The team keeps its tracker. | must (gh); should (others) |
| R12 | F | Setup is guided: the assistant asks before configuring anything, suggests options and applies no silent default, including the stack, where each agent runs and which tracker is used. | Nothing is configured behind the user's back. | should |
| R13 | F | When a wave closes, the worktrees and merged branches it created are removed, and what could not be removed is reported. | The machine stays clean without manual cleanup. | should |
| R14 | F | When the human intervenes in a task, the assistant mediates by default and, if mediation drags on, offers a named Orca session in the task's worktree; afterwards it resumes from the traces without redoing work. | The owner can steer any task without losing its context. | must |
| R15 | F | During an auto-mode run the human receives no prompts, only the final answer. | The owner can step away while the team works. | must |
| R16 | F | Setup identifies the session owner by git user and addresses them by an alias stored in the developer catalog. | The assistant knows who it is talking to and addresses only the session owner. | must |
| R17 | F | The assistant keeps per-user interaction preferences in two layers and asks about interaction comfort at each wave close. | The number and style of interactions fit the owner. | must |
| R18 | F | Only the assistant's Orca terminal opens by default; more open on request. | A calm workspace with one conversation. | must |
| R19 | F | Amended (owner 2026-11-03, per-change-sdlc): a critical question becomes a plan-change proposal and is never resolved silently. The config sets the widest scope that may change, one task in an intent's `plan.md`, one whole intent (its spec and plan) or several intents and the PRD/TRD conventions, and whether that widest scope may change without the owner. Each proposal has its own issue and PR touching only the product documents and tasks, and is debated with the owner or, when absent, with a clean-context plan-change reviewer. | Large changes never slip in unnoticed, and the owner's absence does not stop small ones. | must |
| R20 | F | Amended (owner 2026-11-03, per-change-sdlc): directiva writes the PRD, the TRD, each intent's `intent.md`, `spec.md` and `plan.md`, the optional `DESIGN.md` and the MIGRATIONS log from its own templates, asking their fixed questions one at a time. A traceability check over intents, specs, plans and tracker tasks blocks the next planning step on a gap, and `gabo check` enforces the intent stage order. | Consistent documents, and gaps are caught before any coder starts. | must |
| R21 | Q | Secrets and credentials never enter agent context or generated files; agents receive only references to where secrets live. | Safe to use with real credentials. | should |
| R22 | Q | The reviewer runs isolated from the coder, with its own clone, home and cache. | The review cannot be influenced by the code it reviews. | should |
| R23 | Q | In auto mode the system deletes only resources it or its subagents created in the workflow. | Nothing of the user's is deleted by the agents. | should |
| R24 | Q | Every task write is verified by re-reading it; an unverified write is reported, never skipped silently. | Tracker updates can be trusted. | should |
| R25 | Q | An interrupted wave resumes from tracker state and memory traces without redoing finished tasks. | Interruptions cost no work. | should |
| R26 | Q | The reviewer reviews each pushed head at most once. | Review cost follows actual changes. | should |
| R27 | Q | Amended (owner 2026-11-03, per-change-sdlc): the plan-change reviewer runs only when a plan change is proposed, at any of the three scopes (one task, one intent, or several intents and the PRD/TRD conventions). | Debate cost is paid only when a plan actually changes. | should |
| R28 | Q | A new team reaches its first wave quickly after install; the time is measured (initial target: one working session). | Setup pays off fast. | should |
| R29 | Q | Every wave produces a report: interactions and reasons, decisions, questions with resolver, cleanup leftovers. | The owner sees what happened without asking. | should |
| R30 | Q | Every auto-mode decision is queryable with its reason, wave and task. | Full accountability for auto mode. | should |
| R31 | C | The gabo CLI offers only `gabo init`, `gabo check`, `gabo version` and `gabo prompt`; agents do everything else with skills. `gabo check` has one additive hook mode, `gabo check --pre-write <path>...` (intent 0001 S11), that the engines' pre-write hooks call to block writing an intent's `plan.md` before its spec is accepted: exit 0 silent when every write passes, 1 with one `✗ <path>: <reason>` line per blocked path (fail closed), 2 on a usage error; `gabo check` without the flag keeps its output and exit codes (TRD 9.4). | A small, universal tool with little to learn. | must |
| R32 | F | `gabo init` sets up or refreshes the copied skill catalog and its version stamp, installs the user-level reviewer skill and prints the setup prompt; it never touches `gabo.yaml` or the user config. Amended (owner 2026-11-04, [intent 0006](intents/0006-engine-integration-at-init/spec.md) S1–S11, S15): for the engines `gabo.yaml` declares in `runtimes.*`, it also writes the project-level gabo integration keys (Claude Code `.claude/settings.json`, Codex `.codex/config.toml`, Pi `.pi/settings.json`), merged, reported and never the user's global engine config, and ends with a report of what it wrote and what the user must still run. Amended (owner 2026-11-04, compatibility tiers, section 6): this applies to full-tier engines only. For a flexible-tier engine it writes nothing engine-specific and reports one line naming the engine as flexible tier (skills and configuration only). Amended (owner 2026-11-06, [intent 0010](intents/0010-gabo-init-writes-the-main-session-key/spec.md) S1 to S5, S8; owner's rationale, verbatim: "one command completes setup; works with guard hooks; gabo init is the single writer of engine config; clear guidance, no loop."): it also writes the main-session key of the engine in `runtimes.assistant` (`agent: "gabo-assistant"` in `.claude/settings.json` for Claude Code, `default_agent: "gabo-assistant"` in the project OpenCode config for OpenCode), removes that gabo value from the other engine's file when the assistant engine changes, and writes `experimental.subagent_depth` (at least 3) in the project OpenCode config when a role runs on OpenCode; a user's own value is kept and reported. `gabo init` is the only writer of engine config keys for full-tier engines; the runtime skills build agent files and links only. | One safe command to start or upgrade a project. | must |
| R33 | F | `gabo check` validates the project read-only and independent of the engine: config schemas, agents of the neutral spec, slot coverage, catalog stamp and agent version lines. Amended (owner 2026-11-03, per-change-sdlc): it also validates the intent folders, including that no plan exists before a human accepted its spec. Amended (owner 2026-11-04, [intent 0006](intents/0006-engine-integration-at-init/spec.md) S11, covered by the owner's approval of the engine-integration amendments, verbatim: "apruebo el spec 3 sobre lo que falta"; record: https://github.com/testeos-z/gabo-ai/issues/261#issuecomment-6633243675): for the engines `gabo.yaml` declares, it also verifies, read-only, the project-level gabo integration keys that `gabo init` writes; every other rule stays independent of the engine. Amended (owner 2026-11-04, compatibility tiers, section 6; this resolves the conflict between the engine-file checks and "independent of the engine"): the engine-file checks apply only to full-tier engines, namely those with project keys (`claude`, `codex`, `pi`; OpenCode has no engine-key check). For a flexible-tier engine `gabo check` stays independent of the engine and reads no engine file. Amended (owner 2026-11-06, intent 0008; [spec](intents/0008-stricter-gabo-check/spec.md) S1 to S9): for each engine `gabo.yaml` declares, through what its runtime skill declares and with no engine name in Go, it also checks per-agent skills (each agent file lists exactly its spec skills plus the fixed-name links of its template, S1, S2), declared names (an agent's declared name equals its spec id, S6), the no-silent-skip version rule (every agent file carries a `gabo-version: x.y.z` line, and without a valid stamp the versions are reported as not compared, never skipped, S7), the reviewer isolation rule (no `gabo-reviewer` in a project skill directory or an agent's skill list, S8) and the leftover-link rule (no gabo link left in a former skill directory such as `.pi/skills` or `.opencode/skills` after the move to the shared `.agents/skills`, S3, S4). A flexible-tier runtime skill that declares no agent skill list gets one warning and is never failed by the per-agent rules. The new rules keep `gabo check` read-only, keep its output shape and exit codes, and are skipped under `GABO_UPGRADE_PR=1` (S9). Amended (owner 2026-11-06, [intent 0010](intents/0010-gabo-init-writes-the-main-session-key/spec.md) S6): a failure whose fix is a key `gabo init` writes (the main-session key, missing, stale or set to another value, and `experimental.subagent_depth`) names `gabo init` and the exact key and file in its `next:` line, read from the runtime skill's `metadata.main_session`; the rules themselves are unchanged. Amended (owner 2026-11-07, [intent 0012](intents/0012-design-norms-template/spec.md) S7, D4; owner, verbatim: "6. que si lo valide."): when the project has the optional design norms `docs/product/DESIGN.md`, it also validates them, read-only and independent of the engine, against the shape of the `gabo-frontend` norms template: the 15 sections, the rule-table columns, an RFC 2119 level that matches the rule's keyword, a verification for every MUST rule, and the exceptions and changelog tables; without the file nothing is checked. Amended (owner 2026-11-07, [intent 0014](intents/0014-agent-name-and-duplicate-skill-checks/spec.md) S1, S2; owner, verbatim: "que falle" and "si no la listo tomare tu propuesta para toda opcion no listada"; spec accepted by the owner 2026-11-07, release 0.2.10): where the runtime skill declares where an agent's name lives, an agent file that declares no name fails, since the engine cannot load it, and a skill an agent file lists twice gets its own failure saying so instead of being reported as not in the agent spec. | One check that agents, the reviewer and CI all trust. | must |
| R34 | F | The user's agent builds the configuration and the agents from gabo-ai skills and fixed templates, without inventing content, then runs `gabo check`. | Any supported engine gets a working team without hand-written agent files. | must |
| R35 | F | Where no runtime skill exists, the runtime builder writes one from verified sources and tests it. Amended (owner 2026-11-04, compatibility tiers, section 6): an engine added this way is in the flexible tier, with gabo skills and configuration only. Using its name in `gabo.yaml` depends on [intent 0005](intents/0005-any-runtime-engine/intent.md). | A new engine needs no gabo-ai release. | must |
| R36 | C | The setup prompt has a single source: the text embedded in the CLI, published unchanged in the documentation. | Both setup entries behave the same. | must |
| R37 | F | Proposed (auto) (issue #107): gabo-ai recommends one memory backend shared by every agent of the project except the reviewer, on every machine and for every developer; a local-only memory stays acceptable. | Agents work from the same context. | should |
| R38 | F | Proposed (auto) (issue #107): setup explains the shared-memory recommendation and helps the user choose a backend, applying nothing silently. | An informed memory choice. | should |
| R39 | F | Proposed (auto) (issue #148): `gabo prompt` prints the embedded setup prompt and changes nothing. | The setup prompt can be reprinted at any time. | should |
| R40 | F | Amended (owner 2026-11-06, [intent 0013](intents/0013-several-engines-in-parallel-in-one-project/spec.md)): several engines run in parallel in one project: every engine the owner lists as an assistant keeps its own main session, coders and the reviewer can run on the engine of the terminal that runs the wave or on one fixed engine, and a wave claims its intent so no two sessions run the same intent. | Two teams on two subscriptions work at the same time without undoing or taking each other's work. | should |

### 8.1 Operating rules

Approved by the owner on 2026-10-25.

1. **Mid-wave task change (R7, R3).** A change directiva makes to a task a coder is working on applies when the coder finishes its current round; the coder is not interrupted.
2. **Task state during a plan-change debate (R19).** Only the affected task is BLOCKED while the proposal is debated or waits for the owner; the rest of the wave continues.
3. **Reviewer-to-coder rounds (R9, R10).** At most 4 fix rounds per PR; after that the task is reported BLOCKED and the question escalates to the assistant.

## 9. Quality requirements

Quality requirements are rows R21–R30 (type Q) in section 8: security (R21–R23), reliability (R24–R25), cost (R26–R27), time-to-value (R28) and observability (R29–R30). Numeric targets are initial and calibrated after the first waves. Performance targets are not set in the first release.

## 10. Constraints

### 10.1 Configuration standard

- **Two files.** The global user file holds personal preferences (alias, interaction level, tone, language): the stable personalization layer. The project file `gabo.yaml` holds team and repository settings. Field-level shapes are in [TRD 9.3](TRD.md#93-configuration-shapes).
- **Precedence.** For project keys the local file wins; for personal keys the global file always wins.
- **Contracts and adapters.** Every configurable slot has a contract shipped by gabo-ai and an adapter supplied by the user or written with the assistant: a Markdown skill or document. Tool-specific behavior is never compiled code.
- **Setup flow.** Ask how each slot is handled → receive the adapter → check it covers the contract → build the agents through the runtime skill → run `gabo check` → verify with a harmless operation by a subagent of the working kind.
- **Setup entries.** (a) Run `gabo init` in the project, then ask the agent to configure gabo-ai; (b) paste the published setup prompt into the agent, which installs the CLI and continues. Both use the same prompt text (R36). There is no setup terminal UI.
- **Config writes.** Agents edit the YAML files directly (during setup through the config-writer skill); validation runs at the next `gabo check`.
- **Go CLI scope.** The CLI copies the skill catalog, installs the user-level reviewer skill, writes the project-level engine integration keys for the declared full-tier engines (R32, amended owner 2026-11-04, intent 0006 and compatibility tiers), validates read-only, prints its version and prints the setup prompt (R31–R33, R39). Everything during the work is done by agents with their own tools, guided by skills and adapters.
- **Personalization (R16, R17).** Stable layer: the global user file, read by every agent. Learning layer: observations in the memory slot under `user/preferences`, read by the assistant at session start; an observation repeated about 3 times is proposed for the stable layer. The comfort answer at wave close adjusts `interaction_level` and `tone`. Secrets and sensitive personal data are never stored.
- **Configurable slots.** Identity, mode (manual or auto), methodology, runtime per agent, tracker, repository host, reviewer placement, reviewer runtime, reviewer signing or marker, coder-side review, persistent memory, notification channel, direct-session manager (Orca), integration branch, parallelism, kept worktrees, wave chaining (`wave.after`).
- **Recommended options, not defaults.** Setup recommends gentle-ai ODD with RDD for the methodology slot and Engram for the memory slot; a recommendation is offered first with its reason and installed only if the user accepts. No slot gets a silent default (R12).
- **Stack chosen by the user.** The user chooses the technologies of their project and the assistant helps configure them; the TRD records the choice and never imposes one.
- **Shared project memory (R37, R38).** Recommended, not required: one backend for every agent except the reviewer. Credentials and access are the user's own backend setup.

### 10.2 Auto-mode limits

In auto mode the assistant:

- **May merge** only into the integration branch it was assigned, **never into main**.
- **May delete** only resources it or its subagents created in this workflow, never anything else.
- **Changes the plan only within the configured scope** (R19). Amended (owner 2026-11-03, per-change-sdlc): a change to several intents or to the PRD/TRD conventions applies without the owner only when `plan_change.global_without_owner` allows it, and only after the plan-change reviewer agreed; otherwise the proposal waits for the owner. Plan changes always go in their own issue and PR, never in a code PR.
- **Never accepts a spec.** A human accepts each intent's spec before its plan is written, and a human makes the production decision.

## 11. Success metrics

| metric | how it is measured |
|---|---|
| Interactions per wave and their reasons | Counted in each wave report. An input to learn the owner's comfort, not a reduction target. |
| Share of waves closed by the assistant without the human launching or relaying anything | Waves closed without human launch or relay steps, divided by total waves. |
| Questions recorded and resolved, and by whom | Count of questions raised, resolved, and resolver (directiva, assistant, owner). |
| Plan adjustments per wave without re-planning an intent | Count of task adjustments per wave; any intent whose plan was rewritten is reported separately. Amended (owner 2026-11-03, per-change-sdlc). |
| Time from install to the first wave running with the team's tracker and methodology | Elapsed time from install to the start of the first wave. |

**Targets.** The first release measures without targets. After 3 to 5 real waves the baseline is known and the owner sets targets.

## 12. Later

Out of the first release.

- Plane, Trello and other tracker providers.
- Reviewer on a separate server, GitHub Actions or GitLab CI (first release: local).
- Additional reviewer runtimes.
- Direct sessions with other terminal managers (for example herdr).
- Target values for the success metrics.
- Shipped memory adapters for claude-mem and Mem0 (open question 13); until then they are user-supplied adapters.

## 13. Open questions

Questions 1–12 are answered; their answers are in [PRD 0.10](archive/PRD-0.10.md#13-open-questions).

| # | question | status |
|---|---|---|
| 13 | Does gabo-ai ship memory adapters for claude-mem and Mem0 in the first release, or only list them in setup as user-supplied adapters, with Engram the only shipped memory adapter? Proposed (auto): list only. | open (issue #107) |

## 14. Glossary

| term | meaning |
|---|---|
| Owner | The session owner, identified by git user and alias; makes product decisions, accepts specs and validates. |
| Assistant | The default point of contact for the whole flow; starts waves, receives reports, routes questions. Not part of directiva. |
| Directiva | The direction agents: pm, dev-lead, db-master and frontend. They write the product documents, create and adjust tasks, research answers. |
| gabo-cycle | The agent that runs one wave: reads the intent's tasks, launches coders, updates the tracker, reports to the assistant. |
| Coder | A native subagent of the cycle that implements one task and opens its PR. |
| Reviewer | An independent, isolated reviewer: deterministic scripts in the user-level `gabo-reviewer` skill validate each PR and comment the verdict; a reviewer agent invokes them and relays the verdict. |
| Plan-change reviewer | A fresh clean-context session that must agree with a plan change or propose a different improvement. |
| Intent | One change, in `docs/product/intents/<NNNN-slug>/`: its problem (`intent.md`), its spec, its plan and its review findings. |
| Spec | The requirements, acceptance criteria, UX and flow and data changes of one intent; a human accepts it. |
| Plan | The ordered steps and the tasks of one intent, written after its spec is accepted. |
| Wave | The run of one accepted intent's unblocked tasks. Amended (owner 2026-11-03, per-change-sdlc); earlier: the set of all tasks whose dependencies are done. |
| `wave.after` | What the assistant does when a wave closes: `next` starts the next accepted intent, `wait` (default) stops. |
| Critical question | A question whose answer would change an intent's spec or plan, several intents, or the PRD/TRD conventions; it becomes a plan-change proposal. |
| Manual / auto mode | Manual: questions reach the owner through the assistant. Auto: the assistant decides and records each decision with its reason. |
| Slot | A configurable part of the system (for example tracker, memory, reviewer placement). |
| Contract | What the agents must be able to do for a slot; shipped by gabo-ai. |
| Adapter | A Markdown skill or document explaining how to fulfil a contract with the team's tool. |
| Neutral agent spec | The engine-independent list of gabo-ai agents: role, skills, tools and who launches whom (R34). |
| Runtime skill | A shipped skill that builds the agents of the neutral agent spec in one engine's format (R34). |
| Runtime builder | A skill that writes a runtime skill for an engine without one, from verified sources (R35). |
| Full compatibility tier | The supported engines `claude`, `codex`, `pi` and `opencode`: plugin or package, hooks and project-level integration set up by `gabo init`; `gabo check` reads the engine files of those with project keys (`claude`, `codex`, `pi`), while OpenCode is full tier through its runtime skill and stage-gate plugin, with no engine-key check (section 6, R32, R33). |
| Flexible compatibility tier | Any other engine with subagents and skills, or an equivalent: gabo skills and configuration only, no plugin and no hooks; `gabo init` writes nothing engine-specific and `gabo check` stays engine-neutral for it (section 6, R35; usable once intent 0005 lands). |
| Setup prompt | The text embedded in the CLI and published in the documentation that starts setup in the user's agent (R36, R39). |
| Shared project memory | The recommended single memory backend used by every agent of a project except the reviewer (R37). |
| Stable layer / learning layer | Preferences in the global user file / observations in the memory slot under `user/preferences`. |
| Direct session | An extra Orca terminal in a task's worktree, opened only on the owner's request. |
| Integration branch | The branch the assistant may merge into in auto mode; never main. |
