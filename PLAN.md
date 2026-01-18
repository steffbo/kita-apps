# Kita-Apps Projektplan - Knirpsenstadt

## Projektübersicht

| Attribut | Wert |
|----------|------|
| **Projekt** | Kita-Apps für Knirpsenstadt |
| **Ziel** | Zeiterfassung + Dienstplanung für ~15-18 Mitarbeiter |
| **Subdomains** | `plan.knirpsenstadt.de`, `zeit.knirpsenstadt.de` |
| **Architektur** | Monorepo, Spec-First (OpenAPI) |

---

## Tech-Stack

### Backend

| Komponente | Technologie |
|------------|-------------|
| Framework | Spring Boot 3.x |
| Sprache | Java 21 |
| Datenbank | PostgreSQL 16 |
| Auth | Spring Security + JWT |
| API Spec | OpenAPI 3.0 |
| Code-Gen | openapi-generator-maven-plugin |

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
├── backend/
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
├── frontend/
│   ├── apps/
│   │   ├── dienstplan/
│   │   └── zeiterfassung/
│   ├── packages/
│   │   └── shared/
│   ├── package.json
│   └── bun.lockb
├── docker/
│   ├── docker-compose.yml
│   ├── docker-compose.prod.yml
│   ├── Caddyfile
│   ├── Dockerfile.backend
│   ├── Dockerfile.frontend-plan
│   ├── Dockerfile.frontend-zeit
│   └── backup/
│       └── pg-backup.sh
└── scripts/
    └── generate-api.sh
```

---

## Datenmodell

### Entitäten

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

### Admin-Bereich (beide Apps)
- Mitarbeiter anlegen/bearbeiten/löschen
- Arbeitszeiten konfigurieren
- Gruppen verwalten
- Passwort zurücksetzen für MA

### Export
- PDF: Zeitnachweise, Dienstpläne
- Excel: Alle Daten für Lohnabrechnung

---

## Entwicklungsphasen

### Phase 0: Setup & OpenAPI (3-4 Tage)
- [ ] Monorepo initialisieren
- [ ] OpenAPI Spec schreiben
- [ ] Backend-Projekt (Spring Boot)
- [ ] Frontend-Workspace (Bun + Vue)
- [ ] Code-Generierung einrichten
- [ ] Docker Basis-Setup
- [ ] Caddy Konfiguration

### Phase 1: Auth & Users (1 Woche)
- [ ] JWT Authentication
- [ ] Login/Logout
- [ ] Passwort-Reset (E-Mail)
- [ ] User CRUD
- [ ] Rollen-System

### Phase 2: Stammdaten (3-4 Tage)
- [ ] Gruppen CRUD
- [ ] Mitarbeiter-Gruppen-Zuordnung
- [ ] Arbeitszeit-Konfiguration

### Phase 3: Zeiterfassung (1-2 Wochen)
- [ ] Stempel-API (clockIn/clockOut)
- [ ] Pausen-Erfassung
- [ ] Frontend: Stempeluhr
- [ ] Frontend: Tagesübersicht
- [ ] Frontend: Monatsübersicht
- [ ] Admin: Korrektur mit Audit

### Phase 4: Dienstplanung (2 Wochen)
- [ ] Wochenplan-API
- [ ] Feiertage Brandenburg
- [ ] Spezielle Tage
- [ ] Frontend: Wochenansicht
- [ ] Frontend: Drag & Drop
- [ ] Frontend: Gruppenansicht
- [ ] Frontend: Readonly für MA

### Phase 5: Statistiken (1 Woche)
- [ ] Überstunden-Berechnung
- [ ] Resturlaub-Tracking
- [ ] Soll/Ist-Vergleich
- [ ] Wöchentliche Reports
- [ ] Monatliche Reports
- [ ] Dashboard für Leitung

### Phase 6: Export (3-4 Tage)
- [ ] PDF-Generation
- [ ] Excel-Export
- [ ] Zeitnachweise
- [ ] Dienstpläne

### Phase 7: Deployment (2-3 Tage)
- [ ] Production Docker Setup
- [ ] Caddy SSL
- [ ] Backup-Cron
- [ ] Health Checks
- [ ] Dokumentation

---

## Geschätzter Zeitrahmen

| Phase | Dauer |
|-------|-------|
| Phase 0 | 3-4 Tage |
| Phase 1 | 5-7 Tage |
| Phase 2 | 3-4 Tage |
| Phase 3 | 7-10 Tage |
| Phase 4 | 10-14 Tage |
| Phase 5 | 5-7 Tage |
| Phase 6 | 3-4 Tage |
| Phase 7 | 2-3 Tage |
| **Gesamt** | **~6-8 Wochen** |

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
