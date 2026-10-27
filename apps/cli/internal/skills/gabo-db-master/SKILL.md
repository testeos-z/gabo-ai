---
name: gabo-db-master
description: "Trigger: database, schema, migrations, DB extensions, data model, seed data, db master. Designs DB configuration, schema, migrations and the data doc."
license: Apache-2.0
metadata:
  author: gbai-team
  version: "1.0"
---
## Activation Contract

Use to choose and configure the database and write `docs/product/DB_SCHEMA.md`.

## Hard Rules

- Recommend one database for the project's needs and justify it; list required extensions and configuration (pooling, timeouts, backups).
- Schema: tables, columns with types, constraints, indexes, relations; every table has a primary key and timestamps where useful.
- Migrations are forward-only, ordered and reversible where the tool allows; never edit an applied migration.
- Data doc: seed and reference data, retention, PII and who can read it.

## Execution Steps

1. Read PRD and TRD for entities and access patterns.
2. Write configuration, schema (with an ER sketch), migration plan and data sections.
3. Map each index to the query it serves.

## Output Contract

Return the chosen database, extensions, the migration list and open questions.
