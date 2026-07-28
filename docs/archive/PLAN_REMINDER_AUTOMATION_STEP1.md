# Payment Reminder Automation - Step 1 Plan

## Goal
Replace the admin-facing reminder email with a family-based email flow: one email per family, sent directly to parents, for `initial` reminder runs.

## Current State
- `Run()` in `reminder_service.go` sends a single email to the logged-in admin listing all unpaid fees.
- There is no cron — `auto` mode means the backend picks the stage based on day-of-month (5th → `initial`, 10th → `final`). The trigger is always manual (button in UI) or a future external cron (not yet built).
- `householdRepo.GetParents(ctx, householdID)` already exists and returns `[]Parent` with `Email *string`.
- Fees are per-child (`FeeExpectation.ChildID`), children have `HouseholdID`.
- All children are expected to belong to a household. If a household has no parents with emails, it's a skip + warning.

## Confirmed Decisions
- **Recipients**: Parent emails from the household (all valid emails in `To`). No admin copy — admins see sent emails via email log in the UI.
- **Trigger**: `sentBy` stays as `userCtx.UserID` for manual runs. For future automated runs, use a sentinel value (e.g. `nil` → "Automatisch" in the UI).
- **Tone**: Informal Du/ihr/euch. Address parents by first name ("Hallo Anna und Thomas,").
- **Email subject**: "Kita Zahlungserinnerung April 2026" (initial) / "Kita Mahnung April 2026" (final).
- **Email body**: List of open fee line items for that family's children, followed by bank details block.
- **Bank details in body**:
  ```
  Knirpsenstadt e.V.
  IBAN: DE33 3702 0500 0003 3214 00
  BIC: BFSWDE33XXX
  ```
- **No household → error**: All children must belong to a household. If somehow not, skip + log server error.
- **No parent emails → skip + warning**: Show as card on dashboard (out of scope for step 1, but the warning data is emitted).
- **Dry-run**: Returns full preview data (per-family: recipients, subject, body). Frontend shows this as a modal.
- **Email log `to_email`**: Store comma-separated list of recipients.
- **`final` stage**: Untouched. If improvements happen naturally during refactoring, fine, but no intentional scope creep.
- **Scale**: ~5-10 families per run, no batching/rate-limiting needed.

## Scope (Step 1)

### In Scope
- Family-based recipient grouping for `initial` reminder runs.
- One outbound email per family, all valid parent emails in `To`.
- New parent-facing email template (informal, with fee list + bank details).
- Warning tracking for families with no valid emails.
- Extended `ReminderRunResponse` with family-level stats and warnings.
- Dry-run returns per-family preview (recipients, subject, body) for modal display.
- Email log stores comma-separated recipients.

### Out of Scope
- Functional changes to `final` dunning behavior.
- Changes to reminder fee creation logic (still only on `final`).
- Dashboard warning card for families without parent emails (data is emitted, card is later).
- Scheduler/cron rework.

## Backend Changes

### 1) Family-based reminder grouping
- **File**: `backend-fees/internal/service/reminder_service.go`
- **New dependency**: `householdRepo repository.HouseholdRepository` (already exists, has `GetParents`)
- **Process**:
    1. Load unpaid fees via `ListUnpaidByMonthAndTypes` (unchanged).
    2. Load children via `childRepo.GetByIDs` to get `HouseholdID` (currently used for names only, extend to read household).
    3. Group fees by `HouseholdID`.
    4. For each household: call `householdRepo.GetParents(ctx, householdID)`.
    5. Collect valid emails (non-nil, non-empty), deduplicate.
    6. If no valid emails → skip family, add to `warnings[]`.
    7. Build parent-facing email per family.
    8. Send (or preview if dry-run).

### 2) Multi-recipient SMTP send
- **File**: `backend-fees/internal/email/email.go`
- Add `SendTextEmailMulti(to []string, subject, body string) error`.
- `SendTextEmail` delegates to `SendTextEmailMulti` with single-element slice.
- Sets `To` header to comma-joined addresses, sends via SMTP `rcptTo` for each.

### 3) New email template
- **File**: `backend-fees/internal/service/reminder_service.go`
- New function: `buildFamilyReminderEmail(stage, runDate, parentFirstNames []string, items []reminderItem) (subject, body string)`
- **Subject**: "Kita Zahlungserinnerung {Month} {Year}" or "Kita Mahnung {Month} {Year}"
- **Body structure**:
  ```
  Hallo {Anna und Thomas},

  für eure Familie sind noch folgende Beiträge offen:

  - {ChildName}: {FeeType} {Month}/{Year} — {Amount} €
  - {ChildName}: {FeeType} {Month}/{Year} — {Amount} €

  Bitte überweist den Gesamtbetrag von {Total} € auf folgendes Konto:

  Knirpsenstadt e.V.
  IBAN: DE33 3702 0500 0003 3214 00
  BIC: BFSWDE33XXX

  Vielen Dank!
  ```

