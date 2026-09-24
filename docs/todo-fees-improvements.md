# Beiträge: Verbesserungs-Backlog

Ergebnis des Code-Reviews vom 2026-09-23 (`backend-fees` + `frontend/apps/beitraege`).
Wird **von oben nach unten** abgearbeitet. Jeder Schritt ist ein eigener Commit (inkl. Doc-Update
in `docs/status-fees.md`). Erledigte Punkte abhaken und Commit-Hash dahinterschreiben.

Leitplanke für alle Refactorings: **keine fachliche Verhaltensänderung**, außer wo explizit genannt.

## Phase 1 – Quick Wins & Absicherung

- [x] **1. CI: Tests vor Image-Build** (`a725ddc`) – eigener Workflow `ci.yml` (auf PR + `main`):
      `gofmt -l`, `go vet`, `go test ./...` (Integrationstests nutzen testcontainers, Docker ist auf
      GitHub-Runnern vorhanden) für `backend-fees`, `vue-tsc` für `beitraege`. `build-images.yml`
      hängt davon ab (`needs`). Erwartung: +2–4 min zusätzlich zum heutigen Build (~2–2,5 min).
- [x] **2. gofmt** auf die 10 unformatierten Dateien anwenden (reiner Format-Commit). (`49b48d4`)
- [x] **3. tsconfig TS6310** – `tsconfig.node.json` so anpassen, dass `vue-tsc -b` ohne Fehler läuft. (`d2952ab`)
- [x] **4. Import-Token härten** (`804fe6e`, zusammen mit #5) – `CRON_API_TOKEN` (genutzt vom banking-sync-Container,
      Header `X-Import-Token`) über `config` laden statt `os.Getenv`, Vergleich mit
      `subtle.ConstantTimeCompare`.
- [x] **5. Produktions-Defaults absichern** – umgesetzt als `config.Harden()` (zufälliges Secret + Warnung statt Abbruch, siehe status-fees). Ursprünglich: Start abbrechen, wenn `JWT_SECRET` der Dev-Default ist
      oder `CORS_ORIGINS=*` mit Credentials aktiv ist (außer im Dev-Modus).
- [x] **6. Request-Limits** (`ed28648`) – `http.MaxBytesReader` zentral: 1 MB für JSON-Bodies
      (`request.Decode`), 5 MB für CSV-Uploads (Bank- und Kinder-Import).
      Fehler von `ParseMultipartForm` in `import_handler.go` behandeln.
- [x] **7. Zeitzone Europe/Berlin** (`326112e`) – `TZ=Europe/Berlin` + `tzdata` im Image, zusätzlich im Code
      eine zentrale Location/Clock (`util.Now()`, `util.Berlin`), alle `time.Now()` in
      Service/Domain darauf umstellen. Tests für Monatsgrenzen (31.→1., 23:30 UTC).

## Phase 2 – Import robust machen

- [x] **8. Transaktions-Infrastruktur** (`db93ef5`) – `repository.TxManager` mit
      `WithTx(ctx, func(ctx) error)`; die Tx liegt im Context, Repos holen sich per
      `r.q(ctx)` entweder Tx oder DB (`sqlx.ExtContext`). Bestehende Repos bleiben API-kompatibel.
- [x] **9. Transaktionen im Matching** (`7f4f931`) – `AllocateTransaction`, `UnmatchTransaction`,
      `ConfirmMatches`, `CreateManualMatch`, `ResolveWarningWithLateFee`, `DismissTransaction`
      jeweils atomar. `ProcessCSV`: pro Buchung eine Tx (Transaktion + Match + Warnung).
- [x] **10. Fehler sichtbar machen** (`4326420`) – verschluckte Fehler in `import_service.go` beseitigen:
      Kinderliste/Blacklist nicht ladbar → Import bricht mit Fehler ab; Fehler je Buchung landen
      in `ImportResult.Errors` (statt stumm `Skipped`). Hartes Limit von 1000 Kindern entfernen.
      Frontend (ImportPage, BankingSyncCard, Import-Historie) zeigt Fehler klar an.
      banking-sync meldet Upload-Fehler weiterhin über Exit-Code.
