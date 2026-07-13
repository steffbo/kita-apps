---
name: kita-fees-einstufung
description: Analyze, calculate, explain, create, or verify fee classifications (Einstufungen) in the kita-apps fees domain. Use when Codex must evaluate household income evidence, project annual amounts, identify missing figures, determine the applicable childcare contribution, prepare an Einstufung summary or parent message, or create a follow-up Einstufung after an income, care-hours, or household change.
---

# Kita Fees Einstufung

Follow the current application behavior instead of treating a previous case, spreadsheet, or stored Einstufung as the rules source.

## Establish the task boundary

1. Distinguish analysis, drafting, and live mutation.
2. Treat a request to calculate or explain as read-only.
3. Create or change a live Einstufung only when the user explicitly asks for that mutation.
4. Keep every case-specific amount, person, date, assumption, and missing document out of this skill.

## Load only the needed references

- Read [references/code-map.md](references/code-map.md) before calculating or mapping API fields.
- Read [references/case-workflow.md](references/case-workflow.md) when evaluating evidence, projecting a year, presenting a result, or creating a follow-up.
- Use `$kita-live-data-access` for live API authentication, database reads, production writes, and post-write verification.

## Execute the workflow

1. Identify the source Einstufung, calculation year, effective date, child, household, care hours, child count, age category, and special income status.
2. Gather the applicable contribution policy and all available evidence for both parents. Do not infer that an old year's income or benefit continues into the calculation year.
3. Re-read the current domain and service code listed in `code-map.md`. If the code and this skill differ, follow the code and update this skill in the same development increment.
4. Normalize evidence into annual positive EUR amounts for the current `IncomeDetails` fields. Preserve zero as “no amount entered”; describe unknown values as missing instead of silently using zero.
5. Project incomplete-year payroll only from explicit or clearly stated assumptions. Keep year-to-date figures separate from projected remaining months and avoid double-counting.
6. Calculate the household income with the current application code or calculation API. Do not reproduce a stale fee table or benefit exemption from memory.
7. Determine the fee using the current application logic and the effective-month attributes, including age, care hours, child count, voluntary highest rate, and foster-family status.
8. Present an auditable table with source amount, projection or transformation, annual value, and assumption. State the household total, relevant bracket boundary, margin to that boundary, resulting monthly fee, missing evidence, and confidence level.
9. When requested, prepare notes or correspondence that clearly separate confirmed facts from assumptions and requested confirmations.
10. When explicitly requested, create a follow-up through the current follow-up API. Never create a normal Einstufung as a substitute for a follow-up and never write the database directly.
11. Verify the source period, new period, calculated amounts, and fee-expectation side effects after a live mutation.

## Preserve auditability

- Retain the original evidence terminology where it matters, such as total gross, taxable gross, employee social-security share, net earnings, and payout.
- Explain how vouchers, one-off payments, benefits, maintenance, insurance, and other income were mapped or why they remain unresolved.
- Do not mix calendar years merely to reuse a stored value.
- Copy unchanged parent inputs from the source only when the user accepts that assumption; label them as provisional when current evidence is missing.
- Put material assumptions and missing evidence in the Einstufung notes when creating a provisional live record.
