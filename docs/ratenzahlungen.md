# Ratenzahlungen (installment plans)

Feature spec and handoff contract. Visual reference: `docs/design/ratenzahlung-mockup.html`
(clickable mock, accepted 2026-11-01). Status: backend and frontend implemented (2026-11).

## Scope

A Ratenzahlung is a contractual payment plan for a debt. The debtor is either an existing
household (`fees.households`) or an external person who is not in the system (e.g. a former
family; first real case: payment from „Eggert, Daniel“, 19.10.2026, 300 €, purpose
„Ratenzahlung nach 10 C 644/24“).

- Access: `ADMIN` only (API and UI). A future `VORSTAND` role is expected to get the same
  rights as `ADMIN`: backend `domain.InstallmentManagerRoles`, frontend `auth.isAdmin`.
- Other staff (`USER`) share the bank lists: `redactInstallments` removes suggestions and debtor names there
  (a plan-assigned transaction only shows the plain badge „Ratenzahlung“), and unmatching such a transaction
  returns 403.
- Not visible to `USER`, `LEITUNG`, `PARENT_WORK`, `PARENT`.

## Data model (migration `000049_installment_plans`)

`fees.installment_plans`
| column | type | notes |
| --- | --- | --- |
| id | UUID PK | |
| household_id | UUID NULL → households ON DELETE SET NULL | NULL = external debtor |
| debtor_name | TEXT NOT NULL | always set; for households the household name at save time |
| debtor_address | TEXT NOT NULL DEFAULT '' | optional, for letters |
| emails | TEXT[] NOT NULL DEFAULT '{}' | 0..n reminder recipients, validated, deduplicated |
| total_amount | NUMERIC(12,2) NOT NULL > 0 | agreed total (may already include interest) |
| installment_amount | NUMERIC(12,2) NOT NULL > 0 | regular rate |
| interval_months | INT NOT NULL, IN (1,2,3,6,12) | default 1 (monatlich) |
| first_due_date | DATE NOT NULL | |
| final_due_date | DATE NULL | Schlussrate, see schedule |
| reference | TEXT NOT NULL DEFAULT '' | Bezug / Aktenzeichen, used in mails |
| payer_iban | TEXT NULL | normalized (no spaces, upper case); bank suggestion key |
| note | TEXT NOT NULL DEFAULT '' | |
| status | TEXT NOT NULL DEFAULT 'ACTIVE' CHECK IN ('ACTIVE','CLOSED') | |
| close_reason | TEXT NULL | e.g. „Restschuld erlassen“, „An Inkasso/Gericht übergeben“, „Irrtümlich angelegt“ |
| closed_at | TIMESTAMPTZ NULL | |
| created_at, updated_at | TIMESTAMPTZ NOT NULL DEFAULT now() | |
| created_by | UUID NULL | |

`fees.installment_payments`
| column | type | notes |
| --- | --- | --- |
| id | UUID PK | |
| plan_id | UUID NOT NULL → installment_plans ON DELETE CASCADE | |
| transaction_id | UUID NULL UNIQUE → bank_transactions ON DELETE CASCADE | set for bank payments |
| amount | NUMERIC(12,2) NOT NULL > 0 | bank payments: the full transaction amount |
| paid_on | DATE NOT NULL | bank: booking date |
| source | TEXT NOT NULL CHECK IN ('BANK','MANUAL') | MANUAL = cash etc. |
| note | TEXT NOT NULL DEFAULT '' | bank: transaction description |
| created_at | TIMESTAMPTZ NOT NULL DEFAULT now(), created_by UUID NULL | |

`fees.installment_charges` — extra amounts added to the debt (currently only the Mahngebühr).
`id, plan_id (CASCADE), amount > 0, label TEXT, created_at, email_log_id NULL → email_logs SET NULL`.

`fees.installment_contracts` — signed contract PDF, one per plan, stored in the DB (part of the DB backup).
`plan_id PK (CASCADE), filename, size_bytes, content BYTEA, uploaded_at, uploaded_by`.

`fees.email_logs`: new nullable column `installment_plan_id → installment_plans ON DELETE SET NULL`,
new email types `INSTALLMENT_REMINDER_INITIAL`, `INSTALLMENT_REMINDER_FINAL`.

A bank transaction is assigned either to fees (`fees.payment_matches`) or to exactly one plan
(`fees.installment_payments`), never both; the service rejects mixed assignments.

## Schedule (pure function in `internal/domain`, unit-tested)

- Effective total = `total_amount` + sum(charges). Paid = sum(payments). Rest = max(0, effective total − paid).
- Regular rates: n = ceil(effective total / installment_amount); rate i (0-based) is due
  `first_due_date + i*interval_months` (month arithmetic clamps to the month end); all rates are
  `installment_amount` except the last, which is the remainder.
- Schlussrate (`final_due_date` set): rates fully covered by payments (FIFO, see below) stay as they
  are; all other rates are replaced by one final rate due at `final_due_date` for the remaining
  amount (incl. already overdue rates).
- Payments are applied FIFO to the oldest rate; surplus flows into the next rate.
- Rate status: `PAID` (fully covered), `OVERDUE` (due date < Berlin today and not fully covered),
  `PARTIAL` (partly covered, not yet due), `OPEN`.
- Plan state (derived, for list filters): `CLOSED` (status CLOSED), `DONE` (rest = 0),
  `OVERDUE` (any overdue rate), `ACTIVE`.
- Use `util.Today()` for date comparisons; amounts in cents internally where `domain/money.go` helpers exist.

## Bank reconciliation

