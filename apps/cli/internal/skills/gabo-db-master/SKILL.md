---
name: gabo-db-master
description: "Trigger: database, schema, migrations, DB extensions, data model, seed data, db master. Designs DB configuration, schema, migrations and the data doc."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.2"
---
## Activation Contract

Use to choose and configure the database and write `docs/product/DB_SCHEMA.md`, the data and config schema, scaffolded from `templates/DB_SCHEMA.md`. It covers a product with a database and a config-only product whose persistence is files.

## Hard Rules

- Templates are canonical (D19): answer the template's `## Fixed questions` before writing its sections, and never rename or reorder its table columns.
- Fixed questions one at a time: ask the template's `## Fixed questions` in order, one per message, and wait for each answer before the next (manual mode). In auto mode answer each yourself, mark it **Proposed (auto)** with a one-line reason and present them together for one-pass approval.
- Persistence inventory first: every store or file, its location, scope, writer and whether it is committed.
- Recommend one database (or justify config-only) for the project's needs and justify it; list required extensions and configuration (pooling, timeouts, backups).
- Schema per file: tables with columns, types, constraints, indexes and relations (every table has a primary key and timestamps where useful), or config keys with type, required, description and example.
- Migrations are forward-only, ordered and reversible where the tool allows; never edit an applied migration.
- Data doc: seed and reference data, retention, PII and who can read it.

## Traceability check

Run it before each next planning step: before moving to the next document, before creating or updating tasks, and before returning (D19). Read the stable template tables: PRD stories and requirements, TRD decisions and requirement coverage, plan tasks and coverage. Rows marked removed (a title starting `(removed <date>)`, or a status withdrawn or superseded) are ignored on both sides of every link. A gap is:

- a story without requirement;
- a requirement without decision, task or verification;
- a task without story;
- a dependency cycle, a dangling id (a story, requirement, decision or task id no live row defines), or a `depends_on` / `unblocks` mismatch.

Any gap blocks the next planning step: fix it in your own documents or return it as an open question naming the ids; never skip it. A document not written yet is reported as not checked, not as passing.

## Execution Steps

1. Read PRD and TRD for entities and access patterns.
2. Answer the fixed questions, then write storage choice and persistence inventory, schema per file (with an ER sketch or an example file), migration plan, data and open questions.
3. Map each index to the query it serves.
4. Run the traceability check before the next planning step.

## Output Contract

Return the chosen database, extensions, the migration list, traceability gaps (or none) and open questions.
