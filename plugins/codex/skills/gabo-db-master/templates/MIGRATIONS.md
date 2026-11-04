# MIGRATIONS template

Scaffold for `docs/product/MIGRATIONS.md`, the ordered log of every database migration or data script. There is no upfront schema: the schema is the sum of the applied migrations, and each change is documented step by step by the intent that needs it, in its `spec.md` section 4. The log fits a product with a database and a config-only product whose persistence is files. Answer every fixed question before filling its section; an answer you do not have goes to "Open questions", never an invented decision. Keep the table columns exactly as written.

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | Does the product persist data in a database, in files only (config-only), or both, and which engine or formats? | Conventions |
| Q2 | Which tool runs migrations, where do the files live and how are they named? | Conventions |
| Q3 | Which migration or script does each intent add, in which order? | Log: order, intent, file |
| Q4 | What does each migration or script do? | Log: purpose |
| Q5 | How is each one rolled back? | Log: rollback |
| Q6 | What is still undecided? | Open questions |

## Template

```markdown
# Migrations

| field | value |
|---|---|
| status | Draft |
| owner | gabo-db-master |
| date | <YYYY-MM-DD> |
| storage | <database name / config-only / both> |

## Conventions

- **Storage.** <Engine or file formats, required extensions and configuration.>
- **Tool and location.** <Migration tool, directory and file naming.>
- **Forward-only.** Never edit an applied migration; fix it with a new row.

## Log

One row per migration or script, in the order they are applied. Append rows; never reorder or delete them.

| order | intent | file | purpose | rollback |
|---|---|---|---|---|
| 1 | <NNNN-slug> | <path of the migration or script> | <why> | <how to revert, or "none: forward-only"> |

## Open questions

1. <question> (<owner or role that answers it>)
```
