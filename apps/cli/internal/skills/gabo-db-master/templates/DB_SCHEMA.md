# DB_SCHEMA template

Scaffold for `docs/product/DB_SCHEMA.md`, the data and config schema of the product. It fits both cases: a product with a database, and a config-only product whose persistence is files. Answer every fixed question before filling its section; an answer you do not have goes to section 5 as an open question, never an invented decision. Keep the table columns exactly as written.

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | Does the product persist data in a database, in files only (config-only), or both? | 1. Storage choice |
| Q2 | Which database and extensions, or which file formats, fit the access patterns, and why? | 1. Storage choice |
| Q3 | Which configuration does the store need (pooling, timeouts, backups, schema validation)? | 1. Storage choice |
| Q4 | Which stores and files exist, where, who writes them and are they committed? | 1. Storage choice / Persistence inventory |
| Q5 | For each table or file: which columns or keys, types, constraints and examples? | 2. Schema per file |
| Q6 | Which index serves which query? | 2. Schema per file |
| Q7 | Which ordered, forward-only migrations get from today's shape to the target, and how is each rolled back? | 3. Migration |
| Q8 | Which seed and reference data, PII, secrets and retention rules apply, and who can read the data? | 4. Data |
| Q9 | What is still undecided? | 5. Open questions |

## Template

```markdown
# Data and Config Schema

| field | value |
|---|---|
| status | Draft |
| owner | gabo-db-master |
| date | <YYYY-MM-DD> |
| storage | <database name / config-only / both> |
| linked docs | [PRD](PRD.md), [TRD](TRD.md) |

## 1. Storage choice

<Database or config-only, the chosen engine or file formats with the reason, required extensions and configuration (pooling, timeouts, backups, schema validation).>

### Persistence inventory

| store | kind | location | owner / scope | written by | committed? |
|---|---|---|---|---|---|
| <name> | <database / config file / generated file / state dir> | <path or connection> | <one person / team / host> | <who or what writes it> | <yes / no> |

## 2. Schema per file

One subsection per table or file. Database tables use the column and index tables; config files use the key table. Mark keys or columns not implemented yet as **proposed (not implemented)**.

### 2.1 <table or file>

<Database table:>

| column | type | null | default | constraints | description |
|---|---|---|---|---|---|
| id | <uuid> | no | <gen_random_uuid()> | primary key | <meaning> |

| index | columns | query it serves |
|---|---|---|
| <name> | <columns> | <query or access pattern> |

<Config file:>

| key | type | required | description | example |
|---|---|---|---|---|
| `<key>` | <type> | <yes / no> | <meaning and validation> | `<value>` |

<ER sketch (Mermaid) for databases; an annotated example file for config-only products.>

## 3. Migration

Forward-only and ordered; never edit an applied migration, fix it with a new one.

| order | migration | purpose | rollback |
|---|---|---|---|
| 1 | <change> | <why> | <how to revert, or "none: forward-only"> |

<Backward compatibility: how old data or files are read during the transition.>

## 4. Data

- **Seed and reference data.** <What ships, or "none".>
- **PII.** <Which fields, who can read them.>
- **Secrets.** <Never stored; where they come from instead.>
- **Retention.** <How long each store keeps data and how it is removed.>

## 5. Open questions

1. <question> (<owner or role that answers it>)
```
