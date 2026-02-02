# Banking Sync Service

Automatische CSV-Exporte von der SozialBank via Browser-Automatisierung.

## Konzept

Da die SozialBank EBICS statt FinTS/HBCI unterstützt, nutzen wir Playwright für Browser-Automatisierung:

1. **Headless Chrome** loggt sich bei sozialbank.de ein
2. **Exportiert** CSV der letzten 90 Tage
3. **Sendet** Daten an das Backend API
4. **Läuft** per Cron (z.B. täglich 6:00 Uhr)

## Ressourcen

- **RAM**: ~400MB Spitze beim Sync, sonst 0 (Container stoppt nach Sync)
- **CPU**: Moderate Spitze beim Login/Download
- **Storage**: ~1.2GB für Chromium-Image
- **Netzwerk**: Nur beim Sync aktiv

## Deployment

### Option A: Sidecar Container (empfohlen)

Container läuft im gleichen Compose, startet nur für den Sync:

```yaml
# In docker-compose.prod.yml hinzufügen:

  banking-sync:
    build:
      context: ../banking-sync
      dockerfile: Dockerfile
    container_name: kita-banking-sync
    # Wichtig: Kein restart! Container stoppt nach Sync
    environment:
      BANK_URL: https://banking.sozialbank.de
      BANK_USERNAME: ${BANK_USERNAME}
      BANK_PASSWORD: ${BANK_PASSWORD}
      API_URL: http://backend:8080/api/fees/v1
      API_TOKEN: ${CRON_API_TOKEN}
      SYNC_SCHEDULE: "0 6 * * *"  # Täglich 6:00
    depends_on:
      - backend
    profiles: ["banking-sync"]  # Nur bei explizitem Start
```

### Option B: Externer Cron-Service

 separater VPS/Raspberry Pi:
- Weniger Ressourcen auf dem Hauptserver
- Einfacher zu warten
- Kann auch externer Service wie GitHub Actions sein (kostenlos!)

## Files

- `Dockerfile` - Playwright + Node.js
- `sync.js` - Das Automatisierungs-Script
- `entrypoint.sh` - Cron-Setup

## Setup

1. `.env` erstellen:
```bash
BANK_USERNAME=dein-netkey
BANK_PASSWORD=dein-passwort
CRON_API_TOKEN=secure-random-token
```

2. Erstes Mal manuell testen:
```bash
docker compose run --rm banking-sync node sync.js --test
```

3. Cron aktivieren:
```bash
docker compose --profile banking-sync up -d
```

## Vorteile

✅ Funktioniert mit **jedem** Online-Banking (auch EBICS)  
✅ Keine extra Bank-Verträge nötig  
✅ Gleiche Login-Daten wie manuell  
✅ Container stoppt nach Sync (keine Ressourcen-Last)  

## Nachteile

⚠️ 2FA beim ersten Setup nötig (SecureGo Plus App)  
⚠️ Wenn Bank UI ändert, bricht es (selten)  
⚠️ ~1.2GB zusätzliches Docker Image  

## Alternative: FinTS Fallback

Falls du eine andere Bank mit FinTS hast, kannst du das Go-Backend nutzen:
- Entferne/comment den `banking-sync` Service
- Nutze die FinTS-Integration direkt im Backend
