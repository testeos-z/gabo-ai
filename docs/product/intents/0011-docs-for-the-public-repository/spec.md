# Spec 0011 Docs for the public repository

| field | value |
|---|---|
| intent | [0011-docs-for-the-public-repository](intent.md) |
| status | Accepted |
| owner | gabo-pm |
| contributors | — |
| date | 2026-11-06 |

Evidence base (checked 2026-11-06 on `dev` at 8eacf4e):

- **The repository is public.** `gh repo view testeos-z/gabo-ai --json visibility` returns `PUBLIC`. Without any credential (`GIT_CONFIG_GLOBAL=/dev/null`, no `GH_TOKEN`, `GIT_TERMINAL_PROMPT=0`), `git ls-remote https://github.com/testeos-z/gabo-ai.git refs/heads/main` answers `d2d0fa5…`, so Claude Code and Codex can clone the marketplace anonymously. `https://raw.githubusercontent.com/testeos-z/gabo-ai/main/docs/assets/made-with-gabo-ai.png` returns HTTP 200.
- **`go install` works from the public proxy without `GOPRIVATE`.** `https://proxy.golang.org/github.com/testeos-z/gabo-ai/apps/cli/@v/list` lists v0.2.0, v0.2.2, v0.2.6 and v0.2.8; `@latest` is v0.2.8 (`refs/tags/apps/cli/v0.2.8`); `sum.golang.org` has its hash. With `GOPRIVATE`, `GONOPROXY` and `GONOSUMDB` unset, `GOPROXY=https://proxy.golang.org`, an empty git config and a fresh module cache, `go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@latest` (go1.27.2, linux/amd64) installed a binary whose `gabo version` prints `gabo v0.2.8`.
- **Texts that still say private** (`grep -rn -i 'GOPRIVATE\|is private\|while .* private\|setup-git\|can read it'`, excluding `docs/product/`):
  - `README.md` line 15 to 18 (install step: "Go needs your git credentials while the repository is private" and `GOPRIVATE=github.com/testeos-z/* go install …`), the Upgrading list that points to that command, Known limitations "**Private repository.**" (line 128), and the sticker note (line 157: "While `testeos-z/gabo-ai` is private, that image URL only loads for people with access to it …").
  - `docs/compatibility.md` line 39, "**Private repository.**".
  - `docs/setup-prompt.md` line 40, the `GOPRIVATE` sentence of the source build.
  - `plugins/claude-code/README.md` line 16 and its open-question row 3 ("Which marketplace hosts the plugin while the repository is private?"); `plugins/codex/README.md` line 16.
  - `packages/pi/README.md` line 90: "branch protection and GitHub Environments are not available on this private repository's plan".
  - `docs/cloud-agent-flow.md` line 103: "GitHub Free private repos have 2,000 Actions minutes per month and a $0 default spending limit". GitHub's billing docs: standard GitHub-hosted runners are free "In public repositories" (https://docs.github.com/en/billing/concepts/product-billing/github-actions).
  - `gabo init` report steps: `apps/cli/internal/engineconfig/claude.go` `claudeSteps` ("while testeos-z/gabo-ai is private, give git credentials that can read it (Claude Code clones the marketplace from GitHub)") and `codex.go` `codexSteps` ("while testeos-z/gabo-ai is private, set up git credentials that can read it"), with their tests `claude_test.go`, `codex_test.go` and the golden files `apps/cli/internal/slices/initcmd/testdata/engines-all.golden` and `init-engines.golden`.
