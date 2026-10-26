# Product Requirements (PRD)

| field | value |
|---|---|
| status | Draft |
| owner | gbai-pm |
| version | 0.6 |
| date | 2026-10-26 |
| scope | The gbai framework itself |
| linked docs | [TRD](TRD.md), stories `docs/product/stories/`, diagrams `docs/product/diagrams/` |

## Auto-mode proposals (approved by owner 2026-10-25)

This version was completed by the PM in auto mode (R6, R19). Every item below is a proposal, marked **Approved (auto proposal, owner 2026-10-25)** where it appears, with a one-line reason. Nothing previously approved was changed and no existing priority was raised. The owner approves or rejects these in one pass.

| # | section | proposal |
|---|---|---|
| P1 | 2. Problem — Evidence | Evidence from the owner's own project "vulcam" (2026-10-20 to 2026-10-24): disk exhaustion, an unclosable worktree, silently skipped board moves, owner-relayed coordination. |
| P2 | 5. Appetite | First release limited to GitHub issues, local reviewer, Claude Code and Pi, manual and auto modes; no fixed wave budget; as many waves as the scope needs (updated 2026-10-26). |
| P3 | 7. User stories | Eight proposed stories US-001 to US-008 mapped to requirement ids; story files not yet written. |
| P4 | 8. Requirements — stories column | Stories column filled from the US mapping; R1 and R13 have no story yet. |
| P5 | 8.1 Operating rules | Mid-wave task change applies when the coder finishes its current round (answers open question 1). |
| P6 | 8.1 Operating rules | During a plan-change debate only the affected task is BLOCKED (answers open question 2). |
| P7 | 8.1 Operating rules | At most 4 reviewer-to-coder fix rounds per PR, then BLOCKED and escalated (answers open question 3). |
| P8 | 8. Requirements — R21–R23 | Security quality requirements: no secrets in agent context, isolated reviewer, auto mode deletes only its own resources. |
| P9 | 8. Requirements — R24–R25 | Reliability quality requirements: verified task writes, resumable waves. |
| P10 | 8. Requirements — R26–R27 | Cost quality requirements: one review per pushed head, plan-change reviewer only on proposals. |
| P11 | 8. Requirements — R28 | Time-to-value quality requirement: first wave within one working session (initial target). |
| P12 | 8. Requirements — R29–R30 | Observability quality requirements: a report per wave, every auto decision queryable. |
| P13 | 9. Quality requirements | Section points to R21–R30 instead of "not yet defined". |

## 1. Product in one sentence

gbai is a framework that organizes a team of AI agents (direction, development and review) around an assistant that is your default point of contact, so that work moves forward wave after wave without stopping and questions reach whoever can resolve them, using your methodology, your tracker and your infrastructure.

## 2. Problem

In AI-agent development the human is both the strongest and the weakest point of the flow. Strongest, because only they validate code and plans and turn ideas into product. Weakest, because they get exhausted, and today they also act as the glue between agents: they plan with one, launch coders with another, review results and answer every question.

When the human is tired or absent, work stops. Coder questions get lost or are resolved by guessing, the plan is not adjusted between waves, and each team has to adapt to the methodology, tracker and infrastructure a tool imposes.

gbai protects the human's time and energy for what only they can do: validate, plan and decide.

**Evidence.** The problem statement describes current practice: the human plans with one agent, launches coders with another, reviews results and answers every question.

**Approved (auto proposal, owner 2026-10-25):** observed incidents from the owner's own project "vulcam", an agent-built product where the owner coordinated coder and reviewer agents, between 2026-10-20 and 2026-10-24. Reason: first-hand observations of the exact flow gbai targets are the strongest evidence available.

- **Resources left behind filled the disk.** On 2026-10-22 the disk filled: about 6 GB of loose worktrees, about 6 GB of reviewer end-to-end copies and 126 GB in podman, including 121 stopped containers. Host load reached 88. The owner had to clean up by hand.
- **Worktrees could not be closed cleanly.** The worktree for issue #280 could not be closed because of leftover debug files and had to be force-removed by the owner.
- **Tracker updates were silently skipped.** Project board moves (Backlog → Ready) were skipped by the agents without notice; the owner had to notice and move the cards.
- **The human was the relay.** Coordination between the planning session, the coder sessions and the reviewer was relayed by the owner.

