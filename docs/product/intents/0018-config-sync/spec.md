# Spec 0018 Config sync

| field | value |
|---|---|
| intent | [0018-config-sync](intent.md) |
| status | Accepted |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: section 3 and section 4 are not applicable) |
| date | 2026-11-11 |

Mode is `auto`. Owner's standing rule (2026-11-11): the agent applies its recommended answer to every open question and records it as a decision; only merging to main, tags and secrets stay the owner's. Each decision below is marked **Recorded decision (resolver: agent, rule of the owner)** and is saved through the `memory` slot. The owner's own words are the source and are quoted verbatim.

Source, owner verbatim (2026-11-11):

- "una forma mas rapida de configurar tardamos mucho en la configuracion inicial que podamos decirle que revise otro repo y clone la configuracion apnostica al opjetivo del repo solo de como me gusta trabajar"
- "me gustan los dos cosas que gabo pueda hacer poll-config o 'sync config' y que por defecto tome local pero que te pregunta si local o remoto"
- On which intent gets its spec next: "vamos en orden numerico".

Evidence base (checked 2026-11-11 on `origin/dev` at `0fd848e`):

- **Setup today.** `.gabo/skills/gabo-setup/SKILL.md` step 1 asks, one question at a time, the identity, the personal keys, `repo.name`, `repo.integration_branch`, `review.checks`, `parallelism`, `mode`, `plan_change.*`, the runtimes, where agents run (`slots.agent_host`), the tracker and each other slot; step 3 writes through `gabo-config-writer`; step 4 builds the agents (after a change of `runtimes.*`, `gabo init` first); step 5 runs `gabo check`; step 6 is the slot smoke test. Its hard rules: owner answers only (R12), no secrets (R21), four commands only, idempotent (skips every question already answered).
- **Config layers.** TRD 9.3: the loader reads the user layer, then the project layer; the global user file holds the personal keys (`alias`, `interaction_level`, `tone`, `language`); every other key is a project key in `gabo.yaml`. `schemas/gabo.user.schema.json` line 6 references the project `layer` definition, so the user file would accept project keys as a fallback; `gabo-config-writer` forbids writing them there.
- **User file path.** `apps/cli/internal/config/layers.go` `UserPath` (lines 116-121): `$XDG_CONFIG_HOME/gabo/config.yaml`, else `~/.config/gabo/config.yaml`. The loader reads `config.yaml` in that folder and no other file of the folder as a layer.
- **Writer rules.** `.gabo/skills/gabo-config-writer/SKILL.md`: minimal YAML edits, owner values only, in auto mode a recorded decision before a chosen value, no secrets, refuse keys outside the schemas.
- **CLI surface.** TRD 9.4 and D24: four commands (`init`, `check`, `version`, `prompt`); config is edited as YAML by the agent and validated by `gabo check`.
- **Project keys.** TRD 9.3 key table, lines 434-453: `mode`, `repo.name`, `repo.integration_branch`, `runtimes.*`, `slots.<name>.adapter`, `reviewer.placement`, `review.checks`, `review.auto_merge`, `review.human_only_paths`, `developers`, `plan_change.*`, `wave.after`, `parallelism`, `worktree.keep`.
- **Reading another repository.** The `repository_host` contract (`.gabo/skills/gabo-contract-repository-host/SKILL.md`) has the operations open PR, comment on PR, read PR status and checks, merge into the integration branch and delete branch; it has no operation that reads a file of a repository.

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | When the owner asks to sync the config, the `gabo-setup` skill shall ask "local or remote" and take local when the owner does not answer. | Given the owner asks "sync config", When the skill starts the mode, Then its first question is "local or remote". Given no answer, Then the source is local. Given "remote", Then it asks which repository (S3). | skill review against S1; probe on a disposable repository (see Verification below the table) |
| S2 | When the source is local, the skill shall read the work-style profile `profile.yaml` from the user config folder and, when it is missing, say so and offer the remote source or the normal setup, changing nothing. | Given `~/.config/gabo/profile.yaml` exists, Then its work-style keys are the proposal of S4. Given it is missing, Then the skill reports "no profile at <path>", offers remote or normal setup, and writes nothing. | skill review; probe with and without a profile |
| S3 | When the source is remote, the skill shall read the `gabo.yaml` of one repository the owner names, by local path or GitHub URL, read-only: it reads that one file, needs no write access to the repository, and stores nothing from it except the keys that S4 applies. | Given a local path, Then the skill reads `<path>/gabo.yaml`. Given a GitHub URL, Then it reads that one file with the owner's existing GitHub login (D6), without cloning. Given an unreadable or missing file, Then it reports the reason and changes nothing. | skill review; probe with a local path and a GitHub URL, and with an unreadable source |
| S4 | The skill shall propose only the work-style keys: `mode`, `runtimes.coder`, `runtimes.reviewer`, `runtimes.assistant`, the choice of each `slots.<name>.adapter` (D4), `parallelism`, `wave.after`, `plan_change.scope`, `plan_change.global_without_owner`, `review.auto_merge`, `reviewer.placement` and the generic `review.human_only_paths` entries (D4). It shall show the proposal as a diff against the target `gabo.yaml`, keep every value the target already has unless the owner picks the source value, and apply the proposal through `gabo-config-writer` after the owner's confirmation (in auto mode through its recorded-decision path). | Given a source with all style keys and a target that has `parallelism: 3`, Then the diff lists `parallelism` as "differs", keeps 3 by default and changes it only if the owner picks the source value. Given the owner confirms, Then each key is written with the writer's minimal edit and a `gabo check` after it. Given the owner declines a key, Then it is not written. | skill review; probe on a disposable repository, reading back the resulting `gabo.yaml` |
| S5 | The skill shall never copy the repository-specific keys `repo.name`, `repo.integration_branch`, `review.checks`, `review.check_env`, `developers`, `worktree.keep` or any slot adapter path, or any personal key. It shall detect the ones it can from the target repository and ask the rest, one question at a time, as setup does today. | Given a source with `repo.name: other/repo`, Then the target's `repo.name` is read from its git remote or asked, never `other/repo`. Given a source `review.checks`, Then it is not copied and the skill asks. Given `alias` in a source file, Then it is not copied. | skill review; probe: no repository-specific value of the source appears in the target `gabo.yaml` |
| S6 | The skill shall never read, copy, ask for or write a secret or credential: no token, password or key value, and no key that names a credential. A value that looks like a secret is refused and reported by key path only. | Given a source that holds a token-like value, Then the skill refuses it, names the key path and writes nothing. Given the profile written by S7, Then it holds no secret. | skill review; probe with a planted token-like value |
| S7 | The skill shall offer to save the style of a configured repository as the local profile, writing `profile.yaml` with the S4 keys only, and ask before it overwrites an existing profile, showing the difference. | Given a configured repository and the owner's yes, Then `~/.config/gabo/profile.yaml` holds the S4 keys of its `gabo.yaml` and no repository-specific key. Given an existing profile, Then the diff is shown and nothing is overwritten without a yes. | skill review; probe: save, then sync into a second disposable repository |
| S8 | After a sync the skill shall keep the rest of the setup: `gabo init` first when `.gabo/skills/` is missing or `runtimes.*` changed, the agent build for each engine of the engine set, `gabo check` until it passes, and the slot smoke test; none is skipped. | Given a sync that changed `runtimes.*`, Then `gabo init` runs and the agents of every listed engine are built before `gabo check`. Given a `gabo check` failure, Then it is fixed or asked before the smoke test. Given the smoke test of a slot is not run, Then the skill reports the slot as unverified. | skill review; probe: `gabo check` passes and the smoke test runs after a sync |
| S9 | This change shall add no Go command and no change to the Go loader, the schemas or `gabo check`: its output and exit codes stay byte-identical. The strings that the skill tests under `apps/cli/internal/skills/` pin stay, and a test-only addition that pins the new text is allowed. | Given `go -C apps/cli test ./...`, Then it passes with no golden file changed. Given the diff of the change, Then it touches skill files, docs and at most skill tests, and no Go production file. | `go -C apps/cli build ./...`, `vet`, `test`; review of the diff for Go files |

