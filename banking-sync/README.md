# Banking Sync Service

Automatische CSV-Exporte von der SozialBank via Browser-Automatisierung (Playwright) inkl. Import in das Fees-Backend.

## Wie der Flow exakt funktioniert

1. **Start** (manuell oder per Host-Cron)
2. **Playwright startet Chromium** mit persistentem Profil (`USER_DATA_DIR`)
3. **Login** im Online-Banking (ggf. 2FA beim ersten Mal)
4. **Navigation** zur Umsatzansicht
5. **Zeitraum** setzen (Standard: letzte 90 Tage)
6. **CSV exportieren** und im Download-Ordner speichern; der Browser-Download wird bei Fehlern automatisch erneut versucht
7. **Upload** der CSV per `multipart/form-data` an `POST /api/fees/v1/import/upload`
   - Auth via `X-Import-Token: ${CRON_API_TOKEN}`
8. **Push-Ping an Uptime Kuma** (wenn `UPTIME_KUMA_PUSH_URL` gesetzt ist)
   - Success: `status=up`
   - Fehler: `status=down`
9. **Container beendet sich** (bei Host-Cron Variante)

## Anforderungen

- **bun** (lokal)
- `CRON_API_TOKEN` muss im Backend gesetzt sein
- Backend erwartet einen System-User mit UUID `00000000-0000-0000-0000-000000000001` (Migration `000016_seed_import_user.*`)

## Umgebungsvariablen

| Variable | Pflicht | Default | Bedeutung |
|---|---|---|---|
| `BANK_URL` | optional | `https://www.sozialbank-onlinebanking.de/services_cloud/portal/` | Login URL |
| `BANK_USERNAME` | ja | - | NetKey/Username |
| `BANK_PASSWORD` | ja | - | Passwort |
| `API_URL` | optional | `http://localhost:8081/api/fees/v1` | Fees-API Base |
| `CRON_API_TOKEN` | ja | - | Import-Token für `/import/upload` |
| `UPTIME_KUMA_PUSH_URL` | optional | leer | Uptime Kuma Push-URL |
| `HEADLESS` | optional | `true` | Browser im Headless-Modus; `false` zeigt ihn an |
| `DOWNLOAD_DIR` | optional | `./output` | CSV Download-Ordner |
| `USER_DATA_DIR` | optional | `./profile` | Persistentes Browser-Profil |
| `USE_PERSISTENT_CONTEXT` | optional | `false` | Persistenten Browser-Kontext verwenden |
| `TWO_FA_TIMEOUT_SECONDS` | optional | `600` | Timeout für 2FA-Freigabe |
| `LOGIN_TIMEOUT_SECONDS` | optional | `30` | Timeout für Login-Elemente |
| `LOGIN_OUTCOME_TIMEOUT_SECONDS` | optional | `45` | Timeout zur Erkennung des Login-Ergebnisses |
| `WAIT_PROGRESS_INTERVAL_SECONDS` | optional | `10` | Intervall für Fortschrittsmeldungen beim Warten |
| `DEBUG_DIR` | optional | `./output/debug` | Debug-Screenshots und HTML-Snapshots; folgt `DOWNLOAD_DIR` |
| `USER_AGENT` | optional | Chrome 121 UA | User-Agent Override |
| `GLOBAL_TIMEOUT_SECONDS` | optional | `900` | Gesamttimeout für einen Browser-Sync-Versuch |
| `UPLOAD_TIMEOUT_SECONDS` | optional | `120` | Timeout für den Upload ans Fees-Backend |
| `DOWNLOAD_TIMEOUT_SECONDS` | optional | `120` | Timeout für das CSV-Download-Event nach Export-Klick |
| `DOWNLOAD_RETRY_ATTEMPTS` | optional | `3` | Anzahl Versuche; jeder startet eine neue Browser-Session |
| `DOWNLOAD_RETRY_DELAY_SECONDS` | optional | `30` | Wartezeit zwischen Bank-Download-Retries |
| `BROWSER_CLOSE_TIMEOUT_SECONDS` | optional | `20` | Timeout für das Schließen von Browser und Kontext |
| `UPTIME_KUMA_TIMEOUT_SECONDS` | optional | `10` | Timeout für den Uptime-Kuma-Push |
| `PORT` | optional | `3333` | Port für Runner-API |
| `SYNC_API_TOKEN` | optional | leer | Token für Runner-API (Header `X-Sync-Token`) |
| `STATE_DIR` | optional | `./state` | Status/Log-Ordner für Runner-API |
| `LOG_LINES` | optional | `200` | Anzahl Logzeilen im Status |
| `RUN_SYNC_TIMEOUT_SECONDS` | optional | `1200` | Sicherheits-Timeout für einen Runner-Sync-Lauf |
| `CSV_PATH` | optional | - | CSV-Dateipfad für `upload.js`, falls kein Pfadargument übergeben wird |

