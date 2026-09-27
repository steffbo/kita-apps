# Plan: neues Design + Eltern-Zugang

Arbeitsplan, wird nach Abschluss gelöscht (Entscheidungen wandern nach `docs/status-fees.md`).

## Entscheidungen (Stefan, 2026-09-27)

- Rolle `USER` („Benutzer“) bleibt Mitarbeiter-Rolle (alle Kinder/Beiträge, keine Admin-Seiten).
- Neue Rolle für Eltern (Etappe 2): Konto an einen Elternteil gebunden, sieht nur eigenen Haushalt
  (eigene Kinder, eigene Beiträge, eigene Elternstunden). Bleibt in `backend-fees`, kein eigener Dienst.
- Reihenfolge: erst Design für die ganze App, dann Eltern-Oberfläche im neuen Design, dann Konten/Einladung.
- Design angelehnt an knirpsenstadt.de (kein Nachbau): Nunito, Grüntöne der Homepage,
  helle grüne Kopfleiste mit Logo und Menü oben statt Seitenleiste, weiche Karten, optional Dark Mode.
  Die Homepage soll später im oberen Menü auf das Portal verlinken.

Homepage-Farben (aus deren CSS): brand-50 `#f5fbe9`, brand-100 `#eaf6d3`, brand-200 `#d4ecc0`,
brand-400 `#a8d98a`, brand-600 `#86c06a`, brand-700 `#5d9847`, brand-800 `#4a7a3a`, Kopfleiste `#DAEFB4`.

## Etappe 1a: Design-System und Layout

Status: umgesetzt.

- Tokens in `main.css` (hell + dunkel) auf Homepage-Grün, Schrift Nunito (selbst gehostet, `@fontsource`).
- `MainLayout`: Kopfleiste mit Logo, Menügruppen als Dropdowns, Benutzer-Menü rechts; mobil ausklappbar.
- Theme-Umschalter System/Hell/Dunkel, in `localStorage` gespeichert, ohne Aufblitzen beim Laden.
- Login-Seite und Dashboard im neuen Look.
- Fix `USER`: Dashboard darf nicht an Admin-Endpunkten scheitern; Menüpunkte, deren Seiten nur
  Admin-Endpunkte nutzen, für `USER` ausblenden.

## Etappe 1b: Seiten auf semantische Farben

Alle Seiten/Komponenten von festen `gray-*`/`white`-Klassen auf Tokens (`bg-card`, `text-muted-foreground`,
`border-border` …) umstellen, Statusfarben mit `dark:`-Varianten. Ziel: jede Seite in Dunkel lesbar.

## Etappe 2: Eltern-Rolle und Eltern-Oberfläche

Offen, wird nach Etappe 1 detailliert (Verknüpfung Konto ↔ Elternteil/Haushalt, eigene Endpunkte unter
`/me/**`, Startseite mit Kindern, Beitragsstand, Elternstunden).

## Etappe 3: Konten für Eltern

Offen (Einladung per E-Mail vs. Anlage durch Admin, Passwort setzen).
