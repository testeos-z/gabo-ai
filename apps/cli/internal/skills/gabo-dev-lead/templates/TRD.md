# TRD template

Scaffold for `docs/product/TRD.md`, the global technical conventions every intent follows: the stack and tooling the user chose, architecture, code and test conventions, and MADR-style decisions (D14). It holds no per-change design and no per-requirement mapping; those live in each intent's `spec.md` and `plan.md`. The user chooses the technologies and the assistant helps configure them: record a choice, never impose one. Answer every fixed question before filling its section; an answer you do not have goes to section 10 as an open question, never an invented decision. Keep the table columns exactly as written: the traceability check reads them (D19).

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | Which languages, frameworks, runtimes and tools did the user choose, at which versions? | 1. Stack and tooling |
| Q2 | Which technical, organizational and legal constraints are fixed? | 2. Constraints |
| Q3 | How is the system decomposed (C4 context and containers), and which layering and boundary rules apply? | 3. Architecture conventions |
| Q4 | Which naming, error handling, configuration, logging and security conventions apply to code? | 4. Code conventions |
| Q5 | Which test layers exist, what does each cover, and what runs in the repository checks? | 5. Test conventions |
| Q6 | Where and how does it run (environments, hosts, delivery)? | 6. Deployment and environments |
| Q7 | Which global decisions were made, with which options, consequences and status? | 7. Architecture decisions |
| Q8 | Which risks and technical debt exist, and how are they mitigated? | 8. Risks and technical debt |
| Q9 | Which terms need a shared meaning? | 9. Glossary |
| Q10 | What is still undecided? | 10. Open questions |

## Template

```markdown
# Technical Requirements (TRD)

| field | value |
|---|---|
| status | Draft |
| owner | gabo-dev-lead |
| version | 0.1 |
| date | <YYYY-MM-DD> |
| linked docs | [PRD](PRD.md), [MIGRATIONS](MIGRATIONS.md), `DESIGN.md` (optional), intents `docs/product/intents/` |

## 1. Stack and tooling

The user chooses; the assistant helps configure. One row per area.

| area | choice | version | chosen by | notes |
|---|---|---|---|---|
| <language / framework / database / test runner / CI> | <choice> | <version> | <owner / proposed (auto)> | <configuration notes> |

## 2. Constraints

| constraint | kind | source |
|---|---|---|
| <runtime, language, platform> | <technical / organizational / legal> | <PRD constraint, owner decision> |

## 3. Architecture conventions

<C4 system context and container view (Mermaid welcome), layering and dependency rules, module boundaries and where new code goes.>

## 4. Code conventions

<Naming, error handling, configuration, logging and observability, security, persistence (migrations are logged in MIGRATIONS).>

## 5. Test conventions

<Test layers (unit, integration, end-to-end, inspection), what each covers, test-first rule, what runs in the repository checks.>

## 6. Deployment and environments

<Environments, hosts, delivery pipeline and what runs where.>

## 7. Architecture decisions

MADR-style: one row per global decision; superseded decisions stay with their status. A decision for one change only goes in that intent's spec.

| id | decision | options considered | choice | consequences | status | requirement ids |
|---|---|---|---|---|---|---|
| D1 | <question being decided> | <option A; option B> | <chosen option and reason> | <positive and negative consequences> | <proposed / accepted / superseded by D..> | <R.. or —> |

## 8. Risks and technical debt

| id | risk or debt | decision ids | mitigation |
|---|---|---|---|
| <RK1 / TD1> | <description> | <D..> | <action> |

## 9. Glossary

| term | meaning |
|---|---|
| <term> | <meaning> |

## 10. Open questions

1. <question> (<owner or role that answers it>)
```
