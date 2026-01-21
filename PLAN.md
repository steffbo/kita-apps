# Kita-Apps Projektplan - Knirpsenstadt

## Projektübersicht

| Attribut | Wert |
|----------|------|
| **Projekt** | Kita-Apps für Knirpsenstadt |
| **Ziel** | Zeiterfassung + Dienstplanung + Beitragsverwaltung |
| **Subdomains** | `plan.knirpsenstadt.de`, `zeit.knirpsenstadt.de`, `beitraege.knirpsenstadt.de` |
| **Architektur** | Monorepo, Spec-First (OpenAPI für Java), REST API (für Go) |

---

## Aktueller Stand (Januar 2026)

### Was ist fertig?

#### Infrastruktur & Setup
- [x] Monorepo-Struktur aufgesetzt
- [x] OpenAPI Spec (`/openapi/kita-api.yaml`) vollständig definiert
- [x] Backend Spring Boot 3.3 Projekt mit Java 21
- [x] Frontend Bun + Vue 3 Monorepo mit zwei Apps
- [x] Code-Generierung für API-Typen (Backend & Frontend)
- [x] Shared Package für gemeinsame Komponenten
- [x] PostgreSQL Datenbankschema mit Migrationen (V1-V3)
- [x] E2E Testing Setup mit Playwright

#### Beiträge-Backend (Go) - NEU
- [x] Go Backend mit Chi Router auf Port 8081
- [x] Separates `fees` Schema in PostgreSQL
- [x] JWT Authentication (unabhängig vom Java Backend)
- [x] golang-migrate für Datenbankmigrationen
- [x] REST API für Kinder, Eltern, Beiträge, Import

#### Beiträge-Frontend - NEU
- [x] Vue 3 App auf Port 5175
- [x] Dashboard mit Beitragsübersicht
- [x] Kinder-Verwaltung (CRUD, Suche, Detail-Ansicht)
- [x] Eltern-Verwaltung (Grid-Ansicht)
- [x] Beiträge-Tabelle mit Filtern
- [x] CSV-Import für Kontoauszüge
- [x] Playwright E2E Tests

#### Backend - Fertige Features
- [x] JWT Authentication (Login, Refresh, Logout)
- [x] Passwort-Reset Flow
- [x] Passwort ändern
- [x] Mitarbeiter CRUD mit primaryGroupId (Stammgruppe)
- [x] Gruppen CRUD
- [x] Gruppen-Zuweisungen (PERMANENT/SPRINGER)
- [x] Schedule Entries CRUD (Dienstplan-Einträge)
- [x] Bulk Create für Schedule Entries
- [x] Time Tracking (Clock In/Out, Pausen)
- [x] Time Entries CRUD
- [x] Special Days (Feiertage Brandenburg automatisch, Schließzeiten, Teamtage, Events)
- [x] **Statistics Service** - Wochen- und Monatsstatistiken mit Kapazitätsberechnung

#### Frontend - Dienstplan App
- [x] Login/Logout mit JWT
- [x] **SchedulePage** - Wochenansicht mit Drag & Drop (FullCalendar), Gruppenfilter
- [x] **ScheduleEntryDialog** - Eintrag erstellen/bearbeiten mit Validierung
- [x] **EmployeesPage** - Mitarbeiter-Tabelle mit Icon-Buttons (Edit, Reset PW, Deaktivieren)
  - Klickbare Zeilen zum Bearbeiten (nur Admin)
- [x] **GroupsPage** - Gruppen-Karten mit direkter Mitglieder-Anzeige
  - "Springer"-Karte für Mitarbeiter ohne Stammgruppe
  - Icon-Buttons (Pencil, Trash2)
- [x] **SpecialDaysPage** - Feiertage, Schließzeiten, Teamtage, Events
  - Enddate für mehrtägige Schließzeiten
  - **Sektionsspezifische Add-Buttons** (CirclePlus pro Sektion)
- [x] **StatisticsPage** - Monatsübersicht + Wochen-Kapazitätsansicht
  - Vergleich: Vertrags-Stunden vs. Geplant vs. Gearbeitet
  - Progress-Bars mit Farbcodierung (unter/optimal/über Kapazität)
  - Legende für Kapazitätsauslastung
