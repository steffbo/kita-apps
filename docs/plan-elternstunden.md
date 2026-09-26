# Plan: Elternstunden, Stufe 1 (Erfassung durch Tim)

Stand 2026-09-26. Fachliche Regeln und Herkunft: `docs/status-fees.md`, Abschnitt „Elternstunden: fachliche Regeln“.
Dieses Dokument ist der Umsetzungsvertrag; nach Abschluss wandern die bleibenden Fakten nach `AGENTS.md`
bzw. `docs/status-fees.md`, und der Plan wird gelöscht.

## Ziel

Tim Hall (Rolle `PARENT_WORK`) pflegt die Elternstunden in der Beiträge-App statt in Excel: Zettel erfassen,
Übersicht je Familie und Kita-Jahr mit Soll, Ist, Übertrag, offenen Stunden und Fehlbetrag, Vorstand kennzeichnen,
alte Excel-Liste importieren. Er sieht keine Beitrags-, Einkommens- oder Bankdaten.

Nicht in Stufe 1: Eltern-Zugang, Abnahme durch Erzieherinnen, Erinnerungs-Mails, automatische Forderung des
Fehlbetrags. Das Datenmodell lässt diese Stufen zu (Status am Eintrag), baut sie aber nicht.

## Begriffe

- **Kita-Jahr** `Y`: 01.08.Y bis 31.07.(Y+1); in der API als Startjahr `kitaYear` (int), Anzeige „2026/27“.
- **Tertial**: 01.08.–30.11., 01.12.–31.03., 01.04.–31.07.
- **Familie** = `fees.households`. Einträge, Soll und Konto gehören der Familie, nicht dem Kind.
- **Kind der Familie im Jahr Y**: `fees.children.household_id = Familie` und Betreuungszeit
  (`entry_date` bis `exit_date`, offen = unbegrenzt) überschneidet das Kita-Jahr. `is_active` wird ignoriert.
- **Vereinsmitglied der Familie**: `fees.members.household_id = Familie` oder über `fees.parents.member_id`
  eines Elternteils mit `fees.parents.household_id = Familie` (Vereinigung beider Wege, ohne Duplikate).
- **Minuten**: Alle Dauern werden in ganzen Minuten gespeichert; erlaubt sind Vielfache von 15 (> 0).
  Die UI erfasst Stunden in Schritten von 0,25.

## Regeln (Berechnung je Familie und Kita-Jahr Y)

Regelwerk: `fees.parent_work_rules`, versioniert per `valid_from` (DATE, unique). Für Y gilt die Version mit dem
größten `valid_from <= 01.08.Y`. Gibt es keine, ist Y nicht abrechenbar (Soll 0, kein Übertrag, Hinweis in der
API). Felder: `hours_per_child_minutes` (540), `missing_hour_rate_cents` (3000), `max_carry_over_minutes` (180).
Seed: `valid_from = 2025-08-01` mit diesen Werten (Quelle: Betreuungsvertrag 7.3 und Vordruck;
ANNAHME Startdatum: ältere Jahre werden nicht abgerechnet).

1. **Berechnetes Soll** = Σ über Kinder der Familie in Y: `(hours_per_child_minutes / 3) × Anzahl Tertiale,
   die die Betreuungszeit des Kindes schneidet` (ein Tag genügt).
2. **Vorstandsbefreiung**: Schneidet eine Amtszeit (`fees.board_terms`) eines Vereinsmitglieds der Familie das
   Kita-Jahr (ein Tag genügt), ist das Soll 0 und `exemptReason` = „Vorstand: Vorname Nachname (Amt)“.
3. **Manuelles Soll** (`fees.parent_work_overrides`, eindeutig je Familie+Jahr, mit Pflichtbegründung) ersetzt
   das Ergebnis aus 1 und 2 vollständig.
4. **Ist** = Summe `duration_minutes` aller Einträge der Familie mit `status = 'APPROVED'` und `work_date` in Y.
5. **Übertrag in Y** (`carryIn`) = `carryOut(Y-1)`, für das erste abrechenbare Jahr 0.
   `carryOut(Y) = min(max_carry_over_minutes(Y), max(0, Ist(Y) + carryIn(Y) − Soll(Y)), Ist(Y))`.
   Übertragene Stunden werden zuerst verbraucht und nie ein zweites Mal übertragen (Kappung durch `Ist(Y)`).
6. **Offen** = `max(0, Soll − Ist − carryIn)`; **Fehlbetrag** in Cent = `round(Offen × rate / 60)`
   mit `rate = missing_hour_rate_cents(Y)`. Der Fehlbetrag wird nur angezeigt.
