# Intent 0028 Deterministic hooks as locks

| field | value |
|---|---|
| intent id | 0028-deterministic-hooks-as-locks |
| status | Draft. Open questions closed 2026-11-13; the spec has not started (the owner ordered the intents first, then the specs). The owner accepts the spec, not this intent (owner decision D-6, 2026-11-13) |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed; intent issue linked under it, as [#758](https://github.com/testeos-z/gabo-ai/issues/758) was) |
| intent issue | [#765 I0028: Deterministic hooks as locks](https://github.com/testeos-z/gabo-ai/issues/765) |
| date | 2026-11-13 (revised 2026-11-13 after the owner's decisions, see the Decided table) |

## Problem

The owner's decision of 2026-11-13, after reviewing the gaps between gabo and Anthropic's playbook, is to put a set of deterministic locks on what agents do. Owner, verbatim, relayed by the owner's local assistant: "Me encaja ese corte, me parece bien. Prepara los documentos de intents y los issues" [I agree with that cut, it looks good to me. Prepare the intent documents and the issues]. The cut is "B, deterministic hooks ("locks")". The rules live in gabo (Go); each engine's hook is a thin relay that reports the tool call and applies gabo's answer (TRD section 2, `docs/product/TRD.md:44`). The owner cites this as "D9"; TRD D9 is another decision ("Direct session", `docs/product/TRD.md:591`), so this intent cites the thin-layer rule of TRD section 2 instead.

Evidence, checked on 2026-11-13 at `origin/dev` `63b36df`:

- **No engine hook judges a shell command.** Claude Code runs its hooks on `Write|Edit` only (`plugins/claude-code/hooks/hooks.json:6`, `:17`). The OpenCode stage gate covers `edit`, `write` and `patch` (`apps/cli/internal/skills/gabo-runtime-opencode/plugins/gabo-stage-gate.js:18`) and its header says a shell call is not gated (line 15). The Pi extension covers `write` and `edit` only (`packages/pi/extensions/gabo.ts:137`). Codex is the same as Claude Code on `apply_patch` (per the spec 0027 evidence base, `plugins/codex/hooks/hooks.codex.json:4-25`, not re-read here).
- **The coder rules are text.** Never push to `main` or `dev`, never force-push, delete with `-d` only: `.claude/skills/gabo-repository-host-github/SKILL.md:15-17`. Never merge, approve or set commit statuses: `apps/cli/internal/skills/gabo-coder/SKILL.md:15`. Scratch in `mktemp -d`: `apps/cli/internal/skills/gabo-coder/SKILL.md:29`. Nothing stops a shell call that breaks these rules.
- **Release is by hand, but nothing enforces it.** The reviewer merges clean heads into `dev` only (`gabo.yaml:49-52`, which cites TRD D13). A human sets the version in three manifests and runs `claude plugin tag` (`plugins/claude-code/README.md:45-56`, per the spec 0027 evidence). A coder shell call of `git tag`, `npm publish` or `gh release create` is not judged by any hook.
- **Reserved paths are checked only at merge.** `gabo.yaml:53` lists `review.human_only_paths` (`.github/`, `AGENTS.md`, `CLAUDE.md`, `gabo.yaml`, `.claude/`, `.codex/`, `.agents/`). The reviewer's merge script stops a merge that touches them (`merge-head.sh`, per the spec 0027 evidence). A write in a coder worktree is not stopped. The Claude Code write hook is silent for most of these paths: `plugins/claude-code/scripts/gabo-hook-filter.sh:87-90` judges only `gabo.yaml`, `.gabo/*`, `.claude/agents/*`, `.claude/skills/*` and `docs/product/*`.
- **Credentials: gabo ships no list.** The only list of sensitive paths is in the owner's personal Claude Code settings, `~/.claude/settings.json`, under `permissions.deny` (24 entries, checked 2026-11-13). It denies `Read` and `Edit` for `.env`, `.env.*`, `~/.ssh/**`, `~/.credentials/**`, `~/Library/Keychains/**`, `~/.aws/credentials`, `~/.config/gh/hosts.yml`, `**/*.pem`, `**/*.key` and `**/secrets/*`, and denies three `rm -rf` forms. Values are not copied here. It is a personal permission setting, not a gabo file, so it does not reach a project. The agents run with the owner's `gh` login (`gabo.yaml`, reviewer comment: "the scripts run as the owner with the owner's gh login").
- **Test files are not detected.** No script, extension or check in `plugins/`, `packages/pi/extensions/` or `apps/cli/internal/slices/check/` detects an edit to an existing test file (searched 2026-11-13). Playbook, byte-checked: "A hook that blocks edits to test files during a fix task does this." The owner changed this lock to a warning (decision D-4).
- **No per-hook switch exists.** No plugin script, the OpenCode plugin or the Pi extension reads a setting that turns one hook off (searched 2026-11-13). The only existing off switch is gentle-ai's review mode, which the owner controls and the coders never toggle (spec 0027 evidence).
- **Pi already has guardrails, from gentle-pi.** The installed gentle-pi 4.7.0 lists `gitPush`, `gitRebase`, `gitBranchDeleteForce`, `npmPublish` and `piRemove` as guarded commands (`~/.pi/agent/npm/node_modules/gentle-pi/extensions/gentle-ai.ts:1725-1729`), read from `runtime-guardrails.json` (`:1921-1922`). They cover Pi only and are not in the gentle-ai binary.
- **Engine events are unproven for subagents.** Claude Code documents hooks inside subagents and a `SubagentStop` event (spec 0027 evidence table, source https://code.claude.com/docs/en/hooks). Codex, OpenCode and Pi are unknown for tool calls inside a subagent. Nothing has been probed.

Playbook evidence, Anthropic, "The AI-native SDLC playbook", Louis Claxton, 2026-08-21, https://claude.com/resources/articles/the-ai-native-sdlc-playbook. Byte-checked on 2026-11-13 against the page text (tags stripped; the extraction puts a space before some punctuation, which the check ignores):

- "Hooks are the approval gates."
- "A hook can also ask, pausing the action until a specific person approves, which is what release gating needs."
- "The production deploy hook blocks the release until a named release manager authorizes it."
- "Block edits to protected paths such as generated classes or a frozen package;"
- "A block should explain itself, so when a hook stops an action the reason and the route to approval appear in Claude's output."
- "Consider the above a starting point to tailor, rather than a recommendation to copy." The example it follows denies `Read(.env*)`, `Read(./secrets/**)`, `WebFetch`, `Bash(curl *)` and `Bash(wget *)`, and its sandbox block names `~/.ssh`, `~/.aws/credentials` and `GITHUB_TOKEN`.
- "The steps are modular and organizations may choose to prioritize transforming different stages at different times based on their unique needs." The relay of 2026-11-13 wrote "These steps are modular", which the page does not contain; this is the exact text.

The playbook's hook examples name no git or release commands. The shell commands in this intent come from gabo's own coder rules, not from the playbook.

## Owner decisions, 2026-11-13

Relayed by the owner's local assistant (laptop), on the owner's behalf. The owner answered by voice, question by question, on 2026-11-13. Owner words are quoted in the language the owner used, with an English gloss in brackets; everything else in this section and in the Decided table is the assistant's summary of the owner's answers, not an owner quote.

- General criterion, verbatim: "si está en el documento de Anthropic, úsalo. Vamos a cumplir ese documento estrictamente" [if it is in Anthropic's document, use it; we will follow that document strictly]. Modular reading, verbatim: "Me parece bien, lo dejamos así como si fuera modular" [fine, we leave it as if it were modular]. What the playbook prescribes is in; the owner's choices below are the exceptions and additions.
- Questions the owner did not name, verbatim: "aprobado, siga. Tómala a las técnicas, toma las recomendadas" [approved, go ahead. Take it with the techniques, take the recommended ones]. Those questions take their Proposed (auto) option, adjusted where it clashed with a decision in the Decided table.
- Warn state, verbatim: "Avisar me parece bien" [warn sounds fine to me]. The owner said this about the light lock. Assumption for the spec: the light lock is the test-file lock, which D-4 already keeps at warn.
- Specs wait, verbatim: "Quiero que solo terminen los planteamientos de los intents, y luego vamos con los specs" [I want the intents' proposals to finish first, and then we go to the specs].
- D-1 CI on PRs stays out.
- D-2 The reviewer keeps merging only into `dev`; a person approves the spec and `main`. Agents reach only `dev`; the owner does `main` and the versions.
- D-4 The test-file hook warns and never blocks.
- D-6 The owner accepts the spec, not the intent nor the plan.
- D-1, D-2, D-4 and D-6 are the owner's numbered choices as relayed by the assistant; the owner's own wording of them is not in the record, so they are not quoted verbatim.

## Who is affected

- The owner, who releases, owns `main` and the versions, and must be able to switch each lock off and on.
- Every coder in every wave. Each wave runs coders in their own worktrees, and each one can run any shell command today.
- The assistant, which relays each block and warning in the wave report.
- The reviewer, which remains the merge gate and does not run inside a coder session.

## Scope

In scope, one lock each, decided in gabo and relayed by each engine:

1. **Shell guard.** A shell call that pushes to `main` or `dev`, force-pushes, deletes a branch by force, removes a worktree by force, deletes by glob, or merges a pull request into `main` is blocked with its rule and route. Further commands: decided 14.
2. **Release gate.** Agents work only up to `dev`. A version, a tag, a publish, a release, and any merge into `main` are the owner's steps, taken outside agent sessions (D-2). The hook is the technical lock that stops an agent from creating one.
3. **Reserved paths at write time.** Each entry of `review.human_only_paths` is blocked for a coder write, not only at merge.
4. **Base deny list.** A list that gabo ships, adjustable per project, stored in the repository and written in each engine's own format, in paths reserved to a person. Owner, 2026-11-13: "esto se ajustará al repositorio, ¿correcto" [this will be adjusted to the repository, correct?], then "Listo, queda así" [done, it stays like this]. It starts from the categories named in the Problem (`.env` and `.env.*`, `*.pem`, `*.key`, `secrets/`, `~/.ssh`, `~/.aws/credentials`, `~/.config/gh/hosts.yml`, the macOS keychain folder, `~/.credentials`), the same kind of entries as the playbook's example. A coder's read or write of a listed path is blocked. The exact entries are part of the spec.
5. **Test files, warn state.** An edit to an existing test file is detected, recorded in the wave trace, and never blocked (D-4).
6. **Four states and one switch per lock**, the five above. Each lock is off, warn, ask or block (decided 6 and 10). Each lock has its own switch, and one switch turns all of them off together. Only a person changes a switch. The switches live in `gabo.yaml` (decided 9).
7. **First task: a probe** that shows, on each engine (`claude`, `codex`, `pi`, `opencode`), whether a hook fires inside a coder subagent, and whether the engine's yes/no ask dialog appears when the session runs with permissions skipped (decided 17).

## Non-goals

- The folder boundary (former hook 5 of intent 0027). The owner decided it is out: "La carpeta es la única que dejaremos fuera... cada coder debe trabajar en su propio work tree, entonces no veo el choque" (relayed 2026-11-13).
- CI on PRs (D-1) and package publishing. Both are out of all four intents of this batch.
- The coder done gate and the methodology-report gate (hooks 3 and 4 of intent 0027). They stay in 0027 unless the owner moves them.
- Replacing the reviewer's merge gate (PRD R9). Hooks never merge, approve or set statuses.
- Changing the RDD switch or the gentle-pi guardrails. Coders never toggle RDD. gentle-pi keeps its Pi list.
- Not in this batch, by the owner's modular choice: the playbook's sandbox and managed-settings keys, the fresh-context verifier, metrics and traces, and Design-stage security skills.

## Constraints

- Thin-layer rule (TRD section 2, `docs/product/TRD.md:44`): rules in Go, engine hooks relay only.
- PRD R31 (`docs/product/PRD.md:170`): the CLI offers four commands and one additive hook mode. A new hook mode or any new command needs the owner's amendment. The owner gave that permission on 2026-11-13 for `gabo check --pre-command` (decided 10); the amendment of PRD R31 and TRD 9.4 is written with the spec, not in this intent. Intent 0027's spec question 12 uses the same mode choice and follows this decision (0027 is not edited here).
- PRD R32 (`docs/product/PRD.md:171`): engine config is written at project level only, never in the user's global engine config. Hooks ship in each engine's plugin or package.
- PRD R15 (`docs/product/PRD.md:154`, "During an auto-mode run the human receives no prompts, only the final answer."): R15 is not amended. In auto mode the `ask` state is not shown; it behaves as `block` and is recorded (decided 6). Each lock blocks or allows and records; in auto mode it never pauses for approval.
- PRD R9: the reviewer is the only merge gate. These locks add outside checks and never merge.
- Intents 0001 (S7), 0003 (S5, S6) and 0004 (S10) keep their current write-only scopes; the new locks extend them for their own rules only.
- The agents share the owner's `gh` login, so a rule the owner's account can bypass does not bind the agents.
- Intent 0027's hooks 1 and 2 (its spec S16 and S17) and its questions 12, 13, 15, 16, 17 and 20 moved here (0027 spec lines 21 and 202). Its question 17 proposed a release record file; Decided row 2 settles it, and the moved text stays in git. This intent does not edit 0027.
- Names only: this document never copies a secret value or a credential path's contents.
- **Premise check on the switches (decided 9).** Today the Claude Code write hook runs `gabo check --pre-write` on `gabo.yaml` (`plugins/claude-code/scripts/gabo-hook-filter.sh:88`), and that mode blocks only a `plan.md` whose spec is not accepted (`apps/cli/internal/slices/check/prewrite.go:21-25`). So a coder write of `gabo.yaml` is not blocked today, and no shell guard exists. The reviewer does stop a merge that touches `gabo.yaml`: the built-in list in `common.sh:175` and `is_human_only` (`common.sh:257`) mark it human-only, and `gabo.yaml:53` lists it. Until locks 1 and 3 are built, the switches are protected only at merge. Lock 3 and the shell guard are the reserved-path protection the owner relied on; the spec must make them a precondition for the switches to be binding.

## Decided (owner, 2026-11-13)

| # | was question | decision | basis |
|---|---|---|---|
| 1 | 2, block or warn | Block: shell guard, release gate, reserved paths, base deny list. Warn only: test files, never blocked. | D-4 (test files warn, never block). The playbook prescribes blocks: "Hooks are the approval gates."; "Block edits to protected paths such as generated classes or a frozen package;". The test-file block of the playbook is changed by D-4, an owner choice. |
| 2 | 4, how release authorization is recorded | Not recorded in a file. The owner runs versions, tags, publishes, releases and merges into `main` outside agent sessions; the release gate blocks them inside agent sessions. Agents reach only `dev`. | D-2. The playbook: "The production deploy hook blocks the release until a named release manager authorizes it." A record file in the repository is a file an agent could forge. |
| 3 | 5, which credential paths, and where a project extends them | Base deny list that gabo ships (scope 4); extended per project in the repository, in each engine's format, in paths reserved to a person. | Owner: "esto se ajustará al repositorio, ¿correcto" / "Listo, queda así". The playbook: "Consider the above a starting point to tailor, rather than a recommendation to copy." |
| 4 | 16, which reserved paths | Every entry of `review.human_only_paths`, for any coder write. | Playbook: "lists the human approval gates that must survive, such as change management sign-off, release authorization, and edits to protected paths." |
| 5 | 1 a, g, h, part | No agent pushes to `main` or `dev`; no agent merges a pull request into `main`; no agent runs a publish, a plugin tag or a release command. | D-2. |
| 6 | 2, the ask state and PRD R15 | Each lock has four states: off, warn, ask and block. Ask shows the engine's yes/no dialog, which a person answers and the agent cannot. In auto mode the dialog is not shown: ask behaves as block and is recorded. R15 is not amended. The light lock stays at warn. The probe tests per engine whether the dialog appears with permissions skipped (decided 17). | Relayed summary: ask is the engine's yes/no dialog that a person answers and the agent cannot; owner: "Avisar me parece bien" [warn sounds fine to me] for the light lock. PRD R15 (`docs/product/PRD.md:154`). The recommended option for ask, "no ask hook", is replaced by this decision. |
| 7 | 3, an engine or subagent without the event | Recommended option: the probe decides per engine; where a lock does not fire inside a coder subagent, the outcome is `not run: <reason>` in the wave trace and the owner is told. The text rule stays as written. The wave trace shows a visible `not run` line. | Relayed summary of the owner's answer: the recommended option, with the rule kept as text and a visible `not run` line. |
| 8 | 4, fail open or fail closed | Gabo not on the PATH: fail open with a visible warning only. Gabo runs and fails: the blocking locks block; the other locks report. | Relayed summary of the owner's answer: distinguish the two cases; if gabo is not installed the outcome is only a message; if it runs and fails, the blocking locks block. Equals the recommended option. |
| 9 | 5, where the switches live and who changes them | The switches live in `gabo.yaml`, the repository's gabo config file, not in the owner's user config. Each hook reads its switch on every call. The shell guard and the reserved-path hook protect that file (see the premise check in Constraints; they are not built yet). Residual risk, stated by the assistant: until locks 1 and 3 exist, a coder write of `gabo.yaml` is not blocked, and the reviewer stops only the merge. | Owner: "la configuración de Gabo se guarda en archivo de configuración" [Gabo's configuration is stored in the configuration file]. The recommended option (user config, outside the repository) is replaced. `gabo.yaml:53` and `common.sh:175` list `gabo.yaml` as reserved. |
| 10 | 9, the check surface, and 10, the switch granularity | Check surface: a second additive mode, `gabo check --pre-command`, which reads tool-call JSON on stdin like `--pre-write`. The owner gives permission to amend PRD R31 and TRD 9.4; the four commands stay four. This PR edits neither PRD nor TRD. Switch granularity: one switch per lock (five locks, each four-state), one switch that turns all of them off together, and no per-rule switch of the shell guard. Only a person changes a switch. | Relayed summary of the owner's answer: one switch per lock; everything can be switched off, per lock or together, only by a person. Recommended options for the mode and granularity. The amendment is written with the spec. |
| 11 | 11, outcomes in the wave trace | Recorded in `gabo/trace/<run id>`: `blocked`, `warned`, `not run`, and `asked` with the person's answer. An ask in auto mode is recorded as `blocked`. Every outcome except `allowed` is one line with the lock, task, engine and rule. | Recommended option, with the ask state added by decision 6. |
| 12 | 12, gentle-pi's Pi guardrails | Both apply; a call is blocked when either blocks it. | Owner approved the recommended option ("toma las recomendadas"). |
| 13 | 13, which sessions the PR, worktree and force rules judge | The release and `main` rules judge every agent session. The PR, worktree and force rules judge coder sessions only. The reviewer's merge into `dev` is exempt. | Owner approved the recommended option. |
| 14 | 1, further shell commands | Deny each of: `git push` naming `master`; `git push` with `--force`, `-f`, `--force-with-lease`, `--mirror` or a refspec starting with `+`; creating a tag (`git tag <name>`, `git push --tags`); `git branch -D` and `git branch --delete --force`; `git worktree remove` with `--force` or `-f`; `rm` with a recursive flag on a glob (`*`, `?`, `[`) or on a path under `*-worktrees/`. A literal `mktemp -d` path stays allowed. | Owner approved the recommended option. |
| 15 | 6, which files are test files | `*_test.go`, `*.test.*`, `*_test.*`, and any file under `test/` or `tests/`; an existing file is one tracked at the worktree's base commit. Warn state (decision 1). Each warning is one JSON line in the run's `mktemp -d` scratch, which the cycle copies to `gabo/trace/<run id>`. | Owner approved the recommended option; D-4. |
| 16 | 7, how far the shell guard normalises | One level only: `sh -c '...'`, `bash -c '...'` and `git -C <dir>` are unwrapped before matching; aliases, `eval` and `xargs` are not. | Owner approved the recommended option. |
| 17 | 8, what the probe proves, and on which engines | On `claude`, `codex`, `pi` and `opencode`: (a) a shell pre-command hook fires inside a coder subagent and can deny a shell call; (b) the subagent's tool calls fire the same hook with an agent or session id; (c) the hand-off event exists and whether it can block. Added by the owner's item: whether the engine's yes/no ask dialog appears when the session runs with permissions skipped (bypass). | Owner approved the recommended option. The added item is relayed by the assistant, not an owner quote: test per engine whether the ask dialog appears when the session runs with permissions skipped. |

## Open questions

None. Questions 1 to 13 of the previous draft are closed. Each one has a row in the Decided table with the `was question` column that gives its old number; the mapping is 1 to row 14, 2 to row 6, 3 to row 7, 4 to row 8, 5 to row 9, 6 to row 15, 7 to row 16, 8 to row 17, 9 and 10 to row 10, 11 to row 11, 12 to row 12, 13 to row 13. Questions the owner did not name took their Proposed (auto) option, as relayed, adjusted where they clashed with a decision.

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed) |
| intent issue | [#765 I0028: Deterministic hooks as locks](https://github.com/testeos-z/gabo-ai/issues/765) |
| sibling intent, C | [0029 Branch protection on main and dev](../0029-branch-protection-on-main-and-dev/intent.md), [#766](https://github.com/testeos-z/gabo-ai/issues/766). 0029 depends on this intent: its admin exception leaves agents to the local locks here (0029 decision 5). |
| sibling intent, D | [0030 Agent evals when skills change](../0030-agent-evals-when-skills-change/intent.md), [#767](https://github.com/testeos-z/gabo-ai/issues/767) |
| sibling intent, A | [0027 Coders and product QA share one definition of done](../0027-coders-follow-gentle-ai-odd-and-rdd/intent.md), [#758](https://github.com/testeos-z/gabo-ai/issues/758) (its hooks 1 and 2 moved here; its hooks 3 and 4 stay there) |
| related intents | [0001](../0001-claude-code-plugin/intent.md), [0003](../0003-pi-package/intent.md), [0004](../0004-opencode-runtime/intent.md) (current hook scopes) |
| PRD | [PRD](../../PRD.md) (R9, R15, R31, R32) |
| TRD | [TRD](../../TRD.md) section 2 (thin layer, line 44), section 9.1 (`repository_host`, line 373), D13 (line 595) |
| spec | none yet; the owner ordered the specs after the intents (relayed 2026-11-13) |
| plan | none yet |
| sources | [Anthropic, The AI-native SDLC playbook](https://claude.com/resources/articles/the-ai-native-sdlc-playbook), Louis Claxton, 2026-08-21 (quotes byte-checked 2026-11-13); [Claude Code hooks](https://code.claude.com/docs/en/hooks); [Codex hooks](https://learn.chatgpt.com/docs/hooks); [OpenCode plugins](https://opencode.ai/docs/plugins); gentle-pi 4.7.0 at `~/.pi/agent/npm/node_modules/gentle-pi/extensions/gentle-ai.ts:1725-1729`; `origin/dev` at `63b36df` for every repository path cited; `origin/dev` at `23552d5` for the premise check (`gabo.yaml:53`, `common.sh:175`, `prewrite.go:21-25`) |
