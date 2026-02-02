# Playwright Codegen Anleitung

## Schritt 1: Playwright installieren

```bash
cd banking-sync
npm install
npx playwright install chromium
```

## Schritt 2: Code aufnehmen

```bash
# Headless=false damit du siehst was passiert
npx playwright codegen --target=javascript --browser=chromium \
  --viewport-size=1280,720 \
  https://www.sozialbank-onlinebanking.de/services_cloud/portal/
```

**Was du im Browser machst:**
1. **Login**: Username (NetKey) + Passwort eingeben
2. **2FA**: SecureGo Plus App bestätigen (wenn nötig)
3. **Navigation**: Menü → Umsatzanzeige (oder "Kontoumsätze")
4. **Zeitraum**: Letzte 90 Tage auswählen
5. **Export**: CSV-Export Button klicken
6. **Speichern**: Datei speichern

## Schritt 3: Code kopieren

Playwright Inspector zeigt den generierten Code. Wichtige Teile:

```javascript
// BEISPIEL - wird durch deinen generierten Code ersetzt:
await page.goto('https://www.sozialbank-onlinebanking.de/...');
await page.fill('input[name="login"]', 'DE123456789'); // Dein NetKey
await page.fill('input[name="password"]', 'dein-passwort');
await page.click('button[type="submit"]');
// ... 2FA handling ...
await page.click('text=Umsätze');
await page.fill('input[name="von"]', '01.01.2024');
await page.fill('input[name="bis"]', '31.03.2024');
await page.click('button:has-text("CSV")');
```

## Schritt 4: In sync.js einfügen

1. Öffne `sync.js`
2. Ersetze die `downloadCSV()` Funktion mit deinem generierten Code
3. **Wichtig**: Nutze `CONFIG.username` und `CONFIG.password` statt hardcoded Werten!

## Schritt 5: Anpassungen

### 2FA Handling
Falls 2FA (SecureGo Plus) nötig ist:

```javascript
// Warte auf 2FA Seite
const is2FA = await page.$('text=SecureGo');
if (is2FA) {
  console.log('⚠️ Bitte in SecureGo Plus bestätigen...');
  // Warte bis weitergeleitet (Timeout 2 Minuten)
  await page.waitForSelector('text=Willkommen', { timeout: 120000 });
}
```

### Datei-Upload statt Download
Falls du nicht direkt downloaden kannst:

```javascript
// Alternative: Lese aus LocalStorage oder API
const csvData = await page.evaluate(() => {
  // Manche Banken legen CSV im LocalStorage ab
  return localStorage.getItem('exportData');
});
```

### Headless-Modus testen
Nachdem es im sichtbaren Browser klappt:

```bash
# Teste im headless Container
HEADLESS=true node sync.js --test
```

## Schritt 6: Sicherheit

**Niemals** Credentials im Code speichern! Nutze Environment Variables:

```javascript
// ❌ FALSCH:
await page.fill('#username', 'DE123456789');

// ✅ RICHTIG:
await page.fill('#username', CONFIG.username);
```

## Troubleshooting

### "Timeout waiting for selector"
- Bank-Seite hat sich geändert
- Netzwerk zu langsam
- Lösung: Timeout erhöhen oder Selektor anpassen

### "2FA wird nicht erkannt"
- SecureGo Plus pop-up kann Playwright blockieren
- Lösung: Headless=false für ersten Login, dann Session speichern

### "Download funktioniert nicht"
- Manche Banken blockieren automatische Downloads
- Lösung: `acceptDownloads: true` im Context
- Alternative: CSV über API/LocalStorage holen

## Beispiel: Vollständiger Flow

```bash
# 1. Codegen starten
npx playwright codegen https://www.sozialbank-onlinebanking.de/...

# 2. Manuell durchklicken, Code wird generiert

# 3. In sync.js einfügen

# 4. Test mit sichtbarem Browser
HEADLESS=false node sync.js --test

# 5. Test im Container
docker compose --profile banking-sync run --rm banking-sync node sync.js --test
```
