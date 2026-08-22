# Status: Beiträge (backend-fees + frontend/apps/beitraege)

Rolling change log of non-obvious implementation decisions. Newest first.
Basics (ports, commands, layout) live in `AGENTS.md`.

## Backend (`backend-fees`)

### Pagination collapsed to page/perPage (2026-08-22)

- `request.GetPagination` accepted two query styles (`offset/limit` took precedence over `page/perPage`). The `offset/limit` branch is removed; all paginated endpoints now take `page`/`perPage` only (defaults 1/20, perPage capped at 100). The response envelope (`data`, `total`, `page`, `perPage`, `totalPages`) is unchanged.
- Frontend client and all call sites switched from computed offsets to `page`/`perPage`. Wire-level behaviour is identical for every existing caller (they only ever sent page-aligned offsets).
- Known pre-existing quirk, unchanged on purpose: the 100-item cap silently limits "load everything" calls such as ImportPage's `perPage: 1000` merge or EinstufungDetailPage's `perPage: 2000` option loads. If those lists outgrow 100 rows, raise the cap or add real pagination there.

### OpenAPI spec aligned with actual responses (2026-08-22)

- Fee endpoints (`GET/POST /fees`, `GET/PUT /fees/{id}`, `POST /fees/{id}/reminder`) previously documented a fictional `Fee` DTO (`type`, `status`, `childName`); they actually serialize `domain.FeeExpectation` (`feeType`, `isPaid`, joined `child`, `matchedAmount`, …). The dead DTOs were deleted and the annotations now document the real structs; same for `/fees/overview` (now `domain.FeeOverview` incl. `childrenWithOpenFees`).
- Import endpoints now document what they return: `/import/transactions*` → `domain.BankTransaction` (incl. `payerName`/`payerIban`/`matchedAmount`; the old fictional `Transaction` schema claimed `accountHolder`/`status`), `/import/history` → `domain.ImportBatch`, `/import/blacklist` + `/import/trusted` → `domain.KnownIBAN`, `/import/warnings` → `domain.TransactionWarning`.
- `banking-sync/run|status|cancel` were missing from the spec entirely; annotated against a new `BankingSyncStatus` schema mirroring the banking-sync service state object (statuses incl. `cancelled`, which the frontend type did not know).
- Spec regenerated via `swag init` + `swagger2openapi` and frontend `schema.d.ts` regenerated in the same commit. Note: `swagger2openapi` is blocked on the Artifactory npm mirror — install it from `registry.npmjs.org` directly (`bun add swagger2openapi --registry https://registry.npmjs.org/`).

### Partial and sibling-split transaction allocation (2026-08-22)

- One bank transaction can now be allocated across multiple children (e.g. "Essensgeld - Henri 12011 . Niels 12012") and in multiple steps until its amount is fully allocated. Previously allocation was one-shot, single-child, and any match made the transaction vanish from all unmatched lists.
- `ListUnmatched` (`internal/repository/transaction_repository.go`) now filters on `SUM(payment_matches.amount) < bt.amount` instead of "has no match", so partially allocated transactions stay listed with a new joined `matched_amount` field (`domain.BankTransaction.MatchedAmount`). Fully allocated transactions drop out of the unmatched lists automatically.
- `ImportService.AllocateTransaction` (`internal/service/import_service.go`) no longer rejects transactions that already have matches; it validates `existing + new <= tx.Amount` and rejects duplicate (transaction, expectation) pairs. The single-child rule was removed — allocations may target fees of different children; the IBAN is only auto-linked to a child when all allocations of one call belong to a single child.
- The OVERPAYMENT warning on unallocated remainder was removed: the remainder keeps the transaction visible in the unmatched lists instead, which is the actionable signal now.
- Frontend: `ChildDetailPage` caps allocation inputs at the transaction's remaining amount, shows "Bereits zugeordnet / Rest" for partial transactions, and explains that a remainder stays open; `ImportPage` shows an orange "Teilweise zugeordnet · Rest X" badge.
- OpenAPI `domain.BankTransaction` gained optional `matchedAmount`. No migration needed — `payment_matches.amount` (migration 000017) already carried per-match amounts.

### Monthly fees respect enrollment periods (2026-08-02)

