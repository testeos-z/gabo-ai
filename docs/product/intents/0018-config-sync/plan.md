# Plan 0018 Config sync

| field | value |
|---|---|
| intent | [0018-config-sync](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | 2026-11-11 |

The spec was accepted by the owner on 2026-11-11 (spec section 7), verbatim: "ok, apruebo e l0018 y 0019". `gabo.yaml` mode is `auto`: every fixed question below is answered by `gabo-dev-lead`, marked **Proposed (auto)** with a one-line reason, for one-pass approval. The plan decisions the spec left open are recorded under Plan decisions (resolver: agent, owner's directive of 2026-11-11: apply the recommended answer to every unresolved planning question and record it as a plan decision). The intent tracker item is [#493](https://github.com/testeos-z/gabo-ai/issues/493) under product card #492.

## Fixed questions

| id | answer |
|---|---|
| Q1 | **Proposed (auto)**: one serial chain of three skill tasks, then the probe: the source (local or remote) and its reading (T1), the work-style proposal as one diff with its refusals (T2), the saved profile, the rest of setup and the docs (T3), then the probe on two disposable repositories (T4) (Steps). Reason: all three skill tasks edit the one `gabo-setup/SKILL.md` and its codex copy, so serial order avoids two PRs on the same file (PD1). |
| Q2 | **Proposed (auto)**: four tasks, each one PR well under about 400 changed lines (Tasks). Reason: the source, the proposal and the profile are three behaviors with their own RED and GREEN run; the probe is the spec's own verification (S1 to S8). |
| Q3 | **Proposed (auto)**: see the `spec refs` column. Reason: each task names the requirements and decisions it proves. |
| Q4 | **Proposed (auto)**: see `depends_on` and `unblocks`, acyclic: T1 → T2 → T3 → T4. One cross-intent edge: T3 unblocks `0019/T4`, which edits the same `gabo-setup/SKILL.md` and its codex copy for the agent-communication sentence (spec 0019 section 5). No task of intent 0015 (#474, plan in PR #475) edits `gabo-setup`, `setup_test.go`, `setupsync_test.go` or `docs/setup-prompt.md`, so no 0015 block exists (PD2). Intents 0016 and 0017 are not accepted and are ignored. Reason: file overlap checked against every 0015 edit surface. |
| Q5 | **Proposed (auto)**: see `acceptance` and `size`; every task starts from failing assertions on the skill text in a new test file (the skill is prose, the tests pin its headings and rules), and the probe (T4) runs the skill. Reason: TRD test-first convention; the spec allows test-only additions (S9). |
| Q6 | **Proposed (auto)**: yes: S1 (T1), S2 (T1), S3 (T1), S4 (T2), S5 (T2), S6 (T2), S7 (T3), S8 (T3), S9 (T1, T2, T3, T4), and every S# is probed in T4; D1 (T1, T2, T3), D2 (T1), D3 (T1, T3), D4 (T2), D5 (T2), D6 (T1). Reason: checked against the spec table. |
| Q7 | **Proposed (auto)**: see Risks to the plan. |
| Q8 | **Proposed (auto)**: see Open questions. |
| Q9 | **Proposed (auto)**, applying intent 0008 S10 (TRD 8.5): no `gabo check` update is needed; spec S9 keeps the loader, the schemas and `gabo check` untouched, and every task states `gabo check impact: none` in its PR body. The profile file is not a config layer, so no rule reads it. Reason: D1 and D3 of the spec. |

## Plan decisions

| id | what | question | choice | reason | task | resolver |
|---|---|---|---|---|---|---|
| PD1 | Task split | One task for the whole mode, or several? | Three skill tasks in series (source, proposal, profile and finish) and one probe task. | The mode is about 70 lines of skill prose plus its tests plus the codex copy, which doubles the lines; three slices keep each review to one behavior, and series order avoids two PRs on `gabo-setup/SKILL.md`. | T1 to T4 | agent (owner's directive of 2026-11-11) |
| PD2 | Edit surfaces and 0015 overlap | Which files does a task edit, and do any belong to intent 0015's tasks? | Each skill task edits `apps/cli/internal/skills/gabo-setup/SKILL.md`, its mirror `plugins/codex/skills/gabo-setup/` (a `review.human_only_paths` entry: the owner merges, as in 0015) and the new test file `apps/cli/internal/skills/setupsync_test.go`; T3 also edits `docs/setup-prompt.md` outside the embedded prompt block. No task edits `.gabo/skills/` (the catalog copy that `gabo init` refreshes), `README.md`, `apps/cli/internal/setupprompt/` (changing the embedded prompt changes `gabo prompt` output) or any existing test file. | 0015's tasks T4 to T10 edit the assistant, cycle, coder, agent-spec, runtime skills, `check`, `README.md` and `docs/`; none edits these files, so no blocked-by on 0015 is needed. `README.md` stays out because 0015 T2 to T4 edit it. The pinned strings of `setup_test.go` stay untouched. | T1 to T3 | agent (owner's directive of 2026-11-11) |
| PD3 | How a GitHub URL is read | S3 reads one file without cloning: which call? | `gh api -H "Accept: application/vnd.github.raw+json" repos/<owner>/<repo>/contents/gabo.yaml`, adding `?ref=<ref>` only when the URL names a branch (`/blob/<ref>/...` or `/tree/<ref>`), otherwise the default branch. A non-200 result is reported by status and reason and changes nothing. A remote on another host than GitHub: the skill asks the owner for a local path (spec D6). | One read, the owner's existing `gh` login (D6), no clone, no stored credential. | T1 | agent (owner's directive of 2026-11-11) |
| PD4 | Adapter names in the profile and in the target | D4 copies the adapter name and resolves its path in the target: how? | The profile stores `slots.<name>.adapter` as the adapter's skill name (the folder name, for example `gabo-tasks-github`), never a path. In a target the skill uses the path `.gabo/skills/<name>/SKILL.md` when that file exists (a catalog adapter) and otherwise asks the owner which adapter the target uses for that slot, one question. | Catalog adapters live at one path in every project; a project-owned adapter has no portable path, so asking is the only safe resolution (R12). | T2, T3 | agent (owner's directive of 2026-11-11) |
| PD5 | What is a secret-like value (S6) | S6 refuses it but does not define it. | The skill refuses, by key path only: (a) any key whose last segment contains `token`, `secret`, `password`, `passwd`, `credential` or `api_key` (case-insensitive) or ends in `_key`, `_env` or `_login`, which includes `review.check_env`, `review.pi_credential_env` and `coder.login`; (b) any string value that starts with `ghp_`, `gho_`, `github_pat_`, `sk-`, `xox`, `AKIA` or `eyJ`, or is 32 or more characters of base64 or hex with no space. The rule applies to every key of a source before the proposal is built, and to the profile before it is written. | The refusal must be testable; the lists follow the common token shapes and the credential-naming keys of the schema, and the skill's own S4 list holds none of them, so a refusal is a defense in depth. | T2 | agent (owner's directive of 2026-11-11) |
| PD6 | Probe shape | S1 to S8 are verified by a probe on disposable repositories (pattern of I0013 T15): which sources? | Two disposable local repositories created in one `mktemp -d` directory outside the worktree and deleted by their literal path at the end: save a style from the first, sync it into the second locally; then sync from the first by local path; a GitHub URL source is a read-only read of this repository's own `gabo.yaml` on `dev` (`https://github.com/testeos-z/gabo-ai`) through the owner's existing `gh` login; an unreadable source is a path that does not exist and a GitHub URL of a repository that does not exist. No GitHub repository is created. A planted token-like value goes into the first repository's `gabo.yaml`. | A disposable GitHub repository would be a remote write needing the owner's authorization; reading one file of this repository is exactly what S3 and D6 define. | T4 | agent (owner's directive of 2026-11-11) |

## Steps

1. The source: the `gabo-setup` skill gets a `## Sync config` mode that asks "local or remote" and takes local when unanswered, reads the local profile `profile.yaml` or one remote `gabo.yaml` (local path or GitHub URL, read-only, no clone) and reports a missing or unreadable source without changing anything (T1). Delivers S1, S2, S3, D2, D6.
2. The proposal: only the work-style keys become one diff against the target `gabo.yaml`, target values kept unless the owner picks the source value, repository-specific keys detected or asked, secrets refused by key path, applied through `gabo-config-writer` after one confirmation (T2). Delivers S4, S5, S6, D4, D5.
3. The profile and the rest of setup: the mode offers to save a configured repository's style as `profile.yaml`, then runs setup's `gabo init`, agent build, `gabo check` and slot smoke test by reference to those steps; `docs/setup-prompt.md` describes the mode (T3). Delivers S7, S8, D3.
4. The probe on two disposable repositories, results recorded in this plan (T4). Delivers the verification of S1 to S8 and the diff check of S9.
5. `gabo check` update (TRD 8.5): none needed (Q9, spec S9 and D1); each task's PR body states `gabo check impact: none`.

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

Every skill task keeps `docs/product/` unchanged except T4, which touches only this plan (TRD D7). `plugins/` is a `review.human_only_paths` entry: each skill task refreshes the `plugins/codex/skills/gabo-setup/` copy in the same PR (`codexplugin_test.go` names the first drifted path) and merges only with the owner or the coordinator holding the owner's delegation. No task edits a Go production file, the schemas, `gabo check` or an existing test file (spec S9). Every task runs `go -C apps/cli build ./...`, `go -C apps/cli vet ./...`, `go -C apps/cli test ./...`, `gofmt` and `gabo check` (exit 0, no golden file changed).

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | Sync config: ask local or remote and read the source | S1, S2, S3, S9, D1, D2, D3, D6 | — | T2 | Test first: a new `apps/cli/internal/skills/setupsync_test.go` (helpers `setupDoc` and `requireAll` of `setup_test.go`), each assertion written and seen failing on today's skill: `gabo-setup/SKILL.md` has a `## Sync config` section and its Activation Contract and front-matter trigger name "sync config"; the mode's first question is "local or remote" and local is the source when the owner does not answer; a local source reads `profile.yaml` at `$XDG_CONFIG_HOME/gabo/profile.yaml` when that variable is an absolute path, else `~/.config/gabo/profile.yaml`, and a missing profile reports "no profile at <path>", offers the remote source or the normal setup and writes nothing; a remote source asks for one repository by local path or GitHub URL and reads only `<path>/gabo.yaml`, or the one file through `gh api` with the raw accept header (PD3), read-only, without cloning, storing nothing; an unreadable or missing file reports the reason and changes nothing; a remote on another host asks for a local path (D6). Every string `setup_test.go` pins still passes. gabo check impact: none. Edit surfaces: `apps/cli/internal/skills/gabo-setup/SKILL.md`, `apps/cli/internal/skills/setupsync_test.go` (new), `plugins/codex/skills/gabo-setup/` (`review.human_only_paths`). | M |
| T2 | Sync config: propose the work-style keys as one diff and apply them | S4, S5, S6, S9, D1, D4, D5 | T1 | T3 | Test first: new cases in `setupsync_test.go`, seen failing: the mode lists exactly the S4 keys (`mode`, `runtimes.coder`, `runtimes.reviewer`, `runtimes.assistant`, each `slots.<name>.adapter` choice, `parallelism`, `wave.after`, `plan_change.scope`, `plan_change.global_without_owner`, `review.auto_merge`, `reviewer.placement`) and the generic `review.human_only_paths` entries `gabo.yaml`, `.github/`, `AGENTS.md`, `CLAUDE.md`, `.claude/`, `.codex/`, `.agents/` (any other entry, for example `plugins/`, is asked); it shows one diff against the target `gabo.yaml`, marks `mode`, `review.auto_merge` and `plan_change.global_without_owner` as autonomy keys, keeps each target value unless the owner picks the source value, lets the owner drop a key, and writes each confirmed key through `gabo-config-writer` (minimal edit, `gabo check` after it, in auto mode its recorded-decision path); it never copies `repo.name`, `repo.integration_branch`, `review.checks`, `review.check_env`, `developers`, `worktree.keep`, a slot adapter path or a personal key, detects `repo.name` from the git remote or asks, and asks the other repository-specific keys one at a time as setup does; adapter names resolve per PD4; a secret-like key or value (PD5) is refused and named by key path only. The `plugins/codex` copy is refreshed. gabo check impact: none. Edit surfaces: `apps/cli/internal/skills/gabo-setup/SKILL.md`, `apps/cli/internal/skills/setupsync_test.go`, `plugins/codex/skills/gabo-setup/` (`review.human_only_paths`). | M |
| T3 | Sync config: save the style as the local profile, finish with the rest of setup and document the mode | S7, S8, S9, D1, D3 | T2 | T4, 0019/T4 | Test first: new cases in `setupsync_test.go`, seen failing: the mode offers to save a configured repository's style as `profile.yaml` in the user config folder (same path rule as T1) with the S4 keys only, adapters as names (PD4) and no repository-specific, personal or secret-like key (PD5), validates the keys against `gabo.project.schema.json` before writing, and asks before overwriting an existing profile, showing the difference first; after a sync it runs `gabo init` first when `.gabo/skills/` is missing or `runtimes.*` changed, then setup's step 4 (agent build for each engine of the engine set), step 5 (`gabo check` until it passes) and step 6 (slot smoke test) by reference to those steps, none skipped, and reports a slot whose smoke test did not run as `unverified`. `docs/setup-prompt.md` gains a section on the mode outside the embedded prompt block (`TestPublishedPromptMatchesEmbedded` still passes, `setup-prompt.md` unchanged). The `plugins/codex` copy is refreshed. gabo check impact: none. Edit surfaces: `apps/cli/internal/skills/gabo-setup/SKILL.md`, `apps/cli/internal/skills/setupsync_test.go`, `docs/setup-prompt.md`, `plugins/codex/skills/gabo-setup/` (`review.human_only_paths`). | M |
| T4 | Probe config sync on two disposable repositories and record the results | S1, S2, S3, S4, S5, S6, S7, S8, S9 | T3 | — | Probe task, run by the coordinator on the plan-decision PD6 repositories, scratch only in one `mktemp -d` directory removed by its literal path, with the `gabo` built from `dev` after T1 to T3 merged and the catalog copy refreshed in the disposable repositories. Checks, each read back from `gabo.yaml` and `gabo check` (exit 0): save the style of repository A as the profile, then sync it into repository B locally (the default, no answer to "local or remote"); sync from A by local path; sync from the GitHub URL; target `parallelism: 3` stays 3 unless picked; no repository-specific value of the source (`repo.name`, `review.checks`, `developers`, an adapter path, `alias`) appears in B; the planted token-like value is refused by key path and nothing is written; a missing profile and an unreadable source change nothing; after a sync that changed `runtimes.*`, `gabo init` and the agent build ran before `gabo check`, and the smoke test ran or its slot is reported unverified. S9: `go -C apps/cli build ./...`, `vet`, `test` pass and `git diff --name-only` of T1 to T3 lists no Go production file. The PR touches only `docs/product/intents/0018-config-sync/plan.md` (Probe results below, TRD D7). gabo check impact: none. Edit surfaces: `docs/product/intents/0018-config-sync/plan.md` (Probe results only). | S |

## Tracker

Intent tracker item: [#493](https://github.com/testeos-z/gabo-ai/issues/493) under product card #492.

| task | issue |
|---|---|
| T1 | [#499](https://github.com/testeos-z/gabo-ai/issues/499) |
| T2 | [#500](https://github.com/testeos-z/gabo-ai/issues/500) |
| T3 | [#501](https://github.com/testeos-z/gabo-ai/issues/501) |
| T4 | [#502](https://github.com/testeos-z/gabo-ai/issues/502) |

## Probe results

Not run yet. T4 writes the results here.

| probe | result |
|---|---|
| save a style as the local profile (A) | not run (T4) |
| sync locally by default into B | not run (T4) |
| sync from A by local path | not run (T4) |
| sync from a GitHub URL (read-only) | not run (T4) |
| repository-specific keys never copied | not run (T4) |
| secret-like value refused by key path | not run (T4) |
| missing profile and unreadable source change nothing | not run (T4) |
| `gabo init`, agent build, `gabo check`, smoke test after a sync | not run (T4) |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| Intent 0015 changes agents, `gabo-cycle` and the runtime skills while this mode refers to setup's step 4 (build every agent of the neutral agent spec) and step 6 (smoke test). | T3, T4 | The mode refers to those steps and copies no agent name, so 0015's six-agent change needs no edit here; the probe runs on whatever `dev` holds. |
| `gabo-setup/SKILL.md` and its codex copy are also edited by `0019/T4` (one sentence). | T1 to T3, 0019/T4 | `0019/T4` depends on T3; a second PR on the file never runs in parallel. |
| The prose mode gives different results on different engines (a skill is followed, not executed). | T4 | The tests pin the rules; the probe runs the skill end to end and reads back `gabo.yaml`; a mismatch is a task fix, not a silent pass. |
| `plugins/codex/skills/gabo-setup/` drifts from the catalog skill. | T1 to T3 | `codexplugin_test.go` names the first drifted path; each PR refreshes the copy. |
| The profile is read by an agent with a differently shaped file. | T2, T3 | The skill checks every key against `gabo.project.schema.json` before copying, as `gabo-config-writer` does. |
| A GitHub read fails on a private repository the login cannot read. | T1, T4 | The mode reports the status and reason and changes nothing (S3); the probe covers an unreadable source. |

## Open questions

1. None: spec open questions 1 to 6 were answered as decisions (spec section 6), and the planning questions are PD1 to PD6 above (resolver: agent, owner's directive of 2026-11-11). Nothing waits on the owner except the merge of the PRs that touch `plugins/`.
