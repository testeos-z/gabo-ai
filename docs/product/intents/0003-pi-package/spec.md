# Spec 0003 Pi package

| field | value |
|---|---|
| intent | [0003-pi-package](intent.md) |
| status | Draft |
| owner | gabo-pm |
| contributors | — |
| date | 2026-11-03 |

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | The gabo Pi package shall be a Pi package whose `package.json` declares the `pi-package` keyword and a `pi` manifest that lists exactly its skills and extensions (`pi.skills`, `pi.extensions`), declares the Pi host packages only in `peerDependencies` with range `"*"`, and ships no `gabo-reviewer` skill. | Given the package directory When `npm pack --dry-run` runs Then the tarball holds `package.json`, the listed skill directories and extension files and nothing named `gabo-reviewer`; and `package.json` has `keywords` containing `pi-package`, `pi.skills`, `pi.extensions`, and no host package (`@earendil-works/*`, `typebox`) under `dependencies`. | `bun test` manifest test; `npm pack --dry-run` in CI |
| S2 | When the user runs `pi install npm:<package>@<version>` (or the local path during development), Pi shall list the package under `pi list` and load its skills and extensions at the next start, with no file written into the project. | Given the owner's local Pi 1.4.0 When `pi install <source>` then `pi list` run Then the package appears under "User packages"; and `git status` in the open project shows no change made by the install. | manual smoke on local Pi (S9) |
| S3 | The package shall ship one bootstrap skill, `gabo-pi`, whose name never equals a skill or fixed-name link that `gabo-runtime-pi` places in `.pi/skills/`; it tells the agent to install the gabo CLI when `gabo` is missing, run `gabo init` when `.gabo/skills/` is missing, then run `gabo prompt` and follow its output. It holds no copy of the setup prompt text. | Given the package When its skill list is compared with the spec skills and slot link names of `.gabo/skills/gabo-agent-templates/agents.yaml` and the `gabo-<slot>` names Then no name is shared; and the skill body contains `gabo prompt` and no line of `setup-prompt.md`. | `bun test` name-collision and no-copy test |
| S4 | When a Pi session starts in a gabo project (a `gabo.yaml` in the working directory) and no subagent tool is registered (none of the names in the extension's subagent tool list, default gentle-pi's `subagent_run`), the extension shall show a warning notification with the refusal text of `gabo-runtime-pi` (Prerequisite step 4), which recommends gentle-pi and the full Gentle AI framework with `gentle-ai install --agent pi`. Outside a gabo project it does nothing. | Given a project with `gabo.yaml` and Pi started with `--no-extensions -e <package>` When the session starts Then the notification text equals the quoted block of `gabo-runtime-pi`; Given gentle-pi loaded Then no notification; Given no `gabo.yaml` Then no notification. | `bun test` with a fake `ExtensionAPI`; manual smoke (S9) |
| S5 | While no subagent tool is registered in a gabo project, the extension shall block every `write` or `edit` tool call whose target path is under `.pi/agents/`, with the refusal text as the block reason. No subagent fallback exists. | Given no subagent tool When the agent calls `write` on `.pi/agents/gabo-coder.md` Then the call returns `block: true` and the reason is the refusal text; Given gentle-pi loaded Then the same call is not blocked. | `bun test` |
| S6 | When the agent calls `write` or `edit` on `docs/product/intents/<NNNN-slug>/plan.md` and that folder's `spec.md` is missing or its section 7 Acceptance table has no filled row, the extension shall block the call with the reason `intent <NNNN-slug>: plan.md needs a spec.md accepted by a human (section 7)`. When the spec cannot be read or parsed, it blocks (fail closed). | Given an intent with a Draft spec and an empty acceptance table When `write` targets its `plan.md` Then blocked with that reason; Given a filled acceptance row Then not blocked; Given a write to any other path Then not blocked. | `bun test` on the same fixture cases as the Go stage-order test of `gabo check` (parity test) |
| S7 | The package version shall equal the gabo CLI release version it ships with (one version, one release tag). When `.gabo/VERSION` in a gabo project differs from the package version, the extension shall show one informational notification naming both versions and the next step (`gabo init` or a package update), and change nothing. | Given package `0.5.0` and `.gabo/VERSION` `0.4.0` When the session starts Then one notification names `0.5.0`, `0.4.0` and the next step; Given equal versions Then none. | `bun test`; release check that `package.json` version equals the tag |
| S8 | The package shall coexist with `gabo init` and `gabo-runtime-pi`: it writes no file in the project, does not replace the `.gabo/skills/` catalog, the `.pi/agents/` agents or the `.pi/skills/` links, and does not count as a subagent package for the `gabo-runtime-pi` Prerequisite. | Given a project set up by `gabo init` and `gabo-runtime-pi` When the package is installed and a Pi session runs Then `git status` shows no change made by the package and `gabo check` passes as before; and the Prerequisite of `gabo-runtime-pi` still refuses when only the gabo package is installed. | manual smoke (S9); `gabo check` |
| S9 | The package shall have a written verification procedure, run on the owner's local Pi 1.4.0 before each release: install from the local path, `pi list`, one start with gentle-pi (no warning), one start with `--no-extensions -e <package>` (warning shown, `.pi/agents/` write blocked), one stage-gate block, and `pi remove`. | Given the procedure When the owner runs it Then each step's expected output is observed and recorded in the intent's review findings. | manual, recorded |

