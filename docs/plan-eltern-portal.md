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

Status: umgesetzt.

Alle Seiten/Komponenten von festen `gray-*`/`white`-Klassen auf Tokens (`bg-card`, `text-muted-foreground`,
`border-border` …) umstellen, Statusfarben mit `dark:`-Varianten. Ziel: jede Seite in Dunkel lesbar.

## Entscheidungen Etappe 2/3 (Stefan, 2026-09-27)

- Konto ↔ Elternteil automatisch über die E-Mail-Adresse.
- Eltern dürfen ihre Kontaktdaten ändern, ohne Freigabe, aber mit Audit-Log. Alte Werte sieht nur der Admin.
  Admin-Dashboard bekommt ein rein informatives Feed-Widget mit den letzten Änderungen.
- Nachtrag (2026-09-27): Eltern dürfen auch Kinderdaten ändern (Vorname, Nachname, Geburtsdatum, Adresse),
  ebenfalls mit Audit-Log. Nur lesbar: Betreuungszeit, Rechtsanspruch, außerdem Mitgliedsnummer und
  Ein-/Austrittsdatum (Vertragsdaten, Annahme). Eltern können einen Fehler melden (Freitext, optional mit
  Bezug auf Kind/Beitrag/Elternstunden); Meldungen erscheinen im Admin-Feed und werden dort erledigt.
- Nachtrag (2026-09-30, ersetzt die Nachträge zu Kinderdaten und Adressen): Eltern ändern selbst nur noch
  ihre eigene E-Mail (= Login) und Telefonnummern (eigene und die des anderen Elternteils). Namen,
  Geburtsdatum und Adressen von Eltern und Kindern sind nur lesbar; Korrekturen laufen über „Fehler melden“,
  damit der Vorstand falsche Daten abfängt (Geburtsdatum bestimmt z. B. U3/Ü3).
- E-Mail-Änderung ändert auch den Login: Kontakt-E-Mail des Elternteils und Login-E-Mail des verknüpften
  Kontos bleiben immer gleich (auch wenn der Admin die E-Mail des Elternteils ändert).
- Eltern dürfen Elternstunden melden; gemeldete Stunden zählen erst nach Bestätigung (Tim/Admin).
- Passwörter/Zugänge werden später manuell verschickt, erst nach Umzug auf einen VPS. Jetzt nur vorbereiten.

## Etappe 2a: Backend Eltern-Rolle

Status: umgesetzt.

- Rolle `PARENT` (Label „Eltern“). `fees.users.parent_id` (eindeutig, `ON DELETE SET NULL`).
  Beim Anlegen bzw. Wechsel zur Rolle `PARENT` wird der Elternteil über die E-Mail gesucht
  (Groß-/Kleinschreibung egal): genau einer → verknüpfen; keiner → Fehler „Kein Elternteil mit dieser
  E-Mail“; mehrere → Konflikt. Danach bleiben Kontakt- und Login-E-Mail synchron (siehe Nachtrag).
- Endpunkte `/me/**` nur für `PARENT`, Haushalt immer serverseitig aus `parent_id` bestimmt:
  Übersicht (eigene Kontaktdaten, Haushalt, Kinder), Beiträge des Haushalts je Jahr (offen/bezahlt),
  Elternstunden des Haushalts je Kita-Jahr (Soll/Ist/Offen/Befreiung, Einträge), Stunden melden
  (`SUBMITTED`, Quelle `PARENT`), eigene gemeldete Einträge zurückziehen, Kontaktdaten und
  Kinderdaten ändern, Fehler melden und eigene Meldungen sehen.
- Audit: `fees.data_changes` für Eltern- und Kinderdaten (Entität, handelnder Elternteil, Konto,
  Feld, alt, neu, Zeitpunkt), in derselben Transaktion wie die Änderung. Verlauf je Elternteil
  und Kind sowie Feed nur für `ADMIN`. Staff-Änderungen haben keinen handelnden Elternteil und
  erscheinen nicht im Feed.
- `fees.parent_reports`: Eltern melden Fehler zu Kind, Beitrag oder Elternstunden-Eintrag des
  eigenen Haushalts; Admin sieht und erledigt sie. Kontakt und Allgemeines erlauben Freitext ohne Bezug.
- Kontakt-E-Mail und Login-E-Mail des verknüpften Kontos werden bei Änderungen durch Eltern, Staff
  und auf der Benutzer-Seite zusammen aktualisiert; bestehende Refresh-Sessions bleiben gültig.
- Freigabe: gemeldete Einträge bestätigen/ablehnen (mit Grund) durch `ADMIN`/`PARENT_WORK`;
  Übersicht zeigt Anzahl offener Meldungen.
- Keine Einkommens-/Einstufungsdaten, keine internen Notizen, keine Override-Gründe für Eltern.

## Etappe 2b: Eltern-Oberfläche und Staff-Anpassungen

Status: umgesetzt.

- Eltern-Bereich `/familie/**` im neuen Design, handytauglich: Übersicht, Beiträge, Elternstunden
  (mit „Stunden melden“), Meine Daten. `PARENT` sieht nichts anderes.
- Staff: Freigabe gemeldeter Stunden, Feed-Widget im Admin-Dashboard, Änderungsverlauf auf der
  Elternteil-Seite (Admin), Rolle „Eltern“ auf der Benutzer-Seite mit Anzeige des verknüpften Elternteils.
- e2e: Eltern-Flow (sieht nur eigenen Haushalt, meldet Stunden, ändert Kontaktdaten; Admin sieht Feed).

## Etappe 3: Zugänge vorbereiten (nicht aktiv nutzen vor VPS-Umzug)

Eltern-Konten gesammelt aus Elternteilen mit E-Mail anlegen (ohne Passwort), Einmal-Link zum
Passwort-Setzen, den der Admin kopieren und manuell verschicken kann. E-Mail-Versand erst später.
