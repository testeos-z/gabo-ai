# Spec 0001 Claude Code plugin

| field | value |
|---|---|
| intent | [0001-claude-code-plugin](intent.md) |
| status | Draft |
| owner | gabo-pm |
| contributors | — |
| date | 2026-11-03 |

Fixed questions answered here: Q1 and Q2 in section 1, Q3 in section 2, Q4 in section 3, Q5 in section 4, Q6 in section 5, Q7 in section 6. Q8 is for the human owner (section 7). Items marked **Proposed** wait for the owner's decision in section 6.

## Evidence

The design rests on these facts, read on 2026-11-03.

| fact | source |
|---|---|
| `gabo init` copies the embedded catalog (`apps/cli/internal/skills`, without `gabo-reviewer`) into `.gabo/skills/` and writes `.gabo/VERSION`; it never writes config or agents. | TRD 2 (D24), 9.4; `apps/cli/internal/skills/skills.go` |
| `gabo-runtime-claude` links `.claude/skills/<name>` to `../../.gabo/skills/<name>`, builds `.claude/agents/<id>.md` with `skills:` by bare name and `gabo-version:`; `gabo check` requires those links and agents. | `apps/cli/internal/skills/gabo-runtime-claude/SKILL.md`; `apps/cli/internal/slices/check/links.go` |
| `gabo check` is read-only, has no flags, exits 0, 1 (failure) or 2 (usage or read error), and reports `✗ <intent>/plan.md: spec.md is not accepted …` only once a `plan.md` exists. | TRD 9.4; `apps/cli/internal/slices/check/intents.go:113` |
| Local Claude Code is 2.1.317. `claude plugin` offers `validate`, `install`, `details`, `tag`, `update`, `marketplace`. Installed marketplaces live in `~/.claude/plugins/known_marketplaces.json`, clones in `~/.claude/plugins/marketplaces/`, installed copies in `~/.claude/plugins/cache/`. | `claude --version`, `claude plugin --help`, `ls ~/.claude/plugins` |
| A marketplace is a repository with `.claude-plugin/marketplace.json` (`name`, `owner`, `plugins[]` with `name` and `source`); install id is `<plugin>@<marketplace>`; users add it with `claude plugin marketplace add <owner>/<repo>[#ref]`; `--scope project` writes a committed `.claude/settings.json`. | https://code.claude.com/docs/en/plugin-marketplaces, https://code.claude.com/docs/en/plugins/host-marketplace.md |
| Plugin layout: `.claude-plugin/plugin.json` (only `name` required), `skills/<name>/SKILL.md`, `agents/`, `commands/`, `hooks/hooks.json`; every component is namespaced (`<plugin>:<skill>`); hook commands get `CLAUDE_PLUGIN_ROOT`, `CLAUDE_PROJECT_DIR` and `CLAUDE_PLUGIN_OPTION_<KEY>` from `userConfig`; a top-level `bin/` blocks claude.ai organization sync. | https://code.claude.com/docs/en/plugins-reference |
| When a plugin is copied into the cache, a symlink to a target elsewhere in the same marketplace repository is dereferenced (its content is copied); a symlink outside the marketplace is skipped. | https://code.claude.com/docs/en/plugins/host-marketplace.md (Share files within a marketplace with symlinks) |
| Updates are detected by the computed version: `plugin.json` `version` first, then the marketplace entry; with a `version` set, users stay on it until it changes; with none, they track commits. Background auto-update is off by default for third-party marketplaces. | https://code.claude.com/docs/en/plugins/host-marketplace.md (Keep users up to date) |
| A private GitHub marketplace works through the user's own git credentials (SSH key or `gh auth setup-git`). `testeos-z/gabo-ai` is private, default branch `main`. | same page; `gh repo view testeos-z/gabo-ai` |
| Reference plugins: `ghiaog123/ai-native-sdlc` (`plugins/ai-sdlc`, version 0.1.4) ships skills, agents and hooks (`PreToolUse` `Write\|Edit` artifact gate, `Bash` prod guard, `SessionStart`, `UserPromptSubmit`) as `"${CLAUDE_PLUGIN_ROOT}/hooks/<script>.sh"`; engram ships its marketplace from its product repository with the plugin at `plugin/claude-code` and the same `version` in `plugin.json` and the entry. | `gh api repos/ghiaog123/ai-native-sdlc/contents/...`; `~/.claude/plugins/marketplaces/engram` |

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | The gabo-ai repository shall be a Claude Code marketplace named `gabo-ai` that lists one plugin `gabo`. | Given the repository at a release commit When `claude plugin validate .` and `claude plugin validate plugins/claude-code` run Then both end with `Validation passed` (warnings allowed only for fields listed in D2) and `marketplace.json` lists exactly one entry `gabo` with source `./plugins/claude-code`. | Check: `claude plugin validate` on the owner's machine; Go test reads `marketplace.json` and asserts the single entry and source. |
| S2 | When a user runs `claude plugin marketplace add testeos-z/gabo-ai` and then `claude plugin install gabo@gabo-ai`, Claude Code shall install the plugin with the catalog skills. | Given a machine with git access to the repository When both commands run Then `claude plugin list` shows `gabo@gabo-ai` enabled and `claude plugin details gabo` lists every skill of S3. | Manual check on the owner's local Claude Code (section 2, Verification). |
| S3 | The plugin shall ship as skills exactly the catalog skills that `gabo init` copies (every folder of `apps/cli/internal/skills` except `gabo-reviewer`), each byte for byte, under its catalog name. | Given the repository When the Go test lists `plugins/claude-code/skills/` Then its entries equal `skills.Names()` minus `gabo-reviewer`, each resolves to `apps/cli/internal/skills/<name>`, and no entry `gabo-reviewer` exists. Given the installed plugin When `SKILL.md` of any skill in the cache is compared with the catalog file Then they are identical. | Go unit test in `apps/cli` (catalog vs plugin skill set); manual byte comparison of the cache copy. |
| S4 | The plugin shall ship no agents, no commands, no MCP servers and no `bin/` directory. | Given the plugin directory When it is listed Then it has no `agents/`, `commands/`, `bin/` or `.mcp.json`, and `plugin.json` declares none of `agents`, `commands`, `mcpServers`. | Go unit test on the plugin tree. |
| S5 | The plugin's `version` shall equal the gabo CLI release version (`x.y.z`, the value `gabo init` writes to `.gabo/VERSION`), set in `plugin.json` only. | Given a release tag `apps/cli/v<x.y.z>` When `plugins/claude-code/.claude-plugin/plugin.json` is read Then `version` is `x.y.z` and the marketplace entry has no `version`. | Release check (see D5): `claude plugin tag plugins/claude-code` succeeds and its tag `gabo--v<x.y.z>` matches the CLI tag; Go test asserts the entry has no `version`. |
| S6 | Where the plugin is enabled and its `checks` option is on, when Claude edits or writes a gabo file in a project that has `.gabo/VERSION`, the plugin shall run `gabo check` in the project root and return its failure lines to Claude. | Given a gabo project whose intent `0009-x` has a `plan.md` and a `spec.md` with an empty Acceptance table When Claude writes `docs/product/intents/0009-x/plan.md` Then the hook exits 2 and its stderr holds the `gabo check` line `✗ docs/product/intents/0009-x/plan.md: spec.md is not accepted …`. Given the same edit with an accepted spec and a passing project Then the hook exits 0 with no output. | Shell tests of the hook script with a fake `gabo` on `PATH` (same style as `gabo-reviewer/scripts/tests/`); manual check on the owner's machine. |
| S7 | The hook shall stay silent and never block when the edited file is not a gabo file, when the project has no `.gabo/VERSION`, when `gabo` is not on `PATH`, or when the `checks` option is off. | Given each of these four cases When Claude writes a file Then the hook exits 0 with no output. Given `gabo check` exits 2 Then the hook exits 0 and prints one warning line to stderr. | Shell tests of the hook script, one per case. |
| S8 | The plugin shall not change `gabo init`, `gabo check`, the project's catalog copy, `.claude/skills/`, `.claude/agents/`, `gabo.yaml` or any user-level file. | Given a gabo project that passes `gabo check` When the plugin is installed and a session runs Then `git status` shows no change made by the plugin and `gabo check` still passes; built agents still preload the project copies by bare name. | Manual check on the owner's machine; review of the hook script (it only reads and runs `gabo check`). |
| S9 | The repository documentation shall tell a user how to add the marketplace, install the plugin, turn the checks off, and how the plugin relates to `gabo init`. | Given the README When a reviewer reads its install section Then it names `claude plugin marketplace add testeos-z/gabo-ai`, `claude plugin install gabo@gabo-ai`, the `checks` option, the private-repository credential step and that `gabo init` stays required for per-project config, slots, adapters and agents. | Doc review. |

