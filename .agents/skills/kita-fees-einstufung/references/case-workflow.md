# Case Workflow

## Evidence inventory

Collect only what the case needs:

- source Einstufung and stored per-parent input fields
- current calculation-year income evidence for both parents
- payroll year-to-date totals and current monthly figures
- benefits, other income, self-employment, maintenance, and modeled deductions
- applicable contribution policy or order
- child age at the effective month, care hours, child count, entry/exit dates, and special status

Mark each value as confirmed, derived, assumed, unchanged from source, or missing.

## Incomplete-year payroll

1. Prefer payroll year-to-date totals for elapsed months.
2. Project only the remaining months using a stated monthly assumption.
3. Handle one-off payments and recurring benefits separately.
4. Distinguish payroll net earnings from payout adjustments; map components to the application fields rather than using the payout as household income by default.
5. Keep the calculation within one calendar year.
6. Show the calculation so a reviewer can replace any assumption without rebuilding the whole case.

## Result table

Use a compact structure such as:

| Component | Parent 1 | Parent 2 | Basis or assumption |
| --- | ---: | ---: | --- |
| Annual gross and other positive income |  |  |  |
| Social security, insurance, tax, and other deductions |  |  |  |
| Benefits and maintenance |  |  |  |
| Fee-relevant annual income |  |  |  |

Then state:

- household annual income
- current bracket boundary and distance to it
- calculated monthly childcare fee
- effective month
- missing evidence and sensitivity to each missing value
- whether the result is final or provisional

## Follow-up creation checklist

- Confirm explicit authorization to create the live record.
- Load the source immediately before writing and ensure no newer follow-up already exists.
- Use the actual change date and let the backend calculate the effective month.
- Preserve unchanged case attributes only when supported by evidence or an explicit assumption.
- Put material assumptions and missing documents in notes.
- Review the computed annual income and monthly fee before sending.
- Create through the follow-up endpoint.
- Verify the source `valid_until`, successor linkage and period, stored calculation, household aggregate, and synchronized `CHILDCARE` expectations.
- Check credit-review output before describing a fee decrease as fully settled.
