# reviewer-local: example adapter for the gbai reviewer_placement contract

Fills the `reviewer_placement` slot with placement `local` in single-user mode: the scripts of the user-level `gbai-reviewer` skill run on the owner's machine with the owner's `gh` login. Copy it to `.gbai/adapters/reviewer-local.md`, point `slots.reviewer_placement.adapter` at it and set `reviewer.placement: local`.

## Operations

### trigger review

Validate the request is exactly `PR #<n> head <40-hex sha>`, then run one command from any directory:

```bash
bash ~/.claude/skills/gbai-reviewer/scripts/reviewer.sh head <n> <sha>
```

Never run the scripts from a repository checkout and never pass extra arguments or environment.

### report placement

Placement `local`, mode `single-user`: the run happened on this machine as the owner, in the scripts' own state dir (`~/.local/state/gbai/reviewer`).

### return verdict

Relay the last output line unchanged:

```text
VERDICT PR #<n> head <sha> verdict=<clean|blocking|error|skipped> merge=<merged|needs-human-merge|skipped|none>
```

Precede it with `PREFLIGHT-FAILED <reason>` when the preflight refused. The findings live in the reviewer's comment on the PR, which carries the head SHA.
