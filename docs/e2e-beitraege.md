# E2E-Tests (Beiträge, Playwright)

Suite: `frontend/e2e/beitraege/`, Konfiguration `frontend/playwright.config.ts`. Läuft lokal mit einem Befehl und in CI (`ci.yml`, Job `e2e-beitraege`); ein roter Lauf verhindert den Image-Build.

```bash
cd frontend
bunx playwright install chromium   # einmalig
bun run test:e2e                   # headless
bun run test:e2e:ui                # interaktiv
bunx playwright test auth.spec.ts  # einzelne Datei
bun run test:e2e:report            # letzter HTML-Report
```

Voraussetzungen: Docker, Go, Bun. Keine laufende Dev-DB und kein Backend nötig; Ports `18081` (App) und `55432` (DB) müssen frei sein (`E2E_PORT`, `E2E_DB_PORT`).

## Wie der Stack entsteht

Playwright startet als `webServer` das Skript `frontend/e2e/start-beitraege-stack.sh`:

1. PostgreSQL 16 als Wegwerf-Container `kita-e2e-db-<port>`, bei jedem Lauf leer.
2. Frontend per Vite gebaut und mit `-tags embed_frontend` in `backend-fees` eingebettet, wie im Docker-Image. Gleiche Origin und gleiche Cookie-Pfade wie in Produktion, kein Vite-Proxy.
3. Migrationen, dann Backend auf `127.0.0.1:18081`. Der Admin wird aus `USER_NAME`/`USER_PASSWORD` gebootstrappt (Standard `admin@e2e.test` / `e2e-admin-password`, siehe `e2e/beitraege/env.ts`). `CRON_API_TOKEN` ist `e2e-import-token` (`E2E_IMPORT_TOKEN`), für den Upload-Pfad von banking-sync.

Nach dem Lauf beendet Playwright das Skript per SIGTERM, das Skript entfernt den Container. Backend-Log: `frontend/test-results/e2e-backend.log` (in CI bei Fehlern als Artefakt `playwright-report` hochgeladen, zusammen mit Traces und Screenshots).

## Regeln für Tests

- Jeder Test legt seine Daten selbst an, mit eindeutigen Namen (`uniq()`), damit Tests parallel laufen. Kein Seed, keine Aufräum-Skripte.
- Anmelden über die Fixtures (`adminPage`, `loginContext`) statt über gespeicherten `storageState`: Das Refresh-Token rotiert, ein gespeichertes Cookie ist nach der ersten Nutzung ungültig.
- Daten, die nicht Gegenstand des Tests sind, per `adminApi` über die echte API anlegen; den zu prüfenden Ablauf über die Oberfläche.
- Selektoren über Rolle und Beschriftung (`getByRole`, `getByLabel`). Fehlt einem Element ein zugänglicher Name, im Frontend ergänzen (z. B. `role="dialog"` + `aria-label`, Benutzermenü `aria-label="Benutzermenü"`).
- Bank-CSVs mit `bankCsv()` aus `bank.ts` und je Test eigener Zahler-IBAN (`uniqueIban()`): Eine automatisch zugeordnete Zahlung macht ihre IBAN für dieses Kind vertrauenswürdig, eine geteilte IBAN leitet Zahlungen anderer Tests dorthin um.
- Fehlversuche beim Login zählen gegen die Login-Bremse (20 pro IP in 15 Minuten, alle Tests teilen `127.0.0.1`). Tests für falsche Passwörter nutzen eigene Konten (`createUser`) und bleiben sparsam.

## Abgedeckt

- `auth.spec.ts`: falsches Passwort, Session im httpOnly-Cookie (kein Token im `localStorage`, Cookie-Attribute), Reload, Logout, Deep-Link-Redirect, Login-Bremse.
- `users.spec.ts`: Benutzer anlegen, Nicht-Admin ohne Zugriff auf „Benutzer“, eigenes Passwort ändern (andere Sitzungen enden), Deaktivieren, Selbstschutz.
- `fees-flow.spec.ts`: Kind und Elternteil anlegen, Monatsbeiträge generieren, Bank-CSV hochladen, automatische Zuordnung, Beitrag erscheint als bezahlt.
- `banking-sync-import.spec.ts`: der tägliche Upload von banking-sync, ohne Login, nur `X-Import-Token` und Multipart-Feld `file` wie `banking-sync/upload.js`/`sync.js`: falsches Token → 401, Zahlung wird automatisch zugeordnet, erneuter Upload derselben Datei wird übersprungen.

## Legacy

`playwright.management.config.ts` (`bun run test:management`) enthält die alten Tests für die pausierten Apps Dienstplan/Zeiterfassung. Nicht gepflegt, nicht in CI, erwartet ein manuell vorbereitetes `backend-management`. Die frühere Beiträge-Suite gegen eine vorbefüllte DB ist entfernt (Beschreibung: `docs/archive/test-seeding.md`).