- [x] Wochenenden ein-/ausblenden (Mo-Fr / Mo-So Toggle)
- [x] Responsive Design mit Tailwind CSS + shadcn-vue Komponenten

#### Frontend - Zeiterfassung App
- [x] Login/Logout
- [x] Clock In/Out Buttons
- [x] Tagesübersicht
- [x] Monatsübersicht eigener Einträge

#### Wichtige Bug-Fixes (diese Session)
- [x] **Timezone-Bug in `toISODateString()`** - `toISOString()` konvertierte zu UTC, Daten verschoben sich um einen Tag in UTC+1. Jetzt lokal berechnet.
- [x] **Springer-Auswahl im Dialog** - Bei Mitarbeiter ohne primaryGroupId wird Gruppe auf 'none' gesetzt
- [x] **Formular-Validierung** - Submit-Button nur aktiv wenn Mitarbeiter ausgewählt

---

### Was wurde explizit angefordert?

1. **Springer-Selection fixen** - Wenn Mitarbeiter ohne primaryGroupId ausgewählt wird, soll Gruppe auf "Springer (keine Gruppe)" wechseln ✅
2. **Schedule Entries werden nicht gespeichert** - Timezone-Bug und fehlende Validierung ✅
3. **Enddatum für Schließzeiten** - Mehrtägige Schließzeiten (z.B. Sommerschließzeit) ✅
4. **Gruppen-Mitglieder direkt anzeigen** - Kein Expand/Collapse mehr, direkte Anzeige ✅
5. **Mitarbeiter-Zeilen klickbar** - Row-Click öffnet Edit-Dialog (nur Admin) ✅
6. **Icon-Buttons statt Text** - Pencil, KeyRound, UserX, Trash2 für Actions ✅
7. **Wochenenden ausblenden** - Mo-Fr als Default, Toggle für volle Woche ✅
8. **Sektionsspezifische Add-Buttons** - CirclePlus pro Sektion in SpecialDaysPage ✅
9. **Wochen-Kapazitätsansicht** - Vertrags-Stunden vs. Geplant vs. Gearbeitet mit Visualisierung ✅

---

### Worauf haben wir geachtet?

#### Code-Qualität
- **Type Safety** - Strikte TypeScript-Typen aus OpenAPI generiert
- **Validierung** - Formulare validieren vor Submit, Buttons disabled wenn invalid
- **Fehlerbehandlung** - Try/Catch mit User-Feedback, Loading States
- **Lokalisierung** - Deutsche UI-Texte durchgängig

#### UX-Entscheidungen
- **Admin vs. User** - Features nur für Admins sichtbar (`v-if="isAdmin"`)
- **Konsistente Icons** - Lucide Icons durchgängig (Pencil=Edit, Trash2=Delete, etc.)
- **Farbcodierung** - Gruppen haben Farben, Kapazitäts-Status hat Ampelfarben
- **Feedback** - Loading Spinner, Disabled States, Hover Effects

#### Daten-Integrität
- **Timezone-Handling** - Lokale Zeitzone für Datumsberechnung statt UTC
- **Referentielle Integrität** - primaryGroupId statt separater Tabelle für Stammgruppe
- **Soft Delete** - Mitarbeiter werden deaktiviert, nicht gelöscht

---

## Tech-Stack

### Backend (Java - Dienstplan, Zeiterfassung)

| Komponente | Technologie |
|------------|-------------|
| Framework | Spring Boot 3.x |
| Sprache | Java 21 |
| Datenbank | PostgreSQL 16 |
| Auth | Spring Security + JWT |
| API Spec | OpenAPI 3.0 |
| Code-Gen | openapi-generator-maven-plugin |

### Backend (Go - Beiträge)

| Komponente | Technologie |
|------------|-------------|
| Framework | Chi Router |
| Sprache | Go 1.21+ |
| Datenbank | PostgreSQL 16 (fees Schema) |
| Auth | JWT (eigene Implementierung) |
| Migration | golang-migrate |
| CSV Parser | Sparkasse CSV Format |

### Frontend

| Komponente | Technologie |
|------------|-------------|
| Framework | Vue 3 + Composition API |
| Sprache | TypeScript |
| Build | Vite (via Bun) |
| Runtime | Bun |
| UI | Tailwind CSS + shadcn-vue |
| Theme | Nova / Stone / Green / Small Radius |
| State | TanStack Query (Vue) |
| Kalender | FullCalendar |
| Drag & Drop | VueDraggable |
| API Client | openapi-typescript + openapi-fetch |

