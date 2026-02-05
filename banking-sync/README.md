# Banking Sync Service

Automatische CSV-Exporte von der SozialBank via Browser-Automatisierung (Playwright) inkl. Import in das Fees-Backend.

## Wie der Flow exakt funktioniert

1. **Start** (manuell oder per Host-Cron)
2. **Playwright startet Chromium** mit persistentem Profil (`USER_DATA_DIR`)
3. **Login** im Online-Banking (ggf. 2FA beim ersten Mal)
4. **Navigation** zur Umsatzansicht
5. **Zeitraum** setzen (Standard: letzte 90 Tage)
6. **CSV exportieren** und im Download-Ordner speichern
7. **Upload** der CSV per `multipart/form-data` an `POST /api/fees/v1/import/upload`
   - Auth via `X-Import-Token: ${CRON_API_TOKEN}`
8. **Container beendet sich** (bei Host-Cron Variante)

## Anforderungen

- **bun** (lokal)
- `CRON_API_TOKEN` muss im Backend gesetzt sein
- Backend erwartet einen System-User mit UUID `00000000-0000-0000-0000-000000000001` (Migration `000016_seed_import_user.*`)

## Umgebungsvariablen

| Variable | Pflicht | Default | Bedeutung |
|---|---|---|---|
| `BANK_URL` | optional | SozialBank Portal URL | Login URL |
| `BANK_USERNAME` | ja | - | NetKey/Username |
| `BANK_PASSWORD` | ja | - | Passwort |
| `API_URL` | optional | `http://localhost:8081/api/fees/v1` | Fees-API Base |
| `CRON_API_TOKEN` | ja | - | Import-Token für `/import/upload` |
| `USER_DATA_DIR` | optional | `./profile` | Persistentes Browser-Profil |
| `DOWNLOAD_DIR` | optional | `./output` | CSV Download-Ordner |
| `DATE_RANGE_DAYS` | optional | `90` | Zeitraum (Tage) |
| `HEADLESS` | optional | `true` | Browser sichtbar machen |
| `TWO_FA_TIMEOUT_SECONDS` | optional | `600` | Timeout für 2FA-Freigabe |
| `PORT` | optional | `3333` | Port für Runner-API |
| `SYNC_API_TOKEN` | optional | - | Token für Runner-API (Header `X-Sync-Token`) |
| `STATE_DIR` | optional | `./state` | Status/Log-Ordner für Runner-API |
| `LOG_LINES` | optional | `200` | Anzahl Logzeilen im Status |
| `SCREENSHOT_DIR` | optional | `./output` | Ordner für Debug-Screenshots |
| `DEBUG_SCREENSHOTS` | optional | `false` | Immer einen Screenshot/HTML Snapshot beim Login erzeugen |
| `USER_AGENT` | optional | Chrome UA | User-Agent Override (Anti-Bot) |

## Lokal testen (sichtbar)

```bash
cd banking-sync
bun install
bunx playwright install chromium
HEADLESS=false BANK_USERNAME=... BANK_PASSWORD=... CRON_API_TOKEN=... bun sync.js --test
```

`--test` lädt die CSV herunter und zeigt einen Preview-Output, **ohne** Upload.

## CSV später importieren (manueller Upload)

Wenn du eine CSV bereits im Download-Ordner hast, kannst du den Import später ausführen:

```bash
CRON_API_TOKEN=... bun upload.js --file ./output/sozialbank_2026-02-01_to_2026-05-01.csv
```

## Runner-Modus (API)

Für einen UI-Trigger kann der Sync als kleiner HTTP-Runner laufen:

```bash
cd banking-sync
SYNC_API_TOKEN=... BANK_USERNAME=... BANK_PASSWORD=... CRON_API_TOKEN=... bun server.js
```

Endpoints:
- `POST /run` (startet Sync; Header `X-Sync-Token`)
- `GET /status` (Status + letzte Logs; Header `X-Sync-Token`)
- `GET /health` (ohne Auth)

## Docker (Run-Once)

`docker-compose` Service ist auf Run-Once ausgelegt. Beispiel (siehe auch `docker-compose.integration.yml`):

```yaml
  banking-sync:
    build:
      context: ../banking-sync
      dockerfile: Dockerfile
    container_name: kita-banking-sync
    environment:
      BANK_URL: https://banking.sozialbank.de
      BANK_USERNAME: ${BANK_USERNAME}
      BANK_PASSWORD: ${BANK_PASSWORD}
      API_URL: http://backend-fees:8081/api/fees/v1
      CRON_API_TOKEN: ${CRON_API_TOKEN}
      USER_DATA_DIR: /data/profile
      DOWNLOAD_DIR: /data/downloads
      DATE_RANGE_DAYS: "90"
      HEADLESS: "true"
    volumes:
      - banking_sync_data:/data
    depends_on:
      - backend-fees
    profiles: ["banking-sync"]

volumes:
  banking_sync_data:
```

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
- **2FA hängt:** ersten Run mit `HEADLESS=false`, danach profiliertes Login nutzen
- **Upload 401:** `CRON_API_TOKEN` prüfen (Backend + Container müssen identisch sein)

## Sicherheit

- Keine Credentials im Code speichern
- `CRON_API_TOKEN` wie ein Passwort behandeln
