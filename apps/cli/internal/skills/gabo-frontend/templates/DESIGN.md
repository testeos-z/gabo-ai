# DESIGN template

Scaffold for the repository `DESIGN.md`: the design norms every component, template and screen follows. Tokens are the source of truth for values; this document states the rules that use them. Answer every fixed question before filling its section; an answer you do not have goes to section 15 as an open question, never an invented rule. Keep the table columns exactly as written.

Every rule is one row of a rules table, written this way:

- **id**: stable and never reused; a prefix per section (for example `T1` for tokens, `A3` for accessibility).
- **rule**: one idea, with a normative keyword as in RFC 2119: MUST, SHOULD or MAY.
- **level**: the keyword again (MUST, SHOULD or MAY), so the table can be filtered.
- **rationale**: one line on why the rule exists.
- **verification**: how it is checked; either automated (the test or lint rule that fails) or manual (the exact review step).
- **examples**: a do and a don't (a Storybook story id, an image or a short snippet).
- **source**: where the rule comes from (a WCAG success criterion, a PRD requirement, an owner decision with its date).

A rule that cannot state its verification is a SHOULD at most. Exceptions go to section 13 with a reason, an owner and a date, never into code comments alone. Every change to a rule is a row in section 14.

## Fixed questions

| id | question | answered in |
|---|---|---|
| Q1 | Which brand personality and principles guide the visual design, and which standards and references are followed (for example WCAG 2.2 AA, the DTCG token format)? | 1. Principles and sources |
| Q2 | Where do the tokens live, in which tiers, and which file is the source of truth? | 2. Tokens |
| Q3 | Which themes exist (light, dark, high contrast), which is designed first, and which color roles carry meaning? | 3. Color and themes |
| Q4 | Which typefaces and type scale are used, and which size and weight maps to each heading level and body text? | 4. Typography |
| Q5 | Which devices and breakpoints are supported, which page widths and grids apply, and how does each template adapt on small screens? | 5. Layout and responsive behavior |
| Q6 | Which radii, spacing rhythm, control sizes and density apply? | 6. Shape and density |
| Q7 | Which durations, easing and animated properties are allowed, and how do states change (loading, empty, error)? | 7. Motion and state |
| Q8 | Which icon set, stroke and sizes per context are used? | 8. Icons |
| Q9 | How do keyboard navigation, focus order, focus visibility, target size, contrast and screen-reader labels work? | 9. Accessibility |
| Q10 | Which voice and tone apply, and how are buttons, titles, helper text, empty states, errors and long text written? | 10. Content and writing |
| Q11 | How do forms label fields, validate, show errors and confirm success? | 11. Forms |
| Q12 | Which states must every component document, and how are its stories and examples kept? | 12. Components |
| Q13 | Who approves an exception or a rule change, and where is it recorded? | 13. Exceptions, 14. Changelog |
| Q14 | What is still undecided? | 15. Open questions |

## Template

```markdown
# Design norms

| field | value |
|---|---|
| status | Draft |
| owner | gabo-frontend |
| date | <YYYY-MM-DD> |
| linked docs | [UX_UI](docs/product/UX_UI.md), [APP_FLOW](docs/product/APP_FLOW.md), [PRD](docs/product/PRD.md) |

## 1. Principles and sources

- <Principle and the PRD goal or requirement it serves.>
- Standards followed: <WCAG 2.2 AA, DTCG format version, other references>.

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| P1 | <rule with MUST, SHOULD or MAY> | <MUST> | <why> | <automated: test or lint name / manual: review step> | <do / don't> | <WCAG SC, PRD R, owner decision and date> |

## 2. Tokens

<Token tiers, source files and how components consume them.>

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| T1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <source> |

## 3. Color and themes

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| C1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <source> |

## 4. Typography

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| Y1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <source> |

## 5. Layout and responsive behavior

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| L1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <source> |

## 6. Shape and density

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| S1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <source> |

## 7. Motion and state

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| M1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <source> |

## 8. Icons

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| I1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <source> |

## 9. Accessibility

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| A1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <WCAG 2.2 SC> |

## 10. Content and writing

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| W1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <source> |

## 11. Forms

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| F1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <source> |

## 12. Components

| id | rule | level | rationale | verification | examples | source |
|---|---|---|---|---|---|---|
| K1 | <rule> | <level> | <why> | <how checked> | <do / don't> | <source> |

## 13. Exceptions

| rule id | exception | reason | owner | date |
|---|---|---|---|---|
| <T1> | <where the rule does not apply> | <why> | <who approved it> | <YYYY-MM-DD> |

## 14. Changelog

| date | rule ids | change | decision |
|---|---|---|---|
| <YYYY-MM-DD> | <T1, A3> | <added, changed or retired, and how> | <owner decision or plan change> |

## 15. Open questions

1. <question> (<owner or role that answers it>)
```