### Infrastruktur

| Komponente | Technologie |
|------------|-------------|
| Container | Docker + Docker Compose |
| Reverse Proxy | Caddy |
| SSL | Let's Encrypt (via Caddy, auto) |
| Hosting | Eigener VPS |
| Backup | Automatische PostgreSQL Dumps |
| Monitoring | Basic Health Checks + Logs |
| E-Mail | Vorhandener SMTP Server |

---

## Projektstruktur

```
kita-apps/
├── openapi/
│   └── kita-api.yaml
├── backend/                   # Java Backend
│   ├── pom.xml
│   └── src/main/java/de/knirpsenstadt/
│       ├── api/            # Generiert
│       ├── controller/
│       ├── service/
│       ├── repository/
│       ├── model/
│       ├── dto/
│       ├── config/
│       └── util/
├── backend-fees/              # Go Backend
│   ├── cmd/
│   │   ├── server/         # HTTP Server
│   │   └── migrate/        # Migration CLI
│   ├── internal/
│   │   ├── api/            # Router, Handler
│   │   ├── auth/           # JWT
│   │   ├── domain/         # Entities
│   │   ├── repository/     # DB Layer
│   │   └── service/        # Business Logic
│   └── migrations/         # SQL Files
├── frontend/
│   ├── apps/
│   │   ├── dienstplan/
│   │   ├── zeiterfassung/
│   │   └── beitraege/
│   ├── packages/
│   │   └── shared/
│   ├── package.json
│   └── bun.lockb
├── docker/
│   ├── docker-compose.yml
│   ├── docker-compose.prod.yml
│   ├── Caddyfile
│   ├── Dockerfile.backend
│   ├── Dockerfile.backend-fees
│   ├── Dockerfile.frontend-plan
│   ├── Dockerfile.frontend-zeit
│   ├── Dockerfile.frontend-beitraege
│   └── backup/
│       └── pg-backup.sh
└── scripts/
    └── generate-api.sh
```

---

## Datenmodell

### Entitäten (Java Backend - public Schema)

**Employee**
- id, email, firstName, lastName
- passwordHash, role (ADMIN/EMPLOYEE)
- weeklyHours, vacationDaysPerYear, remainingVacationDays

**Group**
- id, name, description

**GroupAssignment**
- id, employeeId, groupId
- type (PERMANENT/SPRINGER)

**ScheduleEntry**
- id, employeeId, date, startTime, endTime
- groupId, type (WORK/VACATION/SICK/SPECIAL)
- notes

**TimeEntry**
- id, employeeId, date
- clockIn, clockOut, breakMinutes
- type, editedBy, editedAt, notes

**SpecialDay**
- id, date, name
- type (HOLIDAY/CLOSURE/TEAM_DAY/EVENT)
- affectsAll, notes

### Entitäten (Go Backend - fees Schema)

**Child**
- id, memberNumber, firstName, lastName
- birthDate, entryDate
- street, houseNumber, postalCode, city
- isActive, createdAt, updatedAt

**Parent**
- id, firstName, lastName, birthDate
- email, phone
- street, houseNumber, postalCode, city
- annualHouseholdIncome
- createdAt, updatedAt

**ChildParent** (Many-to-Many)
- childId, parentId, isPrimary

**User** (separate vom Java Backend)
- id, email, passwordHash
- firstName, lastName, role (ADMIN/USER)
- isActive, createdAt, updatedAt

**FeeExpectation**
- id, childId, feeType (MEMBERSHIP/FOOD/CHILDCARE)
- year, month (null für Jahresbeiträge)
- amount, dueDate, createdAt

**BankTransaction**
- id, bookingDate, valueDate
- payerName, payerIban, description
- amount, currency, importBatchId, importedAt

**PaymentMatch**
- id, transactionId, expectationId
- matchType (AUTO/MANUAL), confidence
- matchedAt, matchedBy

---

## Features nach App

### Zeiterfassung (zeit.knirpsenstadt.de)

**Mitarbeiter:**
- Ein-/Ausstempeln (großer Button)
- Pause erfassen
- Tagesübersicht mit aktuellem Status
- Monatsübersicht eigener Einträge
- Soll/Ist-Vergleich mit Dienstplan