- `FeeService.Generate` selects monthly fee candidates independently of the current `is_active` flag and checks whether each child's enrollment overlaps the billed month.
- Children whose `exit_date` is before the first day of the month, or whose `entry_date` is on or after the first day of the following month, receive neither `FOOD` nor `CHILDCARE` expectations. Partial entry or exit months remain billable.
- Yearly membership generation continues to use the active-child selection. Existing fee expectations are not changed automatically.
- Coverage: `internal/service/fee_service_integration_test.go` includes children leaving before, entering after, and overlapping the billed month, including historical generation for an inactive child.

### Child hours as single source of truth (2026-07-27)

- Migration `migrations/000028_drop_child_hours_columns.up.sql` backfills column-only values into the history tables and drops `fees.children.care_hours`, `legal_hours`, `legal_hours_until`. `fees.child_care_hours_history` and `fees.child_legal_hours_history` are now the only source; the down migration restores the columns from the currently effective periods.
- `domain.Child.CareHours`, `LegalHours`, `LegalHoursUntil` are derived. Reads in `child_repository.go` / `household_repository.go` join `childCurrentHoursJoins` and select `childSelectColumns`, resolving the period covering `CURRENT_DATE`. API/JSON contract unchanged, so no OpenAPI/frontend regeneration was needed.
- Writes go to history only: `Create` records hours from the child's entry date, `Update` compares against today's effective values and writes a new period from today, `UpsertCareHoursHistory` / `UpsertLegalHoursHistory` write explicit periods. The old `syncCurrentCareHoursTx` / `syncCurrentLegalHoursTx` cache updates were removed.
- The `hasWarnings` child filter checks whether *any* hours period exists, not whether one is effective today, so children starting in the future are no longer flagged as missing master data.
- `repository.truncateDate` normalizes history dates to UTC midnight (previously `time.Now()` in UTC+X could store a period one day early).
- Stichtagsmeldung breakdowns (`loadHoursBreakdown`) read history only; children without a period covering the Stichtag count as unknown.
- Coverage: `internal/service/child_hours_history_integration_test.go`; migration verified up/down/up against a scratch PostgreSQL 16 DB.

### Care-hours resolution for Platzgeld (2026-07-27)

- `FeeService.Generate` and `FeeService.calculateChildcareFeeForChild` both call `FeeService.ResolveCareHours`, which reads `fees.child_care_hours_history`, picks the period effective in the billed month, otherwise the closest upcoming period, and only as last resort the `DefaultCareHours` fallback of 45.
- Pure helper `careHoursAt` is unit-tested in `internal/service/care_hours_resolution_test.go`.

### Household consistency for child-parent links (2026-06-10)

- Migration `migrations/000027_validate_child_parent_households.up.sql` adds deferrable constraint triggers rejecting `fees.child_parents` links and `household_id` updates on `fees.children` / `fees.parents` when linked children and parents would end up in different non-null households. The same SQL was applied manually to production before the code deploy, so re-application must stay idempotent.
- `internal/service/child_service.go`: if the child has no household but the parent does, the child joins the parent household; if both have different households, the service returns `ErrHouseholdMismatch` → HTTP 409. Imports linking an existing parent from another household now fail that row instead of splitting data.
- Production data on `infra-dev` was cleaned for the split Thränhardt/Tränhardt and Zeck households; the global mismatch check returned `0` rows afterwards.

Audit query:

```sql
SELECT c.id, c.first_name, c.last_name,
       c.household_id AS child_household_id,
       p.id AS parent_id, p.first_name, p.last_name,
       p.household_id AS parent_household_id
FROM fees.child_parents cp
JOIN fees.children c ON c.id = cp.child_id
JOIN fees.parents p ON p.id = cp.parent_id
WHERE c.household_id IS NOT NULL
  AND p.household_id IS NOT NULL
  AND c.household_id <> p.household_id;
```

### Einstufung history and follow-ups (2026-06-06)

- Migration `migrations/000026_einstufung_periods.up.sql` adds `valid_until`, `source_einstufung_id`, `change_date`, `effective_from_month`, drops the one-Einstufung-per-child/year uniqueness, and enforces non-overlapping child periods with a GiST exclusion constraint.
- Endpoint `POST /api/fees/v1/einstufungen/{id}/follow-ups` stores the entered `changeDate` and derives `effectiveFromMonth`: day 1–14 = same month, day 15+ = following month.
- Creating a follow-up closes the source Einstufung the day before `effectiveFromMonth`, creates the new one with `annual_membership_fee = 0`, and syncs `CHILDCARE` fee expectations from the effective month through year-end or child exit date.
- CHILDCARE sync creates missing expectations, updates unmatched ones, increases already matched ones when no overpayment results, and reports decreases below matched amounts in `creditReviewRequired` without auto-creating credits. `FOOD` and `MEMBERSHIP` are untouched.
- Income calculation treats all entered income components as fee-relevant unless explicitly modeled as deductions. `Basiselterngeld`, `Elterngeld Plus`, `Mutterschaftsgeld` are fully fee-relevant. Income JSON breaks out `minijobIncome`, `unemploymentBenefit`, `capitalIncome`, `rentalIncome`; `otherIncome` is residual. Older stored JSON without these fields reads as zero.
- Coverage: `go test ./...` includes cut-off tests for `2026-06-14` / `2026-06-15`, sync tests for paid increases and credit-review decreases, plus household consistency tests (service + DB trigger).

