# DESIGN template

Scaffold for the repository `DESIGN.md`, the global styles of the product: principles, design tokens, the shared component inventory, accessibility and message rules. It is optional and written only when the product has a UI or a user-facing surface worth styling. UX and app flow are per change: they live in section 3 of each intent's `spec.md`, which references this file instead of repeating it. A product without a GUI marks sections 2 and 3 "not applicable" with the reason and describes its conversation or CLI conventions under section 1. Answer every fixed question before filling its section; an answer you do not have goes to section 6 as an open question, never an invented decision. Keep the table columns exactly as written.

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | Who uses the product, on which devices or terminals, and which principles guide the experience? | 1. Principles |
| Q2 | Which color, spacing, typography, radius and motion tokens exist? | 2. Design tokens |
| Q3 | Which shared atoms, molecules, organisms, templates and pages exist, with which states? | 3. Component inventory |
| Q4 | Which intents use each component? | 3. Component inventory: used by |
| Q5 | How does the keyboard path, focus order, contrast and labeling work? | 4. Accessibility |
| Q6 | How does each error, refusal or empty case tell what happened, why and what to do next? | 5. Messages and errors |
| Q7 | What is still undecided? | 6. Open questions |

## Template

```markdown
# Design

| field | value |
|---|---|
| status | Draft |
| owner | gabo-frontend |
| date | <YYYY-MM-DD> |
| linked docs | [PRD](docs/product/PRD.md), [TRD](docs/product/TRD.md) |

## 1. Principles

- <Principle and the PRD goal it serves.>

## 2. Design tokens

Components use tokens only; this file is the source of truth.

| token | custom property | value | use |
|---|---|---|---|
| <color.primary> | `--color-primary` | <value> | <where it is used> |

## 3. Component inventory

Atomic design: atom, molecule, organism, template, page. Each component gets a Storybook story per state.

| component | level | states | used by |
|---|---|---|---|
| <Button> | <atom> | default, hover, focus, disabled, loading, error, empty | <NNNN-slug> |

## 4. Accessibility

<Keyboard paths, focus order, contrast targets, labels and announcements.>

## 5. Messages and errors

One line first, details after.

| case | what happened | why | what to do next |
|---|---|---|---|
| <case> | <message> | <cause> | <next action for the user> |

## 6. Open questions

1. <question> (<owner or role that answers it>)
```