## 3. Users

The first release covers both users.

1. **Solo developer working with AI agents.** Today they are the glue between agents. They want to plan once, let the agent team work, and receive only the decisions that matter.
2. **Tech lead of a small team mixing people and agents.** The team already has its tracker (GitHub issues, Plane, Trello), methodology and infrastructure. They want to add agents without changing how the team works.

### Roles

| role | who | responsibility |
|---|---|---|
| Owner | The session owner, identified by git user and alias | Product decisions, validation, answers to unresolved questions |
| Assistant | Default spokesperson for the whole flow; not part of directiva | Starts waves, receives reports, routes questions, answers the owner |
| Directiva | pm, dev-lead, db-master and frontend agents | Write `docs/product`, create tasks, adjust the plan, research answers |
| Development | gbai-cycle (runs waves), coders (native subagents), reviewer (independent process) | Implement, review and report task outcomes |

### Journeys

Detailed flows live in the app flow and `docs/product/diagrams/`.

1. **Global planning, once.** Directiva writes `docs/product` and creates every task in the tracker with dependencies (blocked by / unblocks): the work route map.
2. **Waves derive from the map.** A wave is the set of tasks whose dependencies are done.
3. **Each wave.** Assistant starts the wave → cycle launches coders → coders open PRs → reviewer comments the verdict and returns it to the coder → coder reports DONE or BLOCKED → cycle updates the tracker and sends the wave report and questions to the assistant.
4. **Between waves.** The assistant asks directiva for changes or researched answers; directiva adjusts tasks (not a re-plan); the assistant starts the next wave.
5. **End.** The assistant answers the owner.

Task lifecycle: **creation** (directiva) → **read** (cycle, at wave start) ← **modification** (cycle updates status, directiva adjusts). Modified tasks are read again.

## 4. Goals, non-goals and later

**Goals**

1. **Protect the human's energy.** The human receives product decisions, validations and questions the team could not resolve, not coordination work.
2. **Work never stalls.** Waves advance without the human launching, watching or relaying anything.
3. **Questions surface.** Every question reaches whoever can resolve it (directiva, assistant or human). In auto mode each decision is recorded with its reason.
4. **A living plan.** The plan is made once and adjusted between waves or mid-wave without redoing it.
5. **Adapts to the team.** Methodology, tracker, infrastructure and reviewer are chosen by the user; the assistant guides configuration.

**Non-goals (permanent)**

- Imposing a development methodology.
- Replacing the tracker or being one.
- Replacing the human in product decisions or final validation.
- Being an IDE or a coding agent.
- Providing its own infrastructure.
- Providing its own TODO or dashboard tool (tracker views and the assistant cover it; there is no Go status command, TRD D18).

