# Plan 0006 Engine integration at init

| field | value |
|---|---|
| intent | [0006-engine-integration-at-init](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | 2026-11-04 |

The spec was accepted by the owner on 2026-11-04 (spec section 7), with every **Proposed (auto)** answer, the target release 0.2.6 and the Pi integration through `.pi/settings.json`. `gabo.yaml` mode is `auto`: every fixed question below is answered by `gabo-dev-lead`, marked **Proposed (auto)** with a one-line reason, for one-pass approval.

## Fixed questions

| id | answer |
|---|---|
| Q1 | **Proposed (auto)**: the pure engine-config package first (engine set and report, then one engine per task), then the `gabo init` wiring that writes each engine's project config, then the `gabo check` rule, help text and skill lines, the setup prompt step, the 0.2.6 release bump, and the owner smoke (Steps). Reason: every rule is a pure Go function tested offline before any file is written. |
| Q2 | **Proposed (auto)**: eleven tasks, each one PR of at most about 400 changed lines (Tasks). Reason: each engine's merge rule, the init wiring, the check rule, the docs-in-code lines, the setup prompt and the release are separately reviewable behaviors. |
| Q3 | **Proposed (auto)**: see the `spec refs` column. Reason: each task names the requirements and decisions it proves. |
| Q4 | **Proposed (auto)**: see `depends_on` and `unblocks`, acyclic. Cross-intent: the intent 0007 documentation tasks depend on the tasks whose behavior they document (T2, T3, T4, T6, T9, T10). Reason: documentation follows the merged behavior. |
| Q5 | **Proposed (auto)**: see `acceptance` and `size`; every code task starts from a failing Go test. Reason: TRD test-first convention. |
| Q6 | **Proposed (auto)**: yes, S1 to S15 each have at least one task (S13's README half through `0007/T1` and `0007/T4`, on top of T8). Reason: checked against the spec table. |
| Q7 | **Proposed (auto)**: see Risks to the plan. |
| Q8 | **Proposed (auto)**: see Open questions; none blocks a task. |

Plan items the spec left to `gabo-dev-lead`, answered here:

- **Package placement**: **Proposed (auto)** one pure package `apps/cli/internal/engineconfig/` (no ports, no I/O: bytes in, bytes and report lines out) holds the engine set, the per-engine merge rules, the key table and the report shape; the `initcmd` slice writes through its `Store` port (T6) and the `check` slice reads through its own (T7). Reason: both slices use the same key table (spec Shape) and slices must not import each other (AGENTS.md layout), like `internal/config/`.
- **Pi refusal text source**: **Proposed (auto)** T4 moves `SubagentRefusal` from `internal/slices/check/facts.go` into `engineconfig` and the check slice refers to it there, unchanged. Reason: spec S7 asks for the one Go source the hook mode uses.
- **TOML reader (spec open question 5, accepted)**: **Proposed (auto)** `github.com/pelletier/go-toml/v2`, read-only, to detect existing tables and keys; writes stay append-only text. Reason: spec D5.

## Steps

1. Engine set, unknown-engine line and the `engine integration:` report shape in `engineconfig` (T1). Delivers S1, S8, part of S9.
2. One merge rule per engine, pure and golden-tested: Claude Code (T2), Codex with the TOML reader and the fallback flag (T3), Pi with the `.pi/settings.json` package entry and the refusal text (T4), OpenCode report only (T5). Delivers S2, S3, S4, S6, S7 and the parse half of S5.
3. `gabo init` writes each engine's project config on every successful path, refuses symlinks and wrong file kinds, and ends with the report and exit code (T6). Delivers S5, S9, S10.
4. `gabo check` verifies the keys read-only (T7). Delivers S11.
5. Help text and the runtime skills' engine-config lines (T8); the setup prompt runs `gabo init` again after `gabo.yaml` (T9). Delivers S13, S15.
6. Release 0.2.6 bump (T10). Delivers S14.
7. Owner smoke on Claude Code, Codex and Pi (T11). Delivers S12.

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | Engine set from `runtimes.*`, unknown-engine line and the `engine integration:` report shape | S1, S8, S9, D1, D10 | — | T2, T3, T4, T5 | Test first: failing Go tests in `apps/cli/internal/engineconfig/engines_test.go`: `runtimes` `coder: codex`, `reviewer: claude`, `assistant: claude` yields the set `claude`, `codex` in that order, and `opencode`, `pi`, then other names in appearance order; each engine once; an unknown name (`acme`) yields `acme: no gabo integration known; nothing written` and no failure; the report renders `engine integration:`, one section per engine in set order, then `you still need to:` with the collected steps, and a failure flag when any `✗` line exists; a grep test proves the package calls no `exec.LookPath`, imports no `os/exec` and reads no home path. Edit surfaces: `apps/cli/internal/engineconfig/engines.go`, `apps/cli/internal/engineconfig/report.go`, `apps/cli/internal/engineconfig/engines_test.go`, `apps/cli/internal/engineconfig/report_test.go`. | S |
| T2 | Claude Code rule: `extraKnownMarketplaces."gabo-ai"` and `enabledPlugins."gabo@gabo-ai"` merged into `.claude/settings.json` | S2, S4, S5, S9, D3, D5, D6 | T1 | T6, T7, 0007/T4 | Test first: failing golden tests in `apps/cli/internal/engineconfig/claude_test.go`: missing file → exactly the two keys with the spec S2 values and a `wrote .claude/settings.json` line naming both; a file with `permissions`, `hooks` and another `enabledPlugins` entry keeps those bytes and order, keys inserted; both keys present → no change, `unchanged`; `enabledPlugins."gabo@gabo-ai": false` → no change, one `kept` line naming `false`; `{` → no change, one `✗ .claude/settings.json:` line; the `you still need to:` steps of S9 for `claude` (trust the folder; the credential note while the repository is private). Edit surfaces: `apps/cli/internal/engineconfig/claude.go`, `apps/cli/internal/engineconfig/claude_test.go`, `apps/cli/internal/engineconfig/testdata/claude/`, `apps/cli/internal/engineconfig/engines.go` (one table entry). | M |
| T3 | Codex rule: `[marketplaces.gabo-ai]` and `[plugins."gabo@gabo-ai"]` appended to `.codex/config.toml`, with the fallback flag | S3, S4, S5, S9, D4, D5, D7 | T1 | T6, T7, 0007/T4 | Test first: failing golden tests in `apps/cli/internal/engineconfig/codex_test.go`: missing file → exactly the two tables (`source_type = "git"`, `source = "https://github.com/testeos-z/gabo-ai.git"`, `enabled = true`); a file with comments, `model` and `[agents]` is a byte prefix of the result; both tables present (also as dotted keys) → no change, `unchanged`; `enabled = false` → no change, one `kept` line; invalid TOML → no change, one `✗ .codex/config.toml:` line; with the fallback flag set the report prints `codex plugin marketplace add testeos-z/gabo-ai` and `codex plugin add gabo@gabo-ai`; the S9 steps for `codex` (trust the project, trust both hooks in `/hooks`, credential note). Adds `github.com/pelletier/go-toml/v2`, read-only use. Edit surfaces: `apps/cli/internal/engineconfig/codex.go`, `apps/cli/internal/engineconfig/codex_test.go`, `apps/cli/internal/engineconfig/testdata/codex/`, `apps/cli/internal/engineconfig/engines.go` (one table entry), `apps/cli/go.mod`, `apps/cli/go.sum`. | M |
| T4 | Pi rule: `npm:@gabo-ai/pi@<version>` entry in `.pi/settings.json` `packages`, and the refusal text from one Go source | S7, S4, S5, S9, D9, D11 | T1 | T6, T7, 0007/T5 | Test first: failing golden tests in `apps/cli/internal/engineconfig/pi_test.go`: missing file and version `0.2.6` → exactly `{"packages": ["npm:@gabo-ai/pi@0.2.6"]}` and a `wrote .pi/settings.json` line; a file with `theme`, `extensions` and `packages: ["npm:gentle-pi"]` keeps them in order and appends the entry; `npm:@gabo-ai/pi@0.2.5` or an object entry with that `source` → no change, one `kept` line naming both versions; entry present → `unchanged`; invalid JSON → one `✗ .pi/settings.json:` line; a source-build version → no change and `pi: development build <version>; no published @gabo-ai/pi to pin; nothing written`; the `pi` section holds the refusal text byte-equal to the `gabo-runtime-pi` quoted block; `SubagentRefusal` moves into `engineconfig` and `internal/slices/check` uses it unchanged (its existing facts tests pass untouched). Edit surfaces: `apps/cli/internal/engineconfig/pi.go`, `apps/cli/internal/engineconfig/pi_test.go`, `apps/cli/internal/engineconfig/testdata/pi/`, `apps/cli/internal/engineconfig/engines.go` (one table entry), `apps/cli/internal/slices/check/facts.go`. | M |
| T5 | OpenCode rule: report the stage-gate plugin, write nothing | S6, S9 | T1 | T6 | Test first: failing Go tests in `apps/cli/internal/engineconfig/opencode_test.go`: with `.opencode/plugins/gabo-stage-gate.js` present the section says `opencode: stage-gate plugin in place (built by gabo-runtime-opencode)`; missing, it names asking the agent to build the agents with `gabo-runtime-opencode` as the `you still need to:` step; no file change in either case. Edit surfaces: `apps/cli/internal/engineconfig/opencode.go`, `apps/cli/internal/engineconfig/opencode_test.go`, `apps/cli/internal/engineconfig/engines.go` (one table entry). | S |
| T6 | `gabo init` writes each engine's project config on every successful path and ends with the report | S5, S9, S10, S1, D2, D10 | T2, T3, T4, T5 | T8, T9, T10, T11, 0007/T1, 0007/T3, 0007/T4, 0007/T5 | Test first: failing tests of `Run` in `apps/cli/internal/slices/initcmd/usecase_test.go` with the fake store: catalog copied, updated and current paths each run the engine step and add the S9 block (current keeps `catalog is current (…)`); fresh project (no `gabo.yaml`) writes no engine file and prints `engine integration: no gabo.yaml yet; after your agent writes it, run gabo init again` before the setup prompt; invalid `gabo.yaml` or no `runtimes` → one `✗ gabo.yaml:` line, no engine file, exit 1; a symlinked `.claude/settings.json`, a non-regular file, or `.claude`, `.codex` or `.pi` a symlink → untouched, one `✗` line, other engines still run, `.gabo/VERSION` still written, exit 1; only changed files are written; `gabo.yaml` bytes unchanged on every path; a process runner that fails when called is never called; a temporary `HOME`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `PI_CODING_AGENT_DIR` and `GENTLE_PI_AGENT_HOME` get no write; a golden-output test of the command with all four engines on an empty project with a `gabo.yaml`; the newer-stamp path is unchanged. Edit surfaces: `apps/cli/internal/slices/initcmd/usecase.go`, `apps/cli/internal/slices/initcmd/engines.go`, `apps/cli/internal/slices/initcmd/usecase_test.go`, `apps/cli/internal/slices/initcmd/engines_test.go`, `apps/cli/internal/slices/initcmd/ports.go`, `apps/cli/internal/slices/initcmd/command_test.go`, `apps/cli/internal/slices/initcmd/testdata/`, `apps/cli/cmd/gabo/factories.go` (only if the store needs a new read). | M |
| T7 | `gabo check` verifies the engine integration keys read-only | S11, D6 | T2, T3, T4 | T10 | Test first: failing Go tests in `apps/cli/internal/slices/check/engines_test.go`: `runtimes.coder: claude` and a `.claude/settings.json` without `enabledPlugins."gabo@gabo-ai"` → `✗ .claude/settings.json: enabledPlugins."gabo@gabo-ai" missing; run gabo init`; the key set to `false` passes; the same for both Codex tables; `runtimes.coder: pi` without a `@gabo-ai/pi` `packages` entry → `✗ .pi/settings.json: packages entry npm:@gabo-ai/pi missing; run gabo init`, any version passes; engines not in the set are not checked; `gabo check` writes nothing; existing golden outputs of projects without these engines are unchanged. Edit surfaces: `apps/cli/internal/slices/check/engines.go`, `apps/cli/internal/slices/check/engines_test.go`, `apps/cli/internal/slices/check/usecase.go`, `apps/cli/internal/slices/check/testdata/engines/`, `apps/cli/internal/slices/check/golden_test.go` (only new cases). | M |
| T8 | `gabo init` help text and the `gabo-runtime-claude` and `gabo-runtime-pi` engine-config lines | S13 | T6 | T10 | Test first: failing assertions in `apps/cli/internal/slices/initcmd/command_test.go`: `gabo init --help` names `.claude/settings.json`, `.codex/config.toml` and `.pi/settings.json` as the only engine files it writes and says it never writes user-level engine config; failing assertions in `runtimeclaude_test.go` and `runtimepi_test.go`: the lines "Claude Code needs no engine key today, so this skill writes no settings file" and "Pi needs no engine key today, so this skill writes no Pi config" are replaced by "`gabo init` writes the project-level gabo integration keys (intent 0006); this skill writes no settings file" (Pi: "no Pi config, and never runs `pi install`"); the rest of each rule is unchanged. Edit surfaces: `apps/cli/internal/slices/initcmd/command.go`, `apps/cli/internal/slices/initcmd/command_test.go`, `apps/cli/internal/skills/gabo-runtime-claude/SKILL.md`, `apps/cli/internal/skills/gabo-runtime-pi/SKILL.md`, `apps/cli/internal/skills/runtimeclaude_test.go`, `apps/cli/internal/skills/runtimepi_test.go`. | S |
| T9 | Setup prompt runs `gabo init` again after writing `gabo.yaml` | S15, D8 | T6 | T10, 0007/T1, 0007/T2 | Test first: failing assertion in `apps/cli/internal/setupprompt/setupprompt_test.go`: the embedded prompt holds the step "run `gabo init` again" after the `gabo.yaml` write step and before the agent-build step; the existing byte-equality test with the marked block of `docs/setup-prompt.md` passes after the same edit there (only the marked block; the page prose is `0007/T2`). Edit surfaces: `apps/cli/internal/setupprompt/setup-prompt.md`, `apps/cli/internal/setupprompt/setupprompt_test.go`, `docs/setup-prompt.md` (marked prompt block only). | S |
| T10 | Release 0.2.6: CLI, plugin manifests and `@gabo-ai/pi` read `0.2.6` | S14 | T6, T7, T8, T9 | 0007/T6 | Test first: a failing check that `plugins/claude-code/.claude-plugin/plugin.json`, `plugins/codex/.codex-plugin/plugin.json` and `packages/pi/package.json` hold `"version": "0.2.6"` (the existing plugin and package tests, or a new assertion in `apps/cli/internal/skills/plugin_test.go`); `packages/pi/package.json` has `"name": "@gabo-ai/pi"` (from PR #262); `bun install --frozen-lockfile` passes; `gabo version` on the tagged build prints `0.2.6`; `go -C apps/cli test ./...` and `cd packages/pi && bun test` pass. Human-merged: `plugins/` and `.claude/` are `review.human_only_paths`. Edit surfaces: `plugins/claude-code/.claude-plugin/plugin.json`, `plugins/codex/.codex-plugin/plugin.json`, `packages/pi/package.json`, `bun.lock`, `.claude/agents/*.md` (`gabo-version` line only). | S |
| T11 | Owner smoke of `gabo init` engine integration on Claude Code, Codex and Pi | S12, D4, D11 | T6 | — | Owner runs spec S12 in a scratch project with `runtimes` naming `claude`, `codex` and `pi`: `gabo init`, open each engine, trust the project, and records on the task whether the gabo plugin (Claude Code, Codex) or `@gabo-ai/pi` (`pi list`) is listed and its hooks or extension load with no other command; the Codex result sets the T3 fallback flag (a flip is a task-scope change to T3's code in its own PR); the Pi result confirms or refutes D11's assumption (refuted: add the `pi install -l` fallback line, task-scope change). | S |

## Tracker

Intent tracker item: [#261](https://github.com/testeos-z/gabo-ai/issues/261) under product card #171.

| task | issue |
|---|---|
| T1 | [#264](https://github.com/testeos-z/gabo-ai/issues/264) |
| T2 | [#265](https://github.com/testeos-z/gabo-ai/issues/265) |
| T3 | [#266](https://github.com/testeos-z/gabo-ai/issues/266) |
| T4 | [#267](https://github.com/testeos-z/gabo-ai/issues/267) |
| T5 | [#268](https://github.com/testeos-z/gabo-ai/issues/268) |
| T6 | [#269](https://github.com/testeos-z/gabo-ai/issues/269) |
| T7 | [#270](https://github.com/testeos-z/gabo-ai/issues/270) |
| T8 | [#271](https://github.com/testeos-z/gabo-ai/issues/271) |
| T9 | [#272](https://github.com/testeos-z/gabo-ai/issues/272) |
| T10 | [#273](https://github.com/testeos-z/gabo-ai/issues/273) |
| T11 | [#274](https://github.com/testeos-z/gabo-ai/issues/274) |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| PR #262 (rename to `@gabo-ai/pi` and the npm publish pipeline) is not merged into `dev` before T10. | T4, T10, 0007/T5 | T4 writes the name from its own Go constant and does not read `packages/pi/package.json`; T10 checks the name and waits for #262. |
| `@gabo-ai/pi@0.2.6` exists on npm only after the 0.2.6 merge to `main` runs the publish pipeline; an earlier `gabo init` pins a version Pi cannot install yet. | T4, T11 | Released binaries only write released versions; a source build writes nothing for `pi` (S7). T11 runs after the publish. |
| Codex does not install the plugin from the two project tables alone. | T3, T11 | The fallback commands stay printed (D4); T11 records it. |
| Pi does not install a project npm package on trust without `pi install`. | T4, T11 | D11 fallback line added as a task-scope change after T11. |
| `plugins/` and `.claude/` are human-only paths, so T10 cannot be merged by the wave. | T10 | The coder opens the PR and the owner merges it, with the single PR to `main`. |
| T2, T3, T4 and T5 each add one entry to `engineconfig/engines.go`. | T2–T5 | One table line each; the later PR rebases on `dev` (parallelism 3). |

## Open questions

1. D11 assumption (Pi installs a declared project package after trust without `pi install`) is unverified until T11 (owner).
2. The S12 smoke and T10's human-only merge need the owner; the rest of the intent runs unattended (owner).
