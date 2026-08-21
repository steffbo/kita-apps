# Bankabgleich UI Overhaul — Plan

Status: in progress. Increment 1 done. This doc is the working agreement for
the `/beitraege` import/automation UI rework. Implementation happens in small,
individually committed increments. Check off sections as they land and move
finished facts into `docs/status-fees.md`.

## Problem

The daily workflow ("waren die letzten Zahlungen erfolgreich im System?")
currently spans two pages:

1. `/automatisierung` → Banking-Sync card: was the last bank import successful?
2. `/import` → six tabs: which payments were matched automatically, which need attention?

Consequences:

- One mental task is split across two routes; sync result and transaction state are never visible together.
- The actionable data (nicht zugeordnet, Warnungen) hides behind tabs next to reference data (Zugeordnet ~600 rows, Blacklist).
- `AutomationPage.vue` mixes three unrelated concerns: banking sync, Zahlungserinnerungen/Mahnungen, E-Mail-Protokoll.
- `ImportPage.vue` is ~2.270 lines; default tab is "Upload" although manual upload is the rare path (banking-sync does it automatically).

## Target structure

### 1. One "Bankabgleich" page (`/bankabgleich`, replaces `/import`)

Top-to-bottom order mirrors the workflow:

- **BankingSyncCard** (extracted component): last run status badge, Start/Ende,
  "Jetzt synchronisieren", Stoppen while running, polling behaviour unchanged.
  After a successful sync the transaction list below refreshes automatically.
- **Unified transaction list**: one table of all transactions, newest first,
  with a status badge per row:
  - 🟢 Zugeordnet (child name / expectation)
  - 🟡 Nicht zugeordnet (+ action buttons)
  - 🔴 Warnung
- Status filter chips above the list (`Alle | Offen | Warnungen`) act as *filters*, not tabs — the list stays one list.
- Manual CSV upload becomes a secondary header action ("CSV hochladen" button → dialog), not a tab.
- Historie (import batches) moves to a secondary view/link on this page.
- Blacklist moves out of the daily flow (see §3).

### 2. Automatisierung → "Erinnerungen"

The page keeps only what it really is: Zahlungserinnerungen Essens-/Platzgeld,
Vereinsbeitrags-Erinnerungen, E-Mail-Protokoll. Banking-Sync section removed.
Nav label changes accordingly (route can stay `/automatisierung` initially).

### 3. Dashboard widget

Compact card: last bank import (status, timestamp) + counts of offene
Transaktionen/Warnungen, deep links into the filtered views of `/bankabgleich`.
Dashboard becomes the natural "is everything fine?" entry point.

### 4. Sidebar regrouping (`MainLayout.vue`)

Group the flat nav list, e.g.:

- Täglich: Dashboard, Bankabgleich
- Verwaltung: Kinder, Eltern, Mitglieder
- Beiträge: Beiträge, Einstufungen, Erinnerungen

## Data / API notes

- Matched, unmatched and warning transactions come from three endpoints
  (`getMatchedTransactions`, `getUnmatchedTransactions`, `getWarnings`). A unified
  list starts by merging these client-side per page; sorted by date desc.
- If client-side merging proves awkward (double pagination, inconsistent
  totals), add a backend endpoint in `backend-fees`:
  `GET /api/fees/v1/bank/transactions?status=…` returning one paginated stream
  with status included. Decide when implementing increment 2.
- No auth or schema changes needed.

## Increments (one commit each)

1. [x] Extract `BankingSyncCard.vue` from AutomationPage; render it at the top of
       ImportPage. Nav rename "Import" → "Bankabgleich"; `/import` kept as route alias.
       The card emits `sync-finished` after a run transitions running/2FA → success;
       ImportPage then reloads all transaction tabs.
2. [x] Unified transaction list with status badges + filter chips; default chip
       = Offen (nicht zugeordnet + Warnungen). Upload demoted to header action/dialog.
       Decision: matched/unmatched/warnings are merged **client-side** (limits 1000/500/200)
       instead of adding a backend endpoint — data volume is hundreds of rows; revisit
       only if that grows by an order of magnitude.
3. [x] Blacklist moved into a dialog on the Bankabgleich page; Historie likewise
       (both reachable via small links in the toolbar footer).
4. [x] AutomationPage slimmed down to reminders + email log; banking-sync section
       removed, page retitled "Erinnerungen" (route stays `/automatisierung`).
5. [x] Dashboard widget: Bank-Import status card (last run + badge, deep link to
       `/bankabgleich`) as first card in the secondary row; unmatched-transactions
       card now links to `/bankabgleich`.
6. [x] Sidebar grouped into Täglich (Dashboard, Bankabgleich) / Verwaltung
       (Kinder, Eltern, Mitglieder) / Beiträge (Beiträge, Einstufungen, Erinnerungen).
7. [ ] Update `docs/status-fees.md` (and AGENTS.md nav notes if wording changed),
       remove checked-off plan content or mark done.

## Out of scope

- Other frontend apps, portal, backend behaviour changes beyond the optional
  transactions endpoint above.
