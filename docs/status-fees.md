# Status: Beiträge (backend-fees + frontend/apps/beitraege)

Rolling change log of non-obvious implementation decisions. Newest first.
Basics (ports, commands, layout) live in `AGENTS.md`.

## Code-Review-Backlog (ab 2026-09-23)

Abarbeitung von `docs/todo-fees-improvements.md`, ein Commit pro Punkt.

- **Import-Fehler sichtbar (#10):** `ProcessCSV` bricht jetzt ab (HTTP 500 → banking-sync meldet `error` in der Sync-Karte und an Uptime Kuma), wenn Blacklist, Kinder oder Eltern nicht geladen werden können oder der Batch nicht angelegt werden kann. Vorher lief der Import still gegen eine leere Kinderliste weiter. Der Batch wird erst nach dem Laden der Referenzdaten angelegt. Fehler einzelner Buchungen (Duplikatprüfung, Speichern, Warnung speichern, Auto-Match) landen in `ImportResult.errors` (`domain.ImportError`: Datum, Zahler, Betrag, Meldung) statt stumm als `skipped`. Migration `000032` speichert sie an `fees.import_batches` (`error_count`, `errors` JSONB, max. 100 Details). `GET /import/history` liefert beides, Rescan liefert `errors` ebenfalls. Die Kinderliste für das Matching wird seitenweise komplett geladen (vorher hartes Limit 1000). Frontend: Komponente `ImportErrorList`; Banner „Letzter Import … mit Fehlern“ auf der Bankabgleich-Seite (lädt den neuesten Batch), Spalte „Fehler“ mit aufklappbaren Details in der Import-Historie, Fehlerliste im Upload-Ergebnis und nach „Erneut zuordnen“, und in der Banking-Sync-Karte, wenn `uploadResult.errors` gefüllt ist. Fehler beim Laden der Historie werden im Modal angezeigt statt nur in der Konsole.
- **Atomares Matching (#9):** `ImportService` bekommt den `TxManager`. Atomar sind jetzt: manuelles Zuordnen, Bestätigen (je Match), Auto-Match beim Import/Rescan (alle Matches einer Buchung + Folgeaktionen), Verteilen (Validierung + alle Zuordnungen + Folgeaktionen), Zuordnung aufheben, Ablehnen (Blacklist + Löschen offener Buchungen der IBAN), Mahngebühr aus Verspätungs-Warnung (Gebühr + Warnung auflösen). Die Folgeaktionen (`postMatchActions`: IBAN als vertrauenswürdig markieren, Warnungen auflösen, Verspätung prüfen) geben Fehler jetzt zurück statt sie zu verschlucken; schlägt eine fehl, wird der Match mit zurückgerollt. Beim Bestätigen zählt das als `failed` (mit Log), beim Auto-Match bleibt die Buchung als Vorschlag stehen. Beim CSV-Import bleibt das Speichern der Buchung eine eigene Einheit, damit ein fehlgeschlagener Auto-Match die Buchung nicht verliert. Test `import_tx_integration_test.go` erzwingt einen Fehler in der letzten Folgeaktion und prüft, dass weder Match noch IBAN-Eintrag übrig bleiben.
- **Transaktions-Infrastruktur (#8):** `repository.TxManager.WithTx(ctx, fn)` bindet eine `*sqlx.Tx` an den Context. Alle Repository-Statements laufen über `conn(ctx, r.db)` und nutzen damit automatisch die äußere Transaktion; ohne `WithTx` verhalten sie sich wie bisher. Repository-Methoden mit eigener Transaktion (`beginTx`, z. B. `ChildRepository.Update`, Einstufung-Folgeeinstufung) schließen sich einer äußeren Transaktion an; ihr `Commit`/`Rollback` ist dann wirkungslos. Verschachteltes `WithTx` tritt bei. Ein `nil`-TxManager führt `fn` ohne Transaktion aus (für Unit-Tests mit Fakes). Noch kein Service nutzt es, das kommt mit #9.
- **Zeitzone Europe/Berlin (#7):** Der Container lief in UTC, dadurch galt zwischen 0 und 2 Uhr (Sommer) bzw. 0 und 1 Uhr (Winter) noch der Vortag, z. B. für Fälligkeit, Betreuungszeit-Historie, Jahres-/Monatsvorgaben und Mahnlauf-Datum. Neu: `util.Now()` (Berliner Wanduhr, für Jahr/Monat/Zeitstempel) und `util.Today()` (Berliner Kalendertag als UTC-Mitternacht, also im selben Format wie gescannte `DATE`-Spalten; für Datumsvergleiche). Zonendaten sind per `time/tzdata` eingebettet, `util.SetClock` macht Tests deterministisch. Zusätzlich `TZ=Europe/Berlin` im Image (betrifft nur Log-Ausgabe). Bewusste Verhaltensänderung: Die Beitragsübersicht zählt eine Gebühr erst ab dem Tag nach Fälligkeit als überfällig (bisher ab 0 Uhr UTC am Fälligkeitstag), konsistent mit dem Erinnerungs-Workflow. Der Stichtagszähler rechnet in ganzen Kalendertagen. Nachgezogen: U3-Filter der Kinderliste und Fälligkeit der Mahngebühr aus einer Verspätungs-Warnung. Reine Zeitstempel (`created_at` usw.) bleiben `time.Now()`.
- **Request-Limits (#6):** Globale Middleware `MaxBodySize` begrenzt jeden Body auf 5 MB; `request.DecodeJSON` zusätzlich auf 1 MB. CSV-Uploads (Bank- und Kinder-Import) laufen über `parseUpload`: zu groß → 413, sonst kaputt → 400 (vorher wurde der Fehler beim Bank-Upload ignoriert). `scripts/generate-api.sh` nutzt `BUN_CONFIG_REGISTRY` statt des von neueren Bun-Versionen abgelehnten `bunx --registry`. Die zuvor von Hand in `openapi3.yaml` gepflegten Hinweise zu ignorierten Einstufungsfeldern stehen jetzt als Go-Kommentare (`Deprecated: …`) an den DTOs und überleben so die Neugenerierung; swag kann das Feld-Flag `deprecated: true` selbst nicht erzeugen.
- **Import-Token (#4):** `CRON_API_TOKEN` wird über `config.Import.Token` geladen statt per `os.Getenv` in der Middleware und in konstanter Zeit verglichen (`subtle.ConstantTimeCompare`). Leerer Token deaktiviert Token-Auth. Bearer-Parsing ist zwischen `AuthMiddleware` und `ImportAuthMiddleware` geteilt; Tabellen-Test in `import_auth_test.go`.
- **Unsichere Defaults (#5):** Statt Start-Abbruch (würde `go run` ohne `.env` brechen) härtet `config.Harden()` beim Start: fehlendes, zu kurzes (<32 Zeichen) oder altes Dev-Default-`JWT_SECRET` → zufälliges Secret pro Prozess + Warnung (Sessions enden beim Neustart). `CORS_ORIGINS` hat keinen Default mehr; ohne Wert wird keine CORS-Middleware installiert (Frontend ist same-origin bzw. über Vite-Proxy). `*` wird nie mit `AllowCredentials` kombiniert. Prod (infra-dev) setzt `JWT_SECRET` (44 Zeichen) und `CORS_ORIGINS=https://kita.remer.cc` und ist unverändert.
- **CI vor Image-Build (#1):** Neuer Workflow `ci.yml` (PR, `workflow_call`, manuell) prüft `backend-fees` mit `gofmt -l`, `go vet`, `go test ./...` (Integrationstests via testcontainers) und `beitraege` per `typecheck`. `build-images.yml` ruft ihn als ersten Job auf, die Image-Builds hängen per `needs` daran.
- **gofmt (#2):** 10 Dateien formatiert, reiner Format-Commit.
- **Typecheck ohne Emit (#3):** `vue-tsc -b` hat `vite.config.js`/`.d.ts` neben `vite.config.ts` erzeugt und eingecheckt; Vite lädt im Dev-Modus `vite.config.js` bevorzugt, die `.ts` war also potenziell wirkungslos. Die Projekt-Referenz ist entfernt, beide tsconfigs sind `noEmit`, die Artefakte gelöscht. Neues Script `bun run typecheck` (App + `vite.config.ts`), `build` ruft es vor `vite build` auf.

## Eintrittsmonat anteilig berechnen (2026-09-18)

- Die Beitragserzeugung berücksichtigt jetzt § 12 Abs. 3 der Elternbeitragsordnung: Bei Eintritt nach dem 15. werden Platz- und Essensgeld im Eintrittsmonat hälftig angesetzt; bis einschließlich 15. bleibt der volle Monatsbeitrag fällig.
- Die Monatsübersicht der Einstufung und das Druck-PDF zeigen den hälftigen Eintrittsmonat als eigenen Zeitraum und danach wieder die vollen Monatswerte.
- Wird das Datum einer Einstufung korrigiert, werden `change_date`, `effective_from_month` und das Einstufungsjahr konsistent nachgeführt. Bei einer Folgeeinstufung wird außerdem das Ende des Vorgängerzeitraums atomar angepasst; ihre Stichtagslogik bleibt erhalten.
- Das Bearbeitungsformular zeigt als „Gültig ab“ wieder den tatsächlichen Änderungs-/Eintrittstag aus `changeDate`; der intern normalisierte Monatsbeginn aus `validFrom` bleibt ausschließlich die Periodengrenze.
- Beim Öffnen einer ursprünglichen Einstufung lädt die PDF-Vorschau einen vorhandenen direkten Nachfolger mit, sodass das Gesamtdokument Eintrittsmonat, Übergangszeit und den aktuellen Beitrag enthält.
- Die Altersbefreiung beginnt in der PDF – konsistent mit der Gebührenerzeugung – bereits im Monat des 3. Geburtstags und nicht erst im Folgemonat.

## Einstufung aus Kinddaten ableiten (2026-09-18)

- Die Einstufung erfasst keine manuell gesetzte Gültigkeit oder Betreuungszeit mehr. Für eine neue Jahreseinstufung wird der Beginn aus Einstufungsjahr und Eintrittsdatum bestimmt; die Wochenstunden werden für jeden Monat aus `child_care_hours_history` gelesen.
- Änderungen der Betreuungszeit erfordern damit keine Folgeeinstufung. Die API-Monatstabelle berechnet jeden Monat mit den zu diesem Zeitpunkt gültigen Stunden, der Eintrittsmonatsregel und der Altersbefreiung neu; die PDF fasst identische aufeinanderfolgende Monate zu Zeiträumen zusammen.
- Das Frontend blendet Gültigkeits- und Stundenfelder bei normalen Einstufungen aus und weist auf die Herkunft aus dem Kinddatensatz hin. Nur echte Folgeeinstufungen wegen einer geänderten Einkommensentscheidung benötigen weiterhin ein Änderungsdatum.
- Beim Löschen der letzten Folgeeinstufung wird deren direkter Vorgänger atomar wieder geöffnet. Einstufungen mit eigenem Nachfolger können nicht gelöscht werden.

## Backend (`backend-fees`)

### Review-Fixes Erinnerungs-Workflow (2026-09-15)

- **UTF-8-Betreffzeilen:** SMTP-Nachrichten kodieren den `Subject` jetzt nach RFC 2047 (`mime.QEncoding`). Die MIME-Charset-Angabe des Nachrichtentextes gilt nicht für Header; ohne die explizite Header-Kodierung erschienen Umlaute wie „Beiträge" in einzelnen Mail-Clients als „BeitrÃ¤ge". Der Multipart-Versand ist mit einem Decode-Regressionstest abgesichert.

Ergebnis eines externen Review-Agenten (3 Blocker, 5 P2) — alle umgesetzt:

- **Doppelte Mahngebühren bei Parallel-Send (P1)**: Neue Migration 000031 legt einen
  Unique-Partial-Index auf `fee_expectations(reminder_for_id) WHERE fee_type='REMINDER'`
  (mit defensivem Dedupe davor, damit Bestandsdaten den Index nicht blockieren). Der Send
  erkennt den Unique-Verstoß und antwortet mit 409 statt doppelter 10-/5-€-Gebühren; der
  Verlierer eines Rennens entfernt seine bereits erstellten Gebühren wieder (Kompensation).
- **PreviewedAt ist Pflicht (P2)**: `POST .../send` ohne `previewedAt` → 400; der
  Konkurrenzschutz ist damit serverseitig erzwingbar, nicht nur UI-Konvention.
- **Nicht fällige Fees bleiben `never_contacted` (P2)**: `feeWorkflowStatus` prüft jetzt
  erst die Fälligkeit, dann den Kontakt. Eine vorzeitig kontaktierte, noch nicht fällige
  Fee wird nicht mehr `actionable_final`, solange `due_date >= asOf`.
- **Stichtag als Kalendertag (P2)**: Fees, die am Stichtag selbst (nicht nur vor
  Mitternacht) erstellt wurden, sind `history_unknown` — „bis zum Stichtag einschließlich“.
- **SMTP-Fehler erzeugen keine verwaisten Mahngebühren mehr (P2)**: Schlägt der Versand
  fehl, werden die in diesem Request erstellten Mahngebühren best effort gelöscht; ein
  Retry plant sie erneut.
- **Frontend-Races (P1/P2)**: `loadCases`, `refreshPreview` und `loadChronology`
  korrelieren Responses über Request-Sequenznummern — veraltete Antworten (z. B. das
  initiale scope=actionable-Laden nach einem Scope-Wechsel, eine langsame Preview oder die
  Chronik der vorherigen Familie) überschreiben nichts mehr. Der Send-Dialog öffnet nicht,
  während eine Preview lädt. Zusätzlich invalidiert `invalidatePreview()` die Vorschau
  synchron bei jeder Eingabeänderung (Fee-Auswahl, Stufe, QR, Fallwechsel, Fall schließen):
  Die Sequenznummer wird sofort erhöht und die Vorschau geleert, sodass eine noch in der
  Luft hängende alte Response während des 250-ms-Debounce-Fensters nichts mehr
  überschreiben kann (kein kurzzeitiges Anzeigen fremder Beträge/Texte, kein Senden eines
  veralteten Override-Textes).
- **Scope-Wechsler lädt neu (P1)**: `watch(scope)` triggert `loadCases()`; „Alle offenen“
  zeigt jetzt tatsächlich wartende/zukünftige Familien.
- Tests: paralleler Send (genau 1 Gebühr), Send ohne `previewedAt`, SMTP-Kompensation mit
  Retry, `never_contacted` bei Frühkontakt, Stichtags-Kalendertag, geschärfter E2E-Scope-Test
  (seedt eine Nur-Zukunft-Familie). E2E-Stack: `reminder_history_reliable_from` auf
  `2026-01-01` fixiert, siehe `docs/test-seeding.md`.

### Reminder-Fixes aus dem manuellen Test (2026-09-15)

- **Mahngebühren nur bei „Mahnung"**: `prepareCasePlan` plante Gebühren für beide Stufen — bei „Erinnerung" erschienen geplante 10-€-Gebühren in Vorschau/Mail, und ein Send hätte sie erstellt. Jetzt nur noch `stage=final`. Regressionstest ergänzt.
- **SEPA-Verwendungszweck aus Fee-Perioden**: `buildSEPAReference` nutzte den Laufmonat (`Essens- und Platzbeitrag September 2026`) — im Legacy korrekt (Laufmonat = Beitragsmonat), im familienbasierten Workflow mit älteren/offenen Fees aber irreführend. Neu: `reminderReferencePeriod(items)` leitet die Periode aus den Fees ab — monatlich `MM/YYYY` (mehrere Perioden `01/2026+08/2026`), Vereinsbeitrag `YYYY`, Mahngebühren über `BaseYear/BaseMonth`. Mitgliedsnummern und Betrag stehen unverändert im QR-Payload. Fallback ohne Perioden: Laufmonat.
- **Nil-Slices → JSON `null`**: `collectEmails`, geplante Gebühren und Warnungen lieferten Go-nil-Slices; das Frontend crashte beim Rendern (`null.length`). Jetzt immer leere Arrays.
- Playwright-Suite `e2e/tests/beitraege/reminder-cases.spec.ts` (4 Szenarien, seedt Familien selbst über die API: Kind+Parent werden via `POST /households/{id}/children|parents` verknüpft — `CreateChildRequest`/`CreateParentRequest` nehmen keine `householdId`): Arbeitsliste/Suche/Auswahl/Vorschau/QR/Text-Reset/Mahnung-Warnung/Bestätigung (Send → erwartete 503 ohne SMTP), blockierte Familie ohne E-Mail, Scope-Filter. 4/4 grün, `go test ./...` und `bun run build:beitraege` grün.

### Familienbasierte Erinnerungs-UI (2026-09-15)

- `AutomationPage.vue` (**komplett neu geschrieben**, Route bleibt `/automatisierung`, Nav „Erinnerungen"): Die vier globalen Versandaktionen und der **Automatik-Schalter entfallen aus der Oberfläche** (der Wert wird beim Speichern der Zahlungsdaten unverändert zurückgeschrieben). Neue Struktur: Tabs **Arbeitsliste** / **Versandverlauf**, Zahlungsdaten hinter einem Einstellungs-Dialog.
- **Arbeitsliste:** Scope-Umschalter „Handlungsbedarf" (`scope=actionable`) / „Alle offenen" (`scope=all`), clientseitige Familiensuche. Familienzeilen: Name, offener Gesamtbetrag, nächste Aktion (`nextActionAt`), letzter bekannter Kontakt (max. `lastContactAt` über die Fees), Beitrags-Chips im Farbsystem (Vereinsbeitrag violett, Essensgeld orange, Platzgeld blau, Mahngebühr rot) und rotes „Keine E-Mail"-Badge für blockierte Familien. Desktop Master-Detail (`lg:grid-cols-[2fr_3fr]`), mobil öffnet sich der Familienfall als eigene Ansicht mit Zurück-Navigation.
- **Detail:** Vorauswahl = alle Fees mit Status `actionable_initial`/`actionable_final`/`history_unknown`; `never_contacted`/`waiting` sichtbar aber nicht gewählt. Mahnstufe als bewusste Auswahl (Erinnerung grün/Mahnung amber) mit Empfehlungshinweis gemäß Backend-Regel (alle `actionable_initial` → Erinnerung, alle `actionable_final` → Mahnung, gemischt → kein Hinweis). Jede Zeile zeigt Kind, Typ-Chip, Zeitraum, Fälligkeit, Soll und offenen Rest. Frist wird **nicht** editiert — sie kommt aus der Server-Vorschau (`runDate + 7`).
- **Live-Vorschau:** Debounced Re-Fetch bei Änderung von Auswahl, Mahnstufe oder QR-Schalter (`previewReminderCase`). Betreff/Text optional editierbar; Änderungen an Auswahl/Mahnstufe regenerieren den Text und setzen manuelle Änderungen mit sichtbarem Hinweis („Manuelle Textänderungen wurden zurückgesetzt") zurück. Geplante Mahngebühren (10 €/5 €) werden als Liste angezeigt; QR-Bild + Payload sind ausklappbar.
- **Send:** Bestätigungsdialog mit Empfänger, Beitragszahl, Gesamtbetrag, Frist, QR-Status und neu entstehenden Mahngebühren. `previewedAt` (Zeitstempel des letzten Preview-Fetches) wird mitgeschickt; **409** (`ReminderCaseConflictError` mit `feeIds` aus dem Response-Body) zeigt eine klare Fehlermeldung, lädt die Liste neu und regeneriert die Vorschau. Nach erfolgreichem Versand: Erfolgsmeldung, Liste neu laden, automatisch der nächste Fall geöffnet; die Chronik der Familie (neuer `householdId`-Filter auf `GET /fees/email-logs`, 20 Einträge) erscheint im Detailbereich.
- Versand ist deaktiviert ohne Vorschau, ohne Auswahl oder ohne Empfänger. `bun run build:beitraege` grün.

### Familienbasierte Reminder-Endpunkte (2026-09-15)

- **`GET /fees/reminder-cases?asOf=YYYY-MM-DD&scope=actionable|all`** (ADMIN): familienweise offene Forderungen über alle Beitragsarten. Je Fee: Kind, Typ, Zeitraum, Fälligkeit, Sollbetrag, **Restbetrag** (Soll − SUM(payment_matches), Teilzahlungen fließen ein), Status, `actionableAt`, letzter Kontakt, `hasReminder`. Je Familie: Empfänger (Eltern-E-Mails, dedupliziert), Summe Restbeträge, früheste nächste Aktion. Empfänger darf leer sein (blockierte Fälle, Versand im UI deaktiviert).
- **Fachregeln Status (`feeWorkflowStatus`)**: `actionable_initial` = überfällig + zuverlässig nie kontaktiert (Fees nach dem Stichtag); `waiting` = Kontakt vorhanden, 7-Tage-Frist (`payload.runDate + 7`) noch nicht abgelaufen; `actionable_final` = Frist abgelaufen; `history_unknown` = überfällig, aber vor dem Stichtag angelegt ohne zuordenbaren Log (keine Empfehlung); `never_contacted` = noch nicht fällig. Actionability = `due_date < asOf` — Vereinsbeiträge (due 31.03. des Beitragsjahres) werden damit automatisch erst nach dem 31.03. aufgenommen.
- **`POST /fees/reminder-cases/{householdId}/preview`** und **`/send`** (ADMIN), gemeinsamer Request `{stage: initial|final, runDate?, feeIds[], includeQR?, subject?, body?, previewedAt?}`. Die Frist berechnet **immer der Server** als `runDate + 7 Tage` (nicht editierbar). Einheitliche Betreffzeilen „Kita Zahlungserinnerung: offene Beiträge" / „Kita Mahnung: offene Beiträge" (`buildFamilyMixedReminderEmail`); `reminderLine` zeigt Monthly- wie Yearly-Beiträge korrekt.
- **Preview** liefert Mailtext, QR (DataURL + Payload), Gesamtbetrag, Empfehlung (`recommendedStage`: initial nur wenn **alle** ausgewählten `actionable_initial`, final nur wenn alle `actionable_final`, sonst none), Warnungen (nicht erinnert bei Mahnung, laufende Frist, unbekannte Historie, existierende Mahngebühr) und **virtuelle** Mahngebühren. QR/Mail nutzen Restbeträge plus geplante Gebühren.
- **Send** re-validiert: fremde oder inzwischen bezahlte Fee-IDs → **409** `ReminderCaseConflictResponse{message, feeIds}`; doppelte IDs → 400; Mahngebühr, die **nach `previewedAt`** (RFC3339, vom Client mitgeschickt) auf einen ausgewählten Grundbeitrag angelegt wurde → 409 (Konkurrenz-Schutz). Send erstellt pro Grundbeitrag ohne existierende Mahngebühr genau eine: 10 € Essens-/Platzgeld, 5 € Vereinsbeitrag (`GetOpenReminderBaseIDs` prüft bezahlte **und** unbezahlte bestehende Gebühren → Wiederholungsmahnung erzeugt keine zweite). Gebühren-Fälligkeit = Mailfrist. Kein SMTP konfiguriert → 503.
- Neuer Log-Eintrag: `email_type REMINDER_INITIAL/FINAL` (auch für reine Vereinsbeitrags-Sends), `household_id` gesetzt, Payload `{stage, runDate, deadline, householdId, feeIds, qrIncluded, remindersCreated[{baseFeeId, baseFeeType, amount}]}`.
- Neue Repository-Queries: `FeeRepository.ListOpenByHousehold` (offene Fees je Haushalt inkl. `matched_amount`, Typ `OpenFeeRow`), `GetOpenReminderBaseIDs`, `GetReminderBaseIDsCreatedAfter`; `HouseholdRepository.ListAll`.
- Router: statische Routen `/fees/reminder-cases` vor `/{id}` registriert (chi matcht statische Segmente vor Parametern, wie beim bestehenden `/overview`). OpenAPI (`swagger.yaml`, `openapi3.yaml`) und `schema.d.ts` via `scripts/generate-api.sh`-Pipeline regeneriert (lokal: `go install .../swag@latest`, `npx --registry=https://registry.npmjs.org/ swagger2openapi`, `npx openapi-typescript` — bun ist in dieser Umgebung nicht nötig).
- Integrationstests (`reminder_cases_integration_test.go`): Gruppierung/Scope/Statuswechsel (waiting → actionable_final nach Frist), Teilzahlungen, Mixed-Preview mit 10 €/5 €-Gebührenplanung, Send (Mail + Log + Gebühren) und Wiederholungssend ohne Doppelgebühr, Konflikte (fremd/bezahlt/doppelte/konkurrierende Gebühr nach Preview).

### E-Mail-Log-Historie: household_id, Stichtag, Backfill (2026-09-15)

- Migration `000030_email_log_household_history`: `fees.email_logs.household_id` (nullable FK auf `fees.households`, `ON DELETE SET NULL`, Index `idx_email_logs_household_id`).
- Backfill als wiederverwendbare SQL-Funktion `fees.backfill_email_log_households()` (wird von der Migration und von `EmailLogRepository.BackfillHouseholdIDs` genutzt): mappt Logs über `payload.feeIds` auf den Haushalt der referenzierten `fee_expectations`. Logs, deren Fee-IDs zu **keinem oder mehreren** Haushalten auflösen (oder keinen Payload haben), bleiben bewusst ohne household_id — sie existieren nur im globalen Versandverlauf. Wichtig: Postgres hat kein `MIN(uuid)`, daher `ARRAY_AGG(... ORDER BY ... NULLS LAST)[1]` nach `HAVING COUNT(DISTINCT household_id) = 1`.
- Zuverlässigkeits-Stichtag in `fees.app_settings` (`reminder_history_reliable_from`, UTC-Datum der Migration): Fees, die **bis** zum Stichtag ohne zuordenbaren Log angelegt wurden, gelten als `history_unknown`; danach angelegte ohne Log als zuverlässig `never_contacted`. Lesezugriff über `ReminderService.GetHistoryReliableFrom`.
- Neuer Service-Layer `reminder_history.go`: `ResolveFeeContacts(logs)` leitet je Fee den letzten Kontakt aus Reminder-Logs ab (neuester Log gewinnt; Stage + `runDate` aus dem Payload, Deadline daraus in Schritt 3); `ListFeeContacts(householdID)` lädt die Kontakte je Haushalt. `EmailLog`-Domain-Methoden `FeeIDsFromPayload()`/`StageFromPayload()` parsen den Payload tolerant.
- Neue Repository-Queries: `ListByHouseholdAndTypes` (Reminder-Logs je Haushalt, `pq.Array` für den Typen-Filter — lib/pq braucht den Wrapper für Slices) und `ListByHousehold` (Familien-Chronik). `Create`/`List` schreiben/lesen `household_id` mit; das Log-Payload-Format bleibt unverändert.
- Integrationstests (`email_log_history_integration_test.go`): Backfill (eindeutig/ambig/unauflösbar/ohne Payload), neue Logs tragen household_id, Kontakt-Auflösung (neuester gewinnt), Stichtag von der Migration gesetzt.

### Gemeinsamer Reminder-Kern (2026-09-15)

- Erster Schritt des familienbasierten Erinnerungs-Workflows: `ReminderService` ist jetzt der gemeinsame fachliche Kern für beide Beitragsarten-Familien. Aufbau: `reminder_service.go` (Service-Typ, `Run` für Essens-/Platzgeld inkl. Auto-Stufen, `RunMembership` für Vereinsbeiträge, Stage-Parsing, Settings), `reminder_core.go` (Scope-basierte Pipeline `runScope`: Fee-Auswahl, Mahngebühren-Erzeugung, Haushaltsgruppierung, Mailversand, Log), `reminder_email.go` (beide Mail-Text-Builders + Formatierung), `reminder_payment_qr.go` (unverändert).
- Der near-duplicate `MembershipReminderService` (**Datei gelöscht**) wurde in den Kern überführt: ein `reminderScope` beschreibt je Beitragsfamilie Fee-Types, Auswahl-Strategie (Monat vs. fällig-bis), Mahngebührenbetrag/Fälligkeit (10 € am 15. vs. 5 € am Laufenden), Mail-Builder und Log-Typen. Beide Legacy-Endpunkte `POST /fees/reminders/run` und `/fees/membership-reminders/run` behalten Request-/Response-Format und delegieren an denselben Kern; der Handler hält keinen zweiten Service mehr (`NewFeeHandler` ohne Membership-Parameter).
- **Verhaltensänderung Legacy:** Vereinsbeitrags-Mails verwenden jetzt als Standardfrist `runDate + 7 Tage` statt 31.03. des Beitragsjahres (`buildFamilyMembershipReminderEmail` nutzt `defaultReminderDeadline`). Ein manuell übergebener `deadline`-Query-Param überschreibt weiterhin. Die Mahngebühren-Fälligkeit für Vereinsbeiträge bleibt beim Laufdatum (Ende des Tages), die Mailfrist ist davon unabhängig.
- Log-Payload (`fees.email_logs.payload`) unverändert: `stage`, `runDate`, `unpaidCount`, `remindersCreated`, `feeIds` — für beide Scopes identisch erzeugt (`logScopeEmail`). Mail-Texte und Log-Typen (`REMINDER_*` vs. `MEMBERSHIP_REMINDER_*`) bleiben je Scope distinct.
- Alle bestehenden Service-/Integrationstests laufen unverändert grün; Membership-Mail-Tests auf 7-Tage-Frist umgestellt.

### Notizen zu Kindern (2026-09-14)

- Neue Tabelle `fees.child_notes` (Migration `000029_add_child_notes`): `id`, `child_id` (FK auf `fees.children` mit `ON DELETE CASCADE`), `text`, `created_at`, `updated_at` (Trigger `fees.update_updated_at_column()`). Kein Autor-/Owner-Feld — **jeder authentifizierte Benutzer darf alle Notizen bearbeiten und löschen**. Indizes: `(child_id, created_at DESC, id DESC)` für die Kindliste, `(created_at DESC, id DESC)` für die globale Liste.
- Endpunkte (Swagger-Tag `Notes`, Pagination wie überall `page`/`perPage` + `data/total/page/perPage/totalPages`): `POST`/`GET /children/{id}/notes`, `PUT`/`DELETE /children/{id}/notes/{noteId}`, `GET /notes` (globale Liste, inkl. `childName` über Join auf `fees.children`). Statuscodes: 201/200/204, 400 bei ungültiger UUID oder leerem/Whitespace-Text (Backend trimmt), 404 bei unbekanntem Kind/Notiz oder falscher Kind-Notiz-Zuordnung (Update/Delete gegen andere `childId` geben bewusst 404, kein 403).
- Service (`child_note_service.go`) validiert Kind-Existenz über eine schmale `ChildLookup`-Schnittstelle statt des ganzen `ChildRepository`; Repository sortiert stabil nach `created_at DESC, id DESC`. Notizen werden beim Löschen eines Kindes automatisch per Cascade entfernt. Tests: Service-Unit-Tests mit In-Memory-Fakes (`child_note_service_test.go`) + Integrationstests gegen Testcontainers (`child_note_integration_test.go`, Cleanup in `testutil_test.go` ergänzt).
- OpenAPI-Spec und `schema.d.ts` via `scripts/generate-api.sh` regeneriert. Hinweis: das Skript verschluckt sich aktuell am `bunx --registry …`-Aufruf (bunx parst den Flag als Paketnamen); `npx --registry=https://registry.npmjs.org/ swagger2openapi …` + `bun run generate:api` funktionieren.

### Beitragsregel Pflegekinder (Pflegefamilie) (2026-09-14)

- Pflegekinder (Haushalt `income_status = FOSTER_FAMILY`) werden **nicht** nach der einkommensabhängigen Elternbeitragsentlastung (§§ 50 ff. KitaG) behandelt — die Entlastungs-Bracket-Logik greift für sie nicht. Stattdessen gilt § 17 Abs. 1 KitaG: Beitrag = **Durchschnitt aller Satzungssätze** für die Betreuungsstunden (`calculateAverageSatzungRate`), Einkommen wird ignoriert, kein Geschwisterrabatt, `ShowEntlastung: false`. Code: `internal/service/fee_service.go:597-609`, Regeltext „Pflegefamilie (Durchschnittsbeitrag)".
- **Ab dem Monat der Vollendung des 3. Lebensjahres** greift die altersabhängige Beitragsfreiheit nach § 17a KitaG **ausdrücklich auch für Pflegekinder** (§§ 33, 34 SGB VIII). Die Altersprüfung steht in `CalculateChildcareFee` **vor** dem Pflegefamilie-Zweig (`fee_service.go:582-593`), d. h. jedes Kind ab dem Kindergartenalter bekommt unabhängig vom Haushaltsstatus `Beitragsfrei (ab 3 Jahren)` mit 0 €.
- Übergang praktisch handhaben: Folgeeinstufung mit wirksamem Monat = Geburtstagsmonat. Der Care-Type wird am wirksamen Monat bestimmt (`einstufung_service.go:91`), § 188 Abs. 2 BGB zählt den Tag vor dem 3. Geburtstag (`domain/child.go:61-78`), daher ist der Geburtstagsmonat selbst bereits beitragsfrei.
- Pflegefamilien-U3-Kinder zählen nicht in die Einkommens-Bracket-Statistik der Stichtagsmeldung (Frontend filtert sie separat, `DashboardPage.vue:825`).
- Kein Codebedarf: Die Implementierung entspricht dieser Rechtslage bereits. Referenzfall in der Testseeding-Doku (`docs/test-seeding.md` §4) ist U3-only — ein Seeding-Fall „Pflegekind wird 3" existiert nicht.

### Vereinsmitglieder-Nachtrag aus Vereinsliste (2026-09-04)

- Die vollständige Vereinsmitgliederliste des Vereins (33 Einträge inkl. Zuordnung „Vereinsmitglied → Kind/Kinder") wurde über die API nachgetragen. Weg war je Eintrag `POST /parents/{id}/member` (`ParentService.CreateMemberFromParent`): Mitglied erbt Kontaktdaten und Haushalt des gewählten Elternteils, `membership_start` = Eintrittsdatum des ältesten verknüpften Kindes, `households.membership_parent_id`/`membership_assignment_status=CONFIRMED` gesetzt.
- Neuer Nummernkreis M0009–M0041, erzeugt in Reihenfolge des ältesten Kindeintrittsdatums (Gleichstand alphabetisch). Bestand M0001–M0008 blieb unangetastet, außer: M0001 `firstName` „Karo" → „Karola" (Vereinsliste). Insgesamt 41 aktive Mitglieder.
- Mitgliederauswahl pro Familie folgte der Vereinsliste, bei den fünf vom Verein nachgereichten Familien: Gabriel→Nikolas, Waesch→Aileen Damer-Waesch, Omachil→Ewelina, Schulz→Anja, Kelle→Armin Schürer.
- Startdatum-Anomalien mit Service-Default belassen (ältestes Kind inkl. ausgetretener Geschwister): Melanie Zeck M0009 start 2020-07-01 (Maximilian, ausgetreten; Luca trat 2025-09-15 ein), Sandor Akszenovics M0018 start 2023-08-15 (Max, ausgetreten; Zoe 2024-08-20). Fink-Haushalt hat zwei Mitglieder (Pierre M0037, Anika M0039); `membership_parent_id` endet bei Anika.
- Namensabweichungen bewusst in DB-Schreibweise übernommen: „Annabell" Dörksen (Liste: Annabelle), „Anika" Fink (Liste: Annika), Kind „Leon Martin Fink" (Liste: Leon Marvin). Kein Datenfix am Kind.
- Nicht auf der Vereinsliste und deshalb unangetastet: M0004 Sarah Thränhardt, M0006 Christin Hebert, M0008 Andrea Fritze.
- Abdeckung nach Import: ohne Vereinsmitglied-Elternteil bleiben die Familien Strauss (Edgar), Kolenda (Clara), Jäckel (Alma), Babić (Alexander) — Mitgliedschaft vorerst unbekannt.
- Keine MEMBERSHIP-Gebühren für die neuen Mitglieder generiert; läuft über den normalen Jahresgenerierungsprozess.

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

### Notizen zu Kindern (2026-09-14)

- Neue Card „Notizen" in `ChildDetailPage.vue` (paginiert 5/Seite, Create/Edit im gemeinsamen Dialog mit Pflicht-Textarea, separater Löschdialog, ESC schließt erst Lösch- dann Notizdialog, Reload nach jeder Aktion, leere Seite nach Löschen springt auf die letzte gültige Seite, `toLocaleString('de-DE')`, „Bearbeitet"-Marker wenn `updatedAt` > `createdAt`).
- Neue globale Seite `NotesPage.vue` unter `/notizen` nach dem ChildrenPage-Pattern: Seitengrößen 10/25/50/100, gekürzte Textvorschau, Kindname als Link auf `/kinder/:id`, Erstellt-/Änderungszeit, Lade-/Fehler-/Leerzustand, responsive (Bearbeitet-Spalte `hidden md:table-cell`). Navigationseintrag „Notizen" (Lucide `NotebookPen`) unter „Verwaltung" im `MainLayout`.
- API-Zugriff über `api.getChildNotes/createChildNote/updateChildNote/deleteChildNote/getNotes` mit `normalizePaginated`; `ChildNote`-Typ aus dem generierten Schema abgeleitet, Requests handgeschrieben. ESLint ist im Workspace nicht installiert — Gate bleibt `bun run build` (vue-tsc + vite).

### Mitglieder-Stichtagsreport auf der Mitgliederseite (2026-09-09)

- `MembersPage` hat einen „Stichtagsreport"-Button im Header, der ein Modal im Stil des Dashboard-Stichtagsreports öffnet: Date-Input (Default heute) plus „Heute"-Button und drei Karten — Mitglieder am Stichtag, davon aktuell aktiv, davon inaktiv.
- Neuer Backend-Endpoint `GET /members/count?asOf=YYYY-MM-DD` (Handler `MemberHandler.CountAsOf`): zählt Mitglieder, deren Mitgliedschaftszeitraum (`membership_start` ≤ asOf und `membership_end` ≥ asOf bzw. NULL) den Stichtag umfasst — inklusive `is_active = false`-Mitgliedern, da Deaktivieren den Datensatz erhält. `asOf` fehlt → heute; Antworten: `{ asOf, total, active, inactive }`. Implementiert über das bestehende `MemberRepository.ListActiveAt`.
- OpenAPI-Spec und `schema.d.ts` regeneriert; Frontend-Zugriff über `api.getMemberCountAsOf(asOf)`.

### Kinder-Spalte in der Mitgliederliste (2026-09-09)

- `MembersPage` zeigt pro Mitglied die Kinder seines Haushalts in einer eigenen Spalte (komma-separierte Vor- und Nachnamen, „-" ohne Haushaltsverknüpfung). Hintergrund: die Vereinsmitglieder wurden aus der Vereinsliste mit Haushaltszuordnung importiert, die Zuordnung war in der Liste aber nicht sichtbar.
- Umsetzung rein im Frontend ohne Backend-/Spec-Änderung: nach dem Mitglieder-Laden werden für die DISTINCT `householdId`s der aktuellen Seite die Haushalte über `GET /households/{id}` nachgeladen (`Promise.allSettled`, Fehlschläge → leere Liste), gecacht in `childNamesByHousehold`. Bewusst N-Requests pro Seite statt Members-Endpoint-Anreicherung, um swag-Regeneration zu vermeiden; bei Wachstum des Per-Page-Limits den Members-List-Query direkt mit Kindnamen anreichern.

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
