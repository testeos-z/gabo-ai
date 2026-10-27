# gabo-reviewer

Embedded skill with the deterministic reviewer scripts of gabo (TRD D20, plan task T040). Follows the vulcam `agents/pr-reviewer` pattern in single-user local mode: no sudo, no reviewer OS users, Claude Code as the only engine. `gbai init` installs it at user level (T041); it never belongs in a repository.

## Why user level

A pull request can change anything in the repository, including scripts that live there, but not files in the owner's user-level skill dirs. The preflight therefore refuses to run unless `reviewer.sh` resolves (realpath) to `${XDG_DATA_HOME:-~/.local/share}/gbai/skills/gabo-reviewer/scripts` or `~/.claude/skills/gabo-reviewer/scripts` (a runtime link to it), refuses a copy inside the repository under review, and refuses when the skill dir, the scripts dir, a script or `config.env` is not owned by the user or is writable by group or others.

Trust is still single-user: checks and the review agent run as the owner, so PR code could read the owner's files. The marker proves only that the comment was posted by this login with this marker; it stops honest mistakes, not a hostile PR. Use an isolated placement (the `reviewer_placement` adapter) when that matters.

## Use

Run from the repository's main checkout; its `gbai.yaml` names the repository (`repo.name`) and the integration branch (`repo.integration_branch`):

```bash
gh auth status                                   # the owner's login is the bot login
s="${XDG_DATA_HOME:-$HOME/.local/share}/gbai/skills/gabo-reviewer/scripts"
bash "$s/reviewer.sh" preflight
bash "$s/reviewer.sh" head <pr> <40-hex sha>
```

Other modes: `due <pr> <sha>`, `list-due`, `once` (every due head, one at a time). The last line of `head` is always:

```text
VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>
```

A conflict is `blocking`; a retryable failure or a refused preflight is `error`; a moved head, a head with a final status or a verdict comment, or a refused or failed merge is `skipped` (the `SKIP`, `MERGE-SKIPPED` or `MERGE-FAILED` line above it says why).

## What one review does

1. Preflight: no `sudo`; the trusted location; `jq`, `flock`, `git`, `setsid`; fetches the integration branch into its own mirror and moves its own checkout (`<state>/base`) to the tip, refusing when that checkout is not clean; reads the repository settings from `gbai.yaml` there; `reviewer.placement` must be `local`; the base minimum (below); the `gh` login.
2. Skips a head that moved, a draft, a fork, a PR to another base, and a head that already has a final `reviewer/review` status or a verdict comment by the bot login.
3. Fetches GitHub's merge result `refs/pull/<n>/merge` and trusts it only when it is the integration-branch tip merged with the head; a conflict gets a blocking comment.
4. Runs `review.checks` as argv (no shell) in a fresh worktree of the merge result with `env -i`, its own `HOME`, Go caches under the state dir, `CI=true` and no credential. A `gabo check` or `gbai check` entry runs `REVIEWER_GABO_BIN`; `gabo check` never runs unless listed.
5. Reviews the diff with Claude Code, read-only, in the head checkout, with no GitHub token.
6. Posts one comment (first line the verdict, last lines the marker `<!-- <reviewer.signing.marker> sha=<head> -->` and the footer `🤖 gabo reviewer`), sets `reviewer/tests` and `reviewer/review`, and on a clean head runs the merge gate.
7. Merge gate (`merge-head.sh`): only with `review.auto_merge: true`, PR open, base the integration branch (never `main` or `master`), same repository, head still the reviewed SHA, both statuses `success` by the bot login, the newest marker comment clean and signing that SHA, and no human-only path; then `gh pr merge <pr> --merge --match-head-commit <sha>`. Human-only paths get a "needs human merge" comment instead.

## Built-in rules