7. Die Übersicht enthält jede Familie mit Soll > 0, Einträgen in Y, Befreiung oder manuellem Soll in Y.

Die Berechnung ist eine reine Funktion (Eingaben: Regelwerk, Kinder-Zeiträume, Amtszeiten, Override, Einträge,
carryIn) und wird tabellengetrieben unit-getestet, u. a.: volles Jahr 9 h; Eintritt 15.01. → 6 h; ein Tag im
Jahr → 3 h; zwei Kinder → 18 h; Vorstand 6 Monate → 0; Override schlägt Vorstand; Übertrag gekappt auf 3 h;
Übertrag nicht kettenweise (Vorjahr nur carryIn, kein Ist → carryOut 0); 1,5 h offen → 45,00 €.

## Rollen und Rechte

Neue Rolle `PARENT_WORK` („Elternstunden“). `fees.users.role` CHECK erweitern (neue Migration).

| Bereich | ADMIN | USER | PARENT_WORK |
| --- | --- | --- | --- |
| `/auth/me`, `/auth/change-password` | ja | ja | ja |
| `/users` | ja | – | – |
| alle Beitrags-Routen (Kinder, Eltern, Familien, Mitglieder, Beiträge, Einstufungen, Beitragsordnung, Stichtag, Import, Bankabgleich, Notizen) | ja | ja | – |
| darin schon heute nur ADMIN (Banking-Sync, Reminder, E-Mail-Logs, Beitragsordnung schreiben) | ja | – | – |
| `POST /import/upload` per JWT | ja | ja | – (Import-Token unverändert) |
| `/parent-work/**` | ja | – | ja |
| `/parent-work/rules` schreiben | ja | – | – |

Umsetzung im Router mit `RequireRole` auf Gruppenebene, keine verstreuten Einzelprüfungen. Ein Router-Test prüft
für jede Rolle eine repräsentative Route je Bereich (200/403). Frontend: Navigation und Route-Guards nach Rolle;
`PARENT_WORK` landet nach dem Login auf `/elternstunden` und sieht nur die Elternstunden-Gruppe; Benutzer-Dialog
bietet die Rolle an; Rollen-Labels „Administrator“, „Benutzer“, „Elternstunden“.

## Datenmodell (neue Migrationen ab `000038`)

- `fees.parent_work_rules(id, valid_from DATE UNIQUE, hours_per_child_minutes INT >= 0,
  missing_hour_rate_cents INT >= 0, max_carry_over_minutes INT >= 0, created_at, updated_at)` + Seed.
- `fees.board_terms(id, member_id → fees.members ON DELETE CASCADE, office TEXT NOT NULL, start_date DATE NOT
  NULL, end_date DATE NULL (>= start_date), note TEXT, created_at, updated_at)`.
- `fees.parent_work_entries(id, household_id → fees.households, work_date DATE, duration_minutes INT > 0 und
  % 15 = 0, occasion TEXT NOT NULL (nicht leer), member_name TEXT, child_name TEXT, status TEXT CHECK IN
  ('SUBMITTED','APPROVED','REJECTED','VOIDED') DEFAULT 'APPROVED', void_reason TEXT (Pflicht bei VOIDED),
  source TEXT CHECK IN ('MANUAL','IMPORT'), created_by/updated_by → fees.users NULL, created_at, updated_at)`;
  Index auf `(household_id, work_date)`. `member_name`/`child_name` sind Freitext vom Zettel (Information).
- `fees.parent_work_overrides(id, household_id, kita_year INT, required_minutes INT >= 0, reason TEXT NOT NULL,
  created_by, created_at, updated_at, UNIQUE(household_id, kita_year))`.

Löschen einer Familie mit Elternstunden-Einträgen muss scheitern (FK ohne CASCADE) oder vorher geprüft werden;
Verhalten wie bei bestehenden Abhängigkeiten der Familie.

## API (`/api/fees/v1/parent-work`, swag-annotiert, danach `scripts/generate-api.sh`)

- `GET /overview?kitaYear=` → Regelwerk des Jahres, Summen, Liste je Familie: `householdId, householdName,
  children[{id, name, tertials}], requiredMinutes, calculatedMinutes, overrideMinutes?, overrideReason?,
  exemptReason?, doneMinutes, carryInMinutes, openMinutes, carryOutMinutes, missingAmountCents, entryCount`.
  Ohne `kitaYear`: aktuelles Kita-Jahr nach `util.Today()`.
- `GET /households?search=` → schlanke Familienliste für Auswahl/Suche: id, name, Kinder (Vor-/Nachname,
  Zeitraum), Vereinsmitglieder (id, Name), Eltern (Name). Keine Einkommens-, Adress- oder Bankdaten.
