# Current Code Map

Re-read these sources before relying on a calculation or write contract:

| Concern | Canonical source |
| --- | --- |
| Income fields and household formula | `backend-fees/internal/domain/income_calculation.go` |
| Fee limits, tables, care hours, and sibling discounts | `backend-fees/internal/domain/childcare_fee.go` |
| Child age determination | `backend-fees/internal/domain/child.go` |
| Einstufung and follow-up business behavior | `backend-fees/internal/service/einstufung_service.go` |
| Fee expectation synchronization | `backend-fees/internal/service/fee_service.go` |
| HTTP request and response mapping | `backend-fees/internal/api/handler/einstufung_handler.go` |
| API routes | `backend-fees/internal/api/router.go` |
| Published API contract | `openapi/fees/openapi3.yaml` |
| Frontend request mapping | `frontend/apps/beitraege/src/api/client.ts` and `src/pages/EinstufungDetailPage.vue` |
| Period storage and overlap rules | `backend-fees/migrations/000026_einstufung_periods.up.sql` |

## Current invariants to verify

- `IncomeDetails` stores annual positive EUR amounts for each parent.
- Employee income adds gross and other income, then subtracts employee social security, private insurance, tax, and advertising costs.
- Self-employed income and modeled benefits contribute according to the current domain functions; maintenance paid is deducted and maintenance received is added.
- Household annual income is the rounded sum of both parents' fee-relevant income.
- Fee selection depends on effective-month child age, household income, child count, care hours, voluntary highest-rate selection, and foster-family status.
- A follow-up uses `POST /api/fees/v1/einstufungen/{sourceId}/follow-ups`.
- The current cut-off rule maps change days 1–14 to the same month's first day and day 15 onward to the following month's first day.
- Follow-up creation closes the source period, creates the successor, updates household fee metadata, and synchronizes `CHILDCARE` expectations. `FOOD` and `MEMBERSHIP` expectations are not part of that synchronization.

Treat these as navigation aids, not frozen duplicates. If any invariant changed, use the current code and update this reference.