## 2. Design

**Evidence (checked 2026-11-03 on the owner's machine).**

- Pi 1.4.0 (`pi --version`). `pi install <source> [-l]`, `pi remove`, `pi list`, `pi config`; sources `npm:<name>@<version>`, `git:<host>/<path>@<ref>`, a URL, or a local path (Pi docs `docs/packages.md`).
- Manifest: `package.json` with `keywords: ["pi-package"]` and a `pi` object of `extensions`, `skills`, `prompts`, `themes` (paths or globs relative to the package root), optional `image`/`video`. gentle-pi 4.5.1 uses `"skills": ["./skills"]`, `"extensions": ["./extensions"]`; pi-btw 0.8.0 lists one file `./extensions/btw.ts`; @xynogen/pix-gate lists `src/index.ts`. Host packages (`@earendil-works/pi-coding-agent`, `pi-ai`, `pi-tui`, `pi-agent-core`, `typebox`) go in `peerDependencies` with `"*"`, never in `dependencies` (`docs/packages.md`).
- Extensions are TypeScript loaded through `jiti`, no build step. `pi.on("tool_call", …)` can return `{ block: true, reason }` (`ToolCallEventResult` in the 1.4.0 `extensions/types.d.ts`); gate handlers should fail closed (`docs/extensions.md`). `pi.getAllTools()` lists every configured tool; `ctx.ui.notify` shows a notification; `session_start` runs after trust.
- Skill name collisions keep the first discovered skill (`docs/skills.md`). `package-manager.js` `resolve()` adds package resources before auto-discovered project `.pi/skills/`, so a package skill shadows a project skill of the same name.
- Git sources clone the whole repository and run `npm install` at its root (`installGit` in `package-manager.js`); there is no subdirectory selector. The gabo-ai root `package.json` is the private bun workspace root.
- `--no-extensions` disables extension discovery while an explicit `-e <path>` still loads (`pi --help`), so a run without gentle-pi can be simulated without uninstalling it.
- `gabo-runtime-pi` already refuses without a subagent package and recommends gentle-pi plus `gentle-ai install --agent pi` (`apps/cli/internal/skills/gabo-runtime-pi/SKILL.md`, Prerequisite).

**Shape.** A new workspace package `packages/pi/` (the existing bun workspace `packages/*`):

```text
packages/pi/
  package.json        name, version = CLI version, keywords [pi-package], pi.skills, pi.extensions, peerDependencies
  skills/gabo-pi/SKILL.md
  extensions/gabo.ts  session_start guard (S4, S7) and tool_call gates (S5, S6)
  src/*.ts            pure functions (project detection, refusal text, acceptance parser) with bun tests
```

Flow: the user runs `pi install npm:<package>` once per machine. In a project, the `gabo-pi` skill bootstraps with the CLI (`gabo init`, then `gabo prompt`), and `gabo-runtime-pi` builds the agents from the project's own `.gabo/` copy as today. The extension only observes and gates; subagents stay with gentle-pi.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | What the package ships | A: the whole catalog except `gabo-reviewer` / B: one bootstrap skill plus the extension / C: the extension only | B. A package skill shadows a project skill of the same name (evidence above), so shipping the catalog would replace the project's version-pinned `.gabo/skills/` copy that `gabo check` and `gabo-version` lines rely on (D24); C leaves a fresh user with nothing to start from. | The role skills keep reaching Pi only through `gabo init` and `.pi/skills/` links; owner doubt 1 asks whether A is still wanted. |
| D2 | How the subagent package is detected in-session | A: run `pi list` from the extension / B: check `pi.getAllTools()` for the subagent tool names / C: look for `gentle-shell` on `PATH` | B, with a list of tool names that defaults to gentle-pi's `subagent_run`. It sees what this session really loaded, covers `gentle-shell` and project-scope installs, and spawns no process. `gabo-runtime-pi` keeps its `pi list` check at build time. | Another subagent package needs its tool name added to the list (owner doubt 4). |
| D3 | How "refuses to run" is enforced | A: notify only / B: notify and block writes to `.pi/agents/` / C: block every tool call in a gabo project | B. The refusal is visible at start and no gabo agent can be written without subagents; C would make Pi unusable for anything else in that folder. | The `gabo-runtime-pi` refusal stays the primary gate; the extension is a second, deterministic one. |
| D4 | The intent stage gate | A: no gate in the package / B: block `write`/`edit` of `plan.md` on a TS port of the one stage-order rule / C: run `gabo check` after each write and append failures to the tool result | B. A pre-write block needs the rule before the file exists, which `gabo check` cannot answer; the port is one rule, pinned by a parity test on the Go fixtures. | `bash` writes are not gated; `gabo check` in the reviewer stays authoritative (owner doubt 3). |
| D5 | Distribution in the first release | A: npm only / B: git only / C: both | A: `pi install npm:<package>@<version>`. Git sources install the repository root, which is the private workspace root; git needs a root `pi` manifest or a separate repository. | Needs an npm account and token held by the owner (owner doubts 2 and 5). |
| D6 | Versioning | A: lockstep with the CLI / B: independent semver | A: one version per release, the release tag `v<version>`, checked by S7 against `.gabo/VERSION`. | A package release follows every CLI release, even with no package change. |
| D7 | Refusal text source | A: copy in the extension / B: read the skill at runtime | A, with a test that the extension's text equals the quoted block in `gabo-runtime-pi`. The project may not have `.gabo/` yet when the warning matters. | Drift fails the test, not the user. |
| D8 | Local verification | A: unit tests only / B: unit tests plus a recorded manual run on local Pi (S9) | B. Pi has no headless test harness for packages; `--no-extensions -e` simulates the missing subagent package without uninstalling gentle-pi. | One manual step per release. |

## 3. UX and app flow (optional)

Not applicable: no screen or designed journey. The user-facing surface is two Pi notifications (S4, S7) and two block reasons (S5, S6), whose exact text the requirements fix.

## 4. DB migration steps (optional)

Not applicable: the change alters no stored data and no config shape.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| TRD 2 Stack ("The gabo CLI is Go") and TRD 11 testing strategy (Go-centric) | The package adds TypeScript code and `bun test` to the shipped product. | Plan change proposed (`global` scope): the CLI stays Go; runtime packages may be TypeScript in `packages/*`, tested with `bun test`. |
| TRD 2 Go CLI (D24: `gabo init` is the only catalog copier; agents built by runtime skills) | A package that ships skills is a second catalog distribution path. | D1: the package ships one bootstrap skill only and never replaces `.gabo/skills/`, `.pi/agents/` or `.pi/skills/` (S8). |
| D25 (the setup prompt has one source, embedded in the CLI) | The bootstrap skill could copy the prompt. | S3: the skill runs `gabo prompt` and holds no copy. |
| D20 (`gabo-reviewer` never in the project) and TRD 9.4 user-level reviewer | A package could carry a second reviewer copy. | S1: no `gabo-reviewer` in the package; `gabo init` keeps installing the user-level reviewer. |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Intent Q1, package scope: which skills does the package ship? Options: A the whole catalog except `gabo-reviewer` (shadows the project copy and its version pin); B one bootstrap skill `gabo-pi` plus the extension; C the extension only. | answered: B (D1). Recommendation B; carried to owner to confirm, because the intent's outcome says "gabo's skills". |
| 2 | Intent Q2, npm, git or both in the first release? Options: A npm only; B git only, with a `pi` manifest added to the repository root `package.json`; C both; D git from a separate small repository. | answered: A (D5). Recommendation A; carried to owner. |
| 3 | Is the intent stage gate (S6) in the first release? Options: A yes, TS port with a parity test (D4); B no, ship only the subagent guard and add the gate later; C instead of a block, run `gabo check` after writes under `docs/product/intents/` and append its failures. | carried to owner. Recommendation A; keep the same answer as intent 0001 Q2 so Claude Code and Pi gate alike. |
| 4 | Intent Q3, detection: which tool names count as a subagent package? Options: A gentle-pi's `subagent_run` only; B a fixed list including other verified packages (for example `pi-subagents-j0k3r`); C a list the user extends through a key in `gabo.yaml`. | answered: A as the default list (D2). Recommendation A now, B when another package is verified the way `gabo-runtime-pi` Prerequisite step 3 asks; C needs a config change (TRD 9.3). Carried to owner. |
| 5 | npm package name and owner account. Options: A unscoped `gabo-pi`; B scoped `@testeos-z/gabo-pi`; C `gabo-ai-pi`. | carried to owner. Recommendation B: the scope matches the GitHub organisation and the release assets, and avoids a squatted unscoped name. |
| 6 | Does the TS/bun addition need its own plan change before planning starts (section 5, row 1)? Options: A yes, a `global` plan change first; B record it in this spec's acceptance and amend the TRD with the plan. | carried to owner. Recommendation A, per D7. |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
