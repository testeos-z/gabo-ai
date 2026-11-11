---
name: gabo-reviewer-placement-local
description: "Trigger: reviewer placement, local reviewer, where the reviewer runs, single-user review, reviewer_placement adapter. Local single-user adapter for the gabo reviewer_placement slot."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.0"
---
## Activation Contract

Use when a coder needs one review of a pushed PR head. This adapter fills the `reviewer_placement` slot contract (`gabo-contract-reviewer-placement`, TRD 9.1; R9, R22, R26, D6, D20) with placement `local` in single-user mode: the scripts of the user-level `gabo-reviewer` skill run on the owner's machine, as the owner, with the owner's `gh` login. Point `slots.reviewer_placement.adapter` at this skill's `SKILL.md` and set `reviewer.placement: local`.

Isolated two-user mode (a separate reviewer user and identity) is another adapter, not built here; the bundled scripts refuse `REVIEWER_MODE=isolated`. Placements beyond `local` (separate server, GitHub Actions, GitLab CI) are Later.

## Hard Rules

- Accept only a request of exactly the form `PR #<n> head <40-hex sha>`, or `PR #<n> head <40-hex sha> engine <engine>` only when `runtimes.reviewer` is `session` in the main checkout's `gabo.yaml` (intent 0013 S11, D5; `<engine>` is the engine of the wave that requests the review). Anything else is refused: an `engine` field under a fixed `runtimes.reviewer`, a missing `engine` under `session`, extra words or any other shape; refuse it before running any command.
- Pass the engine to the scripts as an argument, never as environment: append `engine <engine>` to the command only when the request carries it. The scripts check it again against the integration branch (bare lowercase word, listed in `runtimes.assistant`, isolation pass marker) and print `PREFLIGHT-FAILED <reason>` when it does not hold.
- Run only the installed user-level copy of the scripts, `${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/` (installed by `gabo init`, T041), never a copy from a repository checkout, never edited, with no other arguments and no extra environment.
- Run from a checkout of the repository (its `gabo.yaml` names `repo.name` and `repo.integration_branch`): the coder runs it from its task worktree, and never through `sudo`. This is safe: the scripts read only `repo.name` and `repo.integration_branch` from the caller's `gabo.yaml` (a human-only path a PR cannot change without ending in `needs-human-merge`) and take `review.checks` and every other setting from `gabo.yaml` in their own clean checkout of the integration branch, never from the PR head (`gabo-reviewer-notes`, Reviewer isolation).
- One run per request: the scripts skip a head that moved or already has a verdict (R26); never retry in a loop.
- The coder runs the scripts itself and never reviews, edits, comments, approves or merges itself, and never posts the reviewer marker; the scripts own the verdict comment and the merge gate.
- Never paste PR output, check logs or secrets into the reply; the findings live in the PR comment.

## Operations

### trigger review

Validate the request (read `runtimes.reviewer` from the checkout's `gabo.yaml` to decide whether an `engine` field is allowed or required), then run one command. For `PR #<n> head <40-hex sha>`:

```bash
bash "${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh" head <n> <sha>
```

For `PR #<n> head <40-hex sha> engine <engine>` under `session`, the same command with the engine as an argument:

```bash
bash "${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh" head <n> <sha> engine <engine>
```

The output goes to stdout; its last line is the `VERDICT` line. The scripts work in their own state dir (`${XDG_STATE_HOME:-$HOME/.local/state}/gabo/reviewer`) with their own clone, home and cache, never in the caller's checkout.

### report placement

Report one line:

```text
placement=local mode=single-user
```

The preflight refuses any other `reviewer.placement` or `REVIEWER_MODE`, so every run that passed it ran here, as the owner, in single-user mode. To confirm the settings without reviewing, run `bash "${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh" preflight` (with `engine <engine>` appended under `session`); its `PREFLIGHT-OK` line shows `mode=`, `engine=` and `placement=`.

### return verdict

Return the last output line unchanged:

```text
VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>
```

When the preflight refused, the output holds `PREFLIGHT-FAILED <reason>`; return that line first, then the `VERDICT` line. The full verdict lives in the reviewer's comment on PR `#<n>`, which carries the reviewed head SHA in its marker.

## Output Contract

Return the placement line, then `PREFLIGHT-FAILED <reason>` when present, then the `VERDICT` line verbatim, and point to the reviewer's comment on the PR for findings.
