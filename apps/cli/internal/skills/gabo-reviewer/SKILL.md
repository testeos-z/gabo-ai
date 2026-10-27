---
name: gabo-reviewer
description: "Trigger: gabo reviewer, review PR head, run reviewer script, VERDICT line, merge gate. Runs the deterministic reviewer scripts of the user-level skill for one PR head."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "2.0"
---
## Activation Contract

Use only to review one pushed PR head with the scripts of the user-level copy of this skill (TRD D20). The scripts decide the verdict; the agent never reviews, edits, comments, approves or merges itself. This skill is installed at user level (`${XDG_DATA_HOME:-~/.local/share}/gbai/skills/gabo-reviewer/`, T041), never into a repository: a PR cannot change the code that reviews it.

## Hard Rules

- Accept only a request of the form `PR #<n> head <40-hex sha>`; refuse anything else without running a command.
- Run exactly one command, from the repository's main checkout (its `gbai.yaml` names `repo.name` and `repo.integration_branch`): `bash "${XDG_DATA_HOME:-$HOME/.local/share}/gbai/skills/gabo-reviewer/scripts/reviewer.sh" head <n> <sha>`.
- Never run a copy of the scripts from a repository checkout (the preflight refuses it), never edit them, never pass extra arguments or environment.
- Relay the last output line unchanged. It is always `VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>`.
- Never paste PR output, check logs or secrets into the reply; the findings live in the PR comment.

## Facts

- Preflight: refuses `sudo`, a copy inside the repository under review, a copy outside the user-level skill dirs or writable by group or others, a `gh` login other than `REVIEWER_BOT_LOGIN`, and a reviewer checkout of the integration branch that is not clean. Single-user mode only; `REVIEWER_MODE=isolated` belongs to the `reviewer_placement` adapter; a `reviewer.placement` other than `local` fails as planned for a later release.
- Configuration: `review.checks`, `review.auto_merge`, `review.human_only_paths` and `reviewer.signing.marker` come from `gbai.yaml` in the reviewer's own clean checkout of the integration branch, never from the PR head.
- Built-in human-only paths: `.gbai/VERSION` and the catalog subtree `.gbai/skills/`, whatever `review.human_only_paths` says.
- Base minimum: the integration branch's `.gbai/VERSION` is the minimum version of the gabo binary (`REVIEWER_GABO_BIN`, `gbai` until T062); an older binary is refused with `go install <module>/cmd/<binary>@v<version>` (TRD 9.4); the scripts never update it; no stamp means no minimum.
- `gabo check` (or `gbai check`) runs only when `review.checks` lists it, with the configured binary.
- Upgrade PR (changes `.gbai/VERSION`): checks run with `GABO_UPGRADE_PR=1`; the comment reports `upgrade PR: only config validated, human merge required` with the changed paths outside `.gbai/skills/` and the declared agent directories; it is never merged by the scripts.
- The review: skips a moved head, a draft, a fork, a base other than the integration branch and a head that already has a verdict; tests GitHub's merge result against the integration-branch tip; runs the checks as argv, no shell, in a scrubbed environment with the reviewer's own clone, home and cache; reviews the diff with Claude Code read-only; posts one comment that opens with `No blocking issues found.` or `Blocking issues found.` and ends with `<!-- <reviewer.signing.marker> sha=<head> -->` and `🤖 gabo reviewer`.
- Merge gate: only with `review.auto_merge: true`, only into the integration branch (never `main`), pinned with `--match-head-commit`; human-only paths get a "needs human merge" comment.
- Details, settings and tests: `README.md`.

## Output Contract

Return the `VERDICT` line verbatim, preceded by `PREFLIGHT-FAILED <reason>` when the preflight refused.