- Unmatched transaction lists exclude transactions that have an installment payment; `DeleteUnmatchedByIBAN`
  must not delete them either.
- Unmatched transactions carry an optional suggestion `installmentPlanSuggestion {planId, debtorName}` when the
  payer IBAN equals the `payer_iban` of an ACTIVE plan with rest > 0. No automatic assignment.
- Matched transaction list includes plan-assigned transactions with `installmentPayment {paymentId, planId, debtorName}`;
  the existing unmatch action (`POST /import/transactions/{id}/unmatch`) also removes an installment payment.
- Creating a plan can assign a transaction as its first payment in the same DB transaction.

## API (`/api/fees/v1`, all `ADMIN`)

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/installment-plans?state=active\|overdue\|done\|all` | list with summary (`active` = ACTIVE+OVERDUE, `done` = DONE+CLOSED) |
| POST | `/installment-plans` | create; optional `transactionId` assigns it as first payment |
| GET | `/installment-plans/{id}` | detail: plan, summary, schedule, payments, charges, reminders (email logs), contract meta |
| PUT | `/installment-plans/{id}` | update master data / terms |
| POST | `/installment-plans/{id}/close` | `{reason}` |
| POST | `/installment-plans/{id}/payments` | `{transactionId}` or `{amount, paidOn, note}` (manual) |
| DELETE | `/installment-plans/{id}/payments/{paymentId}` | |
| PUT | `/installment-plans/{id}/final-installment` | `{dueDate}` sets the Schlussrate |
| DELETE | `/installment-plans/{id}/final-installment` | restores the regular schedule |
| PUT | `/installment-plans/{id}/contract` | multipart `file`, PDF only (`%PDF` magic), max 10 MB; replaces |
| GET | `/installment-plans/{id}/contract` | `application/pdf`, inline, original filename |
| DELETE | `/installment-plans/{id}/contract` | |
| GET | `/installment-plans/reminder-cases` | plans with overdue rates that are due for a reminder |
| POST | `/installment-plans/{id}/reminder/preview` | build mail |
| POST | `/installment-plans/{id}/reminder/send` | send + log (+ charge) |

Create/update body: `householdId?`, `debtorName` (required when no household; for households the server
uses the household name), `debtorAddress`, `emails[]`, `totalAmount`, `installmentAmount`, `intervalMonths`,
`firstDueDate`, `reference`, `payerIban`, `note`.

## Reminders

- `InstallmentDetail.reminderDue` carries this decision; the detail page only offers sending when it is true.
- A plan is a reminder case when it is ACTIVE, has at least one OVERDUE rate, and either no installment
  reminder was sent yet or the deadline of the last one has passed (deadline stored in the email log payload).
- Stage: first mail `initial` (Zahlungserinnerung), afterwards `final` (Mahnung). A Mahngebühr can be charged
  only on `final` and only when explicitly ticked; amount = the existing reminder fee amount used for fee
  reminders; it is stored as `installment_charges` row and increases the effective total.
- Request (preview/send): `{installmentNumbers[], chargeReminderFee, includeQR?, subject?, body?}`; preview
  returns `{recipients, subject, body, deadline, totalAmount, stage, qrImageDataUrl?, warnings[]}`.
- Without e-mail addresses the case is listed but sending is rejected (UI shows a hint and links to editing).
- Reuse the existing reminder infrastructure: email sender, payment settings (Zahlungsdaten) and QR code,
  default deadline, `email_logs`. Households are addressed with „du“ (like existing reminders), external
  debtors with „Sie“ and „Guten Tag <Name>“.
- The Erinnerungen page lists plan cases next to the fee cases (badge „Ratenzahlung“); the Versandverlauf shows
  the installment mails via the existing email log.

## UI (Beiträge app)

- Nav item „Ratenzahlungen“ in group „Beiträge“ (admin only), routes `/ratenzahlungen` and `/ratenzahlungen/:id`.
- List: stats (Restschuld gesamt, davon überfällig), filter chips (Aktiv, Rückstand, Abgeschlossen, Alle),
  rows with debtor, rate/interval, progress, next due date, rest, state badge, „Vertrag fehlt“ hint.
- Detail: overdue note with reminder action, Restschuld progress, Vereinbarung (+ Schlussrate set/clear),
  Vertrag (upload/view/replace/remove), Schuldner, E-Mails, Notiz, Ratenplan, Zahlungen (+ manual payment),
  Erinnerungsverlauf, „Ratenzahlung beenden …“.
- Form (AppSheet): debtor segment Haushalt | Andere Person (household search; emails prefilled from the
  household's parents), emails add/remove, total, rate, interval, first due date, live schedule preview,
  reference, payer IBAN, contract upload, note.
- Bankabgleich: manual-match dialog gets a „Beitrag | Ratenzahlung“ switch (admin only); plan list with
  suggestion, „Neue Ratenzahlung anlegen“ prefilled from the transaction (external debtor, name „First Last“
  from „Last, First“, rate = amount, first due = booking date, reference from purpose, IBAN).
- Haushalt-Zeile auf der Detailseite ist Text (keine Haushaltsseite vorhanden). Aus dem Entwurf noch nicht umgesetzt:
  PDF-Brief im Erinnerungsdialog (ohne E-Mail-Adresse gibt es daher keinen Versandweg aus der App) und
  Bestätigungsmail zur Schlussrate.
- Use existing building blocks (`AppSheet`, `PageHeader`, `EmptyState`, `.btn`, `.badge`, `.note`, `.chip`,
  `.card`, `SearchInput`, status tokens) — see `docs/design/ui-kit.md`.
