# Design norms

| field | value |
|---|---|
| status | Accepted |
| owner | gabo-frontend |
| date | 2026-11-07 |
| linked docs | [PRD](PRD.md), [TRD](TRD.md) |

## 1. Principles and sources

- Clarity first, it serves PRD R1.
- Standards followed: WCAG 2.2 AA, DTCG 2025.10.

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| P1 | Every screen MUST have one primary action. | MUST | Focus | manual: review the screen | one button / two | PRD R1 |

## 2. Tokens

Not applicable: this product is a CLI with no GUI.

## 3. Color and themes

Not applicable: this product is a CLI with no GUI.

## 4. Typography

Not applicable: this product is a CLI with no GUI.

## 5. Layout and responsive behavior

Not applicable: this product is a CLI with no GUI.

## 6. Shape and density

Not applicable: this product is a CLI with no GUI.

## 7. Motion and state

Not applicable: this product is a CLI with no GUI.

## 8. Icons

Not applicable: this product is a CLI with no GUI.

## 9. Accessibility

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| A1 | Focus MUST stay visible on every control. | MUST | Keyboard users | automated: axe focus-visible | ring / none | WCAG 2.4.7 |

## 10. Content and writing

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| W1 | Buttons SHOULD start with a verb. | SHOULD | Clarity | manual: copy review | Save / OK | owner 2026-11-07 |

## 11. Forms

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| F1 | Every field MUST have a visible label. | MUST | Accessibility | automated: axe label | label / placeholder | WCAG 3.3.2 |

## 12. Components

Not applicable: this product is a CLI with no GUI.

## 13. Exceptions

| rule id | exception | reason | owner | date |
|---|---|---|---|---|

## 14. Changelog

| date | rule ids | change | decision |
|---|---|---|---|
| 2026-11-07 | P1, T1 | added | owner decision 2026-11-07 |

## 15. Open questions

1. Which icon set for marketing pages? (gabo-frontend)
