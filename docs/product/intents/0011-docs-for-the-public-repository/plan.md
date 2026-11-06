# Plan 0011 Docs for the public repository

| field | value |
|---|---|
| intent | [0011-docs-for-the-public-repository](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | 2026-11-06 |

The spec was accepted by the owner on 2026-11-06 (spec section 7), with every **Proposed (auto)** answer. The owner answered spec open question 1 (D2) on 2026-11-06 with "ok", accepting proposal A: `packages/pi/README.md` states the public facts only; locking down `pi/pkg-up` and `NPM_TOKEN` is a later change. `gabo.yaml` mode is `auto`: every fixed question below is answered by `gabo-dev-lead`, marked **Proposed (auto)** with a one-line reason, for one-pass approval.

## Fixed questions

| id | answer |
|---|---|
| Q1 | **Proposed (auto)**: the four tasks touch disjoint files and can run in parallel: the `gabo init` report steps (T1), the user and operating docs (T2), the plugin READMEs (T3) and the Pi package README (T4) (Steps). Reason: text-only changes with no order between them; T1 goes first in the wave because intent 0010 T2 edits the same Go file. |
| Q2 | **Proposed (auto)**: four tasks, each one small PR (Tasks). Reason: the report lines carry tests and golden files; `plugins/` is a human-only path and merges separately; the Pi README records an owner decision. |
| Q3 | **Proposed (auto)**: see the `spec refs` column. Reason: each task names the requirements it proves. |
| Q4 | **Proposed (auto)**: see `depends_on` and `unblocks`, acyclic. Cross-intent: T1 unblocks `0010/T2` (both edit `engineconfig/claude.go` and the two `initcmd` golden files), T2 unblocks `0010/T11` (both edit `README.md`, `docs/compatibility.md` and `docs/setup-prompt.md`). Reason: serial edits of the same files avoid merge conflicts. |
| Q5 | **Proposed (auto)**: see `acceptance` and `size`; T1 starts from failing Go tests, the docs tasks from a failing grep and, for T2, the clean-environment install probe. Reason: TRD test-first convention. |
| Q6 | **Proposed (auto)**: yes, S1 to S7 each have at least one task; S8 (release 0.2.10) has none, on the coordinator's instruction of 2026-11-06 (the version bump is not part of this plan). Reason: checked against the spec table. |
| Q7 | **Proposed (auto)**: see Risks to the plan. |
| Q8 | **Proposed (auto)**: see Open questions. |
| Q9 | **Proposed (auto)**, applying intent 0008 S10 (TRD 8.5): no `gabo check` update (spec S6): `gabo check` reads no install text and no report step, and this intent changes no key `gabo init` writes; every task states `gabo check impact: none` with that reason. Reason: owner rule, verbatim: "1. pondremos una regla cada cambio debe revisar si es necesario actualizar el check". |

## Steps

1. `gabo init` stops printing the credential steps (T1). Delivers S3, and records the S5 grep.
2. `README.md`, `docs/setup-prompt.md`, `docs/compatibility.md` and `docs/cloud-agent-flow.md` state the public repository: plain install command verified from the public proxy, no private-repository limitation, no sticker note, the public-repository Actions fact (T2). Delivers S1, S2 (user docs), S4 (`docs/cloud-agent-flow.md`).
3. The Claude Code and Codex plugin READMEs drop the credential sentence and update open-question row 3 (T3). Delivers S2 (plugins).
4. `packages/pi/README.md` states that branch protection and GitHub Environments are available but not configured, with the 2026-11-04 risk acceptance unchanged (T4, D2 A as answered by the owner). Delivers S4 (Pi).
5. Every task keeps accepted intent folders and `docs/product/archive/` unchanged (S7) and states `gabo check impact: none` (S6, Q9).

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

No PR of this plan touches `docs/product/intents/0001-*` to `0009-*` or `docs/product/archive/` (spec S7), nor `.claude/`, `CLAUDE.md`, `AGENTS.md` or `gabo.yaml`. `plugins/` is a `review.human_only_paths` entry: T3 merges only with the owner or the coordinator holding the owner's delegation. No catalog skill changes in this plan (spec S5), so no `plugins/codex/skills/` copy changes. Every PR body states `gabo check impact: none` with the spec S6 reason.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | `gabo init` report drops the private-repository credential steps | S3, S5, S6, S7 | — | 0010/T2 | Test first: failing expectations in `apps/cli/internal/engineconfig/claude_test.go` and `codex_test.go` (the last step of `claudeSteps` and `codexSteps` absent) and in the `initcmd` golden files; then `claudeSteps` and `codexSteps` lose their last entry; `gabo init` on a project with `claude` and `codex` prints no step that mentions credentials or a private repository; `go -C apps/cli test ./...` passes. The PR records the S5 grep `grep -rn -i 'credential\|private' apps/cli/internal/skills/gabo-runtime-*/SKILL.md apps/cli/internal/skills/gabo-setup/SKILL.md apps/cli/internal/setupprompt/` with no match about reading gabo-ai, and changes no skill. gabo check impact: none (spec S6: `gabo check` reads no report step). Edit surfaces: `apps/cli/internal/engineconfig/claude.go`, `apps/cli/internal/engineconfig/claude_test.go`, `apps/cli/internal/engineconfig/codex.go`, `apps/cli/internal/engineconfig/codex_test.go`, `apps/cli/internal/slices/initcmd/testdata/engines-all.golden`, `apps/cli/internal/slices/initcmd/testdata/init-engines.golden`. | S |
| T2 | User and operating docs describe the public repository | S1, S2, S4, S6, S7 | — | 0010/T11 | Test first: `grep -n -i 'GOPRIVATE\|is private\|while .* private\|setup-git' README.md docs/setup-prompt.md docs/compatibility.md docs/cloud-agent-flow.md` matches before the change and finds nothing describing gabo-ai as private after it. `README.md`: the install step is the plain `go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@latest` with no credential sentence, the Upgrading list points to it, Known limitations "**Private repository.**" and the sticker note are removed; `docs/setup-prompt.md`: the source-build note loses its `GOPRIVATE` sentence and keeps `gabo version` (prose outside the marked prompt block; the embedded prompt is unchanged and `go -C apps/cli test ./internal/setupprompt/...` passes); `docs/compatibility.md`: "**Private repository.**" removed; `docs/cloud-agent-flow.md`: the private-repository Actions minutes bullet becomes "standard GitHub-hosted runners are free in public repositories" with https://docs.github.com/en/billing/concepts/product-billing/github-actions. The install probe is run again and quoted in the PR: with `GOPRIVATE`, `GONOPROXY` and `GONOSUMDB` unset, `GOPROXY=https://proxy.golang.org`, `GIT_CONFIG_GLOBAL=/dev/null`, and `GOMODCACHE`, `GOCACHE` and `GOBIN` in one `mktemp -d` directory (removed afterwards with `rm -rf`; never `go clean -modcache`), the documented command installs `gabo` and `gabo version` prints the latest release. gabo check impact: none (spec S6: `gabo check` reads no install text). Edit surfaces: `README.md`, `docs/setup-prompt.md` (prose outside the marked prompt block only), `docs/compatibility.md`, `docs/cloud-agent-flow.md`. | S |
| T3 | Plugin READMEs describe the public repository | S2, S6, S7 | — | — | Test first: `grep -n -i 'GOPRIVATE\|is private\|while .* private\|credential' plugins/claude-code/README.md plugins/codex/README.md` matches before the change; after it, the credential sentence (line 16 of each) is gone and the Claude Code open-question row 3 records that the repository is public since 2026-11-06, so no credentials are needed, and keeps the plugin-directory recommendation as not decided; no other match describes gabo-ai as private; `go -C apps/cli test ./internal/skills/...` passes (`plugin_test.go`, `codexplugin_test.go`). gabo check impact: none (spec S6). Edit surfaces: `plugins/claude-code/README.md`, `plugins/codex/README.md` (`review.human_only_paths`). | S |
| T4 | `packages/pi/README.md` states the public-repository facts | S4, S6, S7, D2 | — | — | Owner answered D2 on 2026-11-06 ("ok", proposal A). Test first: `grep -n -i 'private' packages/pi/README.md` matches line 90 before the change; after it, the text says branch protection and GitHub Environments are available on the public repository but not configured (`pi/pkg-up` is not protected and the repository has no environment, re-checked with `gh api repos/testeos-z/gabo-ai/branches/pi/pkg-up/protection` and `gh api repos/testeos-z/gabo-ai/environments` and quoted in the PR), that locking them down is a later change, and the owner's risk acceptance of 2026-11-04 stays word for word (diff shows the quote unchanged); no repository setting changes. gabo check impact: none (spec S6). Edit surfaces: `packages/pi/README.md`. | S |

## Tracker

Intent tracker item: [#355](https://github.com/testeos-z/gabo-ai/issues/355) under product card #171.

| task | issue |
|---|---|
| T1 | [#369](https://github.com/testeos-z/gabo-ai/issues/369) |
| T2 | [#370](https://github.com/testeos-z/gabo-ai/issues/370) |
| T3 | [#371](https://github.com/testeos-z/gabo-ai/issues/371) |
| T4 | [#372](https://github.com/testeos-z/gabo-ai/issues/372) |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| The public module proxy serves a stale `@latest` while a release is tagged. | T2 | The probe quotes the version it installed; the docs name no version. |
| Intent 0010 edits the same files. | T1, T2 | `0010/T2` waits for T1 and `0010/T11` for T2. |
| `plugins/` is a human-only path. | T3 | The coder opens the PR; the owner or the coordinator with the owner's delegation merges. |
| A cleanup edit reaches an accepted intent folder or the archive. | all | Each PR is checked against S7 before review. |

## Open questions

1. S8 (release 0.2.10) has no plan task: the version bump is done by the coordinator, on the coordinator's instruction of 2026-11-06.
