---
name: gabo-reviewer
description: "Trigger: gabo reviewer, review PR head, run reviewer script, VERDICT line, merge gate. Runs the deterministic reviewer scripts of the user-level skill for one PR head."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "2.0"
---
## Activation Contract

Use only to review one pushed PR head with the scripts of the user-level copy of this skill (TRD D20). The scripts decide the verdict; the agent never reviews, edits, comments, approves or merges itself. This skill is installed at user level (`${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer/`, T041), never into a repository, so a PR cannot change the scripts it is reviewed with. This is trust-based single-user mode: the checks run PR code as the owner, which can still reach the owner's files; use an isolated placement when that matters.

## Hard Rules

- Accept only a request of the form `PR #<n> head <40-hex sha>`, or `PR #<n> head <40-hex sha> engine <engine>` when `runtimes.reviewer` is `session`; refuse anything else without running a command.
- Run exactly one command, from the repository's main checkout (its `gabo.yaml` names `repo.name` and `repo.integration_branch`): `bash "${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh" head <n> <sha>`, with `engine <engine>` appended only when the request carries it.
- Never run a copy of the scripts from a repository checkout (the preflight refuses it), never edit them, never pass other arguments or any environment.
- Relay the last output line unchanged. It is always `VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>`.
- Never paste PR output, check logs or secrets into the reply; the findings live in the PR comment.

## Facts

