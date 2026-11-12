# gabo-reviewer

Embedded skill with the deterministic reviewer scripts of gabo (TRD D20, plan task T040). Follows the vulcam `agents/pr-reviewer` pattern in single-user local mode: no sudo, no reviewer OS users, Claude Code, Pi, Codex or OpenCode as the review engine (`runtimes.reviewer`). It is embedded apart from the catalog (`skills.ReviewerFiles`), so the catalog install never writes it; `gabo init` installs it at user level (T041). It never belongs in a repository.

## Why user level

A pull request can change anything in the repository, including scripts that live there, but not files in the owner's user-level skill dirs. The preflight therefore refuses to run unless `reviewer.sh` resolves (realpath) to `${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer/scripts` or `~/.claude/skills/gabo-reviewer/scripts` (a runtime link to it), refuses a copy inside the repository under review, and refuses when the skill dir, the scripts dir, a script or `config.env` is not owned by the user or is writable by group or others.

Trust is still single-user: checks and the review agent run as the owner, so PR code could read the owner's files. The marker proves only that the comment was posted by this login with this marker; it stops honest mistakes, not a hostile PR. Use an isolated placement (the `reviewer_placement` adapter) when that matters.

## Use

Run from the repository's main checkout; its `gabo.yaml` names the repository (`repo.name`) and the integration branch (`repo.integration_branch`):

```bash
gh auth status                                   # the owner's login is the bot login
s="${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts"
bash "$s/reviewer.sh" preflight
bash "$s/reviewer.sh" head <pr> <40-hex sha>
```

Other modes: `due <pr> <sha>`, `list-due`, `once` (every due head, one at a time) and `prune`: it removes every `worktrees/pr-<n>` of the state dir whose PR is not open, and every one that is not the current PR's and has not been used for more than `REVIEWER_KEEP_DAYS` days, skipping a run in progress (its `pr-<n>.lock` is held) and logging a `WARN` for a path that is not a worktree of the reviewer's mirror. `head` runs it first and removes the `pr-<n>` worktree after a successful merge; `prune` also keeps `cache/go-build`, `cache/go-mod` and `check-home` together under `REVIEWER_CACHE_MAX`: when over, it runs `go clean -cache` with the reviewer's own `GOCACHE`, removes the `check-home` contents older than `REVIEWER_KEEP_DAYS` and prints one line `pruned: <what> (<size freed>)`; under the limit it prints nothing, and it never trims while a review holds a `pr-<n>.lock`. A prune problem is logged and never changes a verdict. The last line of `head` is always:

```text
VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>
```

A conflict is `blocking`; a retryable failure or a refused preflight is `error`; a moved head, a head with a final status or a verdict comment, or a refused or failed merge is `skipped` (the `SKIP`, `MERGE-SKIPPED` or `MERGE-FAILED` line above it says why).

## What one review does

1. Preflight: no `sudo`; the trusted location; `jq`, `flock`, `git`, `setsid`; the review engine binary and, for `codex` and `opencode`, its isolation pass marker (below); fetches the integration branch into its own mirror and moves its own checkout (`<state>/base`) to the tip, refusing when that checkout is not clean; reads the repository settings from `gabo.yaml` there; `reviewer.placement` must be `local`; the base minimum (below); the `gh` login.
2. Skips a head that moved, a draft, a fork, a PR to another base, and a head that already has a final `reviewer/review` status or a verdict comment by the bot login.
3. Fetches GitHub's merge result `refs/pull/<n>/merge` and trusts it only when it is the integration-branch tip merged with the head; a conflict gets a blocking comment.
4. Runs `review.checks` as argv (no shell) in a fresh worktree of the merge result with `env -i`, its own `HOME`, Go caches under the state dir, `CI=true` and no credential, plus each variable named in `review.check_env` taken from the reviewer's own environment (a listed one unset or empty fails `reviewer/tests` naming it, before any check runs; a listed value a failing check prints is redacted to `[review.check_env:<name>]` in the posted failure output). A `gabo check` entry runs `REVIEWER_GABO_BIN`; `gabo check` never runs unless listed.
5. Reviews the diff with the review engine (Claude Code, Pi, Codex or OpenCode, below), read-only, in the head checkout (OpenCode from a reviewer-owned directory outside it), with no GitHub token.
6. Posts one comment (first line the verdict, last lines the marker `<!-- <reviewer.signing.marker> sha=<head> -->` and the footer `🤖 gabo reviewer`), sets `reviewer/tests` and `reviewer/review`, and on a clean head runs the merge gate.
7. Merge gate (`merge-head.sh`): only with `review.auto_merge: true`, PR open, base the integration branch (never `main` or `master`), same repository, head still the reviewed SHA, both statuses `success` by the bot login, the newest marker comment clean and signing that SHA, and no human-only path; then `gh pr merge <pr> --merge --match-head-commit <sha>`. Human-only paths get a "needs human merge" comment instead.