## Frontend (`frontend/apps/beitraege`)

### API types derived from the generated schema; race guards on list pages (2026-08-22)

- `src/api/types.ts` no longer hand-maintains ~100 response interfaces: response shapes now derive from `src/api/schema.d.ts` via a recursive `DeepStrict` helper, so field names/types can no longer drift from `openapi/fees/openapi3.yaml`. Fields that are genuinely sparse (`omitempty`, joined values) are re-loosened explicitly; request payloads stay hand-written because the backend validates them manually. Exception: the child-import preview/row family stays plain-hand-written — vue-tsc resolves `Omit(DeepStrict)+intersection` compositions differently inside SFCs and wrongly required re-added optional fields.
- List pages (Kinder, Eltern, Mitglieder, Beiträge, Einstufungen) previously fired two or three concurrent loads per interaction and had no stale-response protection. Each interaction now triggers exactly one load (explicit handlers for search/filter, conditional first-page reset), every loader carries a monotonic sequence guard against out-of-order responses, and an empty current page after deletes snaps back to the last valid page.
- `ImportPage`'s rescan result uses the honest `RescanResult` type now — rescan suggestions are string-info DTOs, not domain suggestions.
- Removed unused dependencies from the app: `@tanstack/vue-query` (plugin was registered but zero queries existed), `radix-vue`, `@vueuse/core`. Note: `radix-vue` remains a real dependency of `dienstplan`; the other three apps also register `VueQueryPlugin` without using it — same cleanup is likely safe there, not yet done.
- `scripts/generate-api.sh` was rewritten: it previously referenced a Spring Boot/Maven flow from another repository. It now runs the real fees pipeline end-to-end (swag init → swagger2openapi → openapi-typescript) and documents that swagger2openapi must be fetched from registry.npmjs.org because the Artifactory mirror blocks it.
- ESLint is not installed in this workspace (Artifactory blocks the fetch); verification gate here is `bun run build` (vue-tsc + vite). The Playwright e2e suite for beiträge requires the seeded `kita-db-e2e` environment described in `docs/test-seeding.md`.

### Reminder preview editing + email log filters (2026-08-22)

- `POST /fees/reminders/run` and `/fees/membership-reminders/run` accept an optional JSON body `{ includeQR?: boolean, overrides?: { [householdId]: { subject?, body? } } }`. Query-param behaviour unchanged when no body is sent. Blank override fields keep the generated text; overrides apply to both preview and send (`applyReminderOverrides`, unit-tested in `reminder_run_options_test.go`). `includeQR: false` skips QR generation and attachment entirely.
- Dry-run previews now include `qrPayload` — the SEPA payload string encoded in the QR code.
- Erinnerungen page preview modal: "Alle/Keine" selection shortcuts, per-household editable Betreff/Text (edits marked "bearbeitet" and sent as overrides only for selected households), QR toggle that hides previews' codes and passes `includeQR` on send, payload display under each QR code.
- Erinnerungen page: action buttons follow the app design system now — Erinnerung = primary (green), Mahnung = amber escalation, red reserved for destructive actions; buttons right-aligned in the form row, "(Auswahl)" suffix dropped (selection happens in the preview modal anyway).
- `GET /fees/email-logs` gained `emailType` (enum filter), `search` (ILIKE on recipient + subject) and `sortDir=asc|desc` (by `sent_at`); repository takes a `EmailLogFilter`. UI has type dropdown, search box, newest/oldest toggle and page/perPage pagination (20/page) replacing load-more.

### Bankabgleich UI overhaul (2026-08-22)