## 2. Design

**Layout.** One plugin inside the product repository, next to the CLI:

```text
.claude-plugin/marketplace.json          marketplace "gabo-ai", one entry "gabo" -> ./plugins/claude-code
plugins/claude-code/
  .claude-plugin/plugin.json             name "gabo", version x.y.z, userConfig.checks
  README.md
  skills/<name> -> ../../../apps/cli/internal/skills/<name>   one relative symlink per catalog skill, no gabo-reviewer
  hooks/hooks.json                       PostToolUse Write|Edit -> scripts/gabo-check.sh
  scripts/gabo-check.sh                  the only executable; reads stdin JSON, runs gabo check
  scripts/tests/*.sh                     shell tests with a fake gabo
```

**Data flow of the hook.** Claude Code calls `scripts/gabo-check.sh` after each `Write` or `Edit`, with the tool input as JSON on stdin. The script exits 0 at once when `CLAUDE_PLUGIN_OPTION_CHECKS` is `false`, when `$CLAUDE_PROJECT_DIR/.gabo/VERSION` is missing, when `gabo` is not on `PATH`, or when `tool_input.file_path` is outside the gabo paths (`gabo.yaml`, `.gabo/`, `.claude/agents/`, `.claude/skills/`, `docs/product/`). Otherwise it runs `gabo check` in `$CLAUDE_PROJECT_DIR` and maps its exit code: 0 → exit 0, silent; 1 → print the `✗`/`next:` lines to stderr and exit 2, so Claude reads them; 2 → one warning line, exit 0. The check logic stays in `gabo check`; the script only filters and maps (no second implementation of any rule).