- `GET /households/{id}?kitaYear=` → Zeile wie in der Übersicht + Einträge des Jahres (inkl. VOIDED) +
  Amtszeiten der Familienmitglieder.
- `POST /entries`, `PUT /entries/{id}`, `POST /entries/{id}/void` (Body `reason`). Kein hartes Löschen.
- `PUT /households/{id}/override` (Body `kitaYear, requiredMinutes, reason`), `DELETE /households/{id}/override?kitaYear=`.
- `GET/POST /board-terms`, `PUT/DELETE /board-terms/{id}` (Liste mit Mitgliedsname und Familie).
- `GET /rules`, `POST /rules`, `PUT /rules/{id}` (Schreiben nur ADMIN; nur Versionen mit `valid_from` in der
  Zukunft änderbar, wie bei der Beitragsordnung).
- Import: `POST /import/parse` (CSV-Upload, liefert Kopfzeile + Zeilen), `POST /import/preview` (Mapping
  Spalte → Feld; liefert je Zeile erkannte Familie oder Fehler, Dublettenhinweis),
  `POST /import/execute` (Zeilen mit endgültiger `householdId`; legt Einträge mit `source = 'IMPORT'` an).
  Muster wie `child_import` und `internal/csvparser`. Felder: Mitglied, Kind, Datum, Anlass, Stunden.
  Zuordnung: Kindname (Vor- + Nachname, ohne Groß-/Kleinschreibung) → Familie des Kindes; sonst Mitgliedsname
  → Familie des Mitglieds; mehrdeutig/unbekannt = Fehler, im Frontend per Familienauswahl lösbar.
  Stunden: „1,5“, „1.5“, „2“; nicht auf 0,25 h teilbar = Fehler. Dublette = gleiche Familie, Datum, Dauer und
  Anlass wie ein bestehender, nicht stornierter Eintrag (wird standardmäßig übersprungen).

Fehlermeldungen deutsch, Validierung im Service, Zeitlogik mit `util.Today()`/`util.Now()`.

## Frontend (`frontend/apps/beitraege`)

Navigationsgruppe „Elternstunden“ (ADMIN, PARENT_WORK):

- `/elternstunden` Übersicht: Kita-Jahr-Auswahl, Kacheln (Familien, Soll, Ist, offen, Fehlbetrag gesamt),
  Tabelle je Familie mit Status (erfüllt / offen / befreit), Suche, Filter „nur offen“; Button „Stunden erfassen“.
- `/elternstunden/familien/:id?jahr=` Detail: Kinder mit gezählten Tertialen, Befreiung, manuelles Soll
  (setzen/entfernen mit Begründung), Konto (Soll, Übertrag, Ist, offen, Fehlbetrag), Einträge mit
  Bearbeiten/Stornieren, „Stunden erfassen“ vorbelegt mit der Familie.
- Erfassungsdialog: Familie (Suche über Familien-, Kinder- und Mitgliedsnamen), Datum, Stunden (Schritt 0,25),
  Anlass, Mitglied und Kind (Auswahl aus der Familie oder Freitext).
- `/elternstunden/vorstand`: Amtszeiten anlegen/bearbeiten/beenden, Mitglied per Suche.
- `/elternstunden/import`: CSV hochladen, Spalten zuordnen, Vorschau mit Fehlern und Familienauswahl, ausführen.
- `/elternstunden/regeln`: Regelwerk-Versionen anzeigen; ADMIN kann neue Version anlegen.

Stil und Bausteine wie die bestehenden Seiten (Tailwind, lucide-Icons, `api/client.ts`, Typen aus
`schema.d.ts` über `api/types.ts`). Zugängliche Selektoren (`role="dialog"`, `aria-label`) für e2e.

## Etappen

| # | Inhalt | Umsetzung |
| --- | --- | --- |
| 1 | Rolle `PARENT_WORK`, Router-Gruppen, Import-Upload-Einschränkung, Frontend-Guards/Navigation/Benutzerdialog, Router-Test | Codex Sol |
| 2 | Backend Kern: Migrationen, Domain, Berechnung + Tests, Repository, Service, Handler, Routen, OpenAPI | Codex Sol |
| 3 | Frontend: Übersicht, Detail, Erfassung, Vorstand, Regeln | Codex Sol |
| 4 | CSV-Import Backend + Frontend | Codex Luna |
| 5 | Playwright-e2e (Rolle sieht nur Elternstunden, Erfassen → Übersicht), Docs, Gesamtprüfung | Luna + selbst |

Nach jeder Etappe: Ergebnis prüfen, `go test ./...`, Typecheck, `scripts/generate-api.sh`, dann Commit.
Push und Deploy nur nach Freigabe.