- **Runtime and setup skills.** No catalog skill (`apps/cli/internal/skills/gabo-runtime-*`, `gabo-setup`), no `plugins/codex/skills/` copy and no embedded setup prompt (`apps/cli/internal/setupprompt/setup-prompt.md`) tells the user to give git credentials to read gabo-ai; the credential text of the brief lives only in the `gabo init` report lines above.
- `docs/product/PRD.md` and `docs/product/TRD.md` hold no private-repository text (TRD line 558 "private state files" is about engine state, unrelated). Accepted intent folders (0001 spec and plan, 0006 spec open question 7) record the private repository as it was then.

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | The install instructions shall use the plain command `go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@latest` with no `GOPRIVATE` and no credential sentence: `README.md` (install step and Upgrading) and `docs/setup-prompt.md` (the source-build note loses its `GOPRIVATE` sentence and keeps `gabo version`). Owner, verbatim: "Verify go install without GOPRIVATE works from the public module proxy." | Given the docs Then no `GOPRIVATE`; Given a clean environment (no `GOPRIVATE`, `GONOPROXY`, `GONOSUMDB`, `GOPROXY=https://proxy.golang.org`, empty git config, fresh module cache) Then the documented command installs `gabo` and `gabo version` prints the latest release. | doc review; the install probe, run again by the task and quoted in its PR |
| S2 | Every user-facing text that says the repository is private or asks for git credentials to read it shall be removed or updated to the public state: `README.md` Known limitations "**Private repository.**" (removed), the sticker note (removed: the snippet works as written), `docs/compatibility.md` "**Private repository.**" (removed), `plugins/claude-code/README.md` and `plugins/codex/README.md` (the credential sentence removed; open-question row 3 records that the repository is public since 2026-11-06, so no credentials are needed, and keeps the plugin-directory recommendation as not decided). | Given `grep -rn -i 'GOPRIVATE\|is private\|while .* private\|setup-git' README.md docs/*.md plugins/*/README.md packages/*/README.md apps/cli/internal` Then no match that describes gabo-ai as private. | grep; doc review |
| S3 | The `gabo init` report shall stop printing the credential steps: the last entry of `claudeSteps` and of `codexSteps` is removed, and `claude_test.go`, `codex_test.go` and the two `initcmd` golden files follow. Owner, verbatim: "gabo init report lines and runtime/setup skills that tell the user to give git credentials, with their tests". | Given `gabo init` on a project with `claude` and `codex` Then no step mentions credentials or a private repository; Given `go -C apps/cli test ./...` Then it passes. | Go unit tests; golden output |
| S4 | The maintainer docs shall state the facts of a public repository: `packages/pi/README.md` no longer says branch protection and GitHub Environments are unavailable, and states that they are available but not configured (`pi/pkg-up` is not protected and the repository has no environment, checked 2026-11-06), while the owner's risk acceptance of 2026-11-04 stays as recorded (D2); `docs/cloud-agent-flow.md` replaces the private-repository Actions minutes bullet with the public-repository fact (standard GitHub-hosted runners are free in public repositories, with the source URL). | Given the two files Then neither describes gabo-ai as private, and the risk acceptance quote is unchanged. | doc review |
| S5 | Runtime and setup skills shall be checked for credential text: none holds any today (evidence base), so the task records that grep in its PR and changes no skill. | Given `grep -rn -i 'credential\|private' apps/cli/internal/skills/gabo-runtime-*/SKILL.md apps/cli/internal/skills/gabo-setup/SKILL.md apps/cli/internal/setupprompt/` Then no match about reading gabo-ai. | grep in the PR |
| S6 | `gabo check` impact (TRD 8.5): none. `gabo check` reads no install text and no report step; it verifies the keys `gabo init` writes (TRD 9.4), which this intent does not change. The plan and each PR state `gabo check impact: none` with this reason. | Given the plan and PRs Then the line with the reason. | review |
| S7 | Accepted intent folders and `docs/product/archive/` stay unchanged: they record the private repository as it was. | Given the diff Then no file under `docs/product/intents/0001-*` to `0009-*` or `docs/product/archive/`. | review of the diff |
| S8 | The change shall ship in gabo release 0.2.10; the version bump is not a task of this intent (the coordinator does it). | Given the release Then it holds this intent's merged tasks. | release review |

## 2. Design

Text-only change in the docs, two plugin READMEs, one package README and two report step lists, with their tests. No behavior of `gabo init` or `gabo check` changes beyond the two report lines.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Private-repository texts | A: remove them / B: keep them as "earlier, while private" notes | A, **Proposed (auto)**: the owner's brief says "Remove or update", and a historical note in user docs is noise; history stays in the accepted intents and the archive (S7). | Fewer lines; plugin row 3 is updated, not removed, because it is an open-question record. |
| D2 | `packages/pi/README.md` publishing risk | A: update the text to the public facts only; locking down `pi/pkg-up` and `NPM_TOKEN` is a later change / B: also protect `pi/pkg-up` and move `NPM_TOKEN` into a GitHub Environment in this intent | **Owner question (open question 1), not decided.** Proposed answer A: the brief is a docs change, and protecting a branch or moving a secret changes repository settings that only the owner can approve. Doubt research could not settle: the 2026-11-04 risk acceptance was partly justified by the private plan, which no longer applies. | With A, S4 states the facts and the acceptance stands; with B, the plan adds an owner-run settings task. |

## 3. UX and app flow (optional)

Not applicable: documentation and two CLI report lines; no screen or designed journey.

## 4. DB migration steps (optional)

Not applicable: no stored data changes.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| Intent 0006 open question 7 (the credential note while private) and S9 (report steps) | S3 removes the note. | The condition of that answer ("while private") no longer holds; S3 updates the steps. Intent 0006 stays as recorded (S7). |
| Intent 0001 open question 3 (marketplace while private) | S2 updates the plugin README row. | The README row records the public state; the 0001 spec stays as recorded (S7). |
| `review.human_only_paths` (`plugins/`) | S2 edits `plugins/claude-code/README.md` and `plugins/codex/README.md`. | The coder opens the PR; the merge needs the owner or the coordinator with the owner's delegation. |
| TRD 8.5 (`gabo check` impact) | — | S6: none, with the reason. |
| PRD, TRD | — | No PRD or TRD text describes the repository as private; no amendment. |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Intent Q1: does locking down `pi/pkg-up` and `NPM_TOKEN` (branch protection, a GitHub Environment) belong to this intent, now that the public repository has them? | open, passed to the owner with the proposed answer A and its reasoning (D2); the `packages/pi/README.md` task waits for the answer. |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
| leomerida15 (owner) | 2026-11-06 | Accepted by the owner, with every **Proposed (auto)** answer (open question 1 stays with the owner), answering whether to launch intents 0010 and 0011 for 0.2.10 on the cloud agent: "si quiero el PR de dev a main para mi revision". Recorded by gabo-pm on the owner's instruction, relayed verbatim by the coordinator in the owner's session of 2026-11-06. |
