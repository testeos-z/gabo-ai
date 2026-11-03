---
name: gabo-db-master
description: "Trigger: database, schema, migrations, DB extensions, data model, seed data, db master. Designs DB configuration, schema, migrations and the data doc."
license: Apache-2.0
metadata:
  author: gabo-ai
  version: "1.2"
---
## Activation Contract

Use to choose and configure the database and write `docs/product/MIGRATIONS.md`, the ordered log of migrations and data scripts, scaffolded from `templates/MIGRATIONS.md`, plus section 4 of each intent's `spec.md`. It covers a product with a database and a config-only product whose persistence is files.

## Hard Rules

- Templates are canonical (D19): answer the template's `## Fixed questions` before writing its sections, and never rename or reorder its table columns.
- Fixed questions one at a time: ask the template's `## Fixed questions` in order, one per message, and wait for each answer before the next (manual mode). In auto mode answer each yourself, mark it **Proposed (auto)** with a one-line reason and present them together for one-pass approval.
- Persistence inventory first: every store or file, its location, scope, writer and whether it is committed. It lives in the `## Conventions` of `docs/product/MIGRATIONS.md`; a change that adds or alters a store also lists it in its spec's section 4.
- Recommend one database (or justify config-only) for the project's needs and justify it; list required extensions and configuration (pooling, timeouts, backups).
- No upfront schema: the schema is the sum of the applied migrations. Each change documents its data steps in its intent's `spec.md` section 4: the ordered migrations or scripts, with the tables, columns, types, constraints, indexes and relations they touch (every table has a primary key and timestamps where useful), or the config keys with type, required, description and example, and how each step rolls back.
- Every migration or script gets one appended row in the `MIGRATIONS.md` log, citing its intent; migrations are forward-only, ordered and reversible where the tool allows; never edit an applied migration.
- Data doc: seed and reference data, retention, PII and who can read it.

## Traceability check

Run it before each next planning step: before moving to the next document, before creating or updating tasks, and before returning (D19). Read the intent folders `docs/product/intents/<NNNN-slug>/` (`intent.md`, the spec requirements and its acceptance record, the plan task table) and the tracker tasks through the `tasks` contract. Rows marked removed (a title starting `(removed <date>)`, or a status withdrawn or superseded) are ignored on both sides of every link. A gap is:

- an intent without spec;
- an accepted spec without plan;
- a plan task without a tracker task, or the reverse;
- a task whose parent is not exactly one intent;
- a requirement in the spec not covered by any plan task;
- a dependency cycle;
- a dangling id, including a cross-intent `<NNNN>/T<n>` reference;
- a `depends_on`/`unblocks` mismatch.

Any gap blocks the next planning step: fix it in your own documents or return it as an open question naming the ids; never skip it. A document not written yet is reported as not checked, not as passing.

## Execution Steps

1. Read the PRD, the TRD conventions, `MIGRATIONS.md` and the intent's `intent.md` and `spec.md` for entities and access patterns.
2. First time: answer the MIGRATIONS fixed questions, then write its conventions (storage choice, persistence inventory, tool and location) from the template.
3. Per intent: answer the spec's DB migration question, then write the spec's section 4 (ordered steps with an ER sketch or an example file, rollback per step) or "not applicable" with the reason, and append one log row per migration or script.
4. Map each index to the query it serves.
5. Run the traceability check before the next planning step.

## Output Contract

Return the chosen database, extensions, the migration rows added per intent, traceability gaps (or none) and open questions.
