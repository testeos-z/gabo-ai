# Spec 0017 Any runtime engine without a Go change

| field | value |
|---|---|
| intent | [0017-any-runtime-engine-without-a-go-change](intent.md) |
| status | Draft (conditional: awaits the owner's decision on need, then owner acceptance) |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: section 3 and section 4 are not applicable) |
| date | 2026-11-11 |

**Condition.** Owner, verbatim (2026-11-11): "config.go sigue con la lista fija de motores. Es necesario solo si quieres agregar motores sin tocar Go; si no, es prioridad baja." The requirements in section 1 apply only if the owner says yes (open question 1 in section 6). If the owner says no, nothing in this spec is planned.

Mode is `auto`: the decisions marked **Proposed (auto)** are gabo-pm's proposals for the owner's one-pass approval.

Evidence base (checked 2026-11-11 on `origin/dev` at `0fd848e`):

- **Fixed list.** `apps/cli/internal/config/config.go` lines 22-34 define the four engine constants and `var runtimes = []Runtime{RuntimeClaude, RuntimeCodex, RuntimePi, RuntimeOpencode}`; line 241 `engineChoice` names the same four in the error text; `runtimeErrors` (line 248) validates `runtimes.*` against them.
- **Schema.** `schemas/gabo.project.schema.json` lines 251-256: `"engine"` is `enum ["claude", "codex", "pi", "opencode"]` and `"runtime"` is the same enum plus `null`. The copy at `apps/cli/internal/config/schemas/gabo.project.schema.json` is byte-equal (checked with `cmp`).
- **Existing runtime-skill rule.** `apps/cli/internal/slices/check/usecase.go` line 341 already reports `✗ runtime <rt> has no runtime skill gabo-runtime-<rt>` with the next step `NextRuntimeBuilder`. The builder skill (`.gabo/skills/gabo-runtime-builder/SKILL.md` step 7) stops when the installed gabo "rejects the `runtimes.*` value itself", which is the config list.
- **Reviewer engines.** TRD D27 and the owner decision of 2026-11-03 (intent 0002, 0004): the reviewer scripts run `claude`, `codex`, `pi` and `opencode`.
- **Flexible tier.** PRD section 6 and TRD D28 (owner 2026-11-04): an engine outside the four is flexible tier; intent 0006 S8 and S11 define its `gabo init` line and its engine-neutral `gabo check`.

## 1. Requirements

Conditional on the owner's decision (section 6, question 1).

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | When `runtimes.coder` or `runtimes.assistant` names an engine that has a valid runtime skill (D1) in `.gabo/skills/` or `.gabo/runtimes/`, `gabo` shall accept the name without a Go change: config validation accepts any well-formed name (D4), and `gabo check` checks the runtime skill. | Given `runtimes.coder: foo` and a valid `.gabo/runtimes/gabo-runtime-foo/SKILL.md`, When `gabo check` runs, Then no runtime-name failure is printed for `foo`, and the per-agent rules run for `foo`. Given the same name in `runtimes.assistant`, Then the same. | `config` unit test (a well-formed name passes); `check` test with a fixture runtime skill; golden output |
| S2 | When a name has no valid runtime skill, `gabo check` shall fail with today's line `✗ runtime <engine> has no runtime skill gabo-runtime-<engine>` and the next step that names `gabo-runtime-builder`, and exit 1. | Given `runtimes.coder: bar` and no `gabo-runtime-bar` skill, When `gabo check` runs, Then exactly that failure is printed with the builder next step and the exit code is 1. | existing pattern of `check/usecase_test.go` (the `claude` case) extended to an unknown name |
| S3 | `runtimes.reviewer` shall accept only the engines the reviewer scripts run: `claude`, `codex`, `pi` and `opencode` (TRD D27), whether or not a runtime skill exists for another engine (D3). | Given `runtimes.reviewer: bar` with a valid runtime skill for `bar`, When `gabo check` runs, Then the failure names the four engines and the reviewer-engine next step (`gabo-reviewer-config`), and exit 1. Given `runtimes.reviewer: codex`, Then no failure. | `config` unit test and `check` test; golden output |
| S4 | An accepted builder engine is flexible tier: `gabo init` writes nothing engine-specific for it and prints the one flexible-tier line (intent 0006 S8), and `gabo check` reads no engine file for it (intent 0006 S11). | Given `runtimes.coder: foo` with a valid skill, When `gabo init` runs, Then no engine key is written for `foo` and the flexible-tier line names `foo`. When `gabo check` runs, Then it prints no engine-file rule for `foo`. | `initcmd` and `check` tests with a fixture engine; golden output |
| S5 | The four named engines keep the full tier and their output: `gabo init` and `gabo check` output for `claude`, `codex`, `pi` and `opencode` stays byte-identical to the current golden files, and the two schema copies stay equal. | Given `go -C apps/cli test ./...`, Then it passes with only the S1 to S4 tests added and no existing golden file changed. Given the two schema files, Then `cmp` reports them equal. | existing golden tests; a test that the two schema copies are byte-equal |
| S6 | `gabo.yaml` shall accept any engine name that matches the pattern in D2, and the schema shall list the four known names as examples. | Given `runtimes.coder: foo` in `gabo.yaml`, When the schema validates it, Then it is accepted. Given `runtimes.coder: "Foo Bar"`, Then it is refused. | schema validation test; the pattern test in the config package |

## 2. Design

Affected components: `apps/cli/internal/config/config.go` (the runtime list, the validation and the `engineChoice` text), the two schema copies, `apps/cli/internal/slices/check` (the runtime-skill rule, which already exists) and `apps/cli/internal/slices/initcmd` (the flexible-tier line from intent 0006). The four named engines remain constants for the full tier.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | What makes a runtime skill valid enough to accept its engine name | (a) the skill exists in `.gabo/skills/gabo-runtime-<engine>/` or `.gabo/runtimes/gabo-runtime-<engine>/`, its frontmatter parses and it declares `metadata.agent_dir`; (b) (a) plus a recorded probe result from the builder | **Proposed (auto):** (a). Reason: `gabo check` is read-only and runs no probe; the builder runs its probe before it writes the skill. | A broken skill that passes (a) fails later in the per-agent rules. Open question 2 stays open for the owner. |
| D2 | How the schema accepts an open set of names | (a) replace the `enum` with a pattern (lower case letters, digits and `-`, starting with a letter) and list the four known names as `examples`; (b) keep the `enum` and add a builder-only value | **Proposed (auto):** (a). Reason: it lets a builder engine validate in the editor without a schema edit per engine. | Editors may suggest fewer names, and typos are caught by `gabo check` (S2), not by the schema. Open question 3. |
| D3 | Whether `runtimes.reviewer` accepts builder engines | (a) keep the four named engines, as the reviewer scripts run them (TRD D27); (b) accept any engine a reviewer script supports, read from the scripts | **Proposed (auto):** (a). Reason: a builder engine that the scripts cannot run would fail at review time. | `runtimes.reviewer` is unchanged. Open question 4. |
| D4 | Where the runtime-skill check runs | (a) config accepts any well-formed name and `gabo check` checks the skill (the rule at `check/usecase.go` line 341); (b) config reads the file system | **Proposed (auto):** (a). Reason: config stays pure (no file reads), and `gabo check` already owns the skill check. | Config tests change: a well-formed unknown name is no longer refused in config. |

## 3. UX and app flow (optional)

Not applicable: the change is CLI configuration and text output, with no screen. The lines in S2, S3 and S4 follow the existing `gabo check` and `gabo init` line shapes.

## 4. DB migration steps (optional)

Not applicable: `gabo.yaml` keys and their meaning stay the same, and no stored data changes.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| PRD R35 and TRD 2 | Today's code contradicts them: a new engine needs a Go change and a release. | resolved by this spec if the owner says yes. Until then the conflict stays open, and R35 and TRD 2 stay as written. |
| TRD D27 | The reviewer engines are fixed to the scripts' engines. | none: S3 keeps the four engines. |
| TRD D28 and PRD section 6 | An accepted builder engine is flexible tier, without a plugin and hooks. | none: S4. |
| PRD R33 | `gabo check` stays read-only. | none: D1 runs no probe and D4 keeps config pure. |
| intent 0005 (earlier draft, no spec) | The same problem under the old number; its folder has no spec. | not changed by this spec. Its open questions are carried to section 6. Its missing spec is a traceability gap that the owner decides on. |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Does the owner want engines accepted without a Go change? | open, owner decision. The requirements S1 to S6 apply only on yes. |
| 2 | Is a runtime skill valid with the checks in D1 (a), or does it also need a recorded probe result (D1 b)? | open. **Proposed (auto):** D1 (a). |
| 3 | Does the schema pattern in D2 (a) keep editor suggestions acceptable, or should the owner keep an explicit list? | open. **Proposed (auto):** D2 (a). |
| 4 | Should `runtimes.reviewer` accept builder engines once the scripts report their supported engines? | open. **Proposed (auto):** no, D3 (a). |
| 5 | Should `gabo check` report a builder engine as unverified until its probe result is recorded? | open. |

## 7. Acceptance

Filled by a human only. Open before acceptance: question 1 of section 6 (does the owner want engines accepted without a Go change?). If the answer is no, this spec is not planned.

| accepted by | date | notes |
|---|---|---|
