# TRD template

Scaffold for `docs/product/TRD.md`: arc42 sections, C4 views and MADR-style decisions (D14). Answer every fixed question before filling its section; an answer you do not have goes to section 13 as an open question, never an invented decision. Keep the table columns exactly as written: the traceability check reads them (D19).

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | Which PRD requirements and quality goals drive the architecture? | 1. Introduction and quality goals |
| Q2 | Which technical, organizational and legal constraints are fixed? | 2. Constraints |
| Q3 | Which users and external systems does the product talk to, and through which interfaces? | 3. Context and scope |
| Q4 | What is the core solution approach (technology, decomposition, key patterns)? | 4. Solution strategy |
| Q5 | Which building blocks (C4 containers and components) exist, and which interfaces do they expose? | 5. Building block view |
| Q6 | Which runtime scenarios matter most, step by step? | 6. Runtime view |
| Q7 | Where and how does it run (environments, hosts, delivery)? | 7. Deployment view |
| Q8 | Which concepts cut across blocks (configuration, errors, security, observability, data)? | 8. Crosscutting concepts |
| Q9 | Which decisions were made, with which options, consequences and status? | 9. Architecture decisions |
| Q10 | How is each requirement covered by decisions, sections, stories and a test layer? | 9. Architecture decisions / Requirement coverage |
| Q11 | Which quality scenarios and test layers prove the quality goals? | 10. Quality requirements |
| Q12 | Which risks and technical debt exist, and how are they mitigated? | 11. Risks and technical debt |
| Q13 | Which terms need a shared meaning? | 12. Glossary |
| Q14 | What is still undecided? | 13. Open questions |

## Template

```markdown
# Technical Requirements (TRD)

| field | value |
|---|---|
| status | Draft |
| owner | gabo-dev-lead |
| version | 0.1 |
| date | <YYYY-MM-DD> |
| linked docs | [PRD](PRD.md), stories `docs/product/stories/`, [MIGRATIONS](MIGRATIONS.md), `DESIGN.md`, intents `docs/product/intents/` |

## 1. Introduction and quality goals

<What the system does, in two or three sentences, and the PRD goals it serves.>

| quality goal | scenario | requirement ids |
|---|---|---|
| <reliability> | <Given ... When ... Then ... (measurable)> | <R..> |

## 2. Constraints

| constraint | kind | source |
|---|---|---|
| <runtime, language, platform> | <technical / organizational / legal> | <PRD constraint, owner decision> |

## 3. Context and scope

<C4 system context: users and external systems, with the interface to each. Diagram welcome (Mermaid).>

## 4. Solution strategy

<Core technology choices, decomposition, key patterns, each linked to the quality goal it serves.>

## 5. Building block view

<C4 container view, then components where useful: each block, its responsibility and the interfaces it exposes or consumes.>

## 6. Runtime view

### 6.1 <Scenario> (<R..>)

<Numbered steps across building blocks; error paths included.>

## 7. Deployment view

<Environments, hosts, delivery pipeline and what runs where.>

## 8. Crosscutting concepts

<Configuration, error handling, security, observability, data and persistence (details in MIGRATIONS and each intent's spec).>

## 9. Architecture decisions

MADR-style: one row per decision; superseded decisions stay with their status.

| id | decision | options considered | choice | consequences | status | requirement ids |
|---|---|---|---|---|---|---|
| D1 | <question being decided> | <option A; option B> | <chosen option and reason> | <positive and negative consequences> | <proposed / accepted / superseded by D..> | <R..> |

### Requirement coverage

Every PRD requirement maps to at least one decision or section, its stories (or "no story yet") and the test layer that verifies it.

| req | decisions | sections | stories | verification → test layer |
|---|---|---|---|---|
| R1 | <D..> | <section numbers> | <US-0NN> | <PRD verification method> → <unit / integration / end-to-end / inspection> |

## 10. Quality requirements

<Quality scenarios beyond section 1, each measurable.>

### Testing strategy

<Test layers (unit, integration, end-to-end, inspection), what each covers, what runs in the repository checks.>

## 11. Risks and technical debt

| id | risk or debt | requirement and decision ids | mitigation |
|---|---|---|---|
| <RK1 / TD1> | <description> | <R.., D..> | <action> |

## 12. Glossary

| term | meaning |
|---|---|
| <term> | <meaning> |

## 13. Open questions

1. <question> (<owner or role that answers it>)
```
