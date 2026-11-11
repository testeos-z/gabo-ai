---
name: gabo-reviewer-notes
description: "Trigger: how the reviewer works, review layers, coder-side review, review verdict, VERDICT line, reviewer marker comment, why blocked, merge gate. Explains both gabo review layers for other agents."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "2.2"
---
## Activation Contract

Use when an agent needs to understand or explain a review result. Agents never run or influence the reviewer-side gate themselves; the coder runs the user-level reviewer script itself after every push.

## Review layers

| layer | required | who declares it | what it decides |
|---|---|---|---|
| Coder-side review (optional) | no | the `methodology` slot adapter | nothing about merging; it only helps the coder before pushing |
| Reviewer-side gate (mandatory) | yes, for every PR into `dev` | `review.checks` and the user-level `gbai-reviewer` skill | the verdict and the merge gate |

### Coder-side review (optional)

- The `methodology` slot adapter declares whether a coder reviews its own work before pushing and how (for example `gabo-methodology-gentle-ai` runs gentle-ai RDD only when `gentle-ai review mode status` reports it on).
- It runs inside the coder's worktree with the coder's tools. Its outcome (granted, declined, passive, off, unavailable) is reported by the coder, never posted as a verdict.
- It never replaces, skips or weakens the reviewer-side gate. A methodology without coder-side review is valid.

### Reviewer-side gate (mandatory)

- After every push the coder runs the scripts of the user-level `gbai-reviewer` skill itself (`~/.claude/skills/gbai-reviewer/scripts/`) through its shell, no relay agent in between (intent 0015 S3, S4, D5): it validates `PR #<n> head <40-hex sha>` (or `PR #<n> head <40-hex sha> engine <engine>`, see Review engine), runs the script and acts on its final `VERDICT` line (D20). It never reviews, edits, comments or merges as the reviewer.
- The scripts live outside every repository, so PR code cannot modify the code that reviews it. They refuse to run when they are not under the user-level skills dir or are writable by others.
- Each run has two parts: the repository checks (`review.checks` in `gabo.yaml`) and an agent review of the diff. Both must pass for a clean verdict.

### Review engine

- Fixed engine: a fixed `runtimes.reviewer` reviews every wave on that engine; the request carries no `engine` field and one is refused.
- Optional `engine` field (intent 0013 S11): only when `runtimes.reviewer` is `session`, the request is `PR #<n> head <40-hex sha> engine <engine>`, where `<engine>` is the engine of the wave that requests the review (the coder's `gabo-engine:` value). It is required under `session`. The coder passes it to the scripts as an argument; they accept it only when `runtimes.reviewer` on the integration branch is `session` and the engine is listed in `runtimes.assistant` there, else they print `PREFLIGHT-FAILED` with the reason.
- `codex` and `opencode`, fixed or resolved from `session`, need the isolation pass marker of that engine.

## Reviewer isolation

- Own clone: the scripts work in their own state dir (`~/.local/state/gabo/reviewer`) with their own clone of the repository, never the coder's worktree; never edit or build inside it.
- They re-read the PR and skip a head that moved; they test-merge the pinned head into a fresh `dev`. A merge conflict is blocking.
- Checks run with their own `HOME`, their own build and module cache and `CI=true`, in a clean environment (`env -i`) inside a credential-free copy of the merged tree. Only the variables named in `review.check_env` (from the base `gabo.yaml`) are added, with values from the reviewer's own environment; one unset or empty fails the checks. The agent review never gets them.
- The agent review runs read-only in a clean environment with no `gh` or git credentials: with a model token configured it gets an empty `HOME` of its own and only that token; without one it uses the owner's model login.
- This is trust-based, not an OS-user boundary.

## Modes and placement

- Single-user mode (the local default): the scripts run as the owner with the owner's `gh` login; no extra users. The preflight refuses unless the scripts run from a clean checkout of the integration branch. The marker is trust-based in this mode.
- Isolated mode: a separate reviewer user and identity, available only through the `reviewer_placement` or `agent_host` adapter; the bundled scripts refuse `REVIEWER_MODE=isolated`.
- Placement: only `local` is implemented in the first release (D6). Separate server, GitHub Actions and GitLab CI placements are Later.

## Verdict

- The verdict comes from local evidence (checks, conflict, findings), never from an agent's message. One verdict per pushed head (R26); a head that already has one is skipped.
- The verdict comment carries the marker `<!-- gbai-reviewer-bot sha=<head> -->` with the reviewed head SHA and the footer `🤖 gabo reviewer`. It never approves: GitHub forbids approving your own PR.
- The run ends with one `VERDICT` line: `VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>`.
- Merge gate: the newest marker comment by the owner's login must sign the current head and be clean; merges go only into `dev`, pinned to the reviewed SHA, never `main`, never PRs touching `review.human_only_paths`.

## Hard Rules

- Do not argue with a verdict; fix the findings and push a new head.
- Read the marker comment on the PR for findings; the `VERDICT` line only states the outcome.
- Never post a comment, PR body or review containing the marker, even quoted; a forged marker could sign a head the reviewer never checked.
- A coder-side review result is never presented as the gate's verdict.
