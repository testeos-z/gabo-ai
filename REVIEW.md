# Review policy

This file is the review policy of gabo-ai. The reviewer reads it from the
integration branch (`dev`), never from the pull request head, so a PR that edits
it is reviewed against the `dev` copy.

## Passes

Run three passes over the diff in one review, and tag each finding with its
pass: `[bugs]`, `[security]` or `[compliance]`. When no pass finds anything,
state that the three passes ran and found nothing.

### Pass 1: bugs

Logic errors, broken edge cases and subtle regressions, including missing tests
for changed behavior.

### Pass 2: security

Injection risks, authentication and permission gaps, secrets or PII in logs,
and secrets passed anywhere but stdin.

### Pass 3: compliance

Check that the change matches the spec (`spec.md` of the task's intent), the
implementation plan (`plan.md` of the same intent) and the design principles
(`AGENTS.md` and `docs/cloud-agent-flow.md`, including the hexagonal layout and
the hard rules). Name the `spec.md` and `plan.md` you compared against, or write
`compliance: not run: <reason>`.

## What Important means here

Reserve Important for findings that would break behavior, leak data or breach a policy. Style and naming are nits.

Only an Important finding caused by the pull request is blocking.

## Cap the nits

Report at most five nits per review; summarize the rest as a count.

## Do not report

- Findings in generated files.
- Anything `review.checks` in `gabo.yaml` already enforces (build, vet and tests).
