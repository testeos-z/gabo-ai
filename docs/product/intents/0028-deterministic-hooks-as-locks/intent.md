# Intent 0028 Deterministic hooks as locks

| field | value |
|---|---|
| intent id | 0028-deterministic-hooks-as-locks |
| status | Draft. The owner accepts the spec, not this intent (owner decision D-6, 2026-11-13) |
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

- General criterion, verbatim: "si está en el documento de Anthropic, úsalo. Vamos a cumplir ese documento estrictamente" [if it is in Anthropic's document, use it; we will follow that document strictly]. Modular reading, verbatim: "Me parece bien, lo dejamos así como si fuera modular" [fine, we leave it as if it were modular]. What the playbook prescribes is in; the owner's choices below are the exceptions and additions.
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

1. **Shell guard.** A shell call that pushes to `main` or `dev`, force-pushes, deletes a branch by force, removes a worktree by force, deletes by glob, or merges a pull request into `main` is blocked with its rule and route. Further commands: open question 1.
2. **Release gate.** Agents work only up to `dev`. A version, a tag, a publish, a release, and any merge into `main` are the owner's steps, taken outside agent sessions (D-2). The hook is the technical lock that stops an agent from creating one.
3. **Reserved paths at write time.** Each entry of `review.human_only_paths` is blocked for a coder write, not only at merge.
4. **Base deny list.** A list that gabo ships, adjustable per project, stored in the repository and written in each engine's own format, in paths reserved to a person. Owner, 2026-11-13: "esto se ajustará al repositorio, ¿correcto" [this will be adjusted to the repository, correct?], then "Listo, queda así" [done, it stays like this]. It starts from the categories named in the Problem (`.env` and `.env.*`, `*.pem`, `*.key`, `secrets/`, `~/.ssh`, `~/.aws/credentials`, `~/.config/gh/hosts.yml`, the macOS keychain folder, `~/.credentials`), the same kind of entries as the playbook's example. A coder's read or write of a listed path is blocked. The exact entries are part of the spec.
5. **Test files, warn mode.** An edit to an existing test file is detected, recorded in the wave trace, and never blocked (D-4).
6. **One switch per lock**, the five above (open question 5).
7. **First task: a probe** that shows, on each engine (`claude`, `codex`, `pi`, `opencode`), whether a hook fires inside a coder subagent (open question 8).

## Non-goals

- The folder boundary (former hook 5 of intent 0027). The owner decided it is out: "La carpeta es la única que dejaremos fuera... cada coder debe trabajar en su propio work tree, entonces no veo el choque" (relayed 2026-11-13).
- CI on PRs (D-1) and package publishing. Both are out of all four intents of this batch.
- The coder done gate and the methodology-report gate (hooks 3 and 4 of intent 0027). They stay in 0027 unless the owner moves them.
- Replacing the reviewer's merge gate (PRD R9). Hooks never merge, approve or set statuses.
- Changing the RDD switch or the gentle-pi guardrails. Coders never toggle RDD. gentle-pi keeps its Pi list.
- Not in this batch, by the owner's modular choice: the playbook's sandbox and managed-settings keys, the fresh-context verifier, metrics and traces, and Design-stage security skills.

## Constraints

- Thin-layer rule (TRD section 2, `docs/product/TRD.md:44`): rules in Go, engine hooks relay only.
- PRD R31 (`docs/product/PRD.md:170`): the CLI offers four commands and one additive hook mode. A new hook mode or any new command needs the owner's amendment. The mode choice is shared with intent 0027 (its spec question 12); decide it once for both.
- PRD R32 (`docs/product/PRD.md:171`): engine config is written at project level only, never in the user's global engine config. Hooks ship in each engine's plugin or package.
- PRD R15: in auto mode no hook asks a human. Each lock blocks or allows and records; it never pauses for approval (open question 2).
- PRD R9: the reviewer is the only merge gate. These locks add outside checks and never merge.
- Intents 0001 (S7), 0003 (S5, S6) and 0004 (S10) keep their current write-only scopes; the new locks extend them for their own rules only.
- The agents share the owner's `gh` login, so a rule the owner's account can bypass does not bind the agents.
- Intent 0027's hooks 1 and 2 (its spec S16 and S17) and its questions 12, 13, 15, 16, 17 and 20 moved here (0027 spec lines 21 and 202). Its question 17 proposed a release record file; Decided row 2 settles it, and the moved text stays in git. This intent does not edit 0027.
- Names only: this document never copies a secret value or a credential path's contents.

## Decided (owner, 2026-11-13)

| # | was question | decision | basis |
|---|---|---|---|
| 1 | 2, block or warn | Block: shell guard, release gate, reserved paths, base deny list. Warn only: test files, never blocked. | D-4 (test files warn, never block). The playbook prescribes blocks: "Hooks are the approval gates."; "Block edits to protected paths such as generated classes or a frozen package;". The test-file block of the playbook is changed by D-4, an owner choice. |
| 2 | 4, how release authorization is recorded | Not recorded in a file. The owner runs versions, tags, publishes, releases and merges into `main` outside agent sessions; the release gate blocks them inside agent sessions. Agents reach only `dev`. | D-2. The playbook: "The production deploy hook blocks the release until a named release manager authorizes it." A record file in the repository is a file an agent could forge. |
| 3 | 5, which credential paths, and where a project extends them | Base deny list that gabo ships (scope 4); extended per project in the repository, in each engine's format, in paths reserved to a person. | Owner: "esto se ajustará al repositorio, ¿correcto" / "Listo, queda así". The playbook: "Consider the above a starting point to tailor, rather than a recommendation to copy." |
| 4 | 16, which reserved paths | Every entry of `review.human_only_paths`, for any coder write. | Playbook: "lists the human approval gates that must survive, such as change management sign-off, release authorization, and edits to protected paths." |
| 5 | 1 a, g, h, part | No agent pushes to `main` or `dev`; no agent merges a pull request into `main`; no agent runs a publish, a plugin tag or a release command. | D-2. |

## Open questions

Each question has a **Proposed (auto)** option with a one-line reason. None is accepted. Answers go into the spec, which the owner accepts (D-6).

| # | question | status |
|---|---|---|
| 1 | Shell guard: which further commands are blocked, beyond the decided release and `main` rules? | open. **Proposed (auto): deny each of these: `git push` naming `master` (`main` and `dev` are decided); `git push` with `--force`, `-f`, `--force-with-lease`, `--mirror` or a refspec starting with `+`; creating a tag (`git tag <name>`, `git push --tags`); `git branch -D` and `git branch --delete --force`; `git worktree remove` with `--force` or `-f`; `rm` with a recursive flag on a glob (`*`, `?`, `[`) or on a path under `*-worktrees/`. A literal `mktemp -d` path stays allowed.** Reason: each one is already a coder rule, and an allow-list stops a wave on the first new tool. Option: an allow-list. |
| 2 | The playbook's "ask" hook for release gating (`A hook can also ask, pausing the action until a specific person approves, which is what release gating needs.`) conflicts with PRD R15. Which applies? | open. **Proposed (auto): no ask hook. PRD R15 stands, and the owner's release steps run outside agent sessions (decided 2).** Reason: an ask pauses an auto-mode wave for a human, which R15 forbids. Option: allow an ask hook for release in manual mode only, which needs an R15 amendment. |
| 3 | What happens on an engine, or in a subagent, without the event? | open. **Proposed (auto): the probe decides per engine; where a lock does not fire inside a coder subagent, its outcome is `not run: <reason>` in the wave trace and the owner is told; the text rule stays.** Reason: a silent gap is the failure the owner wants removed. Option: refuse that engine as a coder runtime until the event exists (blocks OpenCode coders today). |
| 4 | Fail open or fail closed when gabo is missing or fails? | open. **Proposed (auto): fail open with a visible warning when gabo is not on the PATH; fail closed for the blocking locks when gabo runs and fails.** Reason: a broken gabo must not silently allow a push. Option: fail closed everywhere, so a machine without gabo cannot run a coder. |
| 5 | Where do the switches live, and who changes them? | open. **Proposed (auto): five switches, one per lock; they live in the owner's user config (`~/.config/gabo/config.yaml`, keyed by repository), outside every repository and worktree; the shell guard denies any agent command that names that file; each hook reads the switch on every call.** Reason: agents share the owner's login and shell, so a switch inside the repository is one edit away. Residual risk: a pattern miss lets an agent write the file. Option: in `gabo.yaml`, which blocks merges but not worktree writes. |
| 6 | Which files are test files, and where does the warning go? | open. **Proposed (auto): `*_test.go`, `*.test.*`, `*_test.*`, and any file under `test/` or `tests/`; an existing file is one tracked at the worktree's base commit; each warning is one JSON line in the run's `mktemp -d` scratch, which the cycle copies to `gabo/trace/<run id>`.** Reason: a hook cannot call memory, and the owner wants the record in the wave trace. Option: only `*_test.go` and `*.test.sh`. |
| 7 | How far does the shell guard normalise a command? | open. **Proposed (auto): one level only: `sh -c '...'`, `bash -c '...'` and `git -C <dir>` are unwrapped before matching; aliases, `eval` and `xargs` are not.** Reason: each new level is a new parser. Option: a full shell parser in Go, which adds a dependency (Task Size item 5, a high-risk item). |
| 8 | What must the probe prove, and on which engines? | open. **Proposed (auto): on `claude`, `codex`, `pi` and `opencode`: (a) a shell pre-command hook fires inside a coder subagent and can deny a shell call; (b) the subagent's tool calls fire the same hook with an agent or session id; (c) the hand-off event exists and whether it can block.** Reason: the owner asked for each engine. Option: Claude Code and Codex only, the two shipped plugins, with the others recorded as `not run`. |
| 9 | Which `gabo check` surface carries the lock rules? | open. **Proposed (auto): a second additive hook mode, `gabo check --pre-command`, reading tool-call JSON on stdin like `--pre-write`; the owner amends PRD R31 and TRD 9.4.** Reason: each mode keeps one contract. Option: extend `--pre-write`, which changes its meaning for every caller. |
| 10 | One switch per lock, or one per shell-guard rule? | open. **Proposed (auto): one per lock (five), and the shell guard's rules are not switched one by one.** Reason: the owner asked for one switch per hook. Option: eight switches, one per shell-guard rule, which the owner did not ask for. |
| 11 | Which outcomes are recorded in the wave trace? | open. **Proposed (auto): every outcome except `allowed` (`blocked`, `warned`, `not run`) is one line in `gabo/trace/<run id>` with the lock, task, engine and rule.** Reason: the owner needs to count how often each lock fires per wave. Option: only the test-file warnings. |
| 12 | How does the shell guard combine with gentle-pi's Pi guardrails? | open. **Proposed (auto): both apply, and a call is blocked if either blocks it.** Reason: gentle-pi's list is Pi-only and owned by gentle-pi, and a union cannot allow what one denies. Option: gabo's list replaces gentle-pi's on Pi. |
| 13 | Which sessions do the PR, worktree and force rules judge? | open. **Proposed (auto): the release and `main` rules judge every agent session; the PR, worktree and force rules judge coder sessions only; the reviewer's merge into `dev` is exempt.** Reason: the split lets the reviewer merge while a coder cannot, and D-2 only says agents reach `dev` and the owner does `main`. Option: every rule judges every agent session, with a documented exception for the reviewer's merge. |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed) |
| intent issue | [#765 I0028: Deterministic hooks as locks](https://github.com/testeos-z/gabo-ai/issues/765) |
| sibling intent, C | [0029 Branch protection on main and dev](../0029-branch-protection-on-main-and-dev/intent.md), [#766](https://github.com/testeos-z/gabo-ai/issues/766) |
| sibling intent, D | [0030 Agent evals when skills change](../0030-agent-evals-when-skills-change/intent.md), [#767](https://github.com/testeos-z/gabo-ai/issues/767) |
| sibling intent, A | [0027 Coders and product QA share one definition of done](../0027-coders-follow-gentle-ai-odd-and-rdd/intent.md), [#758](https://github.com/testeos-z/gabo-ai/issues/758) (its hooks 1 and 2 moved here; its hooks 3 and 4 stay there) |
| related intents | [0001](../0001-claude-code-plugin/intent.md), [0003](../0003-pi-package/intent.md), [0004](../0004-opencode-runtime/intent.md) (current hook scopes) |
| PRD | [PRD](../../PRD.md) (R9, R15, R31, R32) |
| TRD | [TRD](../../TRD.md) section 2 (thin layer, line 44), section 9.1 (`repository_host`, line 373), D13 (line 595) |
| spec | none yet; written only after the owner reviews this intent |
| plan | none yet |
| sources | [Anthropic, The AI-native SDLC playbook](https://claude.com/resources/articles/the-ai-native-sdlc-playbook), Louis Claxton, 2026-08-21 (quotes byte-checked 2026-11-13); [Claude Code hooks](https://code.claude.com/docs/en/hooks); [Codex hooks](https://learn.chatgpt.com/docs/hooks); [OpenCode plugins](https://opencode.ai/docs/plugins); gentle-pi 4.7.0 at `~/.pi/agent/npm/node_modules/gentle-pi/extensions/gentle-ai.ts:1725-1729`; `origin/dev` at `63b36df` for every repository path cited |