**Leitung (Admin):**
- Alle Zeiteinträge einsehen
- Einträge korrigieren (mit Audit-Log)
- Fehlende Einträge nachtragen

### Dienstplan (plan.knirpsenstadt.de)

**Mitarbeiter (Readonly):**
- Wochenplan einsehen
- Eigene Schichten sehen
- Gruppenübersicht

**Leitung (Admin):**
- Wochenplan erstellen/bearbeiten
- Drag & Drop für Mitarbeiter
- Zeiten per Drag anpassen
- Gruppenbasierte Ansicht (3 Spalten)
- Abwesenheiten eintragen (Urlaub, Krank)
- Spezielle Tage verwalten:
  - Feiertage Brandenburg (automatisch)
  - Schließzeiten (Sommer, Weihnachten)
  - Bildungstage
  - Events (Busfahrt, Laternenumzug, Übernachtung)
- Wöchentliche/monatliche Statistiken
- Überstunden-Übersicht aller MA
- Resturlaub-Übersicht

### Admin-Bereich (Dienstplan & Zeiterfassung)
- Mitarbeiter anlegen/bearbeiten/löschen
- Arbeitszeiten konfigurieren
- Gruppen verwalten
- Passwort zurücksetzen für MA

### Beiträge (beitraege.knirpsenstadt.de)

**Beitragsarten:**
- **Vereinsbeitrag**: 30,00 € jährlich (Januar fällig)
- **Essensgeld**: 45,40 € monatlich
- **Platzgeld (U3)**: 100,00 € monatlich (einkommensabhängig geplant)

**Dashboard:**
- Übersicht offene/bezahlte/überfällige Beiträge
- Monatliche Aufschlüsselung
- Jahresfilter

**Kinder-Verwaltung:**
- CRUD mit Mitgliedsnummer, Name, Geburtsdatum, Eintrittsdatum
- Adressdaten optional
- U3-Status (automatisch berechnet)
- Aktiv/Inaktiv-Status
- Verknüpfte Eltern anzeigen

**Eltern-Verwaltung:**
- CRUD mit Kontaktdaten
- Haushaltseinkommen für Platzgeld-Berechnung
- Verknüpfung zu Kindern (Many-to-Many)

**Beitrags-Verwaltung:**
- Automatische Generierung für Zeitraum
- Filtern nach Jahr, Monat, Typ, Status
- Bezahlt-Status mit Zahlungsdatum

**CSV-Import (Kontoauszüge):**
- Upload von Sparkasse-CSV
- Automatisches Matching anhand:
  - Mitgliedsnummer im Verwendungszweck
  - Name des Zahlenden
  - Betragshöhe
- Manuelles Matching für nicht erkannte Zahlungen
- Import-Historie

### Export
- PDF: Zeitnachweise, Dienstpläne
- Excel: Alle Daten für Lohnabrechnung

---

## Entwicklungsphasen

### Phase 0: Setup & OpenAPI ✅ FERTIG
- [x] Monorepo initialisieren
- [x] OpenAPI Spec schreiben
- [x] Backend-Projekt (Spring Boot)
- [x] Frontend-Workspace (Bun + Vue)
- [x] Code-Generierung einrichten
- [x] Docker Basis-Setup
- [x] Caddy Konfiguration

### Phase 1: Auth & Users ✅ FERTIG
- [x] JWT Authentication
- [x] Login/Logout
- [x] Passwort-Reset (E-Mail)
- [x] User CRUD
- [x] Rollen-System (ADMIN/EMPLOYEE)

### Phase 2: Stammdaten ✅ FERTIG
- [x] Gruppen CRUD
- [x] Mitarbeiter-Gruppen-Zuordnung (primaryGroupId)
- [x] Arbeitszeit-Konfiguration (weeklyHours)

### Phase 3: Zeiterfassung ✅ FERTIG
- [x] Stempel-API (clockIn/clockOut)
- [x] Pausen-Erfassung
- [x] Frontend: Stempeluhr
- [x] Frontend: Tagesübersicht
- [x] Frontend: Monatsübersicht
- [x] Admin: Korrektur mit Audit

