# Spenden im Bankabgleich

Feature-Spec und Übergabe. Status: Backend und Frontend umgesetzt (2026-11-01).

## Umfang

Ein offener Geldeingang lässt sich im Bankabgleich als Spende zuordnen. Die Seite „Spenden“ listet alle Spenden nach
Buchungsjahr gruppiert (neuestes Jahr zuerst) mit Summe und Anzahl je Jahr und insgesamt. Betrag = voller
Transaktionsbetrag, Jahr = Buchungsdatum (Berlin). Es gibt keine Spender-Stammdaten und keine Zuwendungsbestätigungen
(siehe „Offen“).

- Rollen: `ADMIN` und `USER` (wie der übrige Bankabgleich, Annahme). `LEITUNG` hat keinen Zugriff (Bankdaten),
  `PARENT_WORK`/`PARENT` ebenfalls nicht. Anders als bei Ratenzahlungen gibt es keine Schwärzung: eine Spende zeigt
  nur Zahler, Zweck und Notiz, die `USER` in der Bankliste ohnehin sieht.

## Datenmodell (Migration `000050_transaction_assignments`)

`fees.transaction_assignments`: `transaction_id` PK → `bank_transactions` (ON DELETE CASCADE), `kind` (CHECK
`DONATION`), `note` (max. 500 Zeichen), `assigned_at`, `assigned_by`.

Das Modell ist „Zweck einer Transaktion neben Beitrag und Ratenzahlung“, nicht „Spende“: eine weitere Zuordnungsart
(z. B. Zuschüsse, `docs/todo-fees-improvements.md` Phase 8) ist ein weiterer `kind`-Wert und kommt an denselben Stellen
wie die Spende vorbei. Für Zuschüsse ist nichts vorgebaut.

Eine Transaktion gehört genau einer Art an: Beiträgen (`payment_matches`, auch teilweise), einem Plan
(`installment_payments`) oder einem Zweck. Alle Zuordnungswege sperren die Bankzeile (`FOR UPDATE`) und prüfen
`transactionOccupiedSQL` (Repository); der Service lehnt gemischte Zuordnungen mit `HasExclusiveAssignment` ab.

## Bankabgleich

- Offene Listen und `ListUnmatched` (Rescan/Auto-Match) schließen zugeordnete Transaktionen aus,
  `DeleteUnmatchedByIBAN` löscht sie nicht, die Liste „Zugeordnet“ enthält sie (`matchedAmount` = Betrag).
- Nur Eingänge (Betrag > 0) ohne Beitragszuordnung können Spende werden.
- Aufheben: der vorhandene `POST /import/transactions/{id}/unmatch` entfernt auch die Spende.

## API (`/api/fees/v1`, `ADMIN`/`USER`)

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/import/transactions/{id}/assign` | `{kind: "DONATION", note}`, 204; 400 bei Ausgang, schon zugeordnet oder unbekanntem `kind` |
| GET | `/import/donations` | `{total, count, years: [{year, total, count, transactions[]}]}`, `transactions[].assignment` trägt die Notiz |

## UI

- Bankabgleich-Dialog: Chip „Spende“ (nicht bei teilweise zugeordneten Transaktionen) mit optionaler Notiz
  (`DonationAssignPanel`); Zeilen tragen das Badge „Spende“.
- Seite `/spenden` (Navigation „Beiträge“ → „Spenden“, `requiresFees`), Tabelle je Jahr, Liste auf dem Handy.
- E2E: `frontend/e2e/beitraege/donations.spec.ts`.

## Offen (nicht gebaut)

- Zuwendungsbestätigungen (Spendenquittungen) und Spender als Stammdaten (Adresse, Zuordnung mehrerer Zahlungen).
- Automatischer Vorschlag anhand „Spende“ im Verwendungszweck oder bekannter IBAN.
- Notiz nachträglich ändern (heute: aufheben und neu zuordnen), Export/CSV, Filter auf ein Jahr.
