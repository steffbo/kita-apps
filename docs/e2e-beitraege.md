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

1. PostgreSQL 16 als Wegwerf-Container `kita-e2e-db-<port>`, bei jedem Lauf leer. Bereitschaft wird über TCP geprüft (`pg_isready -h 127.0.0.1`): Beim ersten Start meldet der Init-Server des Images schon über den Socket „bereit“ und startet danach neu, die Migration bekam dann „connection reset by peer“ (CI-Lauf vom 2026-10-30).
2. Frontend per Vite gebaut und mit `-tags embed_frontend` in `backend-fees` eingebettet, wie im Docker-Image. Gleiche Origin und gleiche Cookie-Pfade wie in Produktion, kein Vite-Proxy.
3. Migrationen, dann Backend auf `127.0.0.1:18081`. Der Admin wird aus `USER_NAME`/`USER_PASSWORD` gebootstrappt (Standard `admin@e2e.test` / `e2e-admin-password`, siehe `e2e/beitraege/env.ts`). `CRON_API_TOKEN` ist `e2e-import-token` (`E2E_IMPORT_TOKEN`), für den Upload-Pfad von banking-sync.

Nach dem Lauf beendet Playwright das Skript per SIGTERM, das Skript entfernt den Container. Backend-Log: `frontend/test-results/e2e-backend.log` (in CI bei Fehlern als Artefakt `playwright-report` hochgeladen, zusammen mit Traces und Screenshots).

## Regeln für Tests

- Jeder Test legt seine Daten selbst an, mit eindeutigen Namen (`uniq()`), damit Tests parallel laufen. Kein Seed, keine Aufräum-Skripte.
- Anmelden über die Fixtures (`adminPage`, `loginContext`) statt über gespeicherten `storageState`: Das Refresh-Token rotiert, ein gespeichertes Cookie ist nach der ersten Nutzung ungültig.
- Daten, die nicht Gegenstand des Tests sind, per `adminApi` über die echte API anlegen; den zu prüfenden Ablauf über die Oberfläche.
- Selektoren über Rolle und Beschriftung (`getByRole`, `getByLabel`). Fehlt einem Element ein zugänglicher Name, im Frontend ergänzen (z. B. `role="dialog"` + `aria-label`, Benutzermenü `aria-label="Benutzermenü"`).
- Bank-CSVs mit `bankCsv()` aus `bank.ts` und je Test eigener Zahler-IBAN (`uniqueIban()`): Eine automatisch zugeordnete Zahlung macht ihre IBAN für dieses Kind vertrauenswürdig, eine geteilte IBAN leitet Zahlungen anderer Tests dorthin um.
- Mitgliedsnummern sind `varchar(10)`: nicht aus `uniq()` bauen, sondern kurze Zufallszahlen verwenden (siehe `parent-work.spec.ts`).
- Fehlversuche beim Login zählen gegen die Login-Bremse (20 pro IP in 15 Minuten, alle Tests teilen `127.0.0.1`). Tests für falsche Passwörter nutzen eigene Konten (`createUser`) und bleiben sparsam.

## Abgedeckt

