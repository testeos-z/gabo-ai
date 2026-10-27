---
name: gabo-db-master
description: "Trigger: database, schema, migrations, DB extensions, data model, seed data, db master. Designs DB configuration, schema, migrations and the data doc."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.1"
---
## Activation Contract

Use to choose and configure the database and write `docs/product/DB_SCHEMA.md`, the data and config schema, scaffolded from `templates/DB_SCHEMA.md`. It covers a product with a database and a config-only product whose persistence is files.

## Hard Rules

- Templates are canonical (D19): answer the template's `## Fixed questions` before writing its sections, and never rename or reorder its table columns.
- Persistence inventory first: every store or file, its location, scope, writer and whether it is committed.
- Recommend one database (or justify config-only) for the project's needs and justify it; list required extensions and configuration (pooling, timeouts, backups).
- Schema per file: tables with columns, types, constraints, indexes and relations (every table has a primary key and timestamps where useful), or config keys with type, required, description and example.
- Migrations are forward-only, ordered and reversible where the tool allows; never edit an applied migration.
- Data doc: seed and reference data, retention, PII and who can read it.

## Execution Steps

1. Read PRD and TRD for entities and access patterns.
2. Answer the fixed questions, then write storage choice and persistence inventory, schema per file (with an ER sketch or an example file), migration plan, data and open questions.
3. Map each index to the query it serves.

## Output Contract

Return the chosen database, extensions, the migration list and open questions.