## Aktueller Betriebsstand

- Seit 2026-07-08 startet der Bank-CSV-Download bei Timeout oder anderen Browser-/Exportfehlern standardmäßig bis zu 3 Versuche. Das betrifft `sync.js` direkt und den Runner-API-Modus über `server.js`.
- Der Download-Event nach dem Export-Klick hat einen eigenen Timeout (`DOWNLOAD_TIMEOUT_SECONDS`, Default 120 Sekunden), damit Hänger nicht nur über den globalen Sync-Timeout beendet werden.
- Uptime Kuma wird per Push-URL mit `status=up` nach erfolgreichem Upload und mit `status=down` bei Sync-Fehlern benachrichtigt. Die Push-URL muss im Runtime-Environment wirklich an den Container durchgereicht sein; wenn sie fehlt, schreibt der Runner einmalig `UPTIME_KUMA_PUSH_URL not set`.
- Die Docker-Basis wurde von `oven/bun:1.1.29` auf `oven/bun:1` umgestellt, nachdem die Produktivlogs auf `infra-dev` mehrfach Bun-Segfaults des alten Runners gezeigt hatten.

## Lokal testen (sichtbar)

```bash
cd banking-sync
bun install
bunx playwright install chromium
HEADLESS=false BANK_USERNAME=... BANK_PASSWORD=... CRON_API_TOKEN=... bun sync.js --test
```

`--test` lädt die CSV herunter und zeigt einen Preview-Output, **ohne** Upload und **ohne** Uptime-Kuma-Ping.

## CSV später importieren (manueller Upload)

Wenn du eine CSV bereits im Download-Ordner hast, kannst du den Import später ausführen:

```bash
CRON_API_TOKEN=... bun upload.js --file ./output/sozialbank_2026-02-01_to_2026-05-01.csv
```

## Runner-Modus (API)

Für einen UI-Trigger kann der Sync als kleiner HTTP-Runner laufen:

```bash
cd banking-sync
SYNC_API_TOKEN=... BANK_USERNAME=... BANK_PASSWORD=... CRON_API_TOKEN=... UPTIME_KUMA_PUSH_URL=... bun server.js
```

Endpoints:
- `POST /run` (startet Sync; Header `X-Sync-Token`)
- `GET /status` (Status + Logs vom letzten Lauf; Header `X-Sync-Token`)
- `GET /health` (ohne Auth)

## Docker (Run-Once)

Compose-Beispiele: [`docker-compose.example.yml`](docker-compose.example.yml).

### Run-Once ausführen

```bash
docker compose --profile banking-sync run --rm banking-sync
```

## Scheduler (ressourcensparend)

**Empfohlen:** Host-Cron. Container läuft nur während des Jobs.

```bash
# Server-Zeitzone: Europe/Berlin
0 6 * * * cd /opt/kita-apps && docker compose --profile banking-sync run --rm banking-sync
```

## Troubleshooting

- **Timeout bei Login:** Selektoren in `sync.js` per Playwright Codegen anpassen
- **Timeout/Fehler beim CSV-Export:** `DOWNLOAD_RETRY_ATTEMPTS`, `DOWNLOAD_RETRY_DELAY_SECONDS` und `DOWNLOAD_TIMEOUT_SECONDS` prüfen. Jeder Retry startet eine neue Browser-Session.
- **2FA hängt:** ersten Run mit `HEADLESS=false`, danach profiliertes Login nutzen
- **Upload 401:** `CRON_API_TOKEN` prüfen (Backend + Container müssen identisch sein)
- **Kein Kuma-Ping:** `UPTIME_KUMA_PUSH_URL` setzen und in Compose an den Runner durchreichen (sonst nur einmalige Startup-Warnung)

## Sicherheit

- Keine Credentials im Code speichern
- `CRON_API_TOKEN` wie ein Passwort behandeln