- The former `/import` page is now **Bankabgleich** (`/bankabgleich`, `/import` kept as route alias). Banking-Sync status + "Jetzt synchronisieren" live at the top of this page via the extracted `src/components/BankingSyncCard.vue`; after a run transitions running/2FA → success the card emits `sync-finished` and the transaction list reloads. No more hopping to the automation page to check sync results.
- One unified transaction list replaces the six tabs: all transactions newest-first with per-row status badges (Zugeordnet green / Nicht zugeordnet amber / n Warnungen orange, expandable in-row for warning details and actions). Filter chips **Offen** (default) / **Warnungen** / **Zugeordnet** / **Alle** act as filters, not tabs.
- Merge is client-side: `getMatchedTransactions({limit:1000})`, `getUnmatchedTransactions({limit:500})`, `getWarnings(0,200)` are combined by transaction id (warning rows win). Fine at current data volume (~hundreds of rows); if it grows by an order of magnitude, add a backend endpoint returning one paginated stream with status included instead.
- Search, sorting (date/payer/description/amount) and pagination (50/page) are client-side over the loaded sets. Legacy `?tab=…` query params still work: they map to filters (`unmatched`→Offen etc.) or open dialogs (`upload`, `history`, `blacklist`).
- CSV upload became a toolbar button opening a dialog (same dropzone + suggestion flow as before). Import-Historie and Blacklist are dialogs reachable from small links under the filter chips; Blacklist actions unchanged ("Ignorieren" on a transaction still adds the IBAN).
- `AutomationPage.vue` keeps only reminders (Essens-/Platzgeld + Vereinsbeiträge) and E-Mail-Protokoll; retitled "Erinnerungen", sidebar entry renamed accordingly (route stays `/automatisierung`). Sidebar is grouped Täglich / Verwaltung / Beiträge.
- Dashboard gained a Bank-Import status card (last run badge + timestamp, deep link to `/bankabgleich`); the unmatched-transactions card links there too.

### Einstufungs-PDF (2026-07-27)

- `src/components/EinstufungPDF.vue` + `EinstufungPDF.css` render a monochrome serif Geschäftsbrief (Briefkopf with logo, Datumszeile, Betreff, Anrede, Angaben zum Kind, Beitragstabelle, Hinweiskasten, zweispaltiges Kleingedrucktes, Grußformel, Fußzeile).
- Logo: `src/assets/knirpsenstadt-logo.png` (480 px, grayscale, ~35 KB), inlined as data URL before printing because the print popup is a blank document; absolute URL is the fallback.
- One-page geometry: `@page` margin `13mm 15mm 10mm`, `.page` width `180mm`, base font `9pt`. Only ~10 mm slack — re-verify one-page fit after any spacing/font change, including follow-ups with four fee columns.
- The fee table is period-based (`Aug. 2026 – Feb. 2027`, `ab März 2027`, follow-up previous period marked `(bisher)`) with Bereich, Std./Woche, Platzgeld, Essensgeld. No monthly total column; the yearly Vereinsbeitrag is a note line below the table (due with first payment, or already paid for follow-ups).
- Period boundaries come from `effectiveFromMonth`/`validFrom`, optional `validUntil`, and the Krippe→Kindergarten transition month (first full month after the third birthday).

### Care hours in the UI (2026-07-27)

- `src/pages/ChildDetailPage.vue` resolves effective care hours from `child.careHours` and, when empty because the child starts later, from the closest upcoming `care-hours-history` entry. Without care hours it shows `Betreuungszeit nicht hinterlegt` instead of calculating; when a fee is shown, the basis (`Basis: 35 Std./Woche (ab 1.8.2026)`) is displayed.
- The Betreuungszeiten block shows `ab <Datum>: <Stunden>` instead of `Unbekannt` when only future periods exist. `careHours`/`legalHours` from the API always describe the period effective today.

### Follow-up navigation (2026-07-12)

- `EinstufungDetailPage.vue` reloads and resets record state on every route change, so navigating from an existing Einstufung to `/einstufungen/neu?sourceId=...` loads the source record and submits via `POST /einstufungen/{id}/follow-ups` instead of keeping stale edit state.

### Mobile layout guardrails (2026-06-26)

- `src/layouts/MainLayout.vue` prevents root horizontal overflow, uses tighter mobile padding, and opens the nav drawer from the right.
- `src/assets/main.css` constrains app-wide horizontal overflow and adds touch-friendly internal scrolling for wide tables.
- `src/pages/ImportPage.vue` wraps import-history, unmatched, blacklist and matched tables in horizontal scroll containers; the tab bar scrolls internally. If another page shifts the viewport sideways, look for raw `<table>` markup or unbounded tab/filter rows missing an `overflow-x-auto` container.