## Review engines

`runtimes.reviewer` on the integration branch picks the engine (`REVIEWER_ENGINE` overrides it); anything other than `claude`, `codex`, `pi`, `opencode` or `session`, or a missing binary, fails the preflight. Under `session` every mode needs a trailing `engine <engine>` (the requesting wave's engine) listed in `runtimes.assistant` on the integration branch; a fixed engine refuses that argument (see `SKILL.md`). Every engine gets the same prompt (on stdin, or for OpenCode as an attached file) and must answer with the same verdict line, so the comment, the statuses and the once-per-head rule do not depend on the engine.

- **Claude Code** (`REVIEWER_CLAUDE`, default `claude`): tools `Read`, `Grep`, `Glob` and `Bash` limited to `git diff/log/show/status`; only user settings, all hooks off, no MCP servers, no slash commands, no saved session. With `CLAUDE_CODE_OAUTH_TOKEN` it runs with an empty `HOME`; without it, with the owner's `HOME`, so the owner's Claude Code login is used.
- **Pi** (`REVIEWER_PI`, default `pi`): `pi --mode json --no-session --no-approve --no-extensions --no-skills --no-prompt-templates --no-themes --no-context-files --no-mcp --tools read,grep,find,ls --exclude-tools 'bash,edit,write,mcp__*'` (plus `--model` from `REVIEWER_PI_MODEL`). No write or shell tool exists; `--no-approve` ignores the PR's `.pi/settings.json`, `.pi/mcp.json`, `.pi/extensions/` and the rest of its project files; the other flags keep the owner's user-level extensions, packages, skills and MCP servers out too, and `AGENTS.md`/`CLAUDE.md` are read only as data. The verdict is the text of Pi's last assistant message; any `stopReason` other than `stop` is an error (Pi exits 0 on a provider error).
  - **Reused Pi files.** Without `review.pi_credential_env` the review keeps the owner's `HOME` (and `PI_CODING_AGENT_DIR` when set), like the Claude path without a token, so no extra API key is needed: Pi reads `auth.json` (the owner's login, for example OpenAI browser OAuth), `settings.json` (`defaultProvider`, `defaultModel`) and `models.json` from its agent dir (default `~/.pi/agent`). The files are used in place rather than copied because Pi refreshes an OAuth login by rewriting `auth.json`; a refresh in a copy can rotate the refresh token and leave the owner's own login stale.
  - **Separate credential.** With `review.pi_credential_env: <NAME>`, the value of `<NAME>` in the reviewer's environment is passed to the review step only (never to the checks), with an empty `HOME` under the state dir, so none of the owner's Pi files are read; an empty `<NAME>` fails the preflight.
