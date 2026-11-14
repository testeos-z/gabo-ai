## Commands

The reviewer runs these commands on every PR (`review.checks` in `gabo.yaml`). Run each one from the repository root before you push. A line starting with `$` is the command; the lines under it are its healthy output, and each output ends with `exit: 0`. A command passed when it prints that output.

```text
{{commands}}
```

## Repeated mistakes

**Two-times rule:** a mistake seen once is recorded, not added here, so that a second occurrence can be recognized. When an agent makes the same mistake a second time, its correction is proposed for this section, naming both occurrences (the PR or task of each). No correction is added on the first occurrence.

No entries yet.

## Review findings

When a review flags a mistake for the second time, the correction is proposed for the "Repeated mistakes" section in a pull request opened by an agent, naming both findings. The owner accepts it or declines it. No agent merges a change to this file.