- **Always human-only:** `.gbai/VERSION` and everything under `.gbai/skills/` (the catalog copy), independent of `review.human_only_paths` (TRD 9.4, issue #79).
- **Base minimum:** the integration branch's `.gbai/VERSION` (read from the reviewer's checkout, never from the PR head) is the minimum version of the gabo binary. An older binary, one without a release version, or none at all is refused with the upgrade command `go install <REVIEWER_GABO_INSTALL>@v<version>` (TRD 9.4). The scripts never update the binary. No stamp on the base means no minimum.
- **Upgrade PR:** a PR that changes `.gbai/VERSION`. Its checks run with `GABO_UPGRADE_PR=1` (the CLI validates only the config); its comment reports `upgrade PR: only config validated, human merge required` and lists the changed paths outside `.gbai/skills/` and the agent directories that the runtime skills of `runtimes.*` declare (`metadata.agent_dir` in `.gbai/skills/<prefix>-runtime-<runtime>/SKILL.md`). The stamp is human-only, so the scripts never merge it.

## Configuration

Repository settings come from `gbai.yaml` on the integration branch; an environment variable of the listed name wins (for tests and one-off runs):

| `gbai.yaml` key | variable | default |
|---|---|---|
| `repo.name` (caller's checkout) | `REVIEWER_REPO` | required |
| `repo.integration_branch` (caller's checkout) | `REVIEWER_BASE_BRANCH` | `dev`; `main` and `master` are refused |
| `review.checks` | `REVIEWER_CHECKS` (`;`-separated) | required |
| `review.auto_merge` | `REVIEWER_AUTO_MERGE` | `false` |
| `review.human_only_paths` | `REVIEWER_HUMAN_ONLY_PATHS` (space-separated) | `.claude/ .github/ gbai.yaml AGENTS.md CLAUDE.md`; entries ending in `/` are prefixes, others file names in any directory |
| `reviewer.signing.marker` | `REVIEWER_MARKER` | `gbai-reviewer-bot` |
| `reviewer.placement` | `REVIEWER_PLACEMENT` | `local` (the only one implemented) |

Script defaults: `scripts/config.env` (plain `KEY=value`, read as data, never executed; the environment wins):

| key | default | meaning |
|---|---|---|
| `REVIEWER_MODE` | `single-user` | Only mode here; `isolated` belongs to the placement adapter |
| `REVIEWER_ENGINE` | `claude` | Only engine here |
| `REVIEWER_BOT_LOGIN` | empty = `gh api user` login | Must equal the active gh login |
| `REVIEWER_GABO_BIN` | `gbai` | The gabo binary (`gabo` after T062) |
| `REVIEWER_GABO_INSTALL` | `github.com/testeos-z/gbai-team/apps/cli/cmd/gbai` | Package of the upgrade command |
| `REVIEWER_MAX_ATTEMPTS`, `REVIEWER_CHECK_TIMEOUT`, `REVIEWER_REVIEW_TIMEOUT`, `REVIEWER_HEAD_TIMEOUT` | `2`, `1800`, `900`, `7200` | Attempts and seconds |
| `REVIEWER_CLAUDE_MODEL`, `REVIEWER_CLAUDE_MAX_TURNS`, `REVIEWER_DIFF_MAX_BYTES` | Claude default, `40`, `200000` | Review engine |
| `REVIEWER_MERGE_TRIES`, `REVIEWER_MERGE_DELAY` | `5`, `3` | Asks for GitHub's merge result |
| `GBAI_REVIEWER_STATE_DIR` | `${XDG_STATE_HOME:-~/.local/state}/gbai/reviewer` | Own mirror, base checkout, worktrees, caches, logs |

Secrets go only in `${XDG_CONFIG_HOME:-~/.config}/gbai/reviewer.env` (mode 600, refused otherwise; override the path with `REVIEWER_ENV_FILE`): `CLAUDE_CODE_OAUTH_TOKEN` (optional; with it the review runs with an empty HOME, without it the owner's Claude Code login is used) and `REVIEWER_GH_TOKEN` (optional; else gh's stored login). Neither reaches the checks.

## Tests

Offline, with fake `gh`, `claude`, `go` and `gbai`, real `git` and a local bare repository standing in for GitHub:

```bash
for t in scripts/tests/*.test.sh; do bash "$t"; done
```

`go test ./internal/skills/` runs them too when `bash`, `git`, `jq`, `flock` and `setsid` are available.