- `auth.spec.ts`: falsches Passwort, Session im httpOnly-Cookie (kein Token im `localStorage`, Cookie-Attribute), Reload, Logout, Deep-Link-Redirect, Login-Bremse.
- `users.spec.ts` (Mehrfachrollen): Konto mit zwei Rollen startet in der höchsten, wechselt im Benutzermenü und behält die Rolle über Reload und neuen Login; nicht zugewiesene Rolle ergibt 400.
- `board.spec.ts`: Vorstand sieht Beitragsbereich und Lese-Hinweis ohne Admin-Seiten, Meldungen, Änderungen, Regeln und Vorstandsämter ohne Schreib-Bedienelemente, liest per API, jeder Schreibzugriff 403 (Berechnungen erlaubt); `USER` ist nicht zuweisbar.
- `users.spec.ts`: Benutzer anlegen, Nicht-Admin ohne Zugriff auf „Benutzer“, eigenes Passwort ändern (andere Sitzungen enden), Deaktivieren, Selbstschutz, Liste filtert nach Suche und Rolle.
- `fees-flow.spec.ts`: Kind und Elternteil anlegen, Monatsbeiträge generieren, Bank-CSV hochladen, automatische Zuordnung, Beitrag erscheint als bezahlt.
- `einstufung.spec.ts`: neue Einstufung, Kind per Suche (Mindestlänge, Mehrwortsuche), Vorbelegung von Einkommen und Kinderzahl aus der Einstufung des Geschwisterkinds, Speichern; zweite offene Einstufung für dasselbe Kind gesperrt mit Link zur Folgeeinstufung.
- `banking-sync-import.spec.ts`: der tägliche Upload von banking-sync, ohne Login, nur `X-Import-Token` und Multipart-Feld `file` wie `banking-sync/upload.js`/`sync.js`: falsches Token → 401, Zahlung wird automatisch zugeordnet, erneuter Upload derselben Datei wird übersprungen.
- `parent-portal.spec.ts`: zwei Familien, Eltern-Routing und Datentrennung, Stundenmeldung und Freigabe
  (mit Prüfer), Telefon-Audit im Admin-Feed, Telefon-Änderung am zweiten Elternteil, nur lesbare
  Kinderdaten sowie Fehlermeldung mit
  Antwort, Haken im Feed und Anzeige der Antwort in der Eltern-Übersicht.
- `navigation.spec.ts`: Seitenleiste mit Aufgaben-Reitern und `aria-current`, Suche per Strg+K bis zur Kinder-Detailseite; Handy-Viewport: Tab-Leiste, „Mehr“-Sheet, Konto-Sheet mit Design-Umschalter, Esc.
- `parent-work.spec.ts` (Elternstunden): Rolle `PARENT_WORK` sieht nur den Elternstunden-Bereich (ohne Vorstand, mit Verwaltung) und bekommt 403 auf `/children` und `/parent-work/board-terms`; Eintrag erfassen senkt „Offen“, Stornieren stellt es wieder her; Vorstandsamt befreit die Familie; Vorstandstabelle filtert (Suche, aktiv/nicht aktiv); CSV-Vorschau (Seite Verwaltung) ordnet über das Kind zu und importiert einen Eintrag; CSV-Import mit abweichenden Spaltennamen (manuelle Zuordnung, ignorierte Spalten), Drag and Drop und Sammelwarnung für nicht zuordenbare Familien; Zeilen-Button öffnet den Dialog mit vorgewählter Familie (fest, ohne Suche); in der Suche gewählte Familie bleibt fest; Karten filtern die Tabelle; Sortierung (deckt `useTableSort`/`SortTh` für alle Tabellen ab); Konto-Karten der Detailseite (Fehlbetrag nur für Admin und Vorstand, Amtszeiten nur mit Amtszeit).
- `installments.spec.ts` (Ratenzahlungen): Zahlung eines unbekannten Zahlers wird über den Bankabgleich zur ersten Zahlung eines neuen Plans (Vorbelegung, Vertrag hochladen, Schlussrate); IBAN-Vorschlag und Zuordnung zu bestehendem Plan; Fall in den Erinnerungen, ohne Adresse kein Versand; `USER` sieht weder Pläne noch Schuldnernamen, 403 auf Aufheben.
- `leitung.spec.ts` (Rolle `LEITUNG`): eigenes Dashboard mit Stichtagsmeldung (Filter U3/Ü3, Stichtag zurückblättern, Warnung bei fehlender Adresse, „Listen drucken“; API ordnet ein Kind als zurückgestellt bzw. letztes Kita-Jahr ein, meldet die Adresse als (un)vollständig, liefert die Kinderlisten als PDF, `stats`/`report`/`children` 403), Navigation nur Start, Stammdaten und Elternstunden-Übersicht, Beitragsseiten leiten auf `/` um, Kinderseite ohne Einkommen/Beiträge/Notizen; API ohne Einkommensfelder, 403 auf Beitrags-, Bank-, Notiz- und Löschrouten, Einkommen in Anfragen wird ignoriert; Elternstunden: Verwaltung leitet auf die Übersicht um, Familienseite ohne Soll- und Eintragsaktionen, Stunden erfassen klappt, 403 auf Bearbeiten, Stornieren, Bestätigen, Soll und Import.