### 4) Updated run result & API response
- **Files**:
    - `backend-fees/internal/service/reminder_service.go` — new `ReminderRunResult` fields
    - `backend-fees/internal/api/handler/fee_handler.go` — serialize new fields
    - `frontend/apps/beitraege/src/api/types.ts` — update `ReminderRunResponse`
- **New response shape** (additive, no breaking changes):
  ```json
  {
    "stage": "initial",
    "date": "2026-04-05",
    "dryRun": true,
    "unpaidCount": 12,
    "familiesProcessed": 6,
    "familiesEmailed": 5,
    "familiesSkippedNoEmail": 1,
    "warnings": [
      { "householdName": "Müller", "reason": "keine gültige E-Mail-Adresse" }
    ],
    "previews": [
      {
        "householdName": "Schmidt",
        "recipients": ["anna@example.com", "thomas@example.com"],
        "subject": "Kita Zahlungserinnerung April 2026",
        "body": "Hallo Anna und Thomas, ..."
      }
    ]
  }
  ```
- `previews[]` is only populated when `dryRun == true`.
- Old fields (`recipient`, `emailSent`, `reminderCreated`) stay for backward compat during transition but become less relevant.

### 5) Handler changes
- **File**: `backend-fees/internal/api/handler/fee_handler.go`
- `RunReminders` handler: remove `recipient = userCtx.Email` as the send target. Pass `sentBy = userCtx.UserID` for logging only.
- The `Run()` signature changes — it no longer needs `recipient string` since recipients come from household data.

### 6) Email log update
- **File**: `backend-fees/internal/service/reminder_service.go` (where log entries are created)
- `to_email` stores comma-separated list: `"anna@example.com, thomas@example.com"`.
- No schema migration needed — field is already `TEXT`.

## Frontend Changes

### 1) Automation page wording
- **File**: `frontend/apps/beitraege/src/pages/AutomationPage.vue`
- Remove any reference to "recipient is logged-in user".
- Update German labels to reflect family-based sending.

### 2) Dry-run preview modal
- **File**: `frontend/apps/beitraege/src/pages/AutomationPage.vue`
- When dry-run result returns, show a modal with:
    - Summary stats (families processed / emailed / skipped)
    - Warnings list
    - Expandable preview per family: recipients, subject, body
- "Jetzt senden" button in modal to re-run without dry-run.

### 3) Result display after real run
- Show: families emailed count, families skipped count, warning details.
- Replace or extend the current simple result card.

### 4) Types update
- **File**: `frontend/apps/beitraege/src/api/types.ts`
- Add new fields to `ReminderRunResponse`:
  ```typescript
  familiesProcessed: number;
  familiesEmailed: number;
  familiesSkippedNoEmail: number;
  warnings: Array<{ householdName: string; reason: string }>;
  previews?: Array<{
    householdName: string;
    recipients: string[];
    subject: string;
    body: string;
  }>;
  ```

## Testing

### Backend unit tests
- Family with 2 parents, both have emails → one email, 2 `To` recipients.
- Family with 2 parents, one missing email → send to 1 recipient, warning emitted for missing.
- Family with no valid emails → no send, skip + warning.
- Family with 2 children → one email listing both children's fees.
- Dry-run → no outbound emails, previews populated, no side effects.
- Email log stores comma-separated recipients.

### Email service tests
- `backend-fees/internal/email/email_test.go`: test `SendTextEmailMulti` with 1 and multiple recipients.

### Manual verification
1. Dry-run `initial` from Automation page → modal shows previews.
2. Verify preview content: greeting, fee list, bank details.
3. Run without dry-run → emails sent.
4. Check email log entries: correct recipients, correct body.
5. Verify family with no emails shows warning in result.

## Implementation Order
1. `email.go` — add `SendTextEmailMulti`
2. `reminder_service.go` — add `householdRepo` dependency, family grouping, new email template, updated result struct
3. `fee_handler.go` — update handler to use new `Run()` signature, serialize new response fields
4. `types.ts` — update `ReminderRunResponse`
5. `AutomationPage.vue` — dry-run preview modal, result display, wording update
6. Tests

## Step 2 (later)
- Apply same family-based recipient model to `final` (dunning).
- Dashboard warning card for families without parent emails.