**Coexistence with `gabo init`.** `gabo init` and the per-project copy stay the source of truth inside a project: agents built by `gabo-runtime-claude` keep preloading `.claude/skills/<name>` (linked to `.gabo/skills/<name>`), and `gabo check` keeps checking them against `.gabo/VERSION`. The plugin's skills appear in the session as `gabo:<name>` (namespaced, so no name clash) and serve sessions with no gabo project yet and the first setup. The plugin never writes anything.

**Versioning.** The plugin version is the CLI release version. The skills are symlinks to the catalog at the same commit, so a plugin `x.y.z` carries byte for byte the catalog that `gabo` `x.y.z` copies. A project whose `.gabo/VERSION` differs from the plugin version still works: its agents use the project copy.

**Verification on the owner's local Claude Code.** Recheck `claude --version` (2.1.317 at writing) and, when it differs, the facts of the Evidence table. Then:

1. `claude plugin validate .` and `claude plugin validate plugins/claude-code`.
2. Development loop: `claude plugin marketplace add ./` from the repository root (loads in place), `claude plugin install gabo@gabo-ai`, `claude plugin details gabo`: the skill list equals S3 and one hook is listed.
3. Release path: push the branch, `claude plugin marketplace remove gabo-ai`, `claude plugin marketplace add testeos-z/gabo-ai#<branch>`, `claude plugin install gabo@gabo-ai`; then `cmp` each `~/.claude/plugins/cache/gabo-ai/gabo/<version>/skills/<name>/SKILL.md` with `apps/cli/internal/skills/<name>/SKILL.md` (proves the symlinks were dereferenced, D3).
4. Hooks: in a scratch copy of a gabo project (`mktemp -d`), ask Claude to write a `plan.md` for an intent whose spec has no acceptance row: Claude receives the `gabo check` failure. Repeat outside any gabo project and with `checks` off: no output.
5. `gabo check` in this repository still passes, and `git status` shows no file written by the plugin.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Distribution channel | (a) the gabo-ai repository is its own marketplace, like engram; (b) a separate marketplace repository; (c) Anthropic's plugin directory | **Proposed** (a): one repository, one release, no copy to sync; (c) can follow later from the same plugin directory | Users need git access while the repository is private (owner question 3); the marketplace clone fetches the whole repository |
| D2 | Plugin and marketplace names | `gabo@gabo-ai`; `gabo-ai@gabo-ai`; `gabo-ai-sdlc@…` | `gabo` in marketplace `gabo-ai`: short, matches the CLI, no reserved word; skills keep their catalog names, so they appear as `gabo:gabo-pm` | The doubled prefix in `/gabo:gabo-pm` is accepted; renaming later needs a `renames` entry |
| D3 | Single source for the plugin skills | (a) relative symlinks into `apps/cli/internal/skills`; (b) a copied tree plus a byte-equality test; (c) a generator step at release | (a): Claude Code dereferences in-marketplace symlinks into the cache, so there is one source and no drift; a Go test pins the link set | Relies on the documented symlink behavior; step 3 of Verification proves it on the owner's machine, and (b) is the fallback if it fails |
| D4 | Which skills | (a) every skill `gabo init` copies; (b) a subset (directiva roles and setup); (c) everything including `gabo-reviewer` | **Proposed** (a): one rule ("what `gabo init` copies"), no second list to maintain; `gabo-reviewer` stays out per D20 | About 30 skill descriptions enter every session where the plugin is on; `claude plugin details gabo` reports the token cost (owner question 1) |
| D5 | Version source | (a) `plugin.json` `version` = CLI release version; (b) no version, track commits; (c) version in the marketplace entry | (a) in `plugin.json` only: an update reaches users exactly when a release bumps it; `claude plugin tag` creates `gabo--v<x.y.z>` next to `apps/cli/v<x.y.z>` | The release step must bump it; there is no release pipeline yet (TRD 9.4 Dependency), so the bump is a manual release-checklist step until one exists (owner question 4) |
| D6 | Agents in the plugin | ship role agents; ship none | none: agents need per-project slot adapters, their files must sit in `.claude/agents` for `gabo check`, and plugin agents would be namespaced | The plugin cannot run a wave by itself; `gabo init` plus the runtime skill stay the way to get agents |
| D7 | Commands in the plugin | a `/gabo:setup` command carrying the setup prompt; none | none: the setup prompt has one source embedded in the CLI (PRD R36, D25); a third copy needs its own equality test and is a later intent | A user still runs `gabo prompt` or `gabo init` to start setup |
| D8 | Hook placement for the stage gate | (a) `PreToolUse` that blocks writing `plan.md` before acceptance; (b) `PostToolUse` that runs `gabo check` and returns its failures; (c) `Stop` hook running `gabo check` | **Proposed** (b): reuses `gabo check` unchanged; (a) cannot reuse it, because `gabo check` only flags a `plan.md` that exists and takes no path argument, so (a) needs either a new `gabo check` input (a CLI contract change, R31) or a second implementation of the acceptance rule; (c) can block every stop on unrelated failures | The plan file is written before Claude is told to remove it; the gate is corrective, not preventive (owner question 2) |
| D9 | Making the hooks optional | (a) `userConfig.checks` boolean, default `true`; (b) a second plugin `gabo-checks`; (c) no option | **Proposed** (a): one plugin, one version, off with one `/config` row; the script reads `CLAUDE_PLUGIN_OPTION_CHECKS` | Hooks are on by default but no-ops outside gabo projects (S7) |
| D10 | `gabo` binary for the hooks | ship it in `bin/`; require it on `PATH` | require it on `PATH`, silent no-op when missing: the CLI is installed from its documented link (TRD 2), a `bin/` would block claude.ai organization sync, and binaries are per OS | A plugin-only user gets skills without checks until they install the CLI |

