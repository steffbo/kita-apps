# Status: Beiträge (backend-fees + frontend/apps/beitraege)

Rolling change log of non-obvious implementation decisions. Newest first.
Basics (ports, commands, layout) live in `AGENTS.md`.

## Jahresauswahl: Datenjahre plus Folgejahr (2026-10-31)

- Fehler: In der Elternstunden-Übersicht hing die Kita-Jahr-Liste vom gewählten Jahr ab; nach
  Auswahl von 2025/26 verschwand 2026/27.
- `src/utils/years.ts` liefert jetzt alle Jahreslisten: vom ersten Jahr mit Daten
  (`FIRST_DATA_YEAR = 2025`, Beiträge, Einstufungen und Elternstunden beginnen 2025) bis zum Jahr
  nach dem aktuellen, plus weitere Jahre mit Daten (Elternstunden: Beginn der Regelwerke).
  Das aktuelle Jahr kommt aus dem Berliner Datum (`currentYear`, `currentKitaYear`), nie aus der
  Auswahl.
- Umgestellt: Elternstunden-Übersicht (Verwaltung), „Deine Elternstunden“, Beiträge der Eltern
  (jetzt auch Folgejahr), Jahresübersicht im Dashboard (statt Vorjahr bis Folgejahr) und
  Einstufungsliste (statt 2 Vorjahre bis Folgejahr). Anlage-Dialoge (Beitrag anlegen, Beiträge
  generieren, Einstufung) behalten Vorjahr bis Folgejahr, weil sie Jahre zum Anlegen anbieten.

## Kinderlisten „Krippe“ und „Kita“ für die Leitung (2026-10-31)