Verification of S1 to S8: the skill is prose, so a unit test cannot run it. The check is a probe on two disposable repositories (the pattern of the I0013 T15 probes): save a style from the first, sync it into the second locally, then sync from the first by local path and by GitHub URL, each time reading back `gabo.yaml` and running `gabo check`. The plan assigns the probe to a task.

## 2. Design

Affected files: the `gabo-setup` skill (a new mode), its mirrored catalog copies, and the docs that describe setup. No Go production file. Where the profile is written and read is the skill's work, with the rules of `gabo-config-writer` applied.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Shape of the change | (a) a mode of the `gabo-setup` skill, with no Go; (b) a new `gabo sync` Go command; (c) a Go change so the loader merges a profile layer | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: TRD D24 keeps four commands and has the agent edit YAML through `gabo-config-writer`; the loader reads `config.yaml` as the user layer and nothing else, so a separate profile file needs no Go; every written value is validated by the `gabo check` that setup already runs. | Go is needed only if the owner later wants `gabo check` to validate the profile or `gabo init` to seed from it; each is a new intent. |
| D2 | Meaning of "remote" and the default | (a) another repository, by local path or GitHub URL; (b) another machine | **Recorded decision (resolver: agent, rule of the owner):** (a); the owner's words: "revise otro repo". The operation always asks "local or remote" and takes local when unanswered (owner: "por defecto tome local pero que te pregunta si local o remoto"). | Another machine is out of scope; reaching one is the subject of intent 0019. |
| D3 | Where the local profile lives and its form | (a) a file of its own, `profile.yaml`, in the user config folder next to `config.yaml`, with the same key paths as `gabo.yaml` and only the S4 keys; (b) the same keys in `config.yaml`, which the loader would accept as a project-key fallback | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: (b) would make values invisible to teammates and to CI, which have no user file, could let a repository pass `gabo check` without the key in its own `gabo.yaml` (R12, no silent defaults), and `gabo-config-writer` forbids team keys in the user file. The values are copied into `gabo.yaml`, which is committed and reviewed. | The path follows the user file rule: `$XDG_CONFIG_HOME/gabo/profile.yaml` when that variable is an absolute path, else `~/.config/gabo/profile.yaml`. The Go loader ignores the file. The skill checks the keys against `gabo.project.schema.json` before copying, as the writer does. |
| D4 | Which keys are copied | (a) the style keys of S4, with the repository-specific keys of S5 detected or asked; (b) every key | **Recorded decision (resolver: agent, rule of the owner):** (a), the owner's list. Two points the list leaves open. First, `slots.<name>.adapter` holds a path that differs per engine and repository, so the skill copies the adapter name (for example `gabo-tasks-github`), resolves its path in the target and asks when the target has no such adapter. Second, "generic" `review.human_only_paths` entries are the ones every gabo project has, the gabo config and CI paths of this repository's own `gabo.yaml` (`gabo.yaml`, `.github/`, `AGENTS.md`, `CLAUDE.md`, `.claude/`, `.codex/`, `.agents/`); any other entry (for example `plugins/`) is asked. Keys that name a credential, `review.check_env` included, are never in the list. | The plan fixes the exact generic list and reads it against the schema. A new work-style key later needs a skill edit. |
| D5 | How values are applied | (a) one diff, one confirmation, target values kept unless picked; (b) one question per key | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: speed is the intent's point, the owner's source is the owner's answer under R12 once the diff is confirmed, and keeping the target's existing values matches setup's idempotent rule. The diff marks the autonomy keys (`mode`, `review.auto_merge`, `plan_change.global_without_owner`) so they are seen before the confirmation. | In auto mode the writer's recorded-decision path still applies to each value. The owner can drop any key from the diff. |
| D6 | How a GitHub remote is read | (a) read the one file with the GitHub contents API through the owner's existing `gh` login; (b) clone the repository; (c) add a read-file operation to the `repository_host` contract | **Recorded decision (resolver: agent, rule of the owner):** (a). Reason: it reads one file, needs no write access and stores nothing, and a clone would copy a whole repository for one file; (c) changes a slot contract, which is a global-scope convention change for one read. | A private repository works when the owner's login can read it; no credential is stored or asked for. If another tracker host is configured, the mode asks the owner for a local path. |