- [x] **11. Dedupe per Unique-Index** (`c57e7ac`) (niedrige Priorität, Import läuft praktisch nur automatisch) –
      `dedupe_hash` + `ON CONFLICT DO NOTHING`. Vorher Live-Daten read-only auf echte Duplikate prüfen.

## Phase 3 – Fachliche Absicherung

- [x] **12. Konfigurierbare Beitragstabellen** (`10f565e`) – Tabellen + Einkommensgrenzen aus
      `domain/childcare_fee.go` in die DB (`fees.fee_schedules` mit `valid_from`), Seed = heutige
      Werte. Berechnung wählt die Tabelle nach Stichtag. Eigener Bereich „Beitragsordnung" im
      Frontend (anzeigen, neue Version ab Datum anlegen; alte Versionen read-only).
      Tests: Ergebnisse vor/nach Migration identisch.
- [x] **13. Geldbeträge als Cent** (`b5f71aa`) – umgesetzt als Cent-Arithmetik in Go bei unveränderten Structs/API (siehe status-fees). Ursprünglich: DB bleibt `NUMERIC(…,2)` (ist bereits exakt, **keine
      Datenmigration nötig**). In Go ein Typ `Cents int64` mit `Scan`/`Value`
      (NUMERIC ↔ Cent) und JSON-Ausgabe weiterhin als Euro-Zahl, damit die API unverändert bleibt.
      Epsilon-Vergleiche (`0.01`) entfallen. Schrittweise: zuerst Matching/Allocation, dann Rest.

## Phase 4 – Struktur (ohne Logikänderung)

- [x] **14. Handler entschlacken** (`e22ac6f`, `9a3725c`) – Repo-Zugriffe aus `ChildHandler`/`FeeHandler` in Services
      verschieben; `fee_handler.go`/`import_handler.go` aufteilen.
- [x] **15. `import_service.go` aufteilen** (`0bd55f0`) – `matcher`, `iban_registry`, `warning_service`,
      CSV-Import. Reines Verschieben, Tests müssen unverändert grün bleiben.
- [x] **16. Frontend-Formatierer zentralisieren** (`9d34340`) – `src/utils/format.ts`
      (`formatDate`, `formatDateTime`, `formatCurrency`, `formatDateForInput`, …), alle Seiten umstellen.
- [x] **17. Große Seiten zerlegen** (`a3d65a2`, `d9529f3`, `bed8442`) – `ChildDetailPage.vue` (3076 Z.), `ImportPage.vue`,
      `ChildImportPage.vue`, `FeesPage.vue`, `AutomationPage.vue` in Komponenten + Composables.
- [ ] **18. OpenAPI `required`** – Pflichtfelder in swag-DTOs markieren, Spec + `schema.d.ts`
      neu generieren, `DeepStrict`/`Loose` in `types.ts` abbauen.
- [ ] **19. Handler-/Middleware-Tests** – `httptest` für Auth, `RequireRole`, Import-Token,
      Body-Limits, zentrale Fehlerpfade.

## Phase 5 – Mehrbenutzer

- [ ] **20. Benutzerverwaltung** – Tabelle `fees.users` (wieder) einführen, bcrypt-Hashes in der DB,
      Bootstrap-Admin aus `USER_NAME`/`USER_PASSWORD` nur beim ersten Start. Passwortänderung
      persistent (behebt heutigen Bug: Änderung geht beim Neustart verloren). Admin-UI zum
      Anlegen/Deaktivieren. Rollen vorerst `ADMIN`/`USER`. Voraussetzung für einen Agent-Service-Account.

## Später

- [ ] **21. Tokens aus `localStorage`** – Refresh-Token als httpOnly-Cookie.
