# Lessons

Versioned record of what went wrong in the gabo workflow (intent 0030, S14 and S17). It reaches `dev` through a documents PR like any other planning document (D7).

- An incident is a coder run that reported done and was later found wrong by the owner, the assistant or the reviewer (S13). Each incident gets one entry in `## Incidents`.
- A mistake seen once goes to `## Mistakes seen once`, so that a second occurrence can be recognized (D8).

## Entry template

Copy the block, fill every field, and add the entry under `## Incidents`. Write `cause: unknown` when the cause is not known. Keep `eval task: pending fix` until the fix ships; then name the eval task (S15).

```md
### <YYYY-MM-DD> <short title>

- date: <YYYY-MM-DD>
- task and PR: <task issue #n, PR #n>
- what was reported: <what the coder reported as done>
- what was found wrong: <what was later found wrong>
- cause: <cause, or `cause: unknown`>
- draft intent: <link to docs/product/intents/<NNNN-slug>/intent.md>
- eval task: pending fix
```

## Incidents

No incident recorded yet.

## Mistakes seen once

A mistake seen once is recorded here with the task and PR of its first occurrence. When the same mistake is seen again, the two-times rule applies: a proposal for the repeated-mistakes section of the instruction file names both occurrences (S17). Record the second occurrence in the same entry when it is seen.

```md
### <short description of the mistake>

- first occurrence (task and PR): <task issue #n, PR #n>
- second occurrence (task and PR): none yet
```

No mistake seen once yet.
