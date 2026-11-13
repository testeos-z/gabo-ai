# Intent 0028 Deterministic hooks as locks

| field | value |
|---|---|
| intent id | 0028-deterministic-hooks-as-locks |
| status | Draft |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed; intent issue linked under it, as [#758](https://github.com/testeos-z/gabo-ai/issues/758) was) |
| intent issue | [#765 I0028: Deterministic hooks as locks](https://github.com/testeos-z/gabo-ai/issues/765) |
| date | 2026-11-13 |

## Problem

The owner's decision of 2026-11-13, after reviewing the gaps between gabo and Anthropic's playbook, is to put a set of deterministic locks on what agents do. Owner, verbatim, relayed by the owner's local assistant: "Me encaja ese corte, me parece bien. Prepara los documentos de intents y los issues" [I agree with that cut, it looks good to me. Prepare the intent documents and the issues]. The cut is "B, deterministic hooks ("locks")". The rules live in gabo (Go); each engine's hook is a thin relay that reports the tool call and applies gabo's answer (TRD section 2, `docs/product/TRD.md:44`). The owner cites this as "D9"; TRD D9 is another decision ("Direct session", `docs/product/TRD.md:591`), so this intent cites the thin-layer rule of TRD section 2 instead.

Evidence, checked on 2026-11-13 at `origin/dev` `63b36df`:

- **No engine hook judges a shell command.** Claude Code runs its hooks on `Write|Edit` only (`plugins/claude-code/hooks/hooks.json:6`, `:17`). The OpenCode stage gate covers `edit`, `write` and `patch` (`apps/cli/internal/skills/gabo-runtime-opencode/plugins/gabo-stage-gate.js:18`) and its header says a shell call is not gated (line 15). The Pi extension covers `write` and `edit` only (`packages/pi/extensions/gabo.ts:137`). Codex is the same as Claude Code on `apply_patch` (per the spec 0027 evidence base, `plugins/codex/hooks/hooks.codex.json:4-25`, not re-read here).
- **The coder rules are text.** Never push to `main` or `dev`, never force-push, delete with `-d` only: `.claude/skills/gabo-repository-host-github/SKILL.md:15-17`. Never merge, approve or set commit statuses: `apps/cli/internal/skills/gabo-coder/SKILL.md:15`. Scratch in `mktemp -d`: `apps/cli/internal/skills/gabo-coder/SKILL.md:29`. Nothing stops a shell call that breaks these rules.
- **Release is by hand, but nothing enforces it.** The reviewer merges clean heads into `dev` only (`gabo.yaml:49-52`, which cites TRD D13). A human sets the version in three manifests and runs `claude plugin tag` (`plugins/claude-code/README.md:45-56`, per the spec 0027 evidence). A coder shell call of `git tag`, `npm publish` or `gh release create` is not judged by any hook.
- **Reserved paths are checked only at merge.** `gabo.yaml:53` lists `review.human_only_paths` (`.github/`, `AGENTS.md`, `CLAUDE.md`, `gabo.yaml`, `.claude/`, `.codex/`, `.agents/`). The reviewer's merge script stops a merge that touches them (`merge-head.sh`, per the spec 0027 evidence). A write in a coder worktree is not stopped. The Claude Code write hook is silent for most of these paths: `plugins/claude-code/scripts/gabo-hook-filter.sh:87-90` judges only `gabo.yaml`, `.gabo/*`, `.claude/agents/*`, `.claude/skills/*` and `docs/product/*`.
- **Credentials: gabo ships no list.** The only list of sensitive paths is in the owner's personal Claude Code settings, `~/.claude/settings.json`, under `permissions.deny` (24 entries, checked 2026-11-13). It denies `Read` and `Edit` for `.env`, `.env.*`, `~/.ssh/**`, `~/.credentials/**`, `~/Library/Keychains/**`, `~/.aws/credentials`, `~/.config/gh/hosts.yml`, `**/*.pem`, `**/*.key` and `**/secrets/*`, and denies three `rm -rf` forms. Values are not copied here. It is a personal permission setting, not a gabo file, so it does not reach a project. The agents run with the owner's `gh` login (`gabo.yaml`, reviewer comment: "the scripts run as the owner with the owner's gh login").
- **Test files are not detected.** No script, extension or check in `plugins/`, `packages/pi/extensions/` or `apps/cli/internal/slices/check/` detects an edit to an existing test file (searched 2026-11-13). Playbook, extraction: "the block on the agent editing test files during a fix". The owner asks for a warning here, not a block.
- **No per-hook switch exists.** No plugin script, the OpenCode plugin or the Pi extension reads a setting that turns one hook off (searched 2026-11-13). The only existing off switch is gentle-ai's review mode, which the owner controls and the coders never toggle (spec 0027 evidence).
- **Pi already has guardrails, from gentle-pi.** The installed gentle-pi 4.7.0 lists `gitPush`, `gitRebase`, `gitBranchDeleteForce`, `npmPublish` and `piRemove` as guarded commands (`~/.pi/agent/npm/node_modules/gentle-pi/extensions/gentle-ai.ts:1725-1729`), read from `runtime-guardrails.json` (`:1921-1922`). They cover Pi only and are not in the gentle-ai binary.
- **Engine events are unproven for subagents.** Claude Code documents hooks inside subagents and a `SubagentStop` event (spec 0027 evidence table, source https://code.claude.com/docs/en/hooks). Codex, OpenCode and Pi are unknown for tool calls inside a subagent. Nothing has been probed.

Playbook evidence, Anthropic, "The AI-native SDLC playbook", Louis Claxton, 2026-08-21, https://claude.com/resources/articles/the-ai-native-sdlc-playbook. Quoted from a page extraction on 2026-11-13, not checked byte for byte: "Hooks are the approval gates."; "A hook can also ask, pausing the action until a specific person approves, which is what release gating needs."; "The production deploy hook blocks the release until a named release manager authorizes it."; "Block edits to protected paths such as generated classes or a frozen package;"; "A block should explain itself, so when a hook stops an action the reason and the route to approval…". Summarized, not quoted: the playbook uses hooks as gates for protected edits and for release, and names no shell commands. The commands in this intent come from gabo's own coder rules, not from the playbook.

## Who is affected

- The owner, who releases, owns `main`, and must be able to switch each lock off and on.
- Every coder in every wave. Each wave runs coders in their own worktrees, and each one can run any shell command today.
- The assistant, which relays each block and warning in the wave report.
- The reviewer, which remains the merge gate and does not run inside a coder session.

## Scope

In scope, one lock each, decided in gabo and relayed by each engine:

1. **Shell guard.** A shell call that pushes to `main` or `dev`, force-pushes, deletes a branch by force, removes a worktree by force, or deletes by glob is blocked with its rule and route.
2. **Release gate.** Agents work alone up to `dev`. A version, a tag, a publish or a release is the owner's action. The hook is only the technical lock that stops an agent from creating one.
3. **Reserved paths at write time.** Each entry of `review.human_only_paths` is blocked for a coder write, not only at merge.
4. **Credentials.** A list of sensitive paths that gabo ships, which a project can extend. A coder's read or write of a listed path is blocked.
5. **Test files, warn mode.** An edit to an existing test file is detected in code, recorded in the wave trace, and never blocked.
6. **One switch per lock**, the five above, which can be turned on and off one at a time, and only a human can change them.
7. **First task: a probe** that shows, on each engine (`claude`, `codex`, `pi`, `opencode`), whether a hook fires inside a coder subagent.

## Non-goals

- The folder boundary (former hook 5 of intent 0027). The owner decided it is out: "La carpeta es la única que dejaremos fuera... cada coder debe trabajar en su propio work tree, entonces no veo el choque" (relayed 2026-11-13).
- CI and package publishing. Both are out of all four intents of this batch (owner, 2026-11-13).
- The coder done gate and the methodology-report gate (hooks 3 and 4 of intent 0027). They stay in 0027 unless the owner moves them.
- Replacing the reviewer's merge gate (PRD R9). Hooks never merge, approve or set statuses.
- Changing the RDD switch or the gentle-pi guardrails. Coders never toggle RDD. gentle-pi keeps its Pi list.
- The playbook topics not yet discussed: the Maintain stage, permissions and isolation, CLAUDE.md practices, the fresh-context verifier, metrics and traces, and Design-stage security skills.

## Constraints

- Thin-layer rule (TRD section 2, `docs/product/TRD.md:44`): rules in Go, engine hooks relay only.
- PRD R31 (`docs/product/PRD.md:170`): the CLI offers four commands and one additive hook mode. A new hook mode or any new command needs the owner's amendment. The mode choice is shared with intent 0027 (its question 12); decide it once for both.
- PRD R32 (`docs/product/PRD.md:171`): engine config is written at project level only, never in the user's global engine config. Hooks ship in each engine's plugin or package.
- PRD R15: in auto mode no hook asks a human. Each lock blocks or allows and records; it never pauses for approval. The playbook's "ask" hook is therefore out of this intent (question 2).
- PRD R9: the reviewer is the only merge gate. These locks add outside checks and never merge.
- Intents 0001 (S7), 0003 (S5, S6) and 0004 (S10) keep their current write-only scopes; the new locks extend them for their own rules only.
- The agents share the owner's `gh` login (question 6), so a rule the owner's account can bypass does not bind the agents.
- Intent 0027's hooks 1 and 2 (spec S16 and S17, questions 12 to 17 and 20 of that spec) are the same text as this intent's shell guard and release gate. The sister narrowing of intent 0027 may renumber them. This intent takes them once the owner decides where they live.
- Names only: this document never copies a secret value or a credential path's contents.

## Open questions

Each question has a **Proposed (auto)** option with a one-line reason. None is accepted; the owner decides at review.

| # | question | status |
|---|---|---|
| 1 | Shell guard: a deny-list or an allow-list, and which exact commands? | open. **Proposed (auto): a deny-list, every entry named.** Reason: an allow-list stops a wave on the first new tool (a test runner, a linter). The list: (a) any `git push` whose refspec names `main`, `master` or `dev` (`git push origin HEAD:main`, `git push origin :main`, `git push origin main`); (b) any `git push` with `--force`, `-f`, `--force-with-lease`, `--mirror` or a refspec starting with `+`; (c) `git push --tags` and `git tag` when it creates a tag (`git tag -l` runs); (d) `git branch -D` and `git branch --delete --force`; (e) `git worktree remove` with `--force` or `-f`; (f) `rm` with a recursive flag whose target has a glob (`*`, `?`, `[`), or any path under `*-worktrees/`, while a literal scratch path under `mktemp -d` stays allowed; (g) `gh pr merge` with base `main`; (h) release commands: `npm publish`, `bun publish`, `pnpm publish`, `claude plugin tag`, `gh release create`. |
| 2 | Does each lock block or only warn? | open. **Proposed (auto): block for the shell guard, the release gate and reserved paths; block for a credential path; warn only for test files (lock 5).** Reason: the owner asked for locks, and a warning cannot stop a push. A lock on an engine without its event warns with `not run` (question 3). An "ask" outcome is excluded (PRD R15). |
| 3 | What happens on an engine, or in a subagent, without the event? | open. **Proposed (auto): the probe decides per engine; where a hook does not fire inside a coder subagent, its outcome is `not run: <reason>` in the wave trace and the wave report, and the owner is told; the text rule stays.** Reason: a silent gap is the failure the owner wants removed. Options: (a) as proposed; (b) refuse that engine as a coder runtime until the event exists (blocks OpenCode coders today); (c) run the check in the parent session only. |
| 4 | How is the owner's release authorization recorded? | open. **Proposed (auto): none is needed. The owner runs every version, tag, publish and release step outside an agent session, and the gate blocks every such command in an agent session.** Reason: the owner's own steps stay in the owner's shell, and a record file in the repository is a file an agent can forge. Options: (b) a record file the owner writes by hand outside any worktree, matched to the exact version, tag and commit, with its path denied to agents; (c) a GitHub environment approval, which depends on intent 0029. This differs from intent 0027 question 17, which proposed (b). The two must be reconciled before either is accepted. |
| 5 | Which credential paths does gabo ship, and where does a project extend the list? | open. **Proposed (auto): gabo ships the categories named in the Problem (`.env` and `.env.*`, `*.pem`, `*.key`, `secrets/`, `~/.ssh`, `~/.aws/credentials`, `~/.config/gh/hosts.yml`, the macOS keychain folder, `~/.credentials`); a project adds patterns with an optional key `hooks.sensitive_paths` in `gabo.yaml`.** Reason: the owner's own list is the best evidence of what matters, and an optional key is additive. Options: (b) the list lives only in each project's `gabo.yaml`, with nothing shipped; (c) shipped only, with no project extension. |
| 6 | How is a human-only switch protected from agents? | open. **Proposed (auto): the switches live in the owner's user config (`~/.config/gabo/config.yaml`, keyed by repository), outside every repository and worktree; the shell guard denies any agent command that names that file; each hook reads the switch on every call.** Reason: the agents run with the owner's `gh` login and shell, so a switch inside the repository (for example `gabo.yaml`, which agents can edit in their worktree) is one edit away. Residual risk, stated: an agent with shell access can still write the file if the guard's pattern misses it. Options: (b) in `gabo.yaml`, which blocks merges but not worktree writes; (c) Claude Code managed settings, which an administrator owns and which the playbook uses for non-negotiable hooks (extraction), but which is outside gabo's scope (R32). |
| 7 | When gabo is missing or fails, does a hook fail open or fail closed? | open. **Proposed (auto): fail open with a visible warning when gabo is not on the PATH; fail closed for the blocking locks when gabo runs and fails.** Reason: a machine without gabo is not blocked, but a broken gabo does not silently allow a push. Options: (a) fail open everywhere with a warning (today's rule for Claude Code and Codex, `gabo-pre-write.sh:38-44`); (c) fail closed everywhere, so a machine without gabo cannot run a coder. |
| 8 | Which files count as test files, what counts as an existing one, and where does the warning go? | open. **Proposed (auto): test files are `*_test.go`, `*.test.*`, `*_test.*`, and any file under `test/` or `tests/`; an existing file is one tracked at the worktree's base commit; the hook appends one JSON line per warning to a file under the run's `mktemp -d` scratch, and the cycle copies those lines into `gabo/trace/<run id>`.** Reason: a hook cannot call memory, and the owner asked for the record in the wave trace. Options: (b) only `*_test.go` and `*.test.sh`. |
| 9 | How far does the shell guard normalise a command? | open. **Proposed (auto): one level only: `sh -c '...'`, `bash -c '...'` and `git -C <dir>` are unwrapped before matching; aliases, `eval` and `xargs` are not.** Reason: each new level is a new parser; the probe and the review test the list. Option (b): a full shell parser in Go, which adds a dependency (Task Size item 5, a high-risk item). |
| 10 | What does the first task, the probe, have to prove, and on which engines? | open. **Proposed (auto): on all four full-tier engines (`claude`, `codex`, `pi`, `opencode`), that (a) a shell pre-command hook fires inside a coder subagent and can deny a shell call, (b) the subagent's tool calls fire the same hook with an agent or session id, and (c) the hand-off event exists and whether it can block.** Reason: the owner asked for each engine, and the flexible tier is out of scope. Option (b): Claude Code and Codex only, the two shipped plugins, with the others as `not run`. |
| 11 | Which `gabo check` surface carries the lock rules? | open. **Proposed (auto): a second additive hook mode, `gabo check --pre-command`, reading the tool-call JSON on stdin like `--pre-write`; the owner amends PRD R31 and TRD 9.4.** Reason: each mode keeps one contract. Option (a): extend `--pre-write`, which changes its existing meaning for every caller. The same decision as intent 0027 question 12. |
| 12 | Which sessions do the locks judge? | open. **Proposed (auto): the release and main rules (questions 1 a to h) judge every agent session; the merge-into-`dev`, PR and worktree rules judge coder sessions only, identified by a coder session's working directory inside a task worktree (intent 0027 question 21).** Reason: the reviewer merges into `dev` from an agent session with the owner's login, and the owner's own release steps are outside agent sessions. Option (b): every rule judges coder sessions only, which leaves the assistant free to push to `main`. Option (c): every rule judges every agent session, which needs a documented exception for the reviewer's merge. |
| 13 | How does the shell guard combine with gentle-pi's Pi guardrails? | open. **Proposed (auto): both apply, and a call is blocked if either blocks it.** Reason: gentle-pi's list is Pi-only and owned by gentle-pi; the union cannot allow what one denies. Option (a): gabo's list replaces gentle-pi's on Pi. Option (c): gabo defers to gentle-pi on Pi. |
| 14 | Are block and warning outcomes recorded in the wave trace, or only the test-file warnings? | open. **Proposed (auto): every outcome (`allowed` is not recorded; `blocked`, `warned` and `not run`) is one line in `gabo/trace/<run id>` with the lock's name, the task, the engine and the rule.** Reason: the owner needs to count how often each lock fires per wave. Option (b): only test-file warnings, as the owner named them. |
| 15 | How many switches, and at what granularity? | open. **Proposed (auto): five switches, one per lock named by the owner (shell guard, release gate, reserved paths, credentials, test-file warn); the shell guard's rules are not switched one by one.** Reason: the owner asked for "one switch per hook". Option (b): one switch per shell-guard rule (eight switches), which the owner did not ask for. |
| 16 | Which reserved paths does the write-time lock cover? | open. **Proposed (auto): every entry of `review.human_only_paths` (`.github/`, `AGENTS.md`, `CLAUDE.md`, `gabo.yaml`, `.claude/`, `.codex/`, `.agents/`), for any coder write.** Reason: the owner named that list, and the reviewer already refuses it at merge. A coder that needs a change there reports it as blocked for the owner, which is the handoff the adapter and the coder agent under `.claude/` need. Option (b): block only the entries the spec of a task does not name, which lets a coder change `.claude/agents/` silently. |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed) |
| intent issue | [#765 I0028: Deterministic hooks as locks](https://github.com/testeos-z/gabo-ai/issues/765) |
| sibling intent, C | [0029 Branch protection on main and dev](../0029-branch-protection-on-main-and-dev/intent.md), [#766](https://github.com/testeos-z/gabo-ai/issues/766) |
| sibling intent, D | [0030 Agent evals when skills change](../0030-agent-evals-when-skills-change/intent.md), [#767](https://github.com/testeos-z/gabo-ai/issues/767) |
| sibling intent, A | [0027 Coders and product QA share one definition of done](../0027-coders-follow-gentle-ai-odd-and-rdd/intent.md), [#758](https://github.com/testeos-z/gabo-ai/issues/758) (its hooks 1 and 2 moved here; its hooks 3 and 4 stay there; its spec is being narrowed by a sister agent) |
| related intents | [0001](../0001-claude-code-plugin/intent.md), [0003](../0003-pi-package/intent.md), [0004](../0004-opencode-runtime/intent.md) (current hook scopes) |
| PRD | [PRD](../../PRD.md) (R9, R15, R31, R32) |
| TRD | [TRD](../../TRD.md) section 2 (thin layer, line 44), section 9.1 (`repository_host`, line 373), D13 (line 595) |
| spec | none yet; written only after the owner reviews this intent |
| plan | none yet |
| sources | [Anthropic, The AI-native SDLC playbook](https://claude.com/resources/articles/the-ai-native-sdlc-playbook), Louis Claxton, 2026-08-21 (extraction, not byte-checked); [Claude Code hooks](https://code.claude.com/docs/en/hooks); [Codex hooks](https://learn.chatgpt.com/docs/hooks); [OpenCode plugins](https://opencode.ai/docs/plugins); gentle-pi 4.7.0 at `~/.pi/agent/npm/node_modules/gentle-pi/extensions/gentle-ai.ts:1725-1729`; `origin/dev` at `63b36df` for every repository path cited |