- **Codex** (`REVIEWER_CODEX`, default `codex`): `codex exec --ignore-user-config --ignore-rules -s read-only --ephemeral -C <head checkout> --output-last-message <state log> -` (plus `-m` from `REVIEWER_CODEX_MODEL`), the flags of the isolation test it passed. The run is `env -i` with only `PATH`, a reviewer-owned `HOME` (`<state>/review-home`) and `CODEX_HOME` (`<state>/codex-home`), which holds only a link to the owner's login (`${CODEX_HOME:-~/.codex}/auth.json`): no GitHub token, none of the owner's Codex config, rules, skills or MCP servers, no saved session, and every command the model runs is read-only. The verdict is Codex's final message; a non-zero exit or no final message is an error.
- **OpenCode** (`REVIEWER_OPENCODE`, default `opencode`): `opencode run --standalone --format json --agent gabo-review --file <state log>/prompt.md <short message>` (plus `-m` from `REVIEWER_OPENCODE_MODEL`, `provider/model[#variant]`), the `run --standalone` of the isolation test it passed, nothing on stdin. The run is `env -i` with only `PATH`, a reviewer-owned `HOME` (`<state>/review-home`) and `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME` of its own per PR (`<state>/review-home/opencode/pr-<n>/{config,data,state,cache}`, recreated on every run, so parallel reviews of other PRs share no config, sessions or cache); the data dir holds only a link to the owner's login (`${XDG_DATA_HOME:-~/.local/share}/opencode/auth.json`), so no GitHub token and none of the owner's OpenCode config, agents or plugins reach it. OpenCode loads `opencode.json` and `.opencode/` from its working directory and every directory above it up to the git worktree root, so the run starts in an empty reviewer-owned directory (`<state>/opencode-run/pr-<n>`) outside the checkout that is a git repository of its own with one empty commit: the PR's `opencode.json`, `.opencode/agents/` and `.opencode/plugins/` are never loaded, and neither is an `opencode.json` or `.opencode/` above the state dir, which by default sits in the owner's home. The scripts write the reviewer-owned config on every run: one primary agent `gabo-review` (and the same global rules) with `edit`, `write`, `patch`, `webfetch`, `websearch`, `task` and `skill` denied, the shell (written under both `bash`, the documented key, and `shell`, its v2 name; 2.0.32 maps both to the `shell` action) limited to `git diff`, `git log`, `git show` and `git status`, plain or as `git -C <checkout>`, with `--output`, `--ext-diff`, `--textconv` and `git diff --no-index` denied, and `external_directory` allowing only the head checkout. The shell starts in the run directory, so the prompt names the checkout and tells the agent to run `git -C <checkout> ...`. The verdict is the text of OpenCode's last assistant message; a non-zero exit or an `error` event is an error.

### Isolation pass marker (`codex`, `opencode`)

A PR's `AGENTS.md`, `.codex/` or `.opencode/` files cannot be switched off by a flag, so `codex` and `opencode` are trusted only after the one-time isolation test of the `gabo-reviewer-config` skill (`isolation-test.sh <engine>`) passed for the installed version. On pass it writes `${XDG_CONFIG_HOME:-~/.config}/gabo/reviewer-isolation/<engine>` holding the engine's `--version` line. The preflight (and `review-head.sh`) runs `<engine> --version` in the reviewer-owned home and refuses the engine, naming the missing isolation test, when the marker is absent or holds another version; upgrading the engine therefore needs a new isolation test. `claude` and `pi` need no marker. The scripts never write the marker.

## Built-in rules

- **Always human-only:** `.gabo/VERSION`, everything under `.gabo/skills/` (the catalog copy) and the reviewer's own source (any `skills/gabo-reviewer/` dir), independent of `review.human_only_paths` (TRD 9.4, issue #79).
- **Base minimum:** the integration branch's `.gabo/VERSION` (read from the reviewer's checkout, never from the PR head) is the minimum version of the gabo binary. An older binary, one without a release version, or none at all is refused with the upgrade command `go install <REVIEWER_GABO_INSTALL>@v<version>` (TRD 9.4). The scripts never update the binary. No stamp on the base means no minimum.
- **Upgrade PR:** a PR that changes `.gabo/VERSION`. Its checks run with `GABO_UPGRADE_PR=1` (the CLI validates only the config); its comment reports `upgrade PR: only config validated, human merge required` and lists the changed paths outside `.gabo/skills/` and the agent directories that the runtime skills of `runtimes.*` declare (`metadata.agent_dir` in `.gabo/skills/<prefix>-runtime-<runtime>/SKILL.md`). The stamp is human-only, so the scripts never merge it.

## Configuration

Repository settings come from `gabo.yaml` on the integration branch; an environment variable of the listed name wins (for tests and one-off runs):