## 3. UX and app flow (optional)

Not applicable: there is no screen or component. The journey is a conversation with the setup skill, described by S1 to S8: it asks local or remote, shows one diff, asks one confirmation, and then runs the same build, check and smoke test as setup. Wording follows the skill's existing one-question-at-a-time style.

## 4. DB migration steps (optional)

Not applicable: the change stores no data in a database. The profile is a new optional user file; without it nothing changes, so no migration or rollback is needed.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| PRD R12 (owner answers only, no silent defaults) | Values come from a source, not from a question. | resolved by D5: the owner picks the source and confirms the diff; that is the owner's answer. |
| PRD R21 (no secrets) | A remote `gabo.yaml` may hold a secret-like value. | resolved by S5, S6 and D4: credential keys are never copied; a secret-like value is refused. |
| TRD 9.3 and `gabo-config-writer` (team keys only in `gabo.yaml`, personal keys only in the user file) | A user-level file holds team keys. | resolved by D3: the profile is its own file, not a config layer; the values land in `gabo.yaml`. The writer's skill keeps writing only the two config files; the setup mode writes the profile and applies the same refusal and no-secret rules. |
| TRD 9.4 and D24 (four commands; config edited as YAML by the agent) | A sync could suggest a new command. | none: D1 adds no command. |
| `gabo-setup` hard rule "idempotent" | A sync could overwrite answers. | resolved by D5: existing target values are kept unless the owner picks the source value. |
| PRD R32 (`gabo init` never touches `gabo.yaml` or the user config) | Sync writes `gabo.yaml`. | none: the skill writes it, not `gabo init`; `gabo init` runs only as setup runs it today. |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Does "remoto" mean another repository or another machine? | answered: another repository, by local path or GitHub URL (D2). Recorded decision (resolver: agent, rule of the owner). |
| 2 | Which keys count as work style and which are repository-specific? | answered: S4, S5 and D4. Recorded decision (resolver: agent, rule of the owner). |
| 3 | Should the same operation also save a configured repository's style as the local profile? | answered: yes (S7). Recorded decision (resolver: agent, rule of the owner). |
| 4 | Where does the local profile live? | answered: `profile.yaml` in the user config folder (D3). Recorded decision (resolver: agent, rule of the owner). |
| 5 | Does the change need Go? | answered: no (D1). Recorded decision (resolver: agent, rule of the owner). |
| 6 | How is a GitHub remote read when the `repository_host` contract has no read-file operation? | answered: D6. Recorded decision (resolver: agent, rule of the owner). |

Nothing is left that needs the owner except the acceptance in section 7.

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
| leomerida15 (owner, alias Leo) | 2026-11-11 | Accepted by the owner as the spec stands in PR #498, with its recorded decisions (resolver: agent, rule of the owner). Owner, verbatim: "ok, apruebo e l0018 y 0019" (this sentence accepts intents 0018 and 0019). Relayed unedited by the assistant from the owner's own message in the session of 2026-11-11; recorded by gabo-pm. The owner reviews the PR before it merges. No platform statement is in this spec; the owner has not decided that point. |
