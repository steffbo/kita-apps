# Spenden im Bankabgleich

Feature-Spec und Übergabe. Status: Backend und Frontend umgesetzt (2026-11-01).

## Umfang

Ein offener Geldeingang lässt sich im Bankabgleich als Spende zuordnen. Die Seite „Spenden“ listet alle Spenden nach
Buchungsjahr gruppiert (neuestes Jahr zuerst) mit Summe und Anzahl je Jahr und insgesamt. Betrag = voller
Transaktionsbetrag, Jahr = Buchungsdatum (Berlin). Optional wird die Spende einem Haushalt zugeordnet (Eltern/Großeltern
spenden meist für die Familie). Es gibt keine Spender-Stammdaten und keine Zuwendungsbestätigungen
(siehe „Offen“).

- Rollen: Zuordnen `ADMIN`, Gesamtübersicht `ADMIN` und `BOARD` nur lesend (wie der übrige Bankabgleich). `LEITUNG` und
  `PARENT_WORK` haben keinen Zugriff (Bankdaten). Anders als bei Ratenzahlungen gibt es keine Schwärzung: eine Spende zeigt
  nur Zahler, Zweck und Notiz, die der Vorstand in der Bankliste ohnehin sieht.
- `PARENT` sieht die dem eigenen Haushalt zugeordneten Spenden: Datum, Zahlername, Betrag, Jahressumme. IBAN,
  Verwendungszweck und Notiz (intern) gehören nicht dazu; `service.OwnDonation` ist dafür die Positivliste. Der Zahlername
  ist bewusst sichtbar (Entscheidung 2026-11-01), auch für beide Elternteile eines Haushalts.

## Datenmodell (Migration `000050_transaction_assignments`)

`fees.transaction_assignments`: `transaction_id` PK → `bank_transactions` (ON DELETE CASCADE), `kind` (CHECK
`DONATION`), `note` (max. 500 Zeichen), `household_id` NULL → `households` (ON DELETE SET NULL, Migration `000051`),
`assigned_at`, `assigned_by`.

Das Modell ist „Zweck einer Transaktion neben Beitrag und Ratenzahlung“, nicht „Spende“: eine weitere Zuordnungsart
(z. B. Zuschüsse, `docs/todo-fees-improvements.md` Phase 8) ist ein weiterer `kind`-Wert und kommt an denselben Stellen
wie die Spende vorbei. Für Zuschüsse ist nichts vorgebaut.

Eine Transaktion gehört genau einer Art an: Beiträgen (`payment_matches`, auch teilweise), einem Plan
(`installment_payments`) oder einem Zweck. Alle Zuordnungswege sperren die Bankzeile (`FOR UPDATE`) und prüfen
den Bestand im Repository: Die SQL-Bedingung „Plan oder Zweck“ steht einmal in `inOtherLedger`, Listen und
`DeleteUnmatchedByIBAN` bauen darauf auf; Plan und Zweck prüfen zusätzlich Beitragszuordnungen (`transactionOccupiedSQL`).
Der Service lehnt gemischte Zuordnungen mit `HasExclusiveAssignment` ab.

## Bankabgleich

- Offene Listen und `ListUnmatched` (Rescan/Auto-Match) schließen zugeordnete Transaktionen aus,
  `DeleteUnmatchedByIBAN` löscht sie nicht, die Liste „Zugeordnet“ enthält sie (`matchedAmount` = Betrag).
- Nur sichtbare Eingänge (Betrag > 0) ohne Beitragszuordnung können Spende werden. Zuordnen löst offene Warnungen der
  Transaktion wie eine Beitragszuordnung. Spenden und Planzahlungen lassen sich nicht ausblenden (`POST …/hide` gibt 400).
- Der Auto-Match läuft beim Import vor jeder Spendenzuordnung. Bucht er eine Spende von einer bekannten IBAN als Beitrag,
  muss die Zuordnung erst aufgehoben werden. Das ist so akzeptiert (Entscheidung 2026-11-01).
- Aufheben: der vorhandene `POST /import/transactions/{id}/unmatch` entfernt auch die Spende.

## Haushalt und IBAN-Vorschlag

- Offene Eingänge tragen `householdSuggestion {householdId, name}`, wenn die Zahler-IBAN einem Haushalt bekannt ist:
  zuerst der Haushalt der letzten Spende von dieser IBAN, sonst der Haushalt eines Kindes, für das die IBAN als
  vertrauenswürdig geführt wird (`known_ibans`). Berechnet in `enrichHouseholdSuggestions`, nur für die Liste des
  Bankabgleichs (`ListTransactions`), nicht für Einzelabrufe oder den Rescan.
- Der Vorschlag ist nur vorbelegt, nie automatisch gesetzt: im Spenden-Panel ist der Haushalt vorgewählt, bleibt
  abwählbar („Haushalt entfernen“), die Zuordnung bestätigt der Nutzer.
- Notiz und Haushalt lassen sich später auf der Seite „Spenden“ ändern (`PUT …/assign`, Stift/Zeile).

## API (`/api/fees/v1`, `ADMIN`, `BOARD` nur lesend)

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/import/transactions/{id}/assign` | `{kind: "DONATION", note, householdId?}`, 204; 404 bei unbekanntem Haushalt; 400 bei Ausgang, schon zugeordnet oder unbekanntem `kind` |
| PUT | `/import/transactions/{id}/assign` | Notiz und Haushalt einer bestehenden Spende ersetzen, 204 |
| GET | `/import/donations?householdId=` | `{total, count, years: [{year, total, count, transactions[]}]}`, `transactions[].assignment` trägt die Notiz; optional auf einen Haushalt eingeschränkt |
| GET | `/households/options` | alle Haushalte mit Namen von Kindern, Eltern, Mitgliedern für `HouseholdPicker`; dieselbe Antwort wie `/parent-work/households`, das `USER` nicht erreicht |

`GET /me/donations` (`PARENT`): `[{year, total, count, transactions: [{id, bookingDate, amount, payerName}]}]`, nur der eigene
Haushalt; ohne Haushalt eine leere Liste.

## UI

- Bankabgleich-Dialog: Chip „Spende“ (nicht bei teilweise zugeordneten Transaktionen) mit optionaler Notiz
  und Haushaltssuche (`HouseholdPicker`, gemeinsam mit Ratenplan-Formular und Elternstunden; `DonationAssignPanel`); Zeilen tragen das Badge „Spende“.
- Seite `/spenden` (Navigation „Beiträge“ → „Spenden“, `requiresFees`), Tabelle je Jahr, Liste auf dem Handy.
- Eltern: Abschnitt „Spenden“ auf `/familie/beitraege`, unabhängig vom gewählten Jahr, ohne Spenden ausgeblendet.
- Haushalt ansehen (`ADMIN`/`USER`): `HouseholdDonationsCard` auf der Kindseite und auf `/elternstunden/familien/:id`.
  Eine eigene Haushaltsseite gibt es nicht. Die Familienseite erreicht `USER` nicht, dort bleibt die Kindseite.
- E2E: `frontend/e2e/beitraege/donations.spec.ts`.

## Offen (nicht gebaut)

- Zuwendungsbestätigungen (Spendenquittungen) und Spender als Stammdaten (Adresse, Zuordnung mehrerer Zahlungen).
- Spende ohne Klick zuordnen (auch der Haushalt ist nur Vorschlag), Vorschlag „ist Spende“ anhand des Verwendungszwecks.
- Spenden je Haushalt auswerten/gruppieren, Export/CSV, Filter auf ein Jahr.