## 3. UX and app flow (optional)

Not applicable: the change adds no screen or app flow. The only user-visible surfaces are two Claude Code shell commands (S2), the plugin's `checks` row in `/config` (D9) and `gabo check` lines relayed unchanged by the hook (UX_UI section 4 shape, owned by `gabo check`).

## 4. DB migration steps (optional)

Not applicable: the change alters no stored data, no config shape and no catalog layout; `.gabo/VERSION`, `gabo.yaml` and the JSON Schemas are unchanged.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| PRD R31, TRD 9.4 (`gabo check` has no flags) | A preventive `PreToolUse` gate needs `gabo check` to judge a `plan.md` that does not exist yet. | D8 (b): `PostToolUse` reuses `gabo check` unchanged; a preventive gate is owner question 2 and would need a plan change. |
| TRD D20 (`gabo-reviewer` only at user level, installed by `gabo init`) | Shipping the whole catalog would put the reviewer scripts in a channel the PR under review could change. | `gabo-reviewer` is excluded (S3, D4). |
| PRD R36, TRD D25 (one source for the setup prompt) | A plugin command with the setup prompt would be a third copy. | No commands in this intent (D7). |
| TRD 9.4 project-relative spec | In a gabo project the session sees both the project copy (`gabo-pm`) and the plugin copy (`gabo:gabo-pm`), possibly of different versions. | Agents keep preloading the project copy by bare name, so waves are unaffected; how the main session picks between them is owner question 5. |
| Intent constraint "hooks limited to the deterministic checks" | None in the chosen design: the only hook runs `gabo check`. | A version-mismatch or context hook is out of scope unless the owner widens the constraint (owner question 5). |
| TRD 9.6 Claude Code Plugins row | None: install by `claude plugin marketplace add` and `claude plugin install`, never by editing `~/.claude/plugins` files. | Followed. |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Intent question 1: which catalog skills go into the plugin? Options: (a) all that `gabo init` copies (about 30, one rule, more description tokens per session); (b) only directiva roles and setup skills (`gabo-setup`, `gabo-config-writer`, `gabo-agent-templates`, `gabo-runtime-claude`, `gabo-runtime-builder`, `gabo-doc-guide`, `gabo-pm`, `gabo-dev-lead`, `gabo-frontend`, `gabo-db-master`, `gabo-plan-reviewer`), fewer tokens but a second list to maintain; (c) include `gabo-reviewer` too. Recommendation: (a); never (c) (D20). | carried to owner (proposed answer: a) |
| 2 | Intent question 2: are the hooks in the first release, and how strong is the stage gate? Options: (a) first release with the corrective `PostToolUse` hook of D8 (b), no CLI change; (b) first release with a preventive `PreToolUse` gate, which needs a plan change adding an input to `gabo check` (R31) or a duplicated acceptance rule; (c) skills only now, hooks in a later intent. Recommendation: (a). | open, owner |
| 3 | Intent question 3: which marketplace hosts the plugin, given `testeos-z/gabo-ai` is private? Options: (a) this repository as marketplace, users need git credentials (`gh auth setup-git` or an SSH key) until it is public; (b) a separate public marketplace repository holding a copy of the skills (sync job, drift risk); (c) Anthropic's plugin directory (public listing, submission review, needs a public source). Recommendation: (a) now, (c) when the repository goes public. | open, owner |
| 4 | Which ref do users install from, and who bumps the version? Options: (a) default branch `main`, `version` bumped by hand in the release commit until a release pipeline exists; (b) a `stable` branch or tag pinned with `#stable` on `marketplace add`; (c) omit `version` and track every commit on `main`. Recommendation: (a), with the bump and `claude plugin tag` added to the release checklist. | open, owner |
| 5 | Inside a gabo project, how should the main session treat the duplicate plugin skills? Options: (a) accept both copies (agents are unaffected; the main session may load the plugin copy); (b) widen the hook constraint with a `SessionStart` hook that, when `.gabo/VERSION` exists, tells the session the project copy wins and names a version mismatch; (c) document disabling the plugin per project with `enabledPlugins` `"gabo@gabo-ai": false` in `.claude/settings.json`. Recommendation: (a) for the first release plus the README note of (c); (b) only if the owner widens the constraint. | open, owner |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