### Phase 4: Dienstplanung ✅ FERTIG
- [x] Wochenplan-API
- [x] Feiertage Brandenburg (automatisch)
- [x] Spezielle Tage (Schließzeiten, Teamtage, Events)
- [x] Frontend: Wochenansicht
- [x] Frontend: Drag & Drop
- [x] Frontend: Gruppenansicht
- [x] Frontend: Readonly für MA

### Phase 5: Statistiken ✅ FERTIG
- [x] Überstunden-Berechnung
- [x] Resturlaub-Tracking
- [x] Soll/Ist-Vergleich
- [x] Wöchentliche Reports (Kapazitätsansicht)
- [x] Monatliche Reports
- [x] Dashboard für Leitung

### Phase 6: Beitragsverwaltung ✅ FERTIG
- [x] Go Backend Setup (Chi Router, Port 8081)
- [x] PostgreSQL fees Schema mit Migrationen
- [x] JWT Authentication (unabhängig)
- [x] Kinder-API (CRUD)
- [x] Eltern-API (CRUD)
- [x] Kind-Eltern-Verknüpfung (Many-to-Many)
- [x] Beitrags-API (Generierung, Übersicht)
- [x] CSV-Import für Kontoauszüge
- [x] Payment-Matching (Auto/Manuell)
- [x] Vue Frontend auf Port 5175
- [x] Dashboard mit Statistiken
- [x] Kinder-Seite (Liste, Suche, Detail)
- [x] Eltern-Seite (Grid-Ansicht)
- [x] Beiträge-Seite (Tabelle, Filter)
- [x] Import-Seite (Upload, Matching, Historie)
- [x] Playwright E2E Tests

### Phase 7: Export 🔄 OFFEN
- [ ] PDF-Generation
- [ ] Excel-Export
- [ ] Zeitnachweise
- [ ] Dienstpläne

### Phase 8: Deployment 🔄 TEILWEISE
- [x] Production Docker Setup
- [x] Caddy SSL
- [ ] Backup-Cron
- [ ] Health Checks
- [ ] Dokumentation

---

## Nächste Schritte

### Priorität 1: Export-Funktionen
- [ ] PDF-Export für Dienstpläne (Wochenansicht)
- [ ] PDF-Export für Zeitnachweise (Monat pro Mitarbeiter)
- [ ] Excel-Export für Lohnabrechnung

### Priorität 2: Feinschliff
- [ ] E-Mail-Versand für Passwort-Reset (aktuell nur Console-Log)
- [ ] E-Mail-Benachrichtigung bei Dienstplanänderungen
- [ ] Vacation-Request Workflow (Mitarbeiter beantragt, Admin genehmigt)

### Priorität 3: Production-Readiness
- [ ] Automatische Backups (PostgreSQL Dumps)
- [ ] Health Check Endpoints
- [ ] Error Monitoring / Logging
- [ ] Benutzer-Dokumentation

---

## Geschätzter Zeitrahmen

| Phase | Status | Verbleibend |
|-------|--------|-------------|
| Phase 0-6 | ✅ Fertig | - |
| Phase 7 (Export) | 🔄 Offen | ~3-4 Tage |
| Phase 8 (Deploy) | 🔄 Teilweise | ~1-2 Tage |
| **Verbleibend** | | **~4-6 Tage** |

---

## Konfiguration

### Feiertage Brandenburg (automatisch berechnet)
- Neujahr (1. Januar)
- Karfreitag (variabel)
- Ostermontag (variabel)
- Tag der Arbeit (1. Mai)
- Christi Himmelfahrt (variabel)
- Pfingstmontag (variabel)
- Tag der Deutschen Einheit (3. Oktober)
- Reformationstag (31. Oktober)
- 1. Weihnachtsfeiertag (25. Dezember)
- 2. Weihnachtsfeiertag (26. Dezember)

### Spezielle Tage (manuell pflegbar)
- Schließzeiten Sommerferien
- Schließzeiten Weihnachten
- Bildungstage für Team
- Events: Busfahrt, Laternenumzug, Kita-Übernachtung

### Gruppen
- 3 Gruppen
- Je 2 feste Erzieherinnen pro Gruppe
- 2 Springer (gruppenübergreifend)

### Arbeitszeiten
- Zwischen 20 und 38 Wochenstunden pro Mitarbeiter
- Kombination aus festen Schichten und flexiblen Zeiten
