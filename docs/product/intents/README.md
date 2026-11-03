# Intents

One folder per change to gabo-ai. A change starts as an intent, gets one spec, and once a human accepts the spec it gets one plan and its tasks. The tracker is the global view of the work; there is no global plan.

## Layout

```text
docs/product/intents/
  README.md            this file
  <NNNN-slug>/
    intent.md          the problem, the affected users and the desired outcome (gabo-pm)
    spec.md            requirements, acceptance criteria, UX and flow, data changes, and the human acceptance record
    plan.md            ordered steps and the task table (gabo-dev-lead)
    <review findings>  findings recorded while the change is reviewed
```

- `<NNNN>` is the next free four-digit number. `<slug>` is the title in lower case, with every run of non-alphanumeric characters replaced by one `-`. An intent id is never reused or renumbered.
- The documents are written from the templates of the directiva skills: `gabo-pm` (`intent.md`, `spec.md`), `gabo-dev-lead` (`plan.md`), with UX and flow from `gabo-frontend` and migrations from `gabo-db-master` in the spec.
- A tracker product card can hold several intents. Each task belongs to exactly one intent, and its `blocked-by` links may cross intents (written `<NNNN>/T<n>` in a plan).
- One wave runs one intent ([TRD 5.1](../TRD.md#51-wave-run-r3-r4-r5-r10-r13)). The spec is accepted by a human, one intent at a time.

## Checks

`gabo check` validates every folder here offline: the folder name and a unique number, an `intent.md` in each folder, the stage order (no `plan.md` without a `spec.md` whose acceptance table a human filled) and the plan task table (ids, cross-intent references, `depends_on` mirrored by `unblocks`, no dependency cycle). Plain files in this folder, such as this README, are skipped.

There are no intents yet. Do not create one by hand to fill the folder; directiva opens the first one when the owner asks for a change.