- Preflight: refuses `sudo`, a copy inside the repository under review, a copy outside the user-level skill dirs or writable by group or others, a `gh` login other than `REVIEWER_BOT_LOGIN`, and a reviewer checkout of the integration branch that is not clean. Single-user mode only; `REVIEWER_MODE=isolated` belongs to the `reviewer_placement` adapter; a `reviewer.placement` other than `local` fails as planned for a later release.
- Configuration: `review.checks`, `review.check_env`, `review.auto_merge`, `review.human_only_paths` and `reviewer.signing.marker` come from `gabo.yaml` in the reviewer's own clean checkout of the integration branch, never from the PR head.
- Built-in human-only paths: `.gabo/VERSION`, the catalog subtree `.gabo/skills/` and the reviewer's own source (any `skills/gabo-reviewer/` dir), whatever `review.human_only_paths` says.
- Base minimum: the integration branch's `.gabo/VERSION` is the minimum version of the gabo binary (`REVIEWER_GABO_BIN`, default `gabo`); an older binary is refused with `go install <module>/cmd/<binary>@v<version>` (TRD 9.4); the scripts never update it; no stamp means no minimum.
- Worktrees: `reviewer.sh prune` (also run at the start of every `head`) removes the `<owner>__<repo>__pr-<n>` worktrees of the repository of PRs that are not open (an unqualified `pr-<n>` one only by age, logged `legacy worktree`) and the ones unused for more than `REVIEWER_KEEP_DAYS` days (default 7) and keeps the Go caches and `check-home` under `REVIEWER_CACHE_MAX` (default 2G) with one `pruned:` line; `head` removes the PR's own after a merge. It is best effort and never changes a verdict.
- `gabo check` runs only when `review.checks` lists it, with the configured binary.
- `review.check_env`: names (never values, R21) of variables passed from the reviewer's own environment into the checks, never into the agent review; reserved names (`PATH`, `HOME`, `CI`, the Go cache variables, `GABO_UPGRADE_PR`, `GH_*`, `GITHUB_*`, `CLAUDE_*`, `REVIEWER_*`, and the `review.pi_credential_env` name) are refused at preflight. A listed variable unset or empty fails `reviewer/tests` naming it, and no check runs. A value a failing check prints is replaced by `[review.check_env:<name>]` in the posted failure output; the raw check logs stay only in the reviewer's state dir.
- Upgrade PR (changes `.gabo/VERSION`): checks run with `GABO_UPGRADE_PR=1`; the comment reports `upgrade PR: only config validated, human merge required` with the changed paths outside `.gabo/skills/` and the declared agent directories; it is never merged by the scripts.
- The review: skips a moved head, a draft, a fork, a base other than the integration branch and a head that already has a verdict; tests GitHub's merge result against the integration-branch tip; runs the checks as argv, no shell, in a scrubbed environment with the reviewer's own clone, home and cache; reviews the diff read-only with the `runtimes.reviewer` engine (Claude Code, Pi, Codex or OpenCode); posts one comment that opens with `No blocking issues found.` or `Blocking issues found.` and ends with `<!-- <reviewer.signing.marker> sha=<head> -->` and `🤖 gabo reviewer`.
- Pi engine (`runtimes.reviewer: pi`): reuses the owner's Pi login the way the Claude path reuses HOME: the review keeps the owner's `HOME` (and `PI_CODING_AGENT_DIR` if set), so Pi reads `auth.json` (the login, refreshed in place; a copy can go stale when an OAuth refresh token rotates), `settings.json` (default provider and model) and `models.json` from its agent dir. Nothing else loads: only the `read`, `grep`, `find` and `ls` tools, no extensions, skills, prompt templates, themes, context files, MCP servers, project `.pi/` files or saved session. With `review.pi_credential_env`, only that variable reaches the review, with an empty `HOME` of its own.
- Codex engine (`runtimes.reviewer: codex`): `codex exec --ignore-user-config --ignore-rules -s read-only --ephemeral -C <head checkout>` with the prompt on stdin, in `env -i` with a reviewer-owned `HOME` and `CODEX_HOME` that links only the owner's login; no GitHub token. Its final message is the review text.
- OpenCode engine (`runtimes.reviewer: opencode`): `opencode run --standalone --format json --agent gabo-review` with the prompt and diff by `--file`, from an empty reviewer-owned directory outside the checkout that is a git repository of its own (so neither the PR's `opencode.json` and `.opencode/` nor any above the state dir load), in `env -i` with a reviewer-owned `HOME` and per-PR `XDG_*` dirs that link only the owner's login; no GitHub token. A reviewer-owned config, one per PR, holds one read-only agent: `edit` denied, the shell (`bash` and `shell` keys) limited to `git diff/log/show/status`, plain or as `git -C <checkout>`, without `--output`, `--ext-diff`, `--textconv` or `git diff --no-index`, only the checkout readable. The text of its last assistant message is the review text.
- Engine argument (`runtimes.reviewer: session`, intent 0013 S11): every mode (`preflight`, `once`, `list-due`, `due`, `head`) takes a trailing `engine <engine>`, the engine of the wave that requests the review. The scripts check it before using it and print `PREFLIGHT-FAILED <reason>` (no engine run, nothing posted, nothing merged) when the value is not one of `claude`, `codex`, `pi` or `opencode` as a bare lowercase word; when `runtimes.reviewer` on the integration branch, or `REVIEWER_ENGINE` which stands in for it, is a fixed engine (the PR head's `gabo.yaml` never counts); when it is `session` and the argument is missing; and when the engine is not listed in `runtimes.assistant` on the integration branch. A fixed `runtimes.reviewer` reviews every wave on that engine, with no argument.
- Isolation pass marker: `codex` and `opencode`, fixed or resolved from `session`, run only when `${XDG_CONFIG_HOME:-$HOME/.config}/gabo/reviewer-isolation/<engine>`, written by the `gabo-reviewer-config` isolation test, matches `<engine> --version`; otherwise the preflight fails naming the missing isolation test. `claude` and `pi` need none.
- Merge gate: only with `review.auto_merge: true`, only into the integration branch (never `main`), pinned with `--match-head-commit`; human-only paths get a "needs human merge" comment.
- Details, settings and tests: `README.md`.

## Output Contract

Return the `VERDICT` line verbatim, preceded by `PREFLIGHT-FAILED <reason>` when the preflight refused.