**Later (deferred, out of the first release)** — see [Later](#12-later).

## 5. Appetite

**Approved (auto proposal, owner 2026-10-25):** the first release is scoped to one supported tracker (GitHub issues), the local reviewer, the Claude Code and Pi runtimes, and manual and auto modes. There is no fixed wave budget (updated 2026-10-26, owner): the release takes as many waves as needed to finish its scope. Reason: the Later section already defers other trackers and reviewer placements; scope, not a wave count, bounds the release.

## 6. Assumptions and dependencies

- **Agent runtimes.** Agents run on the supported runtimes Claude Code and Pi (TRD D3). Approved (owner 2026-10-26): Pi core has no subagents, so a Pi target needs a subagent package; gentle-pi (with its `gentle-shell` launcher) is recommended and setup offers to install it when missing; another subagent package is acceptable; `gbai generate` refuses Pi targets when none is present and prints what to install (TRD D21).
- **Direct sessions.** Direct sessions and the single assistant terminal depend on Orca; the first release supports Orca only (R14, R18).
- **Tracker.** The team's tracker is reachable through an adapter (skill plus plugin or MCP); GitHub issues in the first release (R11).
- **Repository host.** A git host (GitHub or GitLab) for PRs, comments and merges (R9, R10).
- **Persistent memory.** A memory provider (for example Engram) records auto-mode decisions (R6).
- **Recommended stack.** Approved (owner 2026-10-26): gbai recommends, but does not require, the gentle-ai stack: gentle-ai with ODD (methodology slot), RDD (receipt-driven review as the coder-side validation inside the methodology adapter) and Engram (memory slot). Setup offers it as the recommended option and installs it only if the user accepts (TRD D21).
- **Notifications.** A chat tool (for example Slack) or a request on the task may be used to notify the owner (R16).

## 7. User stories

The PM proposes user stories (INVEST with Given-When-Then) in `docs/product/stories/` for owner approval.

**Approved (auto proposal, owner 2026-10-25):** the index below. Story files were written on 2026-10-26 and US-009 to US-011 were added (owner approval 2026-10-26). Reason: each story groups the requirements one user needs for one outcome, so every functional requirement except R1 and R13 is traced to at least one story.

| id | title | user | requirement ids | status | file |
|---|---|---|---|---|---|
| US-001 | Configure the team | Solo developer | R12, R16, R8 | approved | [US-001-configure-the-team.md](stories/US-001-configure-the-team.md) |
| US-002 | Plan the product once | Solo developer | R2, R3, R20 | approved | [US-002-plan-the-product-once.md](stories/US-002-plan-the-product-once.md) |
| US-003 | Run a wave without watching | Solo developer | R4, R5, R15 | approved | [US-003-run-a-wave-without-watching.md](stories/US-003-run-a-wave-without-watching.md) |
| US-004 | Resolve questions by mode | Solo developer | R5, R6, R17 | approved | [US-004-resolve-questions-by-mode.md](stories/US-004-resolve-questions-by-mode.md) |
| US-005 | Intervene in a task | Solo developer | R14, R18 | approved | [US-005-intervene-in-a-task.md](stories/US-005-intervene-in-a-task.md) |
| US-006 | Adjust the plan safely | Solo developer | R7, R19 | approved | [US-006-adjust-the-plan-safely.md](stories/US-006-adjust-the-plan-safely.md) |
| US-007 | Use the team's own tools | Tech lead | R8, R11 | approved | [US-007-use-the-teams-own-tools.md](stories/US-007-use-the-teams-own-tools.md) |
| US-008 | Trust what gets merged | Tech lead | R9, R10 | approved | [US-008-trust-what-gets-merged.md](stories/US-008-trust-what-gets-merged.md) |
| US-009 | Single point of contact | Solo developer | R1 | approved | [US-009-single-point-of-contact.md](stories/US-009-single-point-of-contact.md) |
| US-010 | My machine stays clean | Solo developer | R13 | approved | [US-010-my-machine-stays-clean.md](stories/US-010-my-machine-stays-clean.md) |
| US-011 | Resume without losing work | Solo developer | R25 | approved | [US-011-resume-without-losing-work.md](stories/US-011-resume-without-losing-work.md) |

## 8. Requirements

Type: F functional, Q quality, C constraint. All requirements are `must` unless marked otherwise. The stories column lists proposed story ids (section 7).

| id | type | requirement | acceptance criteria | priority | stories | verification |
|---|---|---|---|---|---|---|
| R1 | F | When a session opens, the system shall present the assistant as the human's default interlocutor and shall let the human talk to any agent directly. | A new session opens with the assistant as the human's point of contact. The human can address any directiva or development agent directly without the assistant blocking it. | must | US-009 | demo |
| R2 | F | When given a goal, directiva shall produce the global plan. | Given a goal, directiva produces `docs/product` and creates the tracker tasks with blocked-by / unblocks dependencies. Every task belongs to the dependency map and no task is created outside it. | must | US-002 | test, inspection |
| R3 | F | The system shall handle tasks in three blocks: creation, read, modification. | Directiva creates tasks; the cycle reads them at wave start; the cycle updates status and directiva adjusts content. A task modified after a wave is read again with its changes at the start of the next wave. | must | US-002 | test |
| R4 | F | The assistant shall start each wave and receive its report, without the human relaying anything. | Each wave is started by the assistant and its report reaches the assistant. Across a full wave, the human performs no launch, relay or copy step. | must | US-003 | demo |
| R5 | F | When a coder raises a question, the system shall surface it. | Every question a coder raises reaches the assistant and appears in the wave report. No question is closed without a recorded answer and who gave it. | must | US-003, US-004 | test |
| R6 | F | The system shall support manual and auto modes. | In manual mode, methodology and development questions reach the owner through the assistant. In auto mode, the assistant decides and records the decision and its reason in persistent memory; the owner can query those records later. | must | US-004 | test |
| R7 | F | Directiva shall adjust the plan without re-planning and shall be able to return researched answers. | Directiva can adjust tasks between waves or mid-wave without recreating the global plan (existing tasks keep their identity). Directiva can return a researched answer to a development question. | must | US-006 | test |
| R8 | F | The system shall use the methodology declared in configuration. | (updated 2026-10-26) The project config declares the methodology slot `slots.methodology.adapter` (skill name or path). The agent compiler (`gbai generate`) links it under the fixed name `gbai-methodology`, and the coder skill only instructs loading it. Changing methodology requires editing one line (directly in `gbai.yaml`, or with `gbai config set`) and rerunning `gbai generate` (Approved (owner 2026-10-26), TRD D18). | must | US-001, US-007 | test, inspection |
| R9 | F | The system shall review every PR in two layers with an isolated reviewer. | Coder-side review (for example GGA or gentle-ai receipt-driven review) is optional per methodology. Reviewer-side validation (tests, build and lint as in CI, plus an independent agent review of the PR) is a mandatory merge gate. Approved (owner 2026-10-26): the reviewer-side validation runs in deterministic reviewer scripts bundled in the `scripts/` folder of the `gbai-reviewer` skill, which `gbai generate` installs at user level, outside the repository, so PR code cannot modify the scripts that review it; a generated reviewer agent only validates the review request, invokes the script and relays its final `VERDICT` line to the coder (TRD D20). The scripts are the same everywhere; where they run (local, separate server, GitHub Actions, GitLab CI) is set per repository; the local default is single-user mode, and an isolated two-user mode is available through the reviewer placement or agent host adapter. The reviewer uses its own clone, home and cache and a clean environment, never the coder's. The verdict is commented on the PR and returned to the coder; the merge gate never merges into main. The reviewer skill documents both layers. | must | US-008 | test, inspection |
| R10 | F | When a coder reports, the cycle shall update the tracker status and link the PR to its task. | After each coder report, the task status in the tracker reflects DONE or BLOCKED. Every PR is linked to its task (for example "Closes #n" for GitHub issues, or through the tracker plugin otherwise). | must | US-008 | test |
| R11 | F | The system shall support pluggable trackers. | GitHub issues work in the first release. A new tracker is added through a provider skill plus a plugin or MCP, without changing the agents' skills. Plane, Trello and others are later. | must (gh); should (others) | US-007 | test, inspection |
| R12 | F | When configuring, the assistant shall guide the user and apply no silent defaults. | The assistant asks before configuring anything and applies no silent defaults, only suggestions. It asks where each agent runs and which tracker is used, and explains whether and how the chosen agents can communicate. | should | US-001 | demo |
| R13 | F | When a wave closes, the system shall remove the worktrees and merged branches that wave created. | At wave close, the worktrees and merged branches created by that wave are removed. Anything that could not be cleaned is listed in the wave report. | should | US-010 | test |
| R14 | F | When the human intervenes in a task, the assistant shall mediate by default and offer a direct session if mediation drags on. | Case A (default): the human talks to the assistant, which mediates with the working agent in one session. Case B: if mediation drags on, the assistant offers a direct session; on acceptance it opens a named Orca session in the task's worktree, briefs it with prior context, and the human switches to it. In both cases the assistant then resumes from the traces (commits, task, memory), coordinates with the coder and closes the task without redoing work. The first release supports Orca only. | must | US-005 | demo |
| R15 | F | While an auto-mode run is in progress, the system shall not prompt the human. | During an auto-mode run the human receives no prompts; they receive only the final answer and can query the recorded decisions. | must | US-003 | test, demo |
| R16 | F | When setup runs, the system shall identify the session owner and address them by alias. | Setup detects the git user. The project config keeps a catalog of developers (git user → alias). If no alias exists, the assistant asks once ("call you by your git name X or another name?"), stores the mapping and from then on addresses the session owner by that alias. The assistant addresses only the session owner; it may notify through a chat tool (for example Slack) or a request on the task. Company-internal organization is out of scope. | must | US-001 | test |
| R17 | F | The assistant shall keep per-user interaction preferences and, when a wave closes, ask about interaction comfort. | The assistant keeps, per user, the preferred number of interactions (many or few questions) and speaking style. At each wave close it briefly asks how comfortable the owner felt with the interactions and applies the answer in the next wave. The aim is the interactions the owner needs to feel comfortable, not the fewest. Approved (owner 2026-10-26): preferences live in two layers (see 10.1 Personalization): a stable layer in the global user file read by every compiled agent, and a learning layer of observations in the memory slot read by the assistant at session start; an observation repeated about 3 times is proposed for the stable layer. | must | US-004 | test, demo |
| R18 | F | The system shall open only the assistant's Orca terminal by default. | By default only the assistant's terminal is opened. Additional terminals open only when the user requests them. | must | US-005 | demo |
| R19 | F | When a critical question arises, the system shall turn it into a plan-change proposal and never resolve it silently. | A critical question is one whose answer would drastically change the global plan. In any mode it is never resolved silently. Before work starts the configuration defines which plan scope may change (one task, a group of tasks, or the global plan) and a boolean decides whether the global plan may change without the owner; the assistant asks the owner to set both. Plan changes never travel with code changes: each proposal gets its own issue and its own PR that change only the plan documents and tasks, and its git history explains what changed and why. Every plan change is debated before it applies: with the owner when present, otherwise with a plan-change reviewer, a fresh clean-context session that indexes the repository and must agree or propose a different improvement. A dedicated skill defines how that reviewer behaves. | must | US-006 | test, inspection |
| R20 | F | The system shall create the planning documents from its own templates and block the next planning step on a traceability gap. | The directiva skills shipped by gbai create user stories, PRD, TRD, data and config schema, app flow and implementation plan from their templates (Approved (owner 2026-10-26), TRD D19; no gbai command). Each template carries fixed questions that the assistant asks one at a time; the directiva agent fills the document; the PM recommends likely user stories. Templates follow open conventions (ISO/IEC/IEEE 29148, EARS, INVEST with Given-When-Then, arc42 with C4 and MADR-style decisions). A traceability check, run by the directiva agent as an instruction of its skill, blocks the next planning step when a story has no requirement, a requirement has no decision, task or verification, a task has no story, or the dependency graph has a cycle. | must | US-002 | test, inspection |
| R21 | Q | Approved (auto proposal, owner 2026-10-25): Secrets and credentials shall never enter agent context or generated files. | No agent prompt, generated agent file, plan document, PR, comment or report contains a secret or credential value; agents receive only references to where secrets live. Reason: derived from the reviewer isolation principle (R9) and the setup flow that embeds only adapter instructions (10.1). | should | — | inspection, test |
| R22 | Q | Approved (auto proposal, owner 2026-10-25): The reviewer shall run isolated from the coder. | Every reviewer run uses its own clone, home and cache and shares no writable state with any coder session. Reason: restates R9's isolation as a measurable quality property. | should | — | test, inspection |
| R23 | Q | Approved (auto proposal, owner 2026-10-25): In auto mode the system shall delete only resources it or its subagents created in the workflow. | Every deletion performed in auto mode targets a resource recorded as created by that workflow; a deletion of any other resource is refused and reported. Reason: derived from the auto-mode limits (10.2) and the vulcam cleanup incidents (section 2). | should | — | test |
| R24 | Q | Approved (auto proposal, owner 2026-10-25): Every task write shall be verified by re-reading it. | After each create or update of a task in the tracker, the writer re-reads the task and confirms the change; an unverified write is reported in the wave report, never silently skipped. Reason: derived from R3 and R10 and the silently skipped board moves (section 2). | should | — | test |
| R25 | Q | Approved (auto proposal, owner 2026-10-25): An interrupted wave shall be resumable without redoing finished tasks. | After an interruption, the wave resumes from the tracker state and memory traces; tasks already DONE are not reworked. Reason: derived from R14's resume-from-traces principle. | should | US-011 | test, demo |
| R26 | Q | Approved (auto proposal, owner 2026-10-25): The reviewer shall review each pushed head at most once. | For any PR, no head commit receives more than one reviewer verdict. Reason: keeps reviewer cost proportional to actual changes (R9). | should | — | test |
| R27 | Q | Approved (auto proposal, owner 2026-10-25): The plan-change reviewer shall run only when a plan change is proposed. | Plan-change reviewer sessions are started only by a plan-change proposal (R19) and their count equals the number of proposals debated without the owner. Reason: limits the cost of R19's debate to the cases that need it. | should | — | test, inspection |
| R28 | Q | Approved (auto proposal, owner 2026-10-25): A new team shall reach its first wave quickly after install. | Time from install to the first wave running with the team's tracker and methodology is measured (success metric 5). Initial target: within one working session, to be calibrated after the first waves. Reason: goal 5 (adapts to the team) only pays off if setup is short. | should | — | demo |
| R29 | Q | Approved (auto proposal, owner 2026-10-25): Every wave shall produce a report. | Each wave report lists interactions and their reasons, decisions, questions with resolver, and cleanup leftovers. Reason: derived from R4, R5, R13 and the success metrics (section 11). | should | — | test, inspection |
| R30 | Q | Approved (auto proposal, owner 2026-10-25): Every auto-mode decision shall be queryable. | The owner can list every decision taken in auto mode with its reason, wave and task. Reason: derived from R6 and R15. | should | — | test |

### 8.1 Operating rules (proposed)

These rules answer open questions 1–3. Each is **Approved (auto proposal, owner 2026-10-25)** and applies only once the owner approves it.

1. **Mid-wave task change (R7, R3).** Approved (auto proposal, owner 2026-10-25): a change directiva makes to a task a coder is working on applies when the coder finishes its current round; the coder is not interrupted. Reason: interrupting mid-round wastes partial work and risks inconsistent commits.
2. **Task state during a plan-change debate (R19).** Approved (auto proposal, owner 2026-10-25): only the affected task is BLOCKED while the proposal is debated or waits for the owner; the rest of the wave continues. Reason: goal 2 (work never stalls) applies to every task the proposal does not touch.
3. **Reviewer-to-coder rounds (R9, R10).** Approved (auto proposal, owner 2026-10-25): at most 4 fix rounds per PR; after that the task is reported BLOCKED and the question escalates to the assistant. Reason: matches the current gbai-coder rule and bounds review cost.

## 9. Quality requirements

**Approved (auto proposal, owner 2026-10-25):** quality requirements are rows R21–R30 (type Q) in section 8: security (R21–R23), reliability (R24–R25), cost (R26–R27), time-to-value (R28) and observability (R29–R30). Each is derived only from principles already stated in this PRD and the TRD; numeric targets are initial and calibrated after the first waves. Performance targets are not set in the first release. Reason: completes the PRD without inventing new product scope.

## 10. Constraints

### 10.1 Configuration standard

**Two files.**

- **Global user file**: personal preferences (alias or name, interaction level, tone and speaking style, language); this is the stable personalization layer (see Personalization below).
- **Project file `gbai.yaml`**: team and repository settings (tracker, reviewer placement and runtime, methodology slot, memory provider, notifications, direct sessions, developer catalog).

**Precedence.** For project keys, the local file wins. For personal keys, the global file always wins.

**Contracts and adapters.** Every configurable slot has a contract shipped by gbai (what the agents must be able to do) and an adapter supplied by the user or written with the assistant: a Markdown skill or document explaining how to do it with the team's tool. Tool-specific behavior is never compiled code.

**Setup flow.**

1. **Ask**: the assistant asks how the slot is handled and with what, looking at what is already installed to suggest options.
2. **Receive the adapter**: the user points to a skill or document, or writes one with the assistant.
3. **Check coverage**: the assistant checks the adapter explains every operation of the contract and asks about gaps.
4. **Build the agents**: gbai generates the runtime-native agent files with their tools and the adapter instructions embedded.
5. **Verify**: a subagent of the kind that will do the work runs a harmless operation (for example, read one task).

Switching tool means changing the adapter and rebuilding the agents (edit the YAML, or `gbai config set`, then `gbai generate`). Custom adapters live in the repository.

**Setup entry.** Approved (owner 2026-10-26): there is no setup terminal UI. The user copies a setup prompt (with a link) from the gbai documentation and pastes it into Claude Code or Pi; that agent installs gbai, runs the setup question burst above, writes the config files and runs `gbai generate`. `gbai generate` installs the per-runtime artifacts for each runtime in `generation.targets` (skills, agents, and the runtime's way of adding tools, such as MCP servers, plugins or packages) from one table of what goes where (TRD D21).

**Config writes.** Approved (owner 2026-10-26): during setup and during the work, agents write preferences and config by editing the YAML files directly; JSON Schema validation runs at the next `gbai generate`. `gbai config set` is an owner convenience at configuration time only, never a required writer.

**Approved (owner 2026-10-26):** the gbai Go CLI is used only for this initial configuration: it validates the config, compiles the agents and installs the skills (including the user-level `gbai-reviewer` skill with its scripts), the adapters and the other per-runtime artifacts. Everything during the work is done by agents with their own tools guided by skills and adapters (TRD D18). The planning templates and the traceability check (R20) live in the directiva skills (TRD D19); the reviewer runs on the scripts of the user-level `gbai-reviewer` skill (R9, TRD D20).

**Context cost.** Agents receive their adapters through the generated agent files; skills not in use cost only their one-line description. Personal preferences are a few lines read at session start.

**Documentation.** Each provider's skill is its documentation. Approved (owner 2026-10-26): there is no read-only `gbai config` command; the assistant reads the effective configuration from the files, and `gbai generate` reports schema and coverage errors (section 13, question 7).

**Personalization.** Approved (owner 2026-10-26), R16, R17:

- **Stable layer.** The global user file holds alias or name, `language`, `tone` and `interaction_level`; every compiled agent reads it.
- **Learning layer.** Observations about the user (for example "prefers short answers") are saved in the memory slot (Engram or claude-mem, per adapter) under a fixed topic, `user/preferences`; the assistant reads them at session start.
- **Promotion.** An observation repeated about 3 times is proposed for the stable layer: in manual mode the assistant asks the owner; in auto mode it applies the change to the global file and records the decision in memory (R6, R30).
- **Comfort question.** At wave close the answer adjusts `interaction_level` and `tone` (R17).
- **Scope.** Personal preferences go in the global memory scope; project preferences in the project scope.
- **Never stored.** Secrets, credentials or sensitive personal data (R21).

**Configurable slots.** Identity, mode (manual or auto), methodology slot, runtime per agent, tracker, repository host (GitHub or GitLab), reviewer placement, reviewer runtime (claude, pi), reviewer signing or marker, coder-side review, persistent memory (for example Engram), notification channel, direct-session manager (Orca), integration branch, parallelism, kept worktrees.

**Recommended options, not defaults.** Approved (owner 2026-10-26): for the methodology slot setup recommends gentle-ai ODD with RDD as the coder-side review, and for the memory slot Engram. A recommendation is offered first, with its reason, and is installed and configured only if the user accepts; otherwise the user's own adapter is used. No slot gets a silent default (R12). With RDD, validation keeps two layers: RDD on the coder side and the isolated gbai reviewer as the PR-side merge gate (R9).

### 10.2 Auto-mode limits

In auto mode the assistant:

- **May merge** only into the integration branch it was assigned, **never into main**.
- **May delete** only resources it or its subagents created in this workflow (for example test or development containers, a test database), never anything else.
- **Changes the plan only within the configured scope** (R19). The global plan changes only when the owner's boolean allows it, and only after the plan-change reviewer agreed; otherwise the proposal waits for the owner. Plan changes always go in their own issue and PR, never in a code PR.

## 11. Success metrics

| metric | how it is measured |
|---|---|
| Interactions per wave and their reasons | Counted in each wave report. This is an input to learn the owner's comfort, not a reduction target; the verdict is qualitative: the owner feels comfortable. |
| Share of waves closed by the assistant without the human launching or relaying anything | Waves closed without human launch or relay steps, divided by total waves. |
| Questions recorded and resolved, and by whom | Count of questions raised, resolved, and resolver (directiva, assistant, owner). |
| Plan adjustments per wave without redoing the global plan | Count of task adjustments per wave; any global re-plan is reported separately. |
| Time from install to the first wave running with the team's tracker and methodology | Elapsed time from install to the start of the first wave. |

**Targets.** The first release measures without targets. After 3 to 5 real waves the baseline is known and the owner sets targets; from then on each wave report compares against them.

## 12. Later

Out of the first release.

- Plane, Trello and other tracker providers.
- Reviewer on a separate server, GitHub Actions or GitLab CI (first release: local).
- Additional reviewer runtimes.
- Direct sessions with other terminal managers (for example herdr).
- Target values for the success metrics.

## 13. Open questions

| # | question | status |
|---|---|---|
| 1 | When directiva adjusts a task mid-wave, how does the change reach a coder already working on that task: is the coder interrupted, or does the change apply only after it reports? | answered — approved by owner 2026-10-25 (8.1 Operating rules, rule 1) |
| 2 | While a plan-change proposal (R19) is being debated or waits for the owner, does the affected task stay BLOCKED while the rest of the wave continues? | answered — approved by owner 2026-10-25 (8.1 Operating rules, rule 2) |
| 3 | How many reviewer-to-coder rounds are allowed on a PR before the task is reported BLOCKED and the question escalates to the assistant? | answered — approved by owner 2026-10-25 (8.1 Operating rules, rule 3) |
| 4 | What documented evidence supports the problem (sources, observations, data)? | answered — approved by owner 2026-10-25 (2. Problem, Evidence) |
| 5 | What is the appetite (time and effort budget) for the first release? | answered — approved by owner 2026-10-25 (5. Appetite) |
| 6 | What are the quality requirements (performance, security, reliability, cost) for the first release? | answered — approved by owner 2026-10-25 (R21–R30, 9. Quality requirements) |
| 7 | Does the read-only `gbai config [--json]` command stay in the agent compiler (TRD D18), or is the effective configuration read by the assistant from the files? | answered — Approved (owner 2026-10-26): removed; the assistant reads the files and `gbai generate` reports errors (10.1 Documentation) |
| 8 | Where does the compiler install the reviewer scripts in the repository? Proposed: `.gbai/reviewer/` (TRD D20, DB_SCHEMA). | answered — Approved (owner 2026-10-26): not in the repository; the scripts live in the `scripts/` folder of the `gbai-reviewer` skill, installed at user level so PR code cannot modify them (R9, TRD D20) |
| 9 | Does the terminal UI (TUI) for the initial configuration stay in scope, or is setup done only with the assistant plus `gbai config set` and `gbai generate` (UX_UI section 5, milestone M4)? | answered — Approved (owner 2026-10-26): the TUI is removed; setup starts from a prompt pasted into Claude Code or Pi (10.1 Setup entry, TRD D21) |

## 14. Glossary

| term | meaning |
|---|---|
| Owner | The session owner, identified by git user and alias; makes product decisions and validates. |
| Assistant | The default point of contact for the whole flow; starts waves, receives reports, routes questions. Not part of directiva. |
| Directiva | The direction agents: pm, dev-lead, db-master and frontend. They write `docs/product`, create and adjust tasks, research answers. |
| gbai-cycle | The agent that runs one wave: reads tasks, launches coders, updates the tracker, reports to the assistant. |
| Coder | A native subagent of the cycle that implements one wave task and opens its PR. |
| Reviewer | An independent, isolated reviewer: deterministic scripts bundled in the user-level `gbai-reviewer` skill validate each PR and comment the verdict; a generated reviewer agent invokes them and relays the verdict. |
| Plan-change reviewer | A fresh clean-context session that indexes the repository and must agree with a plan change or propose a different improvement. |
| Wave | The set of tasks whose dependencies are done. |
| Work route map | The global plan: every task with its blocked-by / unblocks dependencies. |
| Critical question | A question whose answer would drastically change the global plan; it becomes a plan-change proposal. |
| Manual / auto mode | Manual: questions reach the owner through the assistant. Auto: the assistant decides and records each decision with its reason. |
| Slot | A configurable part of the system (for example tracker, memory, reviewer placement). |
| Contract | What the agents must be able to do for a slot; shipped by gbai. |
| Adapter | A Markdown skill or document in the repository explaining how to fulfil a contract with the team's tool. |
| Agent builder | The generic gbai code (the agent compiler, the only Go part) that validates the configuration, generates runtime-native agent files and installs the per-runtime artifacts (skills, including the reviewer skill and its scripts, adapters, and tool registrations) from one table of what goes where. |
| Setup prompt | A prompt with a link, published in the gbai documentation, that the user pastes into Claude Code or Pi to have that agent install and configure gbai. |
| Stable layer / learning layer | The two personalization layers: preferences in the global user file, read by every agent; observations in the memory slot under `user/preferences`, read by the assistant at session start. |
| Direct session | An extra Orca terminal in a task's worktree, opened only on the user's request. |
| Integration branch | The branch the assistant may merge into in auto mode; never main. |