- Anlass: Die Leitung führt Papierlisten aller angemeldeten Kinder, getrennt nach Krippe und „Große Gruppe“.
  Sie kommen jetzt aus der App, passend zum Stichtag des Dashboards (Backlog #25).
- `GET /stichtagsmeldung/jugendamt/lists?date=` (`ADMIN`/`USER`/`LEITUNG`, `inline=1` für Anzeige im Tab):
  ein PDF, A4 quer, erst „Krippe“ (U3 am Stichtag), dann „Kita“ (Ü3), jede Liste ab einer neuen Seite. Die
  Kinder und der Stichtag kommen aus demselben Report wie die Zahlen (`GetJugendamtReport`), die Listen passen
  also immer zum Dashboard. Gerendert mit fpdf in `internal/letter/child_lists.go` (Liberation Serif wie der
  Einstufungsbrief, `loadFonts` gemeinsam), nicht im Druckfenster des Browsers: so sind „Seite x von y“ und die
  wiederholten Spaltenköpfe zuverlässig.
- Spalten: Nr., Name, Vorname, Geburtsdatum, Straße (mit Hausnummer), PLZ, Ort, Std./Tag (Wochenstunden / 5,
  z. B. 6,4) und Std./Woche, sortiert nach Name. Kopf: Verein, „Elternverein, gemeinnützig anerkannt“,
  Anschrift, Telefon 030 9446171 und Fax 032 227770160 (Angabe Vorstand, fest im Code), rechts Listentitel,
  Stichtag und Anzahl; Fuß: Erstelldatum, Seitenzahl. Zu lange Werte werden mit „…“ gekürzt.
- Entscheidungen: Titel „Kita“ statt „Große Gruppe“, keine Leerzeilen zum Nachtragen. Fehlende
  Angaben bleiben in der Liste leer.
- `JugendamtChild` liefert jetzt auch `street`, `streetNo`, `postalCode`, `city`, `addressComplete` und
  `careHours` (Wochenstunden aus `child_care_hours_history`, gültig am Stichtag). Das sind keine Beitragsdaten.
  Das Dashboard warnt mit Links zu den Kindern, wenn Straße, Hausnummer, PLZ oder Ort fehlen (live am
  01.09.2026: 2 von 46). Der Button „Listen drucken“ öffnet das PDF im neuen Tab.
- Abgleich mit der Live-Kopie zum 01.09.2026: Krippe 13, Kita 33, 3 Seiten, nichts gekürzt. Die Stunden sind
  noch die gespeicherten Werte. Die Abweichung zur Liste der Leitung (siehe unten) bleibt offen, bis die Daten
  abgeglichen sind.
- Tests: `child_lists_test.go` (Seitenumbruch, leere Listen, Stunden, Kürzen; Muster-PDF mit
  `CHILD_LISTS_SAMPLE_OUT=…`), `TestJugendamtChildAddressComplete`, DB-Test mit Stunden am Stichtag und PDF,
  `TestRouter_RoleAreas`, e2e `leitung.spec.ts`.

## Leitungs-Dashboard: Stichtagsmeldung an den Landkreis (2026-10-31)

- Anlass: Die Leitung meldet quartalsweise an den Landkreis Barnim (Jugendamt): Belegung (Krippe/Kindergarten)
  und die Formularspalten nach § 17a KitaG (Stichtage 1. März, Juni, September, Dezember). Die Zahlen sollen
  sich vom Dashboard ablesen lassen.
- `GET /stichtagsmeldung/jugendamt?date=` (`ADMIN`/`USER`/`LEITUNG`, ohne Beitrags- oder Einkommensdaten;
  `stats`, `report`, `children` unter `/stichtagsmeldung` bleiben `staffOnly`). Ohne `date`: der letzte
  Stichtag bis 30 Tage danach (Meldung wird gerade erstellt), sonst der nächste
  (`domain.StichtagForJugendamt`). Gezählt werden Kinder mit `entry_date <= Stichtag <= exit_date`.
- Einteilung (`domain.ClassifyKitaYear`): U3 = drittes Lebensjahr am Stichtag noch nicht vollendet (wie die
  Admin-Stichtagsmeldung). Für Ü3 zählt das Einschulungsjahr nach § 37 BbgSchulG: schulpflichtig im August,
  wenn das sechste Lebensjahr bis 30.09. vollendet ist; nach § 188 Abs. 2 BGB gilt das auch für den
  Geburtstag 1. Oktober (`domain.SchoolEntryYear`). Kita-Jahr ab 1. August: Einschulung im nächsten August =
  letztes Kita-Jahr, im übernächsten = vorletztes, später = „übrige Ü3“ (Formularspalte „3. Lebensjahr“).
  **Zurückgestellt** wird nicht gepflegt, sondern abgeleitet: Kinder, die nach Alter schon in der Schule
  wären, aber noch angemeldet sind. Sie zählen wie im Formular zusätzlich im letzten Kita-Jahr
  („inkl. Rücksteller“).
- Abgleich mit dem Papierformular zum 01.09.2026 (Live-Daten, read-only): 46 Kinder, 13 U3, 33 Ü3, 9 übrige
  Ü3, 10 vorletztes, 14 letztes inkl. Rücksteller stimmen überein. Abweichung: 2 statt 1 zurückgestellt (beide
  vor dem 01.10.2020 geboren und noch angemeldet); fachlich zu klären, z. B. Austrittsdatum oder
  Geburtsdatum.
- Bewusst nicht enthalten: die Aufteilung „bis/über 6 Stunden“ der Seite 1 (VbE). Die gespeicherten
  Betreuungszeiten weichen am 01.09.2026 oft von der Liste der Leitung ab (Ü3 bis 30 Wochenstunden: 12 statt
  5). Erst nach Datenabgleich ergänzen. Personalplanung (Seite 2/3) ist der nächste Schritt.
- Frontend: `/` rendert über `StartPage.vue` für `LEITUNG` die `LeitungDashboardPage` (sonst das
  Beitrags-Dashboard); Route-Meta `requiresDashboard` (`canAccessFees` oder `isLeitung`). Leitung landet nach
  Login und bei gesperrten Seiten auf `/` statt `/kinder`; Navigation mit „Start“, Handy-Tabs Start, Kinder,
  Eltern, Stunden. Die Karte zeigt Stichtag, Tage bis/seit, Pfeile für ±1 Quartal; ein Klick auf eine Zahl
  filtert die Kinderliste darunter.
- Tests: `stichtagsmeldung_test.go` (Domain, Grenzfälle 30.09./01.10., Geburtstag am Stichtag, Fenster),
  `TestJugendamtReport_CountsEnrolledChildrenByKitaYear` (DB), `TestRouter_RoleAreas`, e2e `leitung.spec.ts`.

## Rolle `LEITUNG`: Stammdaten ohne Beitragsdaten (2026-10-30)

- Anlass: Die Kita-Leitung soll Kinder, Eltern, Haushalte, Mitglieder und Elternstunden pflegen, aber keine
  Beiträge, Einkommen oder Kontodaten sehen. Migration `000048` erweitert `users_role_check` um `LEITUNG`
  (Down schlägt fehl, solange noch ein `LEITUNG`-Konto existiert, wie bei `000038`).
- Zugriff (`internal/api/router.go`): Eine eigene Gruppe „Stammdaten“ (`ADMIN`/`USER`/`LEITUNG`) enthält
  `/children`, `/parents`, `/households`, `/members`. Lesen, Anlegen, Ändern und Verknüpfen sind für `LEITUNG`
  erlaubt. `staffOnly` (`ADMIN`/`USER`) bleiben: **Löschen** von Kind, Elternteil, Haushalt und Mitglied (das
  Löschen eines Kindes löscht auch seine Beitragsdaten), `/children/{id}/ledger`, `/timeline`, `/notes`
  (Kindernotizen), `/children/import/**`, `/notes`, `/fees/**`, `/einstufungen/**`, `/fee-schedules`,
  `/stichtagsmeldung` (außer `jugendamt`, siehe oben), `/import/**` (Bankabgleich), `/banking-sync/**`. In `/parent-work/**` darf `LEITUNG` nur
  lesen (`overview`, `households`, `households/{id}`, `rules`) und Einträge anlegen (`POST /entries`); Bearbeiten,
  Stornieren, Bestätigen/Ablehnen, manuelles Soll und CSV-Import bleiben `ADMIN`/`PARENT_WORK` (Nachtrag
  2026-10-30, Wunsch: Leitung sieht den Stand der Familien und erfasst Stunden, keine Verwaltung).
- Beitragsfelder in den erlaubten Antworten: `middleware.HideFeeData("LEITUNG")` entfernt für diese Rolle aus
  jeder JSON-Antwort der Stammdaten-Gruppe rekursiv `annualHouseholdIncome`, `incomeStatus`,
  `childrenCountForFees`, `openFeesCount` und `einstufungSync` (`middleware.FeeFields`), verwirft dieselben
  Schlüssel in JSON-Anfragen (fehlend heißt „unverändert“, Einkommen lässt sich also auch nicht schreiben) und
  ignoriert in der Kinderliste die Filter `hasOpenFees` und `hasWarnings` (letzterer prüft auch fehlendes
  Einkommen). Andere Rollen laufen unverändert durch.
- Absicherung gegen neue Felder: `TestFeeFieldsCoverDomainPayloads` schlägt fehl, wenn an Kind, Elternteil,
  Haushalt, Mitglied oder den Stundenhistorien ein Feld mit `fee|income|einstuf|amount|balance|iban|price|
  contribution` im JSON-Namen fehlt in `FeeFields`. Neue Beitragsfelder dort eintragen.
- Nebenwirkung, die bleibt: Ändert die Leitung Geburtsdatum oder Betreuungszeiten eines Kindes, gleicht das
  Backend Einstufung und Platzgeld wie bisher an (`ChildService.Update`). Das Ergebnis sieht die Leitung nicht.
- Frontend: `auth.canAccessMasterData` (Stammdaten) neben `canAccessFees` (Beiträge, Einstufungen, Bankabgleich,
  Notizen, Kinderimport); `LEITUNG` zählt zu `canAccessParentWork`, aber nicht zu `canManageParentWork`
  (Seite „Verwaltung“ mit Route-Meta `requiresParentWorkManagement`, in der Familienansicht Soll-Button und
  Aktionsspalte). Route-Meta `requiresMasterData` für
  Kinder, Eltern und Mitglieder; Leitung landete nach Login und bei gesperrten Seiten auf `/kinder` (seit
  2026-10-31 auf dem eigenen Dashboard `/`, siehe oben). Die Navigation zeigt Stammdaten (Kinder, Eltern,
  Mitglieder) und Elternstunden, keine Aufgaben, keine Beitragsgruppe. Kinderseite ohne Einkommen, Platzgeld, IBANs, Beitragsliste, Notizen und Löschen; Kinderliste
  ohne Filter „Mit Hinweisen“ und „Offene Beiträge“ und ohne Import; Haushaltsbearbeitung nur Name.
- Bewusst gleich wie bei `PARENT_WORK`: Die Elternstunden-Regeln zeigen weiter den Satz je Fehlstunde.
- Tests: Routertests (`TestRouter_RoleAreas`, `TestRouter_LeitungWriteRoutes`,
  `TestRouter_LeitungParentWorkRoutes`), Middleware-Tests
  (`hide_fee_data_test.go`) und e2e `frontend/e2e/beitraege/leitung.spec.ts`. Nachgeholt vor dem Merge
  (2026-10-30): `go test ./...` mit Datenbank, Typecheck und alle E2E-Tests grün, `scripts/generate-api.sh`
  ohne Drift; Ansicht als Leitung auf der lokalen Live-Kopie geprüft (Desktop und Handy).
- Bewusst gleich wie bei `PARENT_WORK`: Der Fehlbetrag der Elternstunden (`missingAmountCents`) steht in der
  API-Antwort und wird nur in der Oberfläche ausgeblendet (nur Admin sieht ihn).
- Anlegen eines Kontos (auch `LEITUNG`) verschickt keine Mail; Mails gibt es nur bei Eltern-Einladungen.
- Dialog „Stunden erfassen“ (alle Rollen): Steht die Familie fest (Zeilen-Button, Familienseite, Bearbeiten
  oder in der Suche gewählt), zeigt der Dialog nur noch ihren Namen statt der Suche. Für eine andere Familie
  den Dialog schließen und neu öffnen; das vermeidet Buchungen auf die falsche Familie.

## UI-Redesign: neutrale Basis, Indigo, Inter (2026-10-29)

- Anlass: Die App wirkte wie die verspielte Webseite knirpsenstadt.de (Grün, Nunito, runde Formen). Ziel ist
  ein ruhiges, professionelles Werkzeug, hell und dunkel gleichwertig, mobil zuerst. Referenz-Entwurf:
  `docs/design/ui-mockup.html` (interaktiv, Zustände per Hash, z. B. `#wide&dark&page=kinder`).
- Schritt 1, Farben und Schrift: Zink-Grautöne als Basis, eine Akzentfarbe Indigo (hell `#4f46e5`, dunkel
  `#818cf8`). Grün, Gelb und Rot tragen nur noch Status (bezahlt, offen, überfällig) und haben je Ton,
  Hintergrund und Rahmen als Token (`success|warning|destructive` mit `-soft`/`-line`, dazu `primary-soft`
  und `primary-text`). `brand-*` und `header` sind entfernt. Schrift Inter statt Nunito, Ziffern in Tabellen
  und Eingaben tabellarisch, Radius 10 px. Eingaben auf schmalen Bildschirmen 16 px, sonst zoomt iOS Safari
  beim Fokus hinein.
- Schritt 2, App-Hülle (`layouts/MainLayout.vue`, Navigation zentral in `composables/useNavigation.ts`):
  - Ab `lg` (1024 px) Seitenleiste mit Gruppen (Start, Aufgaben · Stammdaten · Beiträge · Elternstunden ·
    System), Suchfeld und Konto unten. Die bisherigen Split-Buttons (Text öffnet erste Seite, Pfeil das Menü)
    sind weg. Darunter Kopfleiste (Logo, Suche, Konto) und Tab-Leiste unten: Admin Start · Aufgaben · Kinder ·
    Erinnerungen · Mehr; `USER` hat Beiträge statt Erinnerungen (Erinnerungen sind Admin-only); Eltern
    Übersicht · Beiträge · Stunden · Daten. „Mehr“ öffnet die komplette Navigation als Sheet. Tablets im
    Hochformat bekommen bewusst das Handy-Layout, die breiten Tabellen brauchen den Platz.
  - „Aufgaben“ bündelt Bankabgleich, Meldungen und Änderungen von Eltern unter gemeinsamen Reitern
    (`layouts/TasksLayout.vue`, Router-Elternroute mit leerem Pfad). Die URLs `/bankabgleich`, `/import`,
    `/meldungen` und `/aenderungen` bleiben, `/aufgaben` leitet auf den Bankabgleich. Der Navigationspunkt
    springt in den ersten Bereich mit offenen Einträgen. Zähler (`composables/useTaskCounts.ts`): nicht
    zugeordnete Transaktionen, offene Meldungen, neue Änderungen seit dem letzten Besuch (wie die
    Dashboard-Karte). Sie werden beim Seitenwechsel höchstens alle 30 s neu geladen, beim Verlassen oder
    Betreten eines Aufgaben-Bereichs sofort. Die Bereiche haben keine eigene Überschrift mehr.
  - Suche (Strg/⌘+K oder Lupe): Seiten, Kinder und Eltern (Server-Suche ab 2 Zeichen), Pfeiltasten + Enter.
  - Konto-Menü: Design als Segmentschalter System/Hell/Dunkel (statt Rundlauf-Button), Passwort ändern bzw.
    Impersonation beenden, Abmelden. Auf dem Desktop ein Popover über dem Konto-Eintrag, sonst ein Sheet.
    Die Impersonation steht als gelbe Leiste über dem Inhalt, mit eigenem „Beenden“.
  - `components/ui/AppSheet.vue`: Dialog als Bottom Sheet auf dem Handy, zentriert ab `sm`; Esc über
    `[data-dialog-close]`, Scroll-Sperre, Fokus auf `[autofocus]` bzw. den Dialog. Grundlage für Schritt 3.
  - `index.html`: `viewport-fit=cover` (Tab-Leiste respektiert den iPhone-Home-Indikator) und `theme-color`.
- Schritt 3, gemeinsame Bausteine: Klassen `btn`, `badge`, `note`, `card`, `chip`, `label`, `input`, `table`
  in `main.css` und Komponenten `PageHeader`, `AppPagination`, `EmptyState` (Regeln: `docs/design/ui-kit.md`).
  Zellen-Padding in `.table` ist absichtlich fest (Selektor schlägt Utilities). Tabellen auf dem Handy
  scrollen seitlich statt Wörter buchstabenweise umzubrechen (`overflow-wrap: anywhere` entfernt).
  Kinderseite als Referenz: Filter als Chips mit Gesamtzahl statt Checkboxen und Untertitel, Liste statt
  Tabelle auf dem Handy, Dialoge als `AppSheet`, Platzhalter der Suche jetzt „Name oder Mitgliedsnummer“.
- Schritt 3b, alle übrigen Seiten und Komponenten auf die Bausteine umgestellt (in fünf Codex-Lanes, danach
  geprüft: je Datei Handler, Bedingungen, Testids, Labels und sichtbare Texte gegen den alten Stand):
  - Keine Tailwind-Palettenfarben und keine handgebauten Dialoge mehr (Ausnahme: Drag-and-drop-Hinweis im
    Elternstunden-Import). Beitragsarten sind neutrale Badges, nur Mahngebühren rot; Trefferquote und
    Erinnerungsstatus nutzen die Status-Badges (`utils/fees.ts`, `utils/reminders.ts`). Seitenweise Listen
    (Beiträge, Einstufungen, Eltern, Mitglieder, Notizen, Notizen am Kind) nutzen `AppPagination`.
  - Entfallen sind Untertitel ohne eigene Information (Anzahl-Zeilen wie „116 Eltern registriert“,
    Seitenbeschreibungen wie „Übersicht der Beitragszahlungen“) und die „Zurück zur Übersicht“-Buttons
    (ersetzt durch den Zurück-Pfeil im `PageHeader`). Erklärungen mit Regeln (Beitragsordnung,
    Benutzer, Elternstunden-Regeln) stehen weiter da, als `note`.
  - Filter als Chips: Beiträge (Status und Beitragsart, „Alle Typen“), Meldungen (Offen/Erledigt/Alle statt
    Auswahlliste), Mitglieder („Inaktive anzeigen“), Bankabgleich. Bankabgleich und Einstufungen haben die
    Suche in voller Breite oben, Aktionen bzw. Jahr und Status daneben.
  - Dialoge mit eigener Kopfzeile in der alten Modal-Box (Import-Historie, Blacklist, CSV hochladen, manuelle
    Zuordnung, E-Mail-Log, Zahlungsdaten, Stichtagsreport, Sende-Bestätigung) haben jetzt nur den
    `AppSheet`-Titel und einen Schließen-Button.
  - Änderungsverlauf an Kind und Elternteil steht jetzt unter dem Inhalt; vorher landete er als erstes
    Fragment-Element über der Überschrift.
  - Bekannte Lücken: Import-Vorschau der Kinder und Transaktionstabelle haben noch keine Handy-Liste (sie
    scrollen seitlich), einige Bestätigungen nutzen weiter `confirm()` des Browsers.
- Schritt 4, Aufräumen nach dem Umbau (kein neues Verhalten außer den genannten Punkten):
  - Seitenweise Listen laden alle über `composables/usePagedList.ts` (Beiträge, Notizen und Einstufungen
    hatten die Logik nachgebaut); `goToPage` liegt dort, die Seitengrößen kommen aus `AppPagination`. Neue
    Seitengröße springt überall auf Seite 1 (vorher nur bei Notizen). Beiträge behalten die Auswahl jetzt
    für Zeilen, die nach dem Neuladen noch sichtbar sind, statt sie bei jedem Laden zu leeren; Suche bei
    den Einstufungen wartet 150 statt 300 ms.
  - Alle Tabellen in Karten nutzen `.table`; die Zebra-Regel für die übrigen und der Handy-Hack für
    `px-4`-Zellen sind weg. Ausnahmen ohne `.table`: Jahresübersicht im Dashboard und Beitragsauswahl der
    Mahnungsvorschau (eingebettet, eigene Zeilenlinien) sowie das Einstufungs-PDF. Kopfzellen in `.table`
    sind über `:where()` linksbündig, damit `text-right` wieder greift (z. B. „Betrag“, Zahlenspalten).
  - Entfernt: ungenutzte Tokens (`secondary`, `accent-`/`success-`/`warning-foreground`) und Typen,
    eigene Esc-Handler im Bankabgleich und Kinderimport (macht `AppSheet` bzw. `utils/dialogEscape.ts`),
    die Umleitung alter `?tab=`-Links im Bankabgleich (keine Verweise mehr). `formatCareType` liegt in
    `utils/format.ts` statt dreimal lokal.
  - Tests: Sortieren prüft nur noch der Übersichtstest der Elternstunden (alle Tabellen teilen
    `useTableSort`/`SortTh`), Vorstands- und Benutzertabelle prüfen ihre Filter; der Test auf den Regeltext
    der Verwaltung ist entfallen.
- Fix nach dem Deploy: Seitenleiste (Desktop) und Kopfleiste (Handy) scrollten mit, weil
  `overflow-x: hidden` auf `html`, `body` und `#app` diese zu Scroll-Containern machte und `sticky` aushebelte.
  Jetzt `overflow-x: clip` (schneidet ebenfalls ab, ohne Scroll-Container).

## Schnellere Release-Pipeline (2026-10-21)

- Ausgangslage: Push bis Image ca. 3:45 min. Kritischer Pfad war `e2e-beitraege` (gut 2 min, davon 36 s
  Chromium-Installation, 10 s Postgres-Pull, 26 s Stack-Build, 31 s Tests) und danach seriell der Image-Build
  (`backend-fees` 71 s, davon 38 s Cache-Export ohne jeden Cache-Treffer).
- Kein Sharding der Playwright-Suite wie in resk: 36 Tests laufen in rund 30 s, jeder Shard müsste aber die
  komplette Einrichtung (Go, Bun, Chromium, Stack-Build, ca. 85 s) wiederholen. Stattdessen:
  - `build-images.yml`: Job `build` baut beide Images parallel zur CI ohne Push und füllt den Cache, Job
    `push` (`needs: [ci, build]`) baut aus dem Cache neu und pusht. Ein rotes CI liefert weiterhin kein Image.
  - Cache-Scope pro Image (`scope=<name>`): vorher teilten sich beide Images den Standard-Scope und
    überschrieben sich gegenseitig den Cache, deshalb lief `backend-fees` immer komplett neu.
  - Playwright-Browser werden per `actions/cache` (Schlüssel `bun.lock`) zwischengespeichert; die
    Systempakete (`install-deps`) laufen weiter jedes Mal.
  - CI-Worker 2 → 4 (Standard-Runner öffentlicher Repos hat 4 vCPUs).
  - `start-beitraege-stack.sh` startet den Postgres-Container im Hintergrund, damit ein kalter Image-Pull
    parallel zum Frontend-/Backend-Build läuft.

## Einheitliche Sortier-Header, Vorstandstabelle filter- und sortierbar (2026-10-21)

- Alle sortierbaren Tabellenköpfe nutzen `components/SortTh.vue` (Pfeil-Icons, `aria-sort` am `th`). Bisher
  hatten Kinder, Mitglieder, Eltern, Beiträge und Banktransaktionen eigene Header-Buttons (Chevron-Icons,
  bei Beiträge `aria-sort` am Button). `SortTh` ist dafür generisch über den Spaltenschlüssel und kann ein
  Icon vor der Beschriftung (`icon`) und ein anderes Padding (`cellClass`, Transaktionen `px-3`). Die
  Sortierlogik bleibt je Seite: serverseitig (Kinder, Mitglieder, Eltern, Beiträge mit eigener
  Startrichtung je Spalte, Transaktionen) bzw. clientseitig über `useTableSort`.
- Vorstand (`/elternstunden/vorstand`): Suche über Mitglied, Familie und Amt, Filter aktiv/nicht aktiv,
  alle Spalten außer Aktionen sortierbar, Start: „Von“ absteigend.

## Erinnerungen erst nach Eintritt, U3 nur mit Einstufung (2026-10-14)

- Die Arbeitsliste (`GET /fees/reminder-cases`, beide Scopes) und Vorschau/Versand
  (`/fees/reminder-cases/{householdId}/preview|send`) berücksichtigen offene Beiträge eines Kindes nur, wenn
  sein Eintrittsdatum am Stichtag (`asOf` bzw. `runDate`) erreicht ist und es — falls dann unter drei — eine
  Einstufung hat (irgendeinen Zeitraum). Das gilt für alle Beitragsarten samt Mahngebühren, also auch für den
  Vereinsbeitrag, der vor dem Eintritt schon angelegt ist. Austritt spielt keine Rolle.
- Logik: `ReminderService.remindableChildren` in `loadCaseFees`; der `ReminderService` bekommt dafür das
  `EinstufungRepository`. Werden solche Beiträge trotzdem zum Versand ausgewählt, antwortet der Endpunkt mit
  `409` (`CaseConflictError`) und den betroffenen IDs.
- Anlass: zwei Kinder mit Eintritt 01.12.2026 (eins Ü3, eins U3 ohne Einstufung) standen mit dem
  Vereinsbeitrag 2026 schon unter „Handlungsbedarf“.

## Manuelle Zuordnung bucht höchstens den überwiesenen Betrag (2026-10-14)

- `CreateManualMatch` (`POST /import/match`) buchte bisher immer den vollen Gebührenbetrag, auch wenn die
  Überweisung kleiner war oder die Gebühr schon teilweise bezahlt. Jetzt gilt: Match-Betrag = min(offener
  Gebührenbetrag, noch nicht zugeordneter Überweisungsbetrag); ist eins davon 0, antwortet der Endpunkt mit 400.
  Eine Teilzahlung lässt den Rest der Gebühr offen. `allocate` prüfte das schon.
- Aufgefallen bei der Prüfung Härtefallausgleich 2026 (Jugendamt Barnim): Elternbeitragseinnahmen
  Jan–Aug 2026 laut Kontoauszug 14.511,22 €, die App zeigte 141 € mehr. Live korrigiert über die API
  (Unmatch + `allocate`): Middelstaedt Juni-Nachzahlung vom 05.06.2026 (30 €, war mit 150 € gebucht);
  Fink August 2026 Platzgeld 231 € → 210 € laut Einstufung, Zahlung vom 05.08.2026 mit 210 € neu zugeordnet.
  Danach stimmt die App mit der Bank überein; keine neuen Warnungen.
- Noch offen, nicht korrigiert: Abraham August 2026, Zahlung vom 21.09.2026 über 124,00 € ist mit 124,92 €
  gebucht (Korrektur ließe 0,92 € offen).

## Einstufungsliste verlinkt Kinder; E-Mail zur Folgeeinstufung nennt die Änderung (2026-10-12)

- Einstufungsliste: Kindnamen in „Einstufung fehlt“ und in der Tabelle führen zur Kindseite; Haushalt steht
  jetzt unter dem Kindnamen (eigene Spalte entfällt), die Regel darf umbrechen, die Tabelle scrollt
  horizontal statt abgeschnitten zu werden.
- E-Mail-Text (`utils/einstufungEmail.ts`): „willkommen bei uns in der Kita!“ nur, wenn die Einstufung mit
  dem Eintritt beginnt und es keine frühere gibt. Sonst wird gegen die Vorgängerin verglichen (Quelle einer
  Folgeeinstufung, sonst die letzte frühere Einstufung des Kinds) und aufgezählt, was sich geändert hat
  (Bereich, Betreuungszeit, Grundlage, Jahresnettoeinkommen, Anzahl Kinder), dazu Platzgeld neu vs. bisher.
  „Danke für eure Unterlagen.“ nur bei Grundlage Einkommen.
- Live nachgetragen (über die API, ohne Brief/Mail): Einstufungen 2025 für 11089 (Höchstsatz, 2 Kinder),
  11097 (Festsetzung 2024 inkl. Eltern-/Mutterschaftsgeld, 40.637,43 € → 110 €) und 11099 (54.000 € aus den
  Haushaltsdaten als Gesamtbetrag, 210 €; September 2025 aus Kulanz halbiert, siehe Notiz). 11089:
  Betreuungszeit 50 h bis 31.08.2025, 45 h ab 01.09.2025. 11099: Zahlungen vom 03.11.2025/02.02.2026 von
  Feb 2026/Nov 2025 auf Nov 2025/Feb 2026 umgehängt. Erzeugtes Platzgeld blieb überall unverändert.

## Beitragsbuchhaltung beginnt mit November 2025 (2026-10-12)

- Die App führt Beiträge vollständig erst ab November 2025 (erster voller Monat für alle Kinder).
  Ältere Erwartungen sind einzelne Nachbuchungen für spät eingegangene Zahlungen; was damals geschuldet war,
  wurde außerhalb der App abgerechnet (der Bankimport beginnt praktisch Ende Oktober 2025).
- `service.LedgerStartMonth` (2025-11-01): `SyncChildcareExpectations` ändert keine Monate davor. Eine
  nachträglich erfasste Einstufung ab z. B. September 2025 lässt diese Nachbuchungen damit stehen.
- Korrigiert am 2026-10-12: 22 FOOD- und 2 CHILDCARE-Zahlungen (inkl. Luca) waren zwischen November 2025 und
  Februar 2026 vertauscht zugeordnet (Februar-Zahlung auf November und umgekehrt; Beträge gleich). Über die API
  per Unmatch und manuellem Match neu zugeordnet, nach Probe auf der Live-Kopie; jede Gebühr ist weiter voll
  bezahlt, offen bleibt 0. Der Match legte für die Zahlung vom 2025-11-20 eine neue LATE_PAYMENT-Warnung an; sie
  wurde ohne Mahngebühr verworfen, wie die übrigen Warnungen dieser Zahlungen.

## Kinddaten ändern gleicht Einstufung und Platzgeld an; Brief-Einleitung nach Grundlage (2026-10-12)

- Die Monatstabelle einer Einstufung wird bei jedem Lesen aus den Kinddaten berechnet; vorher blieben nur die
  schon erzeugten CHILDCARE-Erwartungen (und ein versendeter Brief) bei einer Stundenänderung stehen.
- Jetzt laufen `PUT /children/{id}` (wenn Geburts-, Eintritts-, Austrittsdatum oder Betreuungszeit gesendet
  werden) und `POST /children/{id}/care-hours-history` über `EinstufungService.ApplyChildChange`
  (`ChildService.UseEinstufungSync`, in `main.go`): Monatstabellen aller Einstufungen des Kinds vor und nach der
  Änderung vergleichen, bei Abweichung in derselben Transaktion `syncFees` (Regeln wie beim Speichern:
  bezahlte Monate → Gutschrift prüfen) und `updated_at` setzen. Ein vorher versendeter Brief zählt damit als
  „seitdem geändert“ (Status „Nicht versendet“, Briefdatum = heute).
- API: `PUT /children/{id}` liefert `ChildUpdateResponse` (Kind + optional `einstufungSync`),
  `POST /children/{id}/care-hours-history` jetzt 200 mit `ChildEinstufungSync` (statt 204):
  geänderte Einstufungen (`id`, `year`, `letterSent`) und `expectationChanges`.
- Frontend: Hinweis „Einstufung durch die Änderung angepasst“ auf der Kindseite nach dem Speichern (Link zur
  Einstufung, „Brief nicht mehr aktuell, bitte neu senden“, Gutschrift-Monate). Einstufungsseite: „Seitdem
  geändert – bitte neu senden.“ neben dem Versanddatum. Sammel-Austrittsdatum auf der Kinderliste gleicht
  ebenfalls an, zeigt aber keinen Hinweis.
- Brief (Server-PDF und Druckansicht): Einleitung nennt „eingereichte Nachweise“ nur bei Grundlage Einkommen;
  Höchstsatz → „auf Grundlage eurer Anerkennung des Höchstsatzes und der Elternbeitragsordnung des Trägers“,
  nicht erforderlich → „auf Grundlage der Elternbeitragsordnung des Trägers“ (`letterIntroduction`).

## Einstufungen: Arbeitsliste fehlender Einstufungen, Suche und Status (2026-10-12)

- `GET /einstufungen/missing` (ADMIN/USER): aktive Kinder, die im aktuellen Monat (bzw. im Eintrittsmonat,
  wenn der Eintritt in den nächsten 3 Monaten liegt) den ganzen Monat unter drei sind und für diesen Monat
  keine gültige Einstufung haben; mit Ende der letzten Einstufung, falls es eine gab. Für sie kommt das
  Platzgeld aus den Haushaltsdaten (Fallback). Live-Kopie am 12.10.: 3 Kinder ohne jede Einstufung.
- `GET /einstufungen` nimmt zusätzlich `search` (Kindname in beiden Reihenfolgen, Mitgliedsnummer,
  Haushaltsname; ILIKE) und `status` (`current` = heute gültig, `provisional`, `unsent` = nie oder vor der
  letzten Änderung versendet); unbekannter Status → 400.
- Frontend `/einstufungen`: Kasten „Einstufung fehlt“ mit Link „Einstufung anlegen“
  (`/einstufungen/neu?childId=…`, Kind vorbelegt), Suchfeld (300 ms entprellt) und Statusauswahl.

## Eltern sehen ihre Einstufungsbriefe (2026-10-12)

- `GET /me/einstufung-letters` (PARENT): die per E-Mail versendeten Briefe für die eigenen Kinder (wie bei
  den übrigen `/me`-Daten über `ParentAccountRepository.Children`), neueste zuerst, nur Kind-Vorname, Datum,
  Betreff, Dateiname. `GET /me/einstufung-letters/{id}/pdf` liefert das gespeicherte PDF; fremde Briefe 404.
- Bewusst nur versendete Briefe, keine Einstufungen selbst: Eltern sehen damit genau, was ihnen mitgeteilt
  wurde, keine Entwürfe, kein Einkommen, keine Notizen. Briefe aus der Zeit vor der Speicherung fehlen.
- Frontend: Abschnitt „Einstufungsbriefe“ auf `/familie/beitraege` (unabhängig vom gewählten Jahr) mit
  PDF-Download.

## Platzgeld aus der gültigen Einstufung (2026-10-12)

- Beitragserzeugung (`POST /fees/generate`) und `POST /fees` ohne Betrag nehmen das Platzgeld eines U3-Kinds aus
  der Einstufung, die im Monat gilt (`effective_from_month <= Monat <= valid_until`), mit deren Kinderzahl und
  Grundlage, Betreuungsstunden des Monats und der Beitragsordnung des Monats – genau wie die Monatstabelle der
  Einstufung. Nur Monate ohne Einstufung rechnen weiter mit den Haushaltsdaten. `FeeService.UseEinstufungen`
  schaltet das ein (in `main.go`); Tests ohne den Aufruf rechnen nur mit Haushaltsdaten.
- Speichern, Ändern und Folgeeinstufung gleichen die schon erzeugten CHILDCARE-Erwartungen ihrer Monate an
  (`FeeService.SyncChildcareExpectations`, in derselben Transaktion). Ein Monat gilt als erzeugt, wenn es für
  das Kind CHILDCARE oder FOOD gibt; spätere Monate legt die Beitragserzeugung an (vorher legte die
  Folgeeinstufung CHILDCARE bis Jahresende im Voraus an). Bezahlte Monate werden nicht gesenkt, sondern als
  „Gutschrift prüfen“ gemeldet; unbezahlte Monate ohne Platzgeld (ab dem 3. Geburtstag) werden gelöscht
  (`removed`). Löschen einer Folgeeinstufung gleicht die Monate an die wieder offene Quelle an (Gutschrift-
  Fälle nur im Log); Löschen einer Einstufung ohne Quelle lässt die Beiträge stehen.
- API: `EinstufungResponse.expectationChanges` bei Anlegen und Ändern, `ChildcareExpectationSyncResult.removed`.
  Frontend: Karte „Beitragsanpassung“ nach jedem Speichern, wenn sich etwas geändert hat (blieb nach einer
  Folgeeinstufung vorher wegen des Neuladens nicht stehen).
- Abgleich auf der Live-Kopie: Alle erzeugten Monate stimmen mit den Einstufungen überein, bis auf zwei bezahlte
  August-Beiträge (zwei Kinder, Platzgeld vor der Einstufung aus Haushaltsdaten erzeugt, einmal mit anderen
  Betreuungsstunden); diese würden beim nächsten Speichern der Einstufung nur als Gutschrift-Prüfung gemeldet.

## Einstufungsbrief: versendete PDFs speichern, Versandhistorie (2026-10-12)

- Neue Tabelle `fees.einstufung_letters` (Migration `000047`): pro versendeter Mail das PDF genau wie
  angehängt (`BYTEA`, ca. 85 KB), Empfänger, Betreff, Dateiname, Absender, Bezug auf Einstufung, Kind,
  Haushalt und `email_logs`-Eintrag. Einstufung/Haushalt/Protokoll `ON DELETE SET NULL`, damit die Historie
  das Löschen einer Einstufung überlebt; Kind `ON DELETE CASCADE`.
- `POST /einstufungen/{id}/send` schreibt Protokoll, gespeicherten Brief und `letter_sent_at` nach dem
  Versand in einer Transaktion.
- `GET /einstufungen/{id}/letters`: alle Briefe zum Kind der Einstufung, neueste zuerst, ohne PDF-Daten.
  `GET /einstufungen/letters/{letterId}/pdf` (`?inline=1`): das gespeicherte PDF. Spätere Änderungen an der
  Einstufung ändern gespeicherte Briefe nicht.
- Frontend: Abschnitt „Versendete Briefe“ in der Karte „Brief und E-Mail an Eltern“ mit Datum, Absender,
  Empfängern, Betreff, Bezug („diese Einstufung“, „vorherige Fassung“, „andere Einstufung“) und PDF
  ansehen/herunterladen. Briefe aus der Zeit vor dieser Änderung (mailto, Druckvorlage) sind nicht enthalten.
- Alle App-Mails stehen weiterhin im E-Mail-Protokoll (Automatisierung), dort ohne Anhang.

## Einstufungsbrief: PDF-Download und Versand aus der App (2026-10-12)

- `GET /einstufungen/{id}/pdf` liefert den Brief als PDF (`Einstufung_<Jahr>_<Nachname>_<Vorname>.pdf`,
  `?inline=1` zum Ansehen im Browser). Der Brief zeigt bei einer Folgeeinstufung die Monate der Quelle als
  „(bisher)“. Grundlagentext nach Einkommensgrundlage und „vorläufig“ wie in `EinstufungPDF.vue`.
- Briefdatum: das Versanddatum, solange die Einstufung danach nicht geändert wurde (`updated_at <=
  letter_sent_at`), sonst heute. Neue Spalte `einstufungen.letter_sent_at` (Migration `000046`), im API als
  `letterSentAt`.
- Vorstand im Fuß aus `fees.board_terms` (am Briefdatum im Amt, Reihenfolge Vorsitz, Bauliches, Finanzen,
  Elternarbeit, Schriftführung, Personal, dann alphabetisch), Bezeichnung wie gepflegt. Bankverbindung aus den
  Zahlungseinstellungen der Erinnerungen (`reminder_payment_*`); der Bankname kommt aus einer kleinen
  BIC-Tabelle im Code (nur BFSWDE33XXX), bei unbekannter BIC bleibt er leer.
- `POST /einstufungen/{id}/send` (`subject`, `body`) schickt die E-Mail an alle Eltern-Adressen des Haushalts
  mit dem PDF als Anhang, protokolliert sie (`email_logs`, Typ `EINSTUFUNG`, Payload mit `einstufungId`) und
  setzt `letter_sent_at`. Ohne SMTP-Konfiguration 503 statt stillem Überspringen. Schlägt nach dem Versand
  das Speichern von Datum oder Protokoll fehl, meldet der Endpunkt einen Fehler, die Mail ist aber raus.
- Frontend: Karte „Brief und E-Mail an Eltern“ mit „PDF ansehen“, „PDF herunterladen“, Versandstatus und
  „E-Mail mit PDF senden“ (Bestätigung, Hinweis auf früheren Versand). Bei ungespeicherten Änderungen sind
  Brief und Versand gesperrt. Mit Folgeeinstufung beziehen sich Brief und Standardtext auf die neueste. Der
  `mailto`-Weg ist entfernt; die alte Browser-Druckvorlage bleibt als „Druckansicht (bisher)“. Liste:
  Spalte „Versendet“. E-Mail-Protokoll kennt den Typ „Einstufung“.
- Vereinsbeitrag bei Folgeeinstufungen: Hinweis nennt den bereits bezahlten Beitrag der Quelle; ohne Beitrag
  entfällt er (vorher „0,00 € jährlich – bereits bezahlt“, auch in der Druckansicht).
- `scripts/live-copy.sh restart` kopiert die Live-DB neu, wenn der Wegwerf-Container weg war (nach Docker-
  oder Rechner-Neustart startete er sonst leer).

## Einstufung: Einkommensgrundlage, vorläufig, Live-Vorschau (2026-10-11)

- Neues Feld `income_basis` (`INCOME` | `HIGHEST_RATE` | `NOT_REQUIRED`) ersetzt die Bedeutung von
  `highest_rate_voluntary` (bleibt als abgeleitetes Feld, wahr genau bei `HIGHEST_RATE`; die API akzeptiert es
  noch als Fallback). `annual_net_income` ist nullable und nur bei `INCOME` gesetzt (CHECK-Constraints).
  „Nicht erforderlich“ ist nur zulässig, wenn das Kind ab dem Wirksamkeitsmonat Kindergartenkind ist.
  Bei Nicht-`INCOME` wird die Einkommensaufstellung geleert.
- `provisional` („Vorläufig – Nachweise fehlen“): reine Kennzeichnung, ändert keinen Betrag; Liste und
  Ergebnis zeigen ein Badge, der Druck-Text erwähnt es.
- Werbungskosten-Pauschale wird höchstens bis zum Bruttolohn angerechnet (analog § 9a EStG; ohne
  Lohneinkommen keine negativen Einkünfte mehr). `calculate-income` liefert die angerechneten Beträge.
- Haushalt folgt der offenen Einstufung (ohne `validUntil`): `INCOME` → `PROVIDED` + Einkommen,
  `HIGHEST_RATE` → `MAX_ACCEPTED` ohne Einkommen, `NOT_REQUIRED` → `NOT_REQUIRED` ohne Einkommen, aber nur aus
  leerem, `PENDING` oder `NOT_REQUIRED` (Geschwister mit Einkommen behalten ihren Status). Pflegefamilien
  behalten ihren Status. Änderungen an abgeschlossenen Einstufungen lassen den Haushalt unberührt; Löschen
  einer offenen Folgeeinstufung wendet die Vorgängerin wieder an. Alles läuft in einer Transaktion.
- `POST /einstufungen/preview` rechnet ein ungespeichertes Formular exakt wie Speichern (neu: `childId` +
  `year`; Folge: `sourceEinstufungId` + `changeDate`; Bearbeiten: `einstufungId`) und liefert Einstufung inkl.
  Monatstabelle, Einkommensaufschlüsselung, `warnings` (Krippe ohne Einkommensangaben oder mit negativem
  Einkommen; vor dem Speichern muss bestätigt werden) und `notices` (Kindergartenkind mit Einkommen, gekappte
  WK-Pauschale). Die Seite fragt entprellt (400 ms) nach und zeigt „Vorschau – noch nicht gespeichert“, sobald
  das Formular vom gespeicherten Stand abweicht; der Aufruf von `calculate-income` aus der Seite entfällt.
- Fachliche Fehler kommen als `service.ValidationError` mit deutscher Meldung (400): Überschneidung
  (`ex_einstufungen_child_period`), Löschen mit Folgeeinstufung, ungültiges Folgedatum, Kind ohne Haushalt.
  `validFrom` und `careHoursPerWeek` im Create-Request sind veraltet: Beginn und Stunden kommen aus dem Kind.
- Migration `000045` korrigiert Bestandsdaten: Höchstsatz-Einstufungen → `HIGHEST_RATE` ohne Einkommen
  (bisher −3.000 € aus der WK-Pauschale), Kindergarten-Einstufungen ohne Einkommensangaben → `NOT_REQUIRED`
  mit Bemerkung „Einkommen unbekannt – nicht abgefragt (Kindergartenkind, beitragsfrei)“; Haushalte
  `MAX_ACCEPTED` ohne Einkommen, `PROVIDED` mit Einkommen ≤ 0 und nur `NOT_REQUIRED`-Einstufungen →
  `NOT_REQUIRED`. Kein Beitrag ändert sich (Höchstsatz ignoriert das Einkommen, betroffene Kinder sind
  beitragsfrei; Sollstellungen bleiben unberührt). Probelauf auf der Live-Kopie: 7 Höchstsatz, 5 nicht
  erforderlich, 6 mit Einkommen. Die Down-Migration stellt die Spalten, nicht die alten Werte wieder her.

## Lokale Live-Kopie zum Testen (2026-10-11)

- `scripts/live-copy.sh up|restart|refresh|down|status` startet den aktuellen Checkout (Frontend eingebettet)
  auf Port 18090 gegen eine Kopie der Live-DB in einem Wegwerf-Container (`kita-live-copy-db`,
  127.0.0.1:55490). Die Live-DB wird nur per `pg_dump` über SSH gelesen; die Migrationen des Checkouts laufen
  gegen die Kopie. Ohne SMTP- und Banking-Sync-Konfiguration (Server startet mit `env -i`), also kein Versand.
  Anmeldung mit dem normalen Live-Konto; JWT-Secret und Log unter `~/.local/state/kita-live-copy/`.
  `down` löscht die Kopie. Die Kopie enthält echte Familiendaten: nur im LAN, nicht weitergeben.

## Einstufungsbrief als Server-PDF, Mail-Anhänge (2026-10-11)

- Neues Paket `internal/letter`: rendert den Einstufungsbrief mit `codeberg.org/go-pdf/fpdf` nach der Vorlage
  `EinstufungPDF.vue` (A4, eingebettete Liberation Serif mit Lizenz, Logo). Es rechnet nichts: Zeiträume
  (`BuildFeePeriods` fasst gleiche Monate zusammen), Grundlagentext, Vorstand und Bankdaten kommen vom
  Aufrufer. Blocksatz ohne Silbentrennung: Zeilen, die zu weite Wortabstände bekämen, bleiben flatternd.
  Beispielbrief: `LETTER_SAMPLE_OUT=/tmp/brief.pdf go test ./internal/letter -run TestWriteSampleLetter`.
- `email.Service.SendTextEmailWithAttachments` verschickt Text-Mails mit Anhängen (multipart/mixed).
- Noch nicht angebunden: Endpunkte für Download und Versand folgen.

## Einstufung: Kindersuche und Vorbelegung (2026-10-11)

- Das Kind wird über ein Suchfeld gewählt (`ChildSearchSelect.vue`): ab zwei Zeichen Treffer aus Vorname,
  Nachname und Mitgliedsnummer, mehrere Suchwörter müssen alle vorkommen; ausgetretene Kinder stehen gedämpft
  am Ende. Gesucht wird clientseitig in der ohnehin geladenen Kinderliste; Pfeiltasten und Enter wählen.
- Bei Auswahl (auch über `?childId=`) wird das Formular neu vorbelegt: Jahr (Eintrittsjahr, wenn es in der
  Zukunft liegt), Kinderzahl = Maximum aus allen bekannten Kindern des Haushalts, `childrenCountForFees` und
  der Quell-Einstufung (Eltern können weitere unterhaltspflichtige Kinder haben; Hinweis im Formular),
  Höchstsatz bei Haushaltsstatus `MAX_ACCEPTED`. Quelle für Einkommen bzw. Höchstsatz ist die neueste
  Einstufung des Haushalts mit Einkommensangaben oder Höchstsatz; ein Banner nennt sie, „Verwerfen“ leert
  die übernommenen Werte. Bemerkungen werden nie übernommen.
- Hat das Kind schon eine offene Einstufung (`validUntil` leer), ist Speichern gesperrt; der Hinweis verlinkt
  die bestehende Einstufung und die Folgeeinstufung (die DB-Sperre `ex_einstufungen_child_period` würde eine
  zweite offene Periode ohnehin ablehnen). Ist das Kind ab Beginn im Kindergartenalter, weist die Seite auf die
  Beitragsfreiheit hin.
- Der Speichern-Knopf heißt jetzt „Einstufung speichern“ / „Folgeeinstufung speichern“ / „Änderungen
  speichern“: Er speichert sofort (inklusive Haushaltseinkommen, -status und Kinderzahl); einen zweiten
  Speicherschritt gibt es nicht. Jahresauswahl: Vorjahr bis Folgejahr relativ zum aktuellen Jahr.
- Einkommensfelder tragen ihre Beschriftung als `aria-label`, die Elternteil-Spalten sind Regionen
  („Elternteil 1/2“). E2E: `einstufung.spec.ts`.

## Backup-Wiederherstellung geprüft (2026-10-11)

- Die frühere Doku („kein App-Dump, VM zurückspielen“) war falsch: Der homelab-B2-Job sichert zweimal täglich
  einen validierten `pg_dump` der `kita`-DB; das Postgres-Datenverzeichnis ist vom Backup ausgeschlossen.
- Probe-Wiederherstellung des neuesten Snapshots aus B2 in einen isolierten Wegwerf-Container: erfolgreich,
  Zeilenzahlen und Migrationsstand wie live (bis auf einen Bankimport nach dem Snapshot). Live-DB nur gelesen,
  Container, Volume und Staging-Dateien danach gelöscht. Ablauf in `docs/deployment-homelab.md`; die
  Wiederherstellung in die Produktions-DB ist beschrieben, aber noch nicht geübt.

## Bankabgleich: serverseitige Listen (2026-10-09)

- `GET /import/transactions?status=offen|zugeordnet|warnungen|alle` bildet die bisherige
  zusammengeführte Liste vor Suche, Sortierung und Pagination in SQL ab. Ohne Status bleibt die
  bisherige Restbetragsliste erhalten; Teilzuordnungen gehören in der UI weiterhin zu „Zugeordnet“.
- Suche umfasst Zahler, Verwendungszweck und IBAN. Sortierfelder sind fest zugelassen;
  die Transaktions-ID stabilisiert die Seiten bei gleichen Sortierwerten.
- Warnungen unterstützen Suche, Sortierung und `transactionIds`; die UI lädt alle Warnungsseiten
  für die aktuelle Transaktionsseite. Statuszähler und Gesamtzahlen stammen vom Server.
- Suche wird um 300 ms entprellt; veraltete Antworten werden verworfen, leere Folgeseiten korrigiert.
  Integrationstests decken Suche, Sortierung, Pagination und überlappende Status ab.
- OpenAPI und Frontend-Schema werden im nachfolgenden Generierungsschritt aktualisiert.

## Einkommensvorschau über das Backend (2026-10-09)

- Die Einstufungsseite berechnet die Einkommensvorschau über `POST /einstufungen/calculate-income`;
  Eingabeänderungen werden um 300 ms entprellt. Die Backend-Ergebnisse inklusive der abweichenden
  Rundung sind maßgeblich; eine zusätzliche Rundung im Frontend entfällt.
- Der letzte Vorschauwert bleibt während neuer Anfragen und bei Fehlern sichtbar. Fehler erscheinen
  inline im Vorschau-Block, auch wenn noch kein Ergebnis vorliegt. Veraltete Antworten werden verworfen;
  beim Verlassen der Seite werden ausstehende Timer und Antworten ungültig.
- Bei freiwilligem Höchstsatz entfällt die Vorschau weiterhin. Speichern und PDF bleiben unverändert.

## Fälligkeitsanzeige nach Berliner Datum (2026-10-09)

- Die Beitragslisten und Kinddetails vergleichen den Datumsanteil des Fälligkeitsdatums mit
  `todayISO()`. Ein Beitrag gilt am Fälligkeitstag selbst noch nicht als überfällig; maßgeblich ist
  der Kalendertag in Europe/Berlin.

## Kinderimport: atomare CSV-Zeilen (2026-10-09)

- Jede Importzeile läuft über `TxManager.WithTx`; Fehler rollen Kind, Eltern, Verknüpfungen und
  Stundenhistorien dieser Zeile zurück. Andere Zeilen werden weiterverarbeitet.
- Fehlermeldungen bleiben erhalten. Die Ergebniszähler (angelegt/aktualisiert/verknüpft) zählen nur
  übernommene Zeilen; Schritte einer zurückgerollten Zeile zählen nicht mit.
- Repositoryinterne Transaktionen übernehmen die Transaktion aus dem Kontext. Verschachteltes
  `WithTx` übernimmt ebenfalls die äußere Transaktion, ohne Savepoint oder eigenen Commit.
- Integrationstest: Nicht vorhandene Eltern-ID erzwingt einen Linkfehler nach Kind- und Elternanlage;
  die fehlgeschlagene Zeile hinterlässt keine Daten, erfolgreiche Zeilen davor und danach bleiben erhalten.

## Fachliche Zeitstempel über Testuhr (2026-10-09)

- Persistierte Zeitpunkte und Refresh-Token-Abläufe verwenden `util.Now()` und folgen damit der
  Europe/Berlin-Uhr sowie `util.SetClock` in Tests. Echtzeit bleibt nur bei JWT-Ausstellung/-Prüfung
  (Bibliotheksvalidierung) und Latenzmessung im HTTP-Logging.

## Erinnerungen: Erinnerung und Mahnung zusammengelegt, Mahngebühr nur per Häkchen (2026-10-05)

- **Eine Aktion statt Erinnerung/Mahnung:** Die Stufenwahl entfällt (`ReminderCaseRequest.stage` und
  `ReminderCasePreview.recommendedStage` sind entfernt). Jeder Beitrag kann beliebig oft erinnert werden.
  Eine Mahngebühr entsteht nur noch für Beiträge in `reminderFeeIds` (Teilmenge von `feeIds`, Standard leer).
  Nichts wird automatisch erhoben, auch wenn die Frist nach einer Erinnerung abgelaufen ist (Kulanz).
- **Regel für das Häkchen:** `ReminderCaseFee.reminderFeeDue` ist wahr für Grundbeiträge mit Status
  `actionable_final` (erinnert, Frist abgelaufen) ohne vorhandene Mahngebühr, auch keine bezahlte, und
  mit konfiguriertem Gebührenbetrag. Mahngebühren selbst werden nicht gemahnt, und je Beitrag gibt es
  höchstens eine Mahngebühr.
  - Häkchen außerhalb der Auswahl → `400`.
  - Häkchen an einem nicht fälligen Beitrag → `409` (`CaseConflictError`).
- **Mailart:** Wird mindestens eine Mahngebühr angelegt, ist die Mail eine Mahnung (`preview.stage = final`,
  Log `REMINDER_FINAL`), sonst eine Zahlungserinnerung (`initial`). Der Schlusssatz der Erinnerung lautet
  neutral „Falls die Zahlung bis zum … nicht eingeht, können für die offenen Beiträge Mahngebühren
  erhoben werden.“
- **Hinweise:** Ist eine Mahngebühr fällig, aber nicht angekreuzt, nennt die Vorschau „<Kind>: <Beitrag> –
  Mahngebühr laut Regeln fällig, wird nicht erhoben“.
- **UI:**
  - Unter dem Status steht die Checkbox „Mahngebühr erheben“, aber nur bei fälligen Beiträgen. Sie ist nur
    aktiv, wenn der Beitrag ausgewählt ist. Die zuerst gebaute eigene Spalte zeigte meist nur „—“ und
    machte die Tabelle breiter als das Panel; sie ist entfallen.
  - Eine Mahngebühr direkt unter ihrem Grundbeitrag hat keine Trennlinie zu ihm und wiederholt den Namen
    nicht. Die Einrückung `pl-4` ist entfallen, weil sie Namen und Labels versetzt hat.
  - Ein Hinweisbalken erklärt nicht angekreuzte fällige Gebühren.
  - Der Button heißt je nach Häkchen „Erinnerung senden“ oder „Mahnung senden“.
  - Statuslabels: „Nicht erinnert“ (statt „Erinnerung fällig“) und „Frist abgelaufen“ (statt „Mahnung fällig“).
- **Entfernt:**
  - die ungenutzten Sammel-Endpunkte `POST /fees/reminders/run` und `/fees/membership-reminders/run`
    samt `reminder_core.go`, Legacy-Mailbuildern, Repo-Methoden `ListUnpaidWithoutReminder*`/
    `ListUnpaidByTypesDueOnOrBefore` und ihren Tests;
  - die Einstellung `autoEnabled`; Migration `000044` löscht `reminder_auto_enabled`.
  - Live war die Automatik aus, und kein Scheduler rief die Endpunkte auf.
  - Erinnerungen laufen nur noch über `/fees/reminder-cases/{householdId}/preview|send`.
- Die Regel „Mahngebühr erst nach Erinnerung“ aus dem Eintrag vom 2026-10-04 gilt weiter. Die Mahnung
  wählt aber nicht mehr selbst aus, welche Beiträge eine Gebühr bekommen; das entscheidet das Häkchen.

## Erinnerungen: Mahngebühren nur nach Erinnerung, Bezug sichtbar, Mail gruppiert (2026-10-04)

- **Regel:** Eine Mahnung legt nur noch für Beiträge mit Status `actionable_final` eine Mahngebühr an,
  also wenn vorher eine Erinnerung verschickt wurde und deren Frist abgelaufen ist (`prepareCasePlan`).
  Weitere ausgewählte Beiträge stehen in der Mahnung, bekommen aber keine Gebühr; die Hinweise nennen
  sie mit „– keine Mahngebühr“ (noch nicht fällig, noch nicht erinnert, Frist läuft, unbekannte Historie).
  Auslöser: eine Mahnung mit dem noch nicht fälligen Essensgeld Oktober hätte dafür 10 € angelegt.
  Beiträge mit unbekannter Historie (angelegt bis `reminder_history_reliable_from`) bekommen damit
  ebenfalls keine automatische Mahngebühr mehr; erst erinnern, dann mahnen.
- **Bezug der Mahngebühr:** `ListOpenByHousehold` lädt den Grundbeitrag (`base_fee_type/year/month`) und
  das Vereinsmitglied (`COALESCE(fe.member_id, base.member_id)`) mit. `ReminderCaseFee` hat dafür
  `reminderForId`, `reminderFor` (`domain.FeeRef`) und `clubMember {id, memberNumber, name}`; die Liste
  stellt jede Mahngebühr direkt hinter ihren offenen Grundbeitrag (`placeRemindersAfterBase`).
- **UI Arbeitsliste:** Spalte „Kind/Mitglied · Beitrag“: Vereinsbeiträge (und ihre Mahngebühren) zeigen
  das Vereinsmitglied mit Link auf `/mitglieder/{id}` statt des Kindes; eine Mahngebühr unter ihrem
  Grundbeitrag ist eingerückt, Zeitraum zeigt „für Essensgeld 9/2026“.
  `plannedReminderFees[].baseLabel` enthält Zeitraum und Person, z. B. „Essensgeld September 2026 (Haily)“.
- **Mailtext** (`familyReminderItemList`): ein Block pro Kind („Haily (Mitgliedsnr. 11090):“) und pro
  Vereinsmitglied („Vereinsmitglied Nadja Moritz:“, ohne M-Nummer, da die Bankzuordnung über die
  Kindernummer läuft); Mahngebühren eingerückt als „zzgl. Mahngebühr — 10,00 EUR“ unter ihrem
  Grundbeitrag, ohne ausgewählten Grundbeitrag als „Mahngebühr für Essensgeld September 2026“; am Ende
  „Gesamtbetrag“. Der SEPA-QR-Verwendungszweck bleibt unverändert (Kindernummern).
  Der Hinweis „Wichtig: Bitte gebt als Empfänger genau … an“ ist aus allen Erinnerungsmails entfallen.
- **Datenprüfung live (read-only):** alle offenen Vereinsbeiträge haben ein Mitglied; die 14
  Vereinsbeiträge 2026 ohne `member_id` gehören ausgetretenen Familien ohne Mitglieder und sind bezahlt.
  Jeder Haushalt mit aktivem Kind hat ein aktives Mitglied.

## Erinnerungen: Kind verlinkt (2026-10-04)

- **Arbeitsliste:** In der Beitragstabelle des Familienfalls ist der Kindname jetzt ein Link auf
  `/kinder/{childId}` (Kinddetail), wie auf der Notizen-Seite. `childId` lieferte `ReminderCaseFee` bereits.

## Esc schließt Dialoge, Dauer-Auswahl für Elternstunden, deutsche Validierung (2026-09-30)

- **Esc:** `utils/dialogEscape.ts` (in `main.ts` installiert) schließt app-weit den obersten offenen Dialog
  (`aria-modal` oder Overlay `fixed inset-0`). Er klickt dessen Schließen-Element: `[data-dialog-close]`, einen
  Button mit `aria-label` „Schließen“ oder einen Button „Abbrechen“/„Schließen“. Neue Dialoge brauchen eines davon.
  Reine Icon-Buttons (X) zum Schließen haben dafür `aria-label="Schließen"` bekommen. Die älteren Esc-Handler
  einzelner Seiten bleiben; der globale Handler läuft vorher und stoppt das Ereignis, damit nicht zwei Dialoge
  auf einmal zugehen.
- **Dauer:** `components/parent-work/DurationPicker.vue` ersetzt das Stunden-Zahlenfeld in „Stunden melden“
  (Eltern) und „Stunden erfassen/bearbeiten“ (Verwaltung). Nach Rückmeldung (Stunden-/Minuten-Kacheln wirkten
  unruhig) ist es ein Textfeld mit ±15-Minuten-Knöpfen, das „2,5“, „2:30“, „2h30“ oder „150 min“ versteht und
  daneben „2 Std. 30 Min.“ ausschreibt, plus Schnellwahl-Chips 0,5/1/1,5/2/2,5/3/4 Std. (2,5 h = ein Klick).
  Anlehnung an übliche Zeiterfassungs-Muster: ein freies Feld mit toleranter Eingabe statt getrennter
  Stunden/Minuten-Picker. Werte bleiben Minuten in Viertelstunden (sonst Hinweis am Feld).
- **Validierung auf Deutsch:** `utils/germanValidation.ts` (in `main.ts`) setzt für alle nativen
  Pflichtfeld-/Format-Meldungen deutsche Texte („Bitte dieses Feld ausfüllen.“ usw.), weil die Browser sonst in
  ihrer Oberflächensprache melden. Vor jedem Absenden werden die Meldungen zurückgesetzt, damit per Code gesetzte
  Werte nicht blockieren.
- **Stunden melden (Eltern):** Die optionalen Felder Mitglied und Kind sind entfallen; die API akzeptiert sie
  weiterhin, die Eltern-Oberfläche sendet sie nicht mehr.

## Einladung auch für bestehende Konten (vorübergehend), kompakte Bankabgleich-Tabelle (2026-09-30)

- **Einladen mit bestehendem Konto (vorübergehend):** Damit Stefan die Einladung auf Production testen kann,
  ohne Daten zu löschen, liefert `GET /users/invitation-candidates` auch Eltern mit aktivem Kind, deren eigenes
  verknüpftes, aktives `PARENT`-Konto schon existiert (`userId` gesetzt). Andere Konten mit derselben E-Mail
  oder Eltern-Verknüpfung schließen weiter aus. `POST /users/invitations` setzt für sie `invitation_pending`
  wieder und schickt einen neuen Passwort-Link; bis das neue Passwort gesetzt ist, ist die Anmeldung gesperrt.
  Schlägt der Mailversand fehl, wird `invitation_pending` zurückgesetzt, das alte Passwort gilt weiter. Im
  Dialog steht ein Hinweis, und „Alle auswählen“ nimmt diese Eltern nicht mit. Nach dem Test wieder entfernen
  (`Candidates` in `account_invitation_repository.go`, `reinviteOne` im Service).
- **Bankabgleich:** Die Tabelle passt ab 1280 px ohne horizontales Scrollen: Aktionen stehen untereinander statt
  nebeneinander, Zellen sind schmaler gepolstert, die Beschreibung wird früher gekürzt.

## Meine Daten nur lesbar bis auf E-Mail/Telefon, Frist bei Beiträgen, Einladungen nur aktive Familien (2026-09-30)

- **Entscheidung (Stefan):** Eltern ändern selbst nur noch die eigene E-Mail (= Login, bleibt mit
  `fees.users.email` synchron) und Telefonnummern (eigene und die des anderen Elternteils). Namen, Geburtsdatum
  und Adressen von Eltern und Kindern sind nur lesbar; Korrekturen laufen über „Fehler melden“ (Kontakt ohne
  Bezug, Kind mit Bezug). So landen keine falschen Stammdaten (z. B. Geburtsdatum → U3/Ü3) ungeprüft im System.
- **Backend:** `PUT /me/children/{id}` ist entfernt. `PUT /me/contact` und `PUT /me/parents/{id}/contact`
  akzeptieren nur `email` und `phone`; andere Felder → 400 „Unbekanntes Kontaktfeld“. Alte Kind-Änderungen in
  `fees.data_changes` bleiben im Feed und in der Historie sichtbar.
- **Meine Daten:** eine Karte pro Person, auf dem Desktop zwei nebeneinander (Eltern und Kinder). Hinweise
  stehen direkt am E-Mail-Label („gleichzeitig dein Login!“ bzw. „kann nicht von dir geändert werden …“).
  Gleiche Adresse aller Personen erscheint einmal als Karte „Adresse der Familie“, sonst je Person als Text.
  Die Bearbeitung der Familienadresse und „Adresse übernehmen“ aus dem Eintrag darunter sind damit entfallen.
- **Beiträge (Eltern):** Offene Beiträge zeigen neben dem Status die Frist („noch 5 Tage“, „heute fällig“,
  „seit 3 Tagen überfällig“, überfällig in Rot), gerechnet mit dem Berliner Datum.
- **Eltern einladen:** `GET /users/invitation-candidates` liefert nur Eltern mit mindestens einem aktiven Kind
  (`is_active` und kein Austrittsdatum vor heute, Berliner Datum) und dazu `children` (Namen der aktiven Kinder),
  die der Dialog unter jedem Elternteil zeigt. Eltern ehemaliger Kinder lassen sich damit auch per
  `POST /users/invitations` nicht mehr einladen („Elternteil ist nicht verfügbar“).
- **Bankabgleich:** Status-Chips der Transaktionen umbrechen nicht mehr („Nicht zugeordnet“ einzeilig).

## Navigation, Seitenhintergrund, Meine Daten mit Familienadresse (2026-09-30)

- **Navigation:** `isActive` vergleicht ganze Pfadsegmente (`/eltern` passt nicht mehr auf `/elternstunden`);
  vorher leuchtete „Verwaltung“ auf der Elternstunden-Übersicht mit.
- **Seitenhintergrund (hell):** `--background` ist ein neutrales Hellgrau (`244 245 241`) statt Hellgrün, damit
  sich die weißen Karten und Tabellen abheben. Grün bleibt im Header, in Tabellenköpfen und Akzenten.
- **Meine Daten:** Alle Felder in zwei gleich breiten Spalten (max. `2xl`) statt 12-Spalten-Raster mit
  unterschiedlichen Breiten; Stammdaten der Kinder (Mitgliedsnummer, Eintritt …) als Liste untereinander.
- **Familienadresse:** Die Adresse bleibt pro Elternteil und Kind gespeichert (kein Datenmodell-Umbau: Admin-Seiten,
  Einstufung und Import arbeiten mit der Adresse pro Person). Haben alle Personen der Familie dieselbe, nicht leere
  Adresse (Vergleich ohne Groß-/Kleinschreibung und Leerzeichen am Rand), zeigt die Seite sie einmal als
  „Adresse der Familie“; Speichern schreibt sie über die bestehenden `/me/**`-Endpunkte an alle (jede Änderung
  wird wie bisher in `fees.data_changes` protokolliert). „Abweichende Adresse für einzelne Personen“ blendet die
  Felder pro Person wieder ein. Weichen Adressen ab, übernimmt „Adresse übernehmen“ die eigene Adresse in die
  Karte des anderen Elternteils oder eines Kindes (Speichern bleibt ein eigener Schritt).

## Eltern-Ansicht: Beiträge, Meine Daten, Mahngebühr-Bezug (2026-09-29)

- **E-Mail des anderen Elternteils** ist für Eltern immer schreibgeschützt, auch ohne dessen Login
  (`ParentAccountService.ParentContact` lehnt Änderungen ab; ändern können sie nur die Person selbst oder der
  Vorstand). `hasLogin` ist aus `GET /me` entfernt.
- **Beiträge (`/familie/beitraege`):** kompakte Tabelle mit farbigem Chip je Beitragsart; die Karte „Offen“
  filtert auf offene Beiträge („Alle anzeigen“ hebt den Filter auf). Bezahlte Beiträge zeigen das Buchungsdatum
  der letzten zugeordneten Überweisung (`paidAt` in `GET /me/fees`, nur wenn vollständig bezahlt).
- **Mahngebühr am Beitrag:** `GET /me/fees` liefert `reminderForId` und Art/Jahr/Monat des Grundbeitrags; eine
  Mahngebühr zählt zum Jahr ihres Grundbeitrags und steht direkt darunter. In der Admin-Beitragsliste liefert
  `GET /fees` `reminderFor` (Art, Jahr, Monat); die Spalte Zeitraum zeigt „zu Essensgeld September 2026“
  (`getFeePeriodLabel`, auch in Kinderdetails für Mahngebühren ohne sichtbaren Grundbeitrag). Die
  Erinnerungsmails nannten den Grundbeitrag bereits.
- **Meine Daten:** gemeinsames 12-Spalten-Raster (Adressfelder bündig über alle Karten), niedrigere Felder,
  Stammdaten der Kinder in einer Zeile; der Hinweis „Änderungen daran bitte über Fehler melden“ ist entfernt.
- **Elternstunden:** Kita-Jahre ab 2025/26 (erste Regel gilt ab 01.08.2025) bis zum nächsten Jahr.
- **Übersicht:** Untertitel unter „Hallo …!“ entfernt.

## Suchfelder, Benutzerliste, Austrittsdatum für mehrere Kinder (2026-09-29)

- **Einheitliche Suchfelder:** Alle Suchen nutzen `components/SearchInput.vue` (Lupe links, optionaler
  Ladeindikator, `size="sm"` für die kompakte Suche auf der Beitragsseite). `class` geht an den Wrapper, alle
  übrigen Attribute und Listener an das `<input type="search">`. Die kopierten Utility-Klassen der einzelnen
  Seiten sind entfernt; auch die Elternstunden-Suchen (Übersicht, Familienauswahl, Vorstand) sehen jetzt so aus.
- **Elternstunden-Übersicht:** Untertitel entfernt; die Suche steht über der Tabelle statt in der Karte.
- **Benutzer:** Freitextsuche (Name, E-Mail, Elternteil), Rollenfilter als Dropdown und sortierbare Spalten
  (`useTableSort` + `SortTh`, clientseitig).
- **Kinder, Sammelaktion „Austrittsdatum setzen“:** setzt für alle ausgewählten Kinder dasselbe Austrittsdatum
  (Vorschlag: 31.07. des laufenden Kita-Jahres) über `PUT /children/{id}` mit nur `exitDate`; ein vorhandenes
  Datum wird überschrieben. Liegt das Datum vor dem Eintritt eines ausgewählten Kindes, ist Speichern gesperrt.
  Kein neuer Backend-Endpunkt.

## Eltern gesammelt einladen (2026-09-29)

- Seite „Benutzer“ → „Eltern einladen“: Der Dialog listet Eltern mit Kontakt-E-Mail ohne Konto (weder per
  `parent_id` verknüpft noch Login-E-Mail gleich, ohne Groß-/Kleinschreibung). Mehrere Eltern mit derselben
  E-Mail erscheinen einmal und ausgegraut, weil die Verknüpfung nicht eindeutig wäre. Für jede Auswahl
  entsteht ein `PARENT`-Konto über dieselbe Verknüpfungslogik wie bei der Einzelanlage, ohne Passwort.
  Das Ergebnis kommt je Elternteil (Teilerfolg möglich); ohne SMTP wird kein Konto angelegt.
- Konten mit `fees.users.invitation_pending` (Migration `000043`) können sich nicht anmelden; der Login
  prüft dabei gegen den Dummy-Hash, damit der Zustand nicht über die Antwortzeit sichtbar wird. Die Liste
  zeigt „Einladung ausstehend“ und „Einladung erneut senden“ (neuer Token, alter ungültig). Ein
  Admin-Passwort-Reset beendet den Zustand ebenfalls.
- Token: 32 Zufallsbytes (hex), in `fees.account_invitations` nur als SHA-256-Hash, ein Token je Konto,
  7 Tage gültig, einmal verwendbar (Löschen und Passwort setzen in einer Anweisung). Link:
  `APP_BASE_URL` + `/passwort-setzen?token=…`; `APP_BASE_URL` enthält den Basispfad (Default
  `http://localhost:5175/beitraege`, live `https://kita.remer.cc/beitraege` in der Homelab-Compose).
  `POST /auth/invitation-password` ist öffentlich und steht unter einer eigenen `LoginLimiter`-Instanz.
  Danach landet man auf dem Login mit Erfolgshinweis.
- Die Mail (du-Form, wie die Erinnerungen) wird als `ACCOUNT_INVITATION` im E-Mail-Log gespeichert, der Link
  darin als `token=[ausgeblendet]`.
- API (`ADMIN`): `GET /users/invitation-candidates`, `POST /users/invitations`,
  `POST /users/{id}/invitation`. Umgesetzt von Codex (gpt-6-sol), danach geprüft und nachgebessert
  (Login-Timing, Mailtext).

## Navigation, Eltern-Änderungen, Bankabgleich, Jahresübersicht (2026-09-29)

- **Hauptmenü als Split-Button:** Der Gruppenname verlinkt auf die erste Unterseite der Gruppe (z. B.
  „Verwaltung“ → Kinder), der Pfeil rechts neben einem senkrechten Strich klappt alle Unterseiten auf
  (`aria-label` „<Gruppe>: alle Seiten“). Gruppen mit nur einer Seite bleiben einfache Links, das mobile
  Menü ist unverändert.
- **Änderungen von Eltern:** eigene Seite `/aenderungen` (Admin, Menü „Verwaltung“) mit bis zu 100 Einträgen
  aus `GET /activity`. Das Dashboard zeigt nur noch eine Karte „x neue Änderungen von Eltern“. „Neu“ heißt:
  jünger als der letzte Besuch der Seite; der Zeitpunkt liegt pro Browser und Konto im `localStorage`
  (`kita-activity-seen:<userId>`), kein Serverzustand. Neue Einträge tragen auf der Seite ein „neu“-Badge.
- **Bankabgleich:** Die Hinweisbox „Letzter Lauf erfolgreich abgeschlossen.“ entfällt, das Status-Badge sagt
  das schon. Fehler (`lastError`) stehen weiter in einer Box, jetzt rot; die 2FA-Aufforderung bleibt gelb.
- **Jahresübersicht im Dashboard:** Beim Jahreswechsel blieben nach einer fehlgeschlagenen Anfrage (Fehler
  wurden verschluckt) die Zahlen des vorherigen Jahres unter dem neuen Jahres-Label stehen. Jetzt landet nur
  die Antwort zum zuletzt gewählten Jahr (Zähler gegen Reihenfolge-Rennen), ein Fehler zeigt „konnte nicht
  geladen werden“ mit „Erneut versuchen“, und die Jahresdaten laden unabhängig vom übrigen Dashboard.
  Live-API und Live-Frontend lieferten beim Nachtest für 2025/2026/2027 korrekte Daten.

## Erinnerungen und globale Flächen (2026-09-29)

- Seitenhintergrund bleibt `--background` (`rgb(245 251 233)`). Formularfelder (`input`, `select`, `textarea`)
  liegen per Grundregel in `main.css` jetzt auf `bg-card` (hell: weiß), nicht mehr auf `bg-background`.
- Alle App-Tabellen erhalten per Grundregel Zebra-Streifen (`--muted` mit 45 % Deckkraft auf geraden
  Zeilen). Ausgenommen sind die PDF-Tabellen `.data-table`/`.fee-table`. Zeilenklassen wie `hover:` oder
  Statusfarben gewinnen weiter, weil sie Utilities sind.
- Erinnerungen: Die Karten der Arbeitsliste sind weiß. „Nächste Aktion“ steht relativ da („in 3 Tagen“,
  „seit 5 Tagen fällig“, `formatDueIn` in `utils/format.ts`), das Datum als Tooltip. „Neu laden“ hat ein
  Icon (dreht sich beim Laden). Der QR-Inhalt steht dauerhaft neben dem QR-Code statt in einem
  aufklappbaren Bereich. Die Tabelle im Versandverlauf steckt in einer Karte mit Kopfzeile.

## Elternstunden-Überarbeitung (2026-09-29)

- **Vorstand nur für Admins:** `/parent-work/board-terms` (GET/POST/PUT/DELETE) ist jetzt auf `ADMIN` beschränkt
  (Routertest `TestRouter_RoleAreas`); `PARENT_WORK` sieht Menüpunkt und Seite `/elternstunden/vorstand` nicht mehr.
  Die Amtszeiten einer Familie erscheinen weiter in der Detailansicht, dort aber nur, wenn es welche gibt.
- **Verwaltung:** „Regeln“ und „Import“ sind eine Seite (`/elternstunden/verwaltung`, Komponenten `RulesSection`/
  `ImportSection`), für `ADMIN` und `PARENT_WORK`; Regeln schreiben bleibt `ADMIN`. Die alten Pfade
  `/elternstunden/regeln` und `/elternstunden/import` gibt es nicht mehr. Der Regelteil erklärt Tertiale
  (1.8.–30.11., 1.12.–31.3., 1.4.–31.7.), die Soll-Berechnung und die Befreiung durch Vorstandstätigkeit.
- **Übersicht:** Kennzahlen-Karten sind zugleich Filter (Klick schaltet um, Suche wirkt zusätzlich): Familien (alle),
  „Alle Stunden geleistet“ (`openMinutes <= 0`), „Stunden noch offen“ (`> 0`), „Noch keine Stunden geleistet“
  (offen und `doneMinutes = 0`, zählt nur bestätigte Stunden) und „Unbestätigte Meldungen“ (Anzahl Meldungen,
  Filter auf Familien mit Meldungen). Soll/Ist/Offen/Fehlbetrag-Summen, „Nur offene“, „Nur mit Meldungen“ und
  die Spalte Fehlbetrag sind entfallen. Jede Zeile hat einen „+“-Button, der den Erfassungsdialog mit
  vorgewählter Familie öffnet. Tabellen sind spaltenweise sortierbar (`useTableSort`, `SortTh`); Übersicht und
  Einträge der Detailseite.
- **Erfassungsdialog** ohne Felder „Mitglied“ und „Kind“. Beim Bearbeiten bleiben vorhandene Werte (z. B. aus
  dem Import) unverändert erhalten.
- **Detailseite:** Konto als Karten (Soll, Geleistet, Übertrag Vorjahr, Offen, Übertrag Folgejahr); der
  Fehlbetrag in Euro erscheint als sechste Karte nur für `ADMIN`. Das ist eine reine Oberflächenregel: die API
  liefert `missingAmountCents` weiterhin auch an `PARENT_WORK`.

## Impersonation durch Admins (2026-09-29)

- `POST /users/{id}/impersonate` (nur `ADMIN`) liefert ein Access-Token, das sich exakt wie der Zielbenutzer
  verhält (Rolle, Daten, Menüs). Ziel darf kein Admin (auch nicht man selbst) und nicht deaktiviert sein,
  damit es weder Rechteausweitung noch Ketten gibt. Das Token trägt den Claim `imp` (Admin-ID), hat die
  normale Laufzeit (15 Min) und gibt es nur als Access-Token: der Refresh-Cookie bleibt immer der des Admins.
- Während der Impersonation liefert `POST /auth/change-password` 403; Admin-Routen sind für das Token
  ohnehin gesperrt (Rolle des Ziels). Start wird per `slog` protokolliert.
- Migration `000042`: `data_changes.impersonated_by`. Änderungen an Eltern-/Kinderdaten während der
  Impersonation stehen mit `user_id` = Zielbenutzer und `impersonated_by` = Admin im Audit (in der API als
  `impersonatedBy` an `DataChange`, in der Oberfläche noch nicht angezeigt). Elternstunden-Einträge und
  Meldungen tragen weiterhin nur den Zielbenutzer.
- UI: Auf der Benutzerseite sind alle Zeilenaktionen Icon-Buttons (`aria-label`/`title`); der
  Impersonate-Button (`VenetianMask`, „Als diesen Benutzer anmelden“) erscheint nur bei aktiven Nicht-Admins.
  Start/Ende laden die Seite komplett neu, damit keine Daten der anderen Identität im Speicher bleiben.
  Der Modus überlebt Reloads über den `sessionStorage`-Marker `fees_impersonating` (Ziel-ID, pro Tab);
  `initialize()` holt per Refresh das Admin-Token und tauscht es erneut. Läuft das Token ab, tauscht
  `api.setTokenExchange` das erneuerte Admin-Token vor dem Retry gegen ein frisches Impersonation-Token;
  scheitert das (Ziel deaktiviert), wird komplett abgemeldet.
- Header: amberfarbener Indikator „Impersonation: {Admin} als {Benutzer}“ (`data-testid="impersonation-indicator"`),
  im Benutzermenü „Impersonation beenden“ (`stop-impersonation`) statt „Passwort ändern“.
- Tests: `internal/service/impersonation_integration_test.go`, `internal/auth/jwt_test.go`,
  Playwright `e2e/beitraege/impersonation.spec.ts`.

## Eltern-Zugang: Rückmeldungen aus dem ersten Test (2026-09-27)

- Migration `000041`: `parent_reports.response` (optionale Antwort des Admins beim Erledigen,
  `POST /parent-reports/{id}/resolve` mit `{response}`) und `parent_work_entries.reviewed_by/_at`.
  `updated_by` reicht nicht als Nachweis der Freigabe, weil jede spätere Bearbeitung ihn überschreibt;
  `reviewed_*` setzt nur Bestätigen/Ablehnen, Bearbeiten übernimmt die Werte. Bereits geprüfte
  Eltern-Meldungen wurden aus `updated_by/updated_at` nachgetragen (beste verfügbare Angabe).
  Die Familienseite (Staff) zeigt „von {Name}, {Datum}“; Eltern bekommen den Namen nicht.
- Meldungen sehen beide Elternteile des Haushalts (mit Absender, Status und Antwort), sonst nur Admins.
  Die Übersicht zeigt offene Meldungen und Antworten der letzten 30 Tage, „Meine Daten“ alle.
  Im Admin-Feed tragen erledigte Meldungen und geprüfte Stunden einen Haken und ihren Status.
- Eltern dürfen die Kontaktdaten des anderen Elternteils im Haushalt ändern
  (`PUT /me/parents/{id}/contact`, Audit mit handelndem Elternteil). Die E-Mail des anderen bleibt
  gesperrt, sobald sie dessen Login ist (sonst könnte ein Elternteil das fremde Login übernehmen).
  `/me` liefert `otherParents` deshalb mit Kontaktdaten und `hasLogin`. Der Feed nennt bei Kind- und
  Fremdänderungen, wessen Daten geändert wurden; Feldnamen sind deutsch (`street_no` → Hausnummer).
- Beiträge der Eltern: neueste zuerst, Jahresauswahl ab 2025 (davor gibt es keine Daten im Portal),
  seit 2026-10-31 bis zum Folgejahr.

## Eltern-Zugang, Etappe 2b (2026-09-27)

- `PARENT` nutzt `/familie/**` mit direktem Menü für Übersicht, Beiträge, Elternstunden und Daten.
  Rollen-Guards leiten Eltern von Staff-Routen und Staff aus dem Familienbereich um. Eltern sehen
  eigene Kinder und Beiträge, melden Stunden und Fehler, ziehen eingereichte Stunden zurück und
  ändern Kontakt- und Kinderdaten. Eine E-Mail-Änderung lädt `/auth/me` neu.
- Staff sieht offene Elternstunden-Meldungen in der Übersicht und kann sie auf der Familienseite
  bestätigen oder mit Pflichtgrund ablehnen. Admins sehen Eltern-Aktivitäten auf dem Dashboard,
  Meldungen unter `/meldungen`, die Verknüpfung auf der Benutzer-Seite und Änderungsverläufe bei
  Elternteil und Kind. Der Feed lädt unabhängig von den übrigen Dashboard-Karten.
- Frontend-Typen kommen vollständig aus `schema.d.ts` (Rollen-Enum inkl. `PARENT` in allen Antworten).

## Beiträge-Design, Etappe 1a (2026-09-27)

- Neues Design-System mit Homepage-Grün, selbst gehostetem Nunito, heller grüner
  Kopfleiste, Logo, Gruppenmenü und zentriertem Inhalt. Login und Dashboard verwenden semantische Farben
  und weichere Karten. Auf schmalen Bildschirmen stehen alle Menüeinträge im ausklappbaren Panel.
- Der Theme-Schalter wechselt zwischen System, Hell und Dunkel, speichert die Wahl unter `kita-theme` und
  setzt die Dunkelklasse vor dem ersten Rendern. Dunkle Grund- und Kartenfarben wurden aus der
  Homepage-Palette abgeleitet: `#243329`, `#2c3d30`, `#354836`, Text `#dcebd5`.
- `USER` ruft den Admin-Endpunkt für den Bank-Sync-Status nicht mehr auf. Dashboard-Anfragen scheitern
  unabhängig voneinander; Erinnerungen und die Banking-Sync-Karte sind Admins vorbehalten.
- Nunito kommt aus `@fontsource-variable/nunito` (im Bundle, kein externer Font-Server).
- Neutrale Flächen und Texte verwenden semantische Tokens; Statusfarben erhalten passende `dark:`-Varianten.
  Formularfelder nutzen eine Grundregel für `bg-card` (bis 2026-09-29 `bg-background`), `text-foreground` und `border-input`.

## Eltern-Zugang, Etappe 2a (2026-09-27)

Backend umgesetzt; Eltern-Oberfläche in Etappe 2b ergänzt.

- Rolle `PARENT`: Verknüpfung beim Anlegen und Rollenwechsel zu `PARENT` über eine eindeutige
  Kontakt-E-Mail. Kontakt- und Login-E-Mail bleiben synchron. Ein Rollenwechsel weg von `PARENT`
  entfernt die Verknüpfung.
- `/me/**` leitet die Familie bei jeder Anfrage aus dem Konto ab. Ohne Haushalt kommen Kinder aus
  `child_parents`; Elternstunden benötigen einen Haushalt. Beitragsstatus: bezahlt bei vollständig
  gedecktem Betrag mit Cent-Toleranz, sonst überfällig vor dem heutigen Berliner Datum, sonst offen.
- Eltern-Meldungen haben Status `SUBMITTED` und Quelle `PARENT`; erst bestätigte Minuten zählen ins Ist.
  `ADMIN` und `PARENT_WORK` bestätigen oder lehnen ab. Der Ablehnungsgrund steht in der neuen Spalte
  `parent_work_entries.reject_reason`; die Übersicht zählt Meldungen je Haushalt und insgesamt.
- Kontaktänderungen wirken sofort. Jedes tatsächlich geänderte Feld wird in derselben Transaktion in
  `data_changes` protokolliert. Alte Werte stehen nur im Admin-Verlauf und Admin-Feed.

## Elternstunden, Stufe 1: Erfassung durch Tim (2026-09-27)

Umgesetzt nach dem (inzwischen gelöschten) Plan `docs/plan-elternstunden.md` in fünf Etappen, jeweils von Codex
gebaut und danach selbst geprüft (Tests, lokale Kopie der Live-Daten, Screenshots).

- **Rolle `PARENT_WORK`** („Elternstunden“, Migration `000038`): sieht nur `/elternstunden/**` und die API
  `/api/fees/v1/parent-work/**`. Dafür trennt der Router jetzt nach Gruppen: Beitrags-Routen nur `ADMIN`/`USER`,
  `/users` nur `ADMIN`, `/parent-work` `ADMIN`/`PARENT_WORK` (Regelwerk schreiben nur `ADMIN`). Vorher reichte für
  alle Beitrags-Routen jede Anmeldung. JWT-Uploads auf `/import/upload` nur `ADMIN`/`USER`; der Import-Token von
  banking-sync bleibt unverändert. `USER` hat keinen Zugriff auf Elternstunden (live gibt es nur einen `ADMIN`).
- **Tabellen** (Migration `000039`): `parent_work_rules` (versioniert per `valid_from`, Seed ab 01.08.2025:
  540 min je Kind, 3000 Cent je Fehlstunde, max. 180 min Übertrag; neue Versionen nur ab einem künftigen 01.08.),
  `board_terms` (Amtszeiten an `fees.members`), `parent_work_entries` (je Familie, Minuten in Viertelstunden,
  Status für spätere Stufen, Storno statt Löschen, Quelle `MANUAL`/`IMPORT`), `parent_work_overrides` (manuelles
  Soll je Familie und Kita-Jahr mit Begründung). Eine Familie mit Elternstunden-Einträgen lässt sich nicht löschen (409).
- **Berechnung** (`domain.CalculateParentWork`, rein, tabellengetrieben getestet): Soll je Kind 3 h je angefangenem
  Tertial; Vorstandsbefreiung, wenn eine Amtszeit eines Vereinsmitglieds der Familie (direkt über
  `members.household_id` oder über `parents.member_id`) das Kita-Jahr schneidet; manuelles Soll ersetzt beides;
  Übertrag `min(max, Ist + Übertrag − Soll, Ist)` – übertragene Stunden werden zuerst verbraucht und nie weitergereicht;
  Fehlbetrag nur angezeigt. Der Service rechnet Jahr für Jahr ab dem ersten Regelwerk-Jahr aus einem gesammelt
  geladenen Snapshot, keine Abfrage je Familie.
- **Import:** Tims Excel als CSV; die Seite nennt die erwarteten Spalten (Datum, Stunden, Anlass, Kind und/oder
  Mitglied). Datei per Button oder per Drag and Drop irgendwo ins Browserfenster (Window-Listener in `ImportSection`).
  Nach dem Einlesen zeigt eine Tabelle jede CSV-Spalte mit Beispielwerten und einem Feld-Dropdown (Vorbelegung über
  Spaltennamen, Rest „Ignorieren“; ein Feld gehört immer nur einer Spalte). Familien werden über den Namen gefunden:
  erst Kind, sonst Mitglied/Elternteil, exakter Vergleich nach Normalisierung (Groß-/Kleinschreibung, Umlaute/ß,
  Leerzeichen, „Nachname Vorname“ bei zweiteiligen Namen), **kein** Fuzzy-Match; 0 oder mehrere Treffer ⇒ Zeile ohne
  Familie. Die Vorschau warnt gesammelt („N Zeilen ohne Familie“, je Name mit Zeilennummern, mehrdeutig vs. nicht
  gefunden); dort wählt man die Familie von Hand, sonst wird die Zeile nicht importiert. Dubletten markiert,
  Ausführung atomar. Das echte Excel lag noch nicht vor; die Annahme „eine Zeile je
  Zettel mit den Feldern des Vordrucks“ ist ungeprüft.
- **Geprüft an einer lokalen Kopie der Live-Daten** (Stand 2026-09-26, `pg_dump --schema=fees`, Wegwerf-Container):
  45 Familien mit 50 Kindern im Kita-Jahr 2026/27, Soll 438 h – deckt sich mit einer unabhängigen SQL-Rechnung.
  11 Kinder ohne Familie sind alle ausgetreten; künftige Fälle zeigt die Übersicht als Hinweis.
- **E2E:** `frontend/e2e/beitraege/parent-work.spec.ts` (Rollenrechte, Eintrag/Storno, Vorstandsbefreiung, CSV-Import).
- **Nicht umgesetzt:** Vorstandsmitglieder und Amtsbeginne sind live noch nicht erfasst (nach dem Deploy auf der
  Vorstandsseite eintragen); Befreiung für die ganze Kita-Zeit nach 2 Jahren Vorstand ist nicht beschlossen und
  nicht gebaut; keine Eltern-Zugänge, keine Abnahme durch Erzieherinnen, keine Erinnerungen, keine Forderung.

### Fachliche Regeln (Stand 2026-09-26)

Ausgangspunkt war die nie deployte Portal-Spec (Tag `archive/pre-cleanup-2026-09`, `docs/portal/SPEC.md`, `backend-portal/internal/service/parent_work_rules.go`); mit Stefan abgeglichen, dazu Betreuungsvertrag Ziff. 7.3 und der Vordruck „Erfassung Elternstunden“:

- **Rechtsgrundlage:** Satzung 2025 § 5: Mitglieder leisten je Kita-Jahr Arbeitsstunden, für nicht geleistete ist ein Entgelt zu zahlen; Stundenzahl, Höhe und Fälligkeit beschließt die Mitgliederversammlung (2/3). Die Elternbeitragsordnung 2025 regelt dazu nichts. Betreuungsvertrag 7.3: „derzeit 9 Stunden (pro Kind)“, „für jede nicht geleistete Stunde … derzeit 30,00 Euro“, nachweispflichtig ist das Vereinsmitglied.
- **Soll:** 9 h je Kind und Kita-Jahr (01.08.–31.07.). Bei Ein- oder Austritt im Jahr 3 h je angefangenem Tertial (01.08.–30.11., 01.12.–31.03., 01.04.–31.07.). Manuelles Überschreiben mit Begründung bleibt möglich.
- **Konto und Einträge je Familie (Haushalt), nicht je Kind:** Das Soll ist die Summe über die Kinder des Haushalts; Einträge gehören dem Haushalt (anders als in der Portal-Spec).
- **Vorstandsbefreiung:** Vorstandsmitglieder werden mit Amtszeiten (von/bis, Amt) am Vereinsmitglied (`fees.members`) gekennzeichnet. Überschneidet sich eine Amtszeit eines Mitglieds des Haushalts mit dem Kita-Jahr, ist der Haushalt für das ganze Kita-Jahr befreit (bewusst einfach; die kürzeste Amtszeit zuletzt waren 6 Monate). Nicht beschlossen, nur Diskussionsstand: Befreiung für die gesamte restliche Kita-Zeit nach einer vollen Amtszeit von 2 Jahren; deshalb erst einmal nicht als Regel umsetzen, das manuelle Überschreiben deckt Einzelfälle ab. Vorstand Stand 2026-09 (alle als aktive Mitglieder mit Haushalt in `fees.members`): André Rüger (1. Vorsitzender), Sarah Thränhardt (2. Vorsitzende / Bauliches), Marcus Rehaag (Finanzen), Stefan Remer (Elternarbeit, seit ca. 2 Jahren), Samantha Lahl (Schriftführerin), Dennis Braak (Personal); genaue Amtsbeginne noch erfassen.
- **Fehlstunden:** 30 € je Stunde. Werden aber fast immer nachgeholt oder anders gelöst; daher zunächst nur anzeigen (inkl. Betrag), keine automatische Forderung. Stundenzahl und Satz sind Beschlüsse der Mitgliederversammlung („derzeit“), gehören also versioniert in die Daten, nicht in den Code.
- **Übertrag:** wie auf dem Vordruck: Mehrstunden über dem Soll werden bis höchstens 3 h ins folgende Kita-Jahr übertragen (nur ein Jahr weit, nicht kettenweise weiter).
- **Genauigkeit:** mindestens halbe Stunden; ob Viertelstunden gebraucht werden, ist offen. Speichern in Minuten hält beides offen.
- **Ablauf heute:** Eltern erledigen Aufgaben (Räume reinigen, Küchendienst, Betreuung während Dienstbesprechungen, Garten …) und füllen den Vordruck aus (Mitglied, Kind, Datum, Anlass, Stunden, Unterschrift Mitglied und Erzieher/in). Tim Hall pflegt die Übersicht für den Vorstand (Excel mit denselben Angaben). Geplante Stufen: (1) Tim erfasst und sieht die Übersicht mit einer eigenen Rolle nur für Elternstunden, (2) Erzieherinnen nehmen digital ab, (3) Eltern reichen selbst ein und bekommen Erinnerungen; danach entfällt Tims Buchführung.
- **Rollen:** Heute sperrt der Router nur einzelne Routen auf `ADMIN`; jeder andere angemeldete Benutzer sieht alle Beiträge. Entscheidung: Rechte klar trennen – Beitrags-Routen nur für `ADMIN`/`USER`, Admin-Routen nur `ADMIN`, die neue Elternstunden-Rolle sieht nur Elternstunden.

## OpenAPI-Drift-Check in CI (2026-09-26)

- Die Quelle für den API-Vertrag ist der Go-Code (swag-Annotationen und Structs), nicht die Spec. `openapi3.yaml` und `schema.d.ts` sind eingecheckte Generate. Bisher prüfte nichts, ob sie nach Änderungen am Backend neu erzeugt wurden: Der Typecheck lief dann gegen den alten Stand.
- Neuer CI-Job `openapi-drift`: führt `scripts/generate-api.sh` aus und bricht bei `git diff` auf beide Dateien ab. `build-images` hängt über `needs: ci` auch an ihm.
- Damit CI und lokale Läufe byte-gleich erzeugen, pinnt das Skript swag (`v1.16.6`, geprüft per `go version -m` auf die Binary, weil `swag --version` bei v1.16.6 noch „v1.16.4“ meldet; der erste CI-Lauf mit echter v1.16.4 ließ die `x-enum-descriptions` weg) und swagger2openapi (`7.0.8`); der Fallback auf ein global installiertes, ungepinntes `swagger2openapi` ist entfernt. openapi-typescript ist über das Lockfile gepinnt. Beim Anheben der swag-Version Skript und `ci.yml` gemeinsam ändern.
- Spec-first (z. B. `oapi-codegen`) wurde erwogen und für den Bestand verworfen; für eine künftige Eltern-API wieder offen.

## Aufräumen: nur noch Beiträge (2026-09-26)

- **Entfernt:** `backend-management` + Dienstplan-/Zeiterfassung-Apps (seit Monaten ungenutzt, live nur Testdaten), Portal-Grundgerüst `backend-portal` + `apps/portal` + `docs/portal` (nie deployt, Stand 2026-05-25), `frontend/packages/shared` (nur von den beiden Management-Apps genutzt), deren e2e-Tests, `openapi/management`, die handgeschriebenen `openapi/*-api.yaml`, der Java/Maven-Rest in `docker/` (Caddyfile, Dockerfiles, `compose.prod`/`compose.ghcr`, Backup-Skript), `docs/archive/` und der abgeschlossene Plan `docs/plan-bankabgleich-ui.md`. Letzter Stand im Tag `archive/pre-cleanup-2026-09`. Grund: Altlasten haben Coding-Agents in die Irre geführt; die Doku markierte sie nur als „pausiert“ oder „Archiv“.
- **Richtung:** Elternstunden und ein Eltern-Zugang kommen als Modul in `backend-fees`/Beiträge-App, nicht als eigener Portal-Dienst: `backend-fees` ist ohnehin Quelle für Kinder/Eltern/Haushalte, hat Benutzer, Mailversand und CI; eine Sync-Schicht mit Quarantäne und zweiter Identität entfällt.
- **OpenAPI:** `openapi/fees/swagger.yaml` ist nicht mehr eingecheckt; `scripts/generate-api.sh` erzeugt es in einem Temp-Verzeichnis und schreibt nur `openapi3.yaml`.
- **Image:** `backend-fees/Dockerfile` kopiert weder `openapi/` noch `frontend/packages` und generiert keine Typen mehr (`schema.d.ts` ist eingecheckt). `build-images.yml` baut nur noch `backend-fees` und `banking-sync`.
- **Homelab (Repo `homelab`):** `backend-management` aus der kita-Compose entfernt. Edge-Caddy leitet `/beitraege/*`, `/api/fees/*`, `/health` an `backend-fees`, alles andere per 302 auf `/beitraege/` (302, weil `/` später ein Eltern-Einstieg werden kann). Dabei behoben: `redir /plan/ permanent` wurde von Caddy als Matcher `/plan/` mit Ziel `permanent` gelesen, `/`, `/health` und `/healthz` lieferten öffentlich eine leere 200 – die Health-Checks in `docs/deployment-homelab.md` prüften also nichts.
- **Live-DB:** Die Management-Tabellen in `public` (Testdaten: 1 Mitarbeiterin, 56 Dienstplan-Einträge, 1 Zeiteintrag, plus `flyway_schema_history` aus der Java-Zeit) und `public.update_updated_at_column()` sind gelöscht; `public` ist leer, `fees` hat keine Abhängigkeiten darauf. Vorher per `pg_dump --schema=public` gesichert auf der Workstation unter `~/backups/kita/kita-public-management-2026-09-26.sql` (enthält E-Mail und Passwort-Hash, Rechte 600).

## Code-Review-Backlog (ab 2026-09-23)

Abarbeitung von `docs/todo-fees-improvements.md`, ein Commit pro Punkt.

- **E2E für den banking-sync-Upload:** Vor dem Push von #21/#22 und dem Vereinsbeitrag pro Mitglied geprüft, ob der tägliche CSV-Import betroffen ist. banking-sync meldet sich nicht an (nur `X-Import-Token`, Route und `ImportAuthMiddleware` unverändert), daher berühren Cookie und Login-Bremse ihn nicht. Das Risiko liegt beim geänderten Zuordnen von Vereinsbeiträgen und bei `member_id` in allen Beitrags-SELECTs; Migration `000037` läuft in Prod vor dem Serverstart (`migrate && server`). Neuer Test `banking-sync-import.spec.ts` bildet den Upload exakt nach (Stack setzt `CRON_API_TOKEN`). Beim Parallel-Lauf fiel auf: Geteilte Zahler-IBANs zwischen Tests leiten Zahlungen über die vertrauenswürdige IBAN zum Kind des anderen Tests. Das ist korrektes Verhalten, deshalb nutzt jetzt jeder Test eine eigene IBAN (`uniqueIban()`).
- **Playwright-E2E (#24):** Bisher wurden UI-Smoke-Tests pro Punkt als Wegwerf-Skripte geschrieben und gelöscht, und die alte Beiträge-Suite lief nur gegen eine von Hand vorbefüllte DB (`kita-db-e2e`) mit veralteten Zugangsdaten und Seiten (z. B. Passwort-Reset). Neu: `frontend/e2e/beitraege/` mit `playwright.config.ts`. Playwright startet `e2e/start-beitraege-stack.sh`: leere Wegwerf-PostgreSQL per `docker run`, Frontend-Build eingebettet ins Backend (wie das Image, also gleiche Origin und Cookie-Pfade wie in Produktion), Admin per Bootstrap. Bewusst keine Test-Container-Bibliothek für Node und kein Vite-Proxy: weniger Abhängigkeiten, näher an Produktion; Nachteil ist ein Build pro Lauf (lokal gut 1 Minute kalt, danach Sekunden). Anmelden per API im Browser-Kontext statt `storageState`, weil das rotierende Refresh-Token ein gespeichertes Cookie nach der ersten Nutzung ungültig macht. Playwright beendet den Webserver standardmäßig mit SIGKILL; ohne `gracefulShutdown: SIGTERM` blieb der DB-Container stehen. 8 Tests (Auth, Benutzer, Kernablauf Kind → Beiträge → Bankimport → bezahlt), in CI als Job `e2e-beitraege`; `build-images` hängt über `needs: ci` auch an ihm. Kleine UI-Änderungen für zugängliche Selektoren: `role="dialog"` + `aria-label` an den Passwort-/Benutzer-Dialogen, `aria-label="Benutzermenü"`. Die alten Beiträge-Specs sind entfernt; die Dienstplan-/Zeiterfassung-Tests liegen unverändert in `playwright.management.config.ts`. Die Beschreibung der alten Seed-DB liegt in `docs/archive/test-seeding.md`.

- **httpOnly-Cookie und Login-Bremse (#21, #22):** Das Refresh-Token steht nicht mehr im Response-Body und nicht mehr im `localStorage`, sondern nur im httpOnly-Cookie `fees_refresh` (`SameSite=Strict`, Pfad `/api/fees/v1/auth`, damit es nur an Login/Refresh/Logout geht; `Secure`, wenn die Anfrage per HTTPS bzw. mit `X-Forwarded-Proto: https` vom Caddy kommt, sonst nicht, damit lokales HTTP funktioniert). Das Access-Token (15 min) liegt nur im Speicher; beim Laden der Seite holt der Router-Guard einmalig per `/auth/refresh` ein neues. Eine XSS-Lücke kann damit keine dauerhafte Sitzung mehr stehlen. CSRF: `SameSite=Strict` plus Pfad-Einschränkung; die übrigen Endpunkte authentifizieren weiter per Bearer-Header. `/auth/logout` ist jetzt öffentlich und arbeitet nur mit dem Cookie (funktioniert also auch mit abgelaufenem Access-Token). `change-password` gibt direkt eine neue Sitzung zurück (neues Access-Token + Cookie), das Frontend muss sich nicht mehr mit dem neuen Passwort neu anmelden. Alte `localStorage`-Einträge werden beim Start gelöscht; nach dem Deploy muss sich jeder einmal neu anmelden. Login-Bremse `auth.LoginLimiter`: feste 15-Minuten-Fenster, 5 Fehlversuche pro IP+Konto (Schlüssel inkl. IP, damit ein Angreifer niemanden global aussperren kann) und 20 pro IP über alle Konten; erfolgreicher Login setzt nur den Konto-Zähler zurück. Antwort `429` mit `Retry-After` und deutscher Meldung, die die Login-Seite direkt anzeigt (vorher erschien bei falschem Passwort „Unauthorized“, weil der Client auch für Login einen Refresh versuchte; Login/Refresh/Logout sind davon jetzt ausgenommen). Die IP kommt über chis `RealIP` aus `X-Forwarded-For`, das der Caddy selbst setzt. Zustand nur im Speicher (eine Instanz). Uploads (Bank-CSV, Kinder-Import) laufen jetzt über den gemeinsamen Request-Pfad und erneuern ein abgelaufenes Access-Token wie alle anderen Aufrufe. Tests: `login_limiter_test.go`, `auth_cookie_integration_test.go` (Cookie-Attribute, kein Token im Body, Rotation, alte Tokens ungültig, Logout, 429).

- **Benutzerverwaltung (#20):** Migration `000036` legt `fees.users` wieder an (bcrypt-Hash, Rolle `ADMIN`/`USER`, `is_active`, E-Mail eindeutig ohne Groß-/Kleinschreibung). Beim Start legt `AuthService.BootstrapAdmin` aus `USER_NAME`/`USER_PASSWORD` ein Admin-Konto an, aber nur, wenn es noch kein Konto mit dieser E-Mail gibt; ein bestehendes Konto wird nie überschrieben. Damit bleibt das bisherige Admin-Login mit gleichem Passwort gültig, und eine Passwortänderung in der App übersteht Neustarts (behoben: vorher ging sie verloren). Das Konto erhält die ID des früheren statischen Admins (`a0eebc99-…`), sodass vorhandene Refresh-Tokens und gespeicherte Akteur-IDs weiter passen und niemand neu anmelden muss. Notfall-Reset: Zeile löschen und neu starten. Keine Fremdschlüssel von `refresh_tokens` auf `users`, weil Tokens vor dem Bootstrap existieren. Login prüft Hash und Aktiv-Flag (mit Dummy-Hash für unbekannte E-Mails gegen Timing-Unterschiede). Refresh lädt das Konto aus der DB: deaktivierte oder gelöschte Konten bekommen keine neuen Tokens, und E-Mail/Rolle im Token kommen aus der DB (eine Rollenänderung greift also spätestens nach 15 min). Eigenes Passwort ändern und Admin-Reset widerrufen alle Refresh-Tokens des Kontos; die Sitzung des ändernden Browsers läuft weiter (seit #21 liefert der Endpunkt dafür direkt eine neue Sitzung). Admins können sich selbst nicht deaktivieren oder herabstufen, dadurch bleibt immer ein aktiver Admin. API: `GET/POST /users`, `PUT /users/{id}`, `POST /users/{id}/password` (nur `ADMIN`); `POST /auth/change-password` war schon da und speichert jetzt tatsächlich. Frontend: Seite „Benutzer“ (Navigation „System“, nur für Admins, Route `/benutzer` mit Admin-Guard) und „Passwort ändern“ im Benutzermenü. Tests: `user_integration_test.go` (Bootstrap mit gleicher ID und ohne Überschreiben, Login, inaktive Konten, Token-Widerruf, Selbstschutz, doppelte E-Mail), `router_test.go` um die neuen Admin-Routen ergänzt; per curl und Playwright gegen das lokale Backend geprüft. Live-DB vorab read-only geprüft: alle 136 Refresh-Tokens gehören der statischen Admin-ID.
- **Handler-/Middleware-Tests (#19):** `middleware/auth_test.go` (fehlender/falscher Header, fremde Signatur, Refresh- statt Access-Token, abgelaufen, gültig inkl. Nutzerkontext; `RequireRole` 401/403/OK), `middleware/body_limit_test.go`, `api/router_test.go` (echter Router mit leeren Handlern: jede nicht-öffentliche Route unter `/api/fees/v1` antwortet ohne Token mit 401 – per `chi.Walk`, schlägt also auch bei neu hinzugefügten ungeschützten Routen an; alle Admin-Routen antworten mit Rolle `USER` 403; `/health`; Upload über 5 MB mit Import-Token → 413), `handler/error_mapping_test.go` (zentrale Fehlerabbildungen inkl. 400 bei fehlender Beitragsordnung). Behobener Fehler: `writeReminderCaseError` verglich mit `switch err` statt `errors.Is`; eingewickelte „nicht gefunden“/„ungültig“-Fehler wurden so zu 500. Auth-Flows sind mit der Benutzerverwaltung (#20) über Integrationstests abgedeckt.
- **OpenAPI-Pflichtfelder (#18):** swag läuft mit `--requiredByDefault` (`scripts/generate-api.sh`); alle JSON-Felder mit `omitempty` oder Zeigertyp tragen `binding:"optional"` (337 Felder per AST-Skript ergänzt, rein dokumentarisch, es läuft kein Validator). Dadurch stimmt die Optionalität in Spec und `schema.d.ts` mit dem überein, was die API sendet. `types.ts` braucht die Hilfstypen `DeepStrict`/`Loose` nicht mehr; übrig sind nur gezielte Überschreibungen (`| null` für Zeigerfelder, auf die sich die UI stützt, und eigene Frontend-Typen wie `Einstufung.incomeCalculation`). Wächter-Test `internal/api/openapi_tags_test.go` meldet neue `omitempty`-/Zeigerfelder ohne Tag. Bekannte Unschärfe: Slices ohne `omitempty` gelten als Pflicht, können aber bei `nil` als `null` kommen (wie bisher mit `DeepStrict`); wo das vorkommt (`ImportBatch.errors`), bleibt die Überschreibung auf optional.
- **Große Seiten zerlegt (#17):** `ChildDetailPage` 3076 → 1368 Zeilen, `ImportPage` 2200 → 1173, `ChildImportPage` 1489 → 200, `FeesPage` 1241 → 851, `AutomationPage` 1233 → 821. Dialoge sind eigene Komponenten unter `components/child/` (Notizen-Karte, Kind bearbeiten, Elternteil anlegen/verknüpfen, Elterndetails, Transaktionsdetails, Zuordnung), `components/import/` (Upload, Historie, Blacklist, manuelle Zuordnung), `components/fees/` (Generieren, Einzelbeitrag) und `components/automation/` (Versandverlauf-Tab, E-Mail-Detail, Zahlungsdaten). Muster: Die Seite rendert den Dialog per `v-if`, der Dialog hält seinen Formularzustand selbst (frisch bei jedem Öffnen) und meldet per Event (`saved`, `changed` …), worauf die Seite neu lädt; ESC schließt weiterhin über die Seite. Der Kinder-Import-Assistent liegt als Composable `composables/useChildImportWizard.ts` vor, die Schritte 2/3 holen ihn per `inject` (`useChildImportWizardContext`). Gemeinsame Helfer: `utils/fees.ts` (Beitragsart-Name/Farbe, Zahlungs-Matches, Restbeträge, IBAN-Maskierung, Konfidenz), `utils/child.ts` (Alter, U3, Einkommensstatus, Betreuungsstunden), `utils/reminders.ts`. Kleine sichtbare Änderungen: Fehler von CSV-Upload, manueller Zuordnung und Blacklist erscheinen jetzt im jeweiligen Modal statt im verdeckten Seitenbanner; „Beiträge generieren“ zeigt Fehler im Dialog; die Zahlungsdaten für den QR-Code werden beim Öffnen des Dialogs geladen (Felder bis dahin gesperrt); die Notizen-Liste springt nach dem Neuladen des Kindes auf Seite 1. Per Playwright-Smoke-Test gegen lokales Backend durchgeklickt (Notizen, Kind/Eltern/Haushalt bearbeiten, Zuordnen, Transaktionsdetails, Bank-CSV-Upload, Historie, Blacklist, manuelle Zuordnung, Kinder-Import, Generieren, Einzelbeitrag, Zahlungsdaten, Versandverlauf).
- **Frontend-Formatierer zentral (#16):** `src/utils/format.ts` bündelt `formatDate`, `formatDateTime`, `formatDateForInput`, `formatCurrency`, `formatCurrencyWhole` (Einkommen ohne Cent), `formatMonthName`, `formatMonthYear` und `todayISO`; die 15 lokalen Kopien in Seiten und Komponenten sind entfernt. Sichtbare Unterschiede: Datumsangaben immer zweistellig (`01.03.2025` statt `1.3.2025`), Zeitstempel explizit in Europe/Berlin, reine Kalenderdaten (`YYYY-MM-DD` bzw. UTC-Mitternacht) werden ohne Zeitzonen-Umrechnung angezeigt, leere Werte als `—`. Behobener Fehler: Vorschlagsdaten für „heute“ (Mitgliedsbeginn, Stichtag, Einstufung „gültig ab“, Mahnlauf) kamen aus `toISOString()` und lagen zwischen 0 und 2 Uhr auf dem Vortag; jetzt `todayISO()` mit Berliner Kalendertag. Bewusst nicht umgestellt: Sonderformate im Einstufungs-PDF und im Bescheidtext („… Euro“).
- **Beitragsordnung ab 01.01.2025, Ü3-Tabelle (Rückmeldung nach Deploy):** Migration `000035` verschiebt die Seed-Version von 2000-01-01 auf 2025-01-01 (seit dann gilt die aktuelle Ordnung, Werte von der Kita bestätigt) und benennt sie „Elternbeitragsordnung ab 01.01.2025“. Sie ändert nur die unveränderte Seed-Zeile und nur, wenn keine andere Version bis 2025-01-01 beginnt. Vorher read-only geprüft: Die frühesten Live-Daten liegen danach (Beiträge ab 2025-04, Buchungen ab 2025-09, Einstufungen ab 2026-02). Für Stichtage vor 2025 gibt es jetzt keine Version mehr (`ErrNoFeeSchedule`). Neu ist das optionale Feld `kindergartenTable` (Ü3-Satzungstabelle, 13 Stufen, 40 h: 61,60–308,00 €). Es dient nur zur Information: Ü3 ist nach dem Elternbeitragsentlastungsgesetz beitragsfrei, deshalb verwendet `CalculateChildcareFee` die Tabelle nicht. Neue Versionen übernehmen sie beim Kopieren. Die Seite „Beitragsordnung“ zeigt sie mit Hinweis an. Fehlt für einen Stichtag eine Version (vor 2025), antworten Rechner, Einstufung und Beitragserzeugung mit 400 und einer verständlichen Meldung statt 500.
- **`import_service.go` aufgeteilt (#15):** Gleicher Typ `ImportService`, Methoden nach Thema auf Dateien verteilt: `import_service.go` (Struct, Konstanten, CSV-Import, Rescan, Historie), `import_matcher.go` (Kind-/IBAN-/Beitragserkennung, Warnungsprüfung beim Import, Vorschläge), `import_match_actions.go` (Bestätigen, manuell zuordnen, Verteilen, Lösen, Verwerfen, Ausblenden, Listen), `import_iban_registry.go` (Blacklist, vertrauenswürdige IBANs), `import_warnings.go` (Warnungen, Verspätungsprüfung, Mahngebühr). Bewusst keine eigenen Service-Typen, damit Konstruktoren und Aufrufer gleich bleiben. Reines Verschieben: alle 60 Deklarationen per AST identisch, Tests unverändert grün.
- **Handler aufgeteilt (#14):** `ChildHandler`/`FeeHandler` bekommen nur noch Services (E-Mail-Log über `ReminderService.ListEmailLogs`). `fee_handler.go` und `import_handler.go` sind nach Thema zerlegt: `fee_reminder_handler.go`, `email_log_handler.go`, `childcare_fee_handler.go`, `import_iban_handler.go`, `import_warning_handler.go`, `import_transaction_handler.go`. Reines Verschieben: per AST verglichen sind alle 81 Deklarationen samt Doc-Kommentaren identisch, die generierte OpenAPI-Spec ist unverändert.
- **Feste Listen-Limits bewusst belassen:** `FeeService.Generate` lädt höchstens 1000 Kinder, das Kindkonto höchstens 1000 Beiträge. Die Kita hat maximal 52 Kinder gleichzeitig; auch mit ehemaligen Kindern ist das Limit auf Jahrzehnte nicht erreichbar. Deshalb nicht umgebaut.

- **Geldbeträge in Cent (#13):** Bewusst pragmatisch umgesetzt: DB (`NUMERIC(…,2)`), Structs und API bleiben Euro-`float64`. Vergleiche und Summen in Go laufen aber exakt über `domain.Cents`/`Euros`/`SumCents`/`IsPaid`, statt mit Float-Epsilon (`0.01`) zu arbeiten. Die fachliche Toleranz „bezahlt, wenn höchstens 1 Cent fehlt“ bleibt als `domain.PaymentToleranceCents` erhalten; sie entspricht den SQL-Bedingungen `matched >= amount - 0.01`, die mit `NUMERIC` ohnehin exakt rechnen und daher unverändert sind. Umgestellt: Bezahlt-Status und Rest (Gebührenliste, Kindkonto), Verteilung von Zahlungen, Einstufungs-Resync, Kindkonto-Summen, Summen der Mahnfälle und des SEPA-QR-Betrags sowie die Monats-Coverage. Behobener Fehler: Die Coverage verglich Float-Summen mit `==`; Gebühren von 45,40 € + 69,40 € und eine Zahlung von 114,80 € ergaben „teilweise bezahlt“ mit einem Rest von 1,4e-14 € (Regressionstest `coverage_cents_test.go`). An exakten 1-Cent-Grenzen können Verteilen und Resync jetzt anders entscheiden als vorher, weil das Float-Epsilon dort zufällig kippte (z. B. 45,40 − 45,39 = 0,0100000000000051 > 0,01); jetzt gilt die Toleranz exakt.
- **Konfigurierbare Beitragsordnung (#12):** Einkommensgrenzen, Entlastungs- und Satzungstabelle, Geschwisterfaktoren, Beitragsfreiheit ab Kinderzahl, Essensgeld und Mitgliedsbeitrag liegen versioniert in `fees.fee_schedules` (Migration `000034`, `valid_from` eindeutig, Werte als JSONB `FeeScheduleConfig`). Der Seed (ursprünglich gültig ab 2000-01-01, seit `000035` ab 2025-01-01) enthält exakt die bisher im Code hinterlegten Werte. Die Berechnung ist eine reine Funktion `FeeScheduleConfig.CalculateChildcareFee`. Welche Version gilt, richtet sich nach dem Stichtag: bei der Beitragserzeugung nach dem abgerechneten Monat (Mitgliedsbeitrag: 1. Januar des Jahres), bei Einstufungen nach dem Wirksamkeitsmonat (die Monatstabelle rechnet jeden Monat mit seiner Version), beim öffentlichen Rechner nach heute oder dem optionalen Parameter `date`, bei der Import-Erkennung (Essensgeld/Mitgliedsbeitrag am Betrag) nach dem Buchungsdatum. Mahngebühren (10 €/5 €) bleiben Konstanten. Absicherung: Eine eingefrorene Kopie der alten Berechnung (`legacy_childcare_fee_reference_test.go`) wird über ein Raster von rund 237.000 Eingaben mit der aus der DB geladenen Seed-Version verglichen, und alle Ergebnisse sind identisch. Einzige theoretische Abweichung: Einkommen mit Bruchteilen unter einem Cent zwischen 35.000,00 und 35.000,01 bzw. 55.000,00 und 55.000,01 fielen vorher in einen „Beitragsfrei“-Fallback und landen jetzt in der nächsten Stufe (die Einkommensberechnung rundet auf Cent). Verwaltung über `GET/POST /fee-schedules`, `PUT/DELETE /fee-schedules/{id}` und die neue Seite „Beitragsordnung“: Versionen, die bereits gelten, sind schreibgeschützt, weil Einstufungs-Monatstabellen live aus ihnen berechnet werden. Neue Versionen beginnen an einem künftigen Monatsersten; geplante Versionen können bearbeitet und gelöscht werden. Das Einstufungs-PDF nennt den Vereinsbeitrag aus der Einstufung statt fest „30,00 €“. Außerdem zeigt der API-Client jetzt die konkrete Fehlermeldung des Backends (`message`) statt nur des HTTP-Statustexts (`error`), und zwar app-weit. Ein alter Test-Kommentar nannte für 40.637,43 € und 45 h laut Excel 66,00 €. Die Kita hat 110,00 € bestätigt (Tabelle und Code, Stufe ab 40.000,01 €); der Test prüft jetzt diesen Betrag.
- **Dedupe per Unique-Index (#11):** Migration `000033` legt `uq_bank_transactions_booking` auf `(booking_date, COALESCE(payer_iban,''), amount, md5(COALESCE(description,'')))` an, also denselben Schlüssel wie `TransactionRepository.Exists`. Live-Daten wurden am 2026-09-23 read-only geprüft (719 Buchungen, keine Duplikate). Enthält eine Dev-DB Duplikate, schlägt die Migration fehl und muss von Hand bereinigt werden (bewusst keine automatische Löschung). `Create` nutzt `ON CONFLICT DO NOTHING` und gibt `repository.ErrDuplicate` zurück; der Import zählt das als übersprungen. Die `Exists`-Vorprüfung bleibt (sie behandelt fehlende IBAN weiterhin als Platzhalter).
- **Import-Fehler sichtbar (#10):** `ProcessCSV` bricht jetzt ab (HTTP 500 → banking-sync meldet `error` in der Sync-Karte und an Uptime Kuma), wenn Blacklist, Kinder oder Eltern nicht geladen werden können oder der Batch nicht angelegt werden kann. Vorher lief der Import still gegen eine leere Kinderliste weiter. Der Batch wird erst nach dem Laden der Referenzdaten angelegt. Fehler einzelner Buchungen (Duplikatprüfung, Speichern, Warnung speichern, Auto-Match) landen in `ImportResult.errors` (`domain.ImportError`: Datum, Zahler, Betrag, Meldung) statt stumm als `skipped`. Migration `000032` speichert sie an `fees.import_batches` (`error_count`, `errors` JSONB, max. 100 Details). `GET /import/history` liefert beides, Rescan liefert `errors` ebenfalls. Die Kinderliste für das Matching wird seitenweise komplett geladen (vorher hartes Limit 1000). Frontend: Komponente `ImportErrorList`; Banner „Letzter Import … mit Fehlern“ auf der Bankabgleich-Seite (lädt den neuesten Batch), Spalte „Fehler“ mit aufklappbaren Details in der Import-Historie, Fehlerliste im Upload-Ergebnis und nach „Erneut zuordnen“, und in der Banking-Sync-Karte, wenn `uploadResult.errors` gefüllt ist. Fehler beim Laden der Historie werden im Modal angezeigt statt nur in der Konsole.
- **Atomares Matching (#9):** `ImportService` bekommt den `TxManager`. Atomar sind jetzt: manuelles Zuordnen, Bestätigen (je Match), Auto-Match beim Import/Rescan (alle Matches einer Buchung + Folgeaktionen), Verteilen (Validierung + alle Zuordnungen + Folgeaktionen), Zuordnung aufheben, Ablehnen (Blacklist + Löschen offener Buchungen der IBAN), Mahngebühr aus Verspätungs-Warnung (Gebühr + Warnung auflösen). Die Folgeaktionen (`postMatchActions`: IBAN als vertrauenswürdig markieren, Warnungen auflösen, Verspätung prüfen) geben Fehler jetzt zurück statt sie zu verschlucken; schlägt eine fehl, wird der Match mit zurückgerollt. Beim Bestätigen zählt das als `failed` (mit Log), beim Auto-Match bleibt die Buchung als Vorschlag stehen. Beim CSV-Import bleibt das Speichern der Buchung eine eigene Einheit, damit ein fehlgeschlagener Auto-Match die Buchung nicht verliert. Test `import_tx_integration_test.go` erzwingt einen Fehler in der letzten Folgeaktion und prüft, dass weder Match noch IBAN-Eintrag übrig bleiben.
- **Transaktions-Infrastruktur (#8):** `repository.TxManager.WithTx(ctx, fn)` bindet eine `*sqlx.Tx` an den Context. Alle Repository-Statements laufen über `conn(ctx, r.db)` und nutzen damit automatisch die äußere Transaktion; ohne `WithTx` verhalten sie sich wie bisher. Repository-Methoden mit eigener Transaktion (`beginTx`, z. B. `ChildRepository.Update`, Einstufung-Folgeeinstufung) schließen sich einer äußeren Transaktion an; ihr `Commit`/`Rollback` ist dann wirkungslos. Verschachteltes `WithTx` tritt bei. Ein `nil`-TxManager führt `fn` ohne Transaktion aus (für Unit-Tests mit Fakes). Noch kein Service nutzt es, das kommt mit #9.
- **Zeitzone Europe/Berlin (#7):** Der Container lief in UTC, dadurch galt zwischen 0 und 2 Uhr (Sommer) bzw. 0 und 1 Uhr (Winter) noch der Vortag, z. B. für Fälligkeit, Betreuungszeit-Historie, Jahres-/Monatsvorgaben und Mahnlauf-Datum. Neu: `util.Now()` (Berliner Wanduhr, für Jahr/Monat/Zeitstempel) und `util.Today()` (Berliner Kalendertag als UTC-Mitternacht, also im selben Format wie gescannte `DATE`-Spalten; für Datumsvergleiche). Zonendaten sind per `time/tzdata` eingebettet, `util.SetClock` macht Tests deterministisch. Zusätzlich `TZ=Europe/Berlin` im Image (betrifft nur Log-Ausgabe). Bewusste Verhaltensänderung: Die Beitragsübersicht zählt eine Gebühr erst ab dem Tag nach Fälligkeit als überfällig (bisher ab 0 Uhr UTC am Fälligkeitstag), konsistent mit dem Erinnerungs-Workflow. Der Stichtagszähler rechnet in ganzen Kalendertagen. Nachgezogen: U3-Filter der Kinderliste und Fälligkeit der Mahngebühr aus einer Verspätungs-Warnung. Fachliche Zeitstempel (`created_at` usw.) verwenden `util.Now()`.
- **Request-Limits (#6):** Globale Middleware `MaxBodySize` begrenzt jeden Body auf 5 MB; `request.DecodeJSON` zusätzlich auf 1 MB. CSV-Uploads (Bank- und Kinder-Import) laufen über `parseUpload`: zu groß → 413, sonst kaputt → 400 (vorher wurde der Fehler beim Bank-Upload ignoriert). `scripts/generate-api.sh` nutzt `BUN_CONFIG_REGISTRY` statt des von neueren Bun-Versionen abgelehnten `bunx --registry`. Die zuvor von Hand in `openapi3.yaml` gepflegten Hinweise zu ignorierten Einstufungsfeldern stehen jetzt als Go-Kommentare (`Deprecated: …`) an den DTOs und überleben so die Neugenerierung; swag kann das Feld-Flag `deprecated: true` selbst nicht erzeugen.
- **Import-Token (#4):** `CRON_API_TOKEN` wird über `config.Import.Token` geladen statt per `os.Getenv` in der Middleware und in konstanter Zeit verglichen (`subtle.ConstantTimeCompare`). Leerer Token deaktiviert Token-Auth. Bearer-Parsing ist zwischen `AuthMiddleware` und `ImportAuthMiddleware` geteilt; Tabellen-Test in `import_auth_test.go`.
- **Unsichere Defaults (#5):** Statt Start-Abbruch (würde `go run` ohne `.env` brechen) härtet `config.Harden()` beim Start: fehlendes, zu kurzes (<32 Zeichen) oder altes Dev-Default-`JWT_SECRET` → zufälliges Secret pro Prozess + Warnung (Sessions enden beim Neustart). `CORS_ORIGINS` hat keinen Default mehr; ohne Wert wird keine CORS-Middleware installiert (Frontend ist same-origin bzw. über Vite-Proxy). `*` wird nie mit `AllowCredentials` kombiniert. Prod (infra-dev) setzt `JWT_SECRET` (44 Zeichen) und `CORS_ORIGINS=https://kita.remer.cc` und ist unverändert.
- **CI vor Image-Build (#1):** Neuer Workflow `ci.yml` (PR, `workflow_call`, manuell) prüft `backend-fees` mit `gofmt -l`, `go vet`, `go test ./...` (Integrationstests via testcontainers) und `beitraege` per `typecheck`. `build-images.yml` ruft ihn als ersten Job auf, die Image-Builds hängen per `needs` daran.
- **gofmt (#2):** 10 Dateien formatiert, reiner Format-Commit.
- **Typecheck ohne Emit (#3):** `vue-tsc -b` hat `vite.config.js`/`.d.ts` neben `vite.config.ts` erzeugt und eingecheckt; Vite lädt im Dev-Modus `vite.config.js` bevorzugt, die `.ts` war also potenziell wirkungslos. Die Projekt-Referenz ist entfernt, beide tsconfigs sind `noEmit`, die Artefakte gelöscht. Neues Script `bun run typecheck` (App + `vite.config.ts`), `build` ruft es vor `vite build` auf.

## Vereinsbeitrag pro Mitglied (2026-09-24)

- Entscheidung: Der Verwendungszweck bleibt bei der Kindernummer, die Mitgliedsnummern (M…) werden nicht kommuniziert und im Import nicht ausgewertet. Die Mitglieder in der App dienen weiter vor allem der Information. Geändert wurde nur, dass ein Haushalt mit mehreren Mitgliedern auch mehrere Vereinsbeiträge schuldet.
- Migration `000037`: `fee_expectations.member_id` (FK auf `fees.members`, `ON DELETE SET NULL`). Rückwirkend gesetzt nur, wo der Haushalt genau ein Mitglied hat (live: 43 von 58 Beiträgen 2026). Eindeutigkeit: ein `MEMBERSHIP`-Beitrag pro Mitglied und Jahr; die alte Regel „einer pro Haushalt und Jahr“ gilt nur noch für Beiträge ohne Mitglied. Die Down-Migration schlägt fehl, solange ein Haushalt mehrere Beiträge pro Jahr hat.
- Jahresgenerierung (`generateHouseholdMembershipFees`): Mitglieder zählen, wenn sie aktiv sind, am Haushalt hängen (`members.household_id`) und ihre Mitgliedschaft das Jahr überschneidet. Haushalte ohne solche Mitglieder bekommen wie bisher einen Beitrag ohne Mitglied. Sonst gilt: Vorhandene Beiträge ohne Mitglied übernimmt das erste Mitglied ohne Beitrag (Reihenfolge Mitgliedsnummer), für die übrigen wird je ein neuer Beitrag angelegt, jeweils am ältesten Kind des Haushalts, das in dem Jahr noch keinen Vereinsbeitrag hat. So passt die übliche Überweisung „mit der Nummer des eigenen Kindes“. Hat der Haushalt weniger Kinder als Mitglieder, landen weitere Beiträge am ältesten Kind. Mitglieder ohne Haushalt bekommen weiterhin keinen Beitrag.
- Import: Hat ein Haushalt mehrere offene Vereinsbeiträge, wird die Zahlung dem Beitrag des erkannten Kindes zugeordnet, wenn dort genau einer offen ist; sonst bleibt die Warnung „mehrere offene Vereinsbeiträge“.
- Familie Fink (M0037 Pierre, M0039 Anika, Kinder 12007/12008): Der bezahlte Beitrag 2026 liegt an 12007 und hat noch kein Mitglied. Eine erneute Jahresgenerierung 2026 ordnet ihn M0037 zu und legt für M0039 einen neuen offenen Beitrag an 12008 an. Andere Haushalte ändern sich dabei nicht (live read-only geprüft: kein Haushalt mit Mitglied 2026 ohne Beitrag).
- `FeeExpectation.memberId` steht im JSON, in der OpenAPI-Spec und in `schema.d.ts`; die UI zeigt das Mitglied noch nicht an.
- Tests: `membership_generation_integration_test.go` (ein Beitrag pro Mitglied, Übernahme des Altbeitrags, beendete Mitgliedschaft, Haushalt ohne Mitglied, Wiederholbarkeit; Import-Zuordnung über das Kind).

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
- **Ab dem Monat der Vollendung des 3. Lebensjahres** greift die altersabhängige Beitragsfreiheit nach § 17a KitaG **ausdrücklich auch für Pflegekinder** (§§ 33, 34 SGB VIII). Die Altersprüfung steht in `FeeScheduleConfig.CalculateChildcareFee` (`domain/childcare_fee.go`) **vor** dem Pflegefamilie-Zweig, d. h. jedes Kind ab dem Kindergartenalter bekommt unabhängig vom Haushaltsstatus `Beitragsfrei (ab 3 Jahren)` mit 0 €.
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
- `ImportService.AllocateTransaction` (`internal/service/import_match_actions.go`) no longer rejects transactions that already have matches; it validates `existing + new <= tx.Amount` and rejects duplicate (transaction, expectation) pairs. The single-child rule was removed — allocations may target fees of different children; the IBAN is only auto-linked to a child when all allocations of one call belong to a single child.
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

### Eltern-Zugang, Etappe 2a Teil 2 (2026-09-27)

- `ParentAccountService` bündelt Kontoauflösung, Sichtbarkeit, Beitragsstatus, Elternstunden,
  Kontaktdaten, Kinderdaten und Fehlermeldungen; Handler parsen und antworten.
- Kontakt-E-Mail und Login-E-Mail eines verknüpften `PARENT`-Kontos ändern sich atomar,
  auch bei Staff- und Admin-Änderungen. Belegte oder leere Anmelde-E-Mails werden abgewiesen;
  Refresh-Sessions bleiben bestehen. Die Elternsuche nach E-Mail erfolgt nur bei Kontoanlage
  und Rollenwechsel zu `PARENT`.
- Migration 000040 erstellt `fees.data_changes` für auditierte Eltern- und Kinderdaten sowie
  `fees.parent_reports`. Eltern bearbeiten eigene Kinderdaten über `PUT /me/children/{id}` und
  melden Fehler über `/me/reports`; fremde Bezüge liefern 404. Admin nutzt
  `/parents/{id}/changes`, `/children/{id}/changes` und `/parent-reports`.
- Der Admin-Feed mischt Kontaktänderungen, Kindänderungen, gemeldete Elternstunden und
  Fehlermeldungen. Nur von Eltern vorgenommene Datenänderungen erscheinen im Feed.

## 2026-11-01: Spenden im Bankabgleich

Eine Banktransaktion lässt sich als Spende zuordnen; die Seite „Spenden“ gruppiert sie nach Buchungsjahr mit Summen.
Spec und offene Punkte: `docs/spenden.md`.

- Entscheidung: statt einer Spenden-Tabelle die allgemeine `fees.transaction_assignments` (Migration 000050, `kind`
  `DONATION`). Weitere Zuordnungsarten (Zuschüsse, Phase 8) sind ein neuer `kind`, nicht neue Prüfstellen. Gebaut
  ist nur die Spende.
- Die „schon belegt“-Prüfung der Zuordnungswege (`payment_matches`, Planzahlung, Zweck) steht einmal in
  `transactionOccupiedSQL`; Service-Seite `BankTransaction.HasExclusiveAssignment`.
- Rollen: `ADMIN`/`USER` wie der Bankabgleich (Annahme, siehe Spec); keine Schwärzung nötig.
- Die Planzahlungs-Hilfe `uploadPayment` der E2E-Tests liegt jetzt in `e2e/beitraege/bank.ts`.

## 2026-11-01: Spenden – Sichtbarkeit für Eltern und im Haushalt, Review

- Eltern sehen die Spenden ihres Haushalts über `GET /me/donations` (Positivliste, ohne IBAN, Zweck, Notiz), Staff in der
  `HouseholdDonationsCard` auf Kindseite und Elternstunden-Familienseite (`GET /import/donations?householdId=`).
- Review der ersten Umsetzung, korrigiert:
  - Enter in der Haushaltssuche speicherte die Spende; Warnungen blieben nach dem Zuordnen offen.
  - Spenden trugen einen Ratenplan-Vorschlag; Einzelabrufe rechneten Haushaltsvorschläge und hielten voll zugeordnete
    Transaktionen für offen. Vorschläge kommen jetzt nur noch in der Liste des Bankabgleichs.
  - Ausgeblendete Eingänge konnten Spende werden, Spenden und Planzahlungen ließen sich ausblenden.
  - DB-Fehler wurden beim Zuordnen zu 404.
- Aufgeräumt: Es gab schon `parent-work/HouseholdPicker`, die zweite Komponente gleichen Namens ist wieder entfernt. Das
  Original liegt jetzt in `components/` und bedient auch Ratenplan-Formular und Spenden. Es lädt alle Haushalte über
  `GET /households/options`, weil `/households` bei 100 Einträgen endet (live 94) und nicht nach Kindern sucht.
  Die Bedingung „Plan oder Zweck“ steht einmal in `inOtherLedger`. Migration 000052 entfernt den wirkungslosen Index
  auf `kind`.

## 2026-11-01: Spenden – Haushalt und IBAN-Vorschlag

Migration 000051: `transaction_assignments.household_id`. Spenden lassen sich einem Haushalt zuordnen und später
ändern (`PUT /import/transactions/{id}/assign`); offene Eingänge mit bekannter IBAN tragen einen `householdSuggestion`,
den das Panel vorwählt. Bewusst kein stilles Auto-Zuordnen. Details: `docs/spenden.md`.

## 2026-11-01: Ratenzahlungen – Frontend

- Seiten `/ratenzahlungen` und `/ratenzahlungen/:id` (Route-Meta `requiresAdmin`). Der Navigationseintrag
  „Ratenzahlungen“ steht im bestehenden `auth.isAdmin`-Block; es gibt kein neues Rollen-Flag.
- Der Bankabgleich-Dialog hat den Schalter „Beitrag | Ratenzahlung“ nur für Admins. Ein Bankvorschlag öffnet
  direkt den Ratenzahlungs-Tab; Zuordnung oder „Neue Ratenzahlung anlegen“ (vorbefüllt aus der Buchung).
- Erinnerungen: Ratenzahlungsfälle stehen über den Beitragsfällen (`InstallmentReminderList`), der Versand
  läuft über dieselbe Dialogkomponente wie auf der Detailseite (`InstallmentReminderSheet`).
- Der Zeitplan kommt vollständig aus der API. Lokal berechnet wird nur der Hinweis „8 Raten à …“ im
  Formular (`schedulePreviewText`).
- `transactionId` wird nur beim Anlegen gesendet; Updates lehnt das Backend ab. Schlägt der Vertrags-Upload
  nach dem Anlegen fehl, bleibt der Dialog offen und ein erneutes Speichern aktualisiert denselben Plan.
- Bankzeilen mit Planzahlung zählen als zugeordnet (`matched` in `useImportTransactions`), damit sie in der
  Liste nicht als „Nicht zugeordnet“ erscheinen. Die Zeile verlinkt auf den Plan.
- Statusfarben: Läuft = `badge-primary`, Rückstand = `badge-danger`, Abgeschlossen = `badge-success`,
  Beendet = neutral.
- Rollen: `domain.InstallmentManagerRoles` ist die eine Stelle fürs Backend (auch Router und Schwärzung), im
  Frontend `auth.isAdmin`. `USER` teilt sich die Banklisten: Vorschläge und Schuldnernamen werden dort entfernt,
  Aufheben einer Planzahlung gibt 403.
- `reminderDue` im Detail vermeidet doppelte Fristlogik; innerhalb der Mahnfrist bietet die Detailseite keinen Versand an.
- Nicht umgesetzt (Entwurf, ohne Backend): PDF-Brief aus dem Erinnerungsdialog, Bestätigungsmail bei der
  Schlussrate. Die Haushalt-Zeile der Detailseite ist Text, weil es noch keine Haushaltsseite gibt.

## 2026-11-01: Ratenzahlungen – Backend

Migration 000049 ergänzt vertragliche Ratenpläne für Haushalte und externe Schuldner, Zahlungen, Mahngebühren und PDF-Verträge. ADMIN-Routen unter `/installment-plans` liefern FIFO-Ratenpläne mit Monatsende-Klemmung und Schlussrate. Bankbuchungen werden atomar entweder Beiträgen oder einem Ratenplan zugeordnet; Banklisten, IBAN-Löschung und Rückzuordnung berücksichtigen Planzahlungen. Erinnerungen verwenden vorhandene E-Mail-, Frist-, Gebühren- und Zahlungsdaten/QR-Helfer; Versand protokolliert Stufe und Frist und legt explizit gewählte Mahngebühren an. PDF-Verträge sind auf 10 MiB begrenzt (nur dieser Upload erhält 11 MiB Request-Limit für Multipart-Overhead; sonst weiter 5 MiB). API-Vertrag: `docs/ratenzahlungen.md`, generiertes OpenAPI und Frontend-Typen. Die Benutzeroberfläche steht im Abschnitt „Frontend“ darüber.

Die generierten Request-Typen kennzeichnen Vorgabewerte sowie die alternativen Bank-/manuellen Zahlungsfelder als optional; Pflichtangaben je Variante werden im Service validiert. `transactionId` beim Erstellen ist optional und wird bei Updates abgelehnt. Datumsangaben in Requests sind `YYYY-MM-DD`, Antworten verwenden Go-Zeitstempel als RFC3339.