| `gabo.yaml` key | variable | default |
|---|---|---|
| `repo.name` (caller's checkout) | `REVIEWER_REPO` | required |
| `repo.integration_branch` (caller's checkout) | `REVIEWER_BASE_BRANCH` | `dev`; `main` and `master` are refused |
| `review.checks` | `REVIEWER_CHECKS` (`;`-separated) | required |
| `review.check_env` | `REVIEWER_CHECK_ENV` (space-separated) | none; names only, the values come from the reviewer's environment; reserved names are refused |
| `review.auto_merge` | `REVIEWER_AUTO_MERGE` | `false` |
| `review.human_only_paths` | `REVIEWER_HUMAN_ONLY_PATHS` (space-separated) | `.claude/ .github/ gabo.yaml AGENTS.md CLAUDE.md`; entries ending in `/` are prefixes, others file names in any directory |
| `reviewer.signing.marker` | `REVIEWER_MARKER` | `gbai-reviewer-bot` |
| `reviewer.placement` | `REVIEWER_PLACEMENT` | `local` (the only one implemented) |
| `runtimes.reviewer` | `REVIEWER_ENGINE` | `claude`; `claude`, `codex`, `pi`, `opencode` or `session` (with the `engine` argument) |
| `review.pi_credential_env` | `REVIEWER_PI_CREDENTIAL_ENV` | empty: a Pi review reuses the owner's Pi login |

Script defaults: `scripts/config.env` (plain `KEY=value`, read as data, never executed; the environment wins):

| key | default | meaning |
|---|---|---|
| `REVIEWER_MODE` | `single-user` | Only mode here; `isolated` belongs to the placement adapter |
| `REVIEWER_BOT_LOGIN` | empty = `gh api user` login | Must equal the active gh login |
| `REVIEWER_GABO_BIN` | `gabo` | The gabo binary |
| `REVIEWER_GABO_INSTALL` | `github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo` | Package of the upgrade command |
| `REVIEWER_MAX_ATTEMPTS`, `REVIEWER_CHECK_TIMEOUT`, `REVIEWER_REVIEW_TIMEOUT`, `REVIEWER_HEAD_TIMEOUT` | `2`, `1800`, `900`, `7200` | Attempts and seconds |
| `REVIEWER_CLAUDE_MODEL`, `REVIEWER_CLAUDE_MAX_TURNS`, `REVIEWER_DIFF_MAX_BYTES` | Claude default, `40`, `200000` | Review engine |
| `REVIEWER_CLAUDE`, `REVIEWER_PI`, `REVIEWER_CODEX`, `REVIEWER_OPENCODE` | `claude`, `pi`, `codex`, `opencode` | Engine binaries |
| `REVIEWER_PI_MODEL`, `REVIEWER_CODEX_MODEL`, `REVIEWER_OPENCODE_MODEL` | the owner's default Pi, Codex and OpenCode models | Engine models |
| `REVIEWER_MERGE_TRIES`, `REVIEWER_MERGE_DELAY` | `5`, `3` | Asks for GitHub's merge result |
| `REVIEWER_KEEP_DAYS` | `7` | Days an open PR's `pr-<n>` worktree may stay unused before `prune` removes it |
| `REVIEWER_CACHE_MAX` | `2G` | Size (K, M or G) `cache/go-build`, `cache/go-mod` and `check-home` may reach together before `prune` trims them |
| `GABO_REVIEWER_STATE_DIR` | `${XDG_STATE_HOME:-~/.local/state}/gabo/reviewer` | Own mirror, base checkout, worktrees, caches, logs |

Secrets go only in `${XDG_CONFIG_HOME:-~/.config}/gabo/reviewer.env` (mode 600, refused otherwise; override the path with `REVIEWER_ENV_FILE`): `CLAUDE_CODE_OAUTH_TOKEN` (optional; with it the review runs with an empty HOME, without it the owner's Claude Code login is used) and `REVIEWER_GH_TOKEN` (optional; else gh's stored login). Neither reaches the checks.

## Migrating from the gbai names

The gabo-ai rename (TRD D26) changed every name these scripts read; nothing falls back to the former ones:

- the project config is `gabo.yaml` (was `gbai.yaml`): a reviewer that still reads `gbai.yaml` finds no `repo.name` on the integration branch and every head ends `verdict=error`, so update or reinstall the user-level scripts (`gabo init`) before the rename reaches the integration branch;
- the user-level copy is `${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer/`; settings in a former `config.env` (for example `REVIEWER_AUTO_MERGE=true`) must be set again in the new one;
- the secrets file is `${XDG_CONFIG_HOME:-~/.config}/gabo/reviewer.env` and the state dir `${XDG_STATE_HOME:-~/.local/state}/gabo/reviewer` (`GABO_REVIEWER_STATE_DIR`); `gabo check` fails while the former config dir still exists;
- a `review.checks` entry `gbai check` is no longer recognised and runs a `gbai` binary as written: change it to `gabo check`.

## Tests

Offline, with fake `gh`, `claude`, `pi`, `codex`, `opencode`, `go` and `gabo`, real `git` and a local bare repository standing in for GitHub:

```bash
for t in scripts/tests/*.test.sh; do bash "$t"; done
```

`go test ./internal/skills/` runs them too when `bash`, `git`, `jq`, `flock` and `setsid` are available.
