# Kita-Apps Knirpsenstadt

Zeiterfassung, Dienstplanung und Beitragsverwaltung für die Kita Knirpsenstadt.

## Projekt-Übersicht

| App | URL | Port (Dev) | Beschreibung |
|-----|-----|------------|--------------|
| Dienstplan | plan.knirpsenstadt.de | 5173 | Wochenplanung, Gruppenübersicht |
| Zeiterfassung | zeit.knirpsenstadt.de | 5174 | Ein-/Ausstempeln, Zeitübersicht |
| Beiträge | beitraege.knirpsenstadt.de | 5175 | Beitragsverwaltung, Zahlungsabgleich |

## Tech-Stack

- **Backend (Go)**: Chi Router, PostgreSQL, JWT Auth
- **Frontend**: Vue 3, TypeScript, Tailwind CSS, shadcn-vue
- **Build**: Bun, Vite
- **Deployment**: Docker, Caddy

## Schnellstart (Entwicklung)

### Voraussetzungen

- Go 1.21+
- Bun 1.x
- Docker & Docker Compose

### 1. Datenbank starten

```bash
cd docker
docker compose up db -d
```

### 2. Backends starten

**Go Backend (Dienstplan, Zeiterfassung):**
```bash
cd backend-management

# Migrationen ausführen (einmalig)
go run cmd/migrate/main.go up

# Server starten
go run cmd/server/main.go
```
Das Backend läuft auf http://localhost:8080

**Go Backend (Beiträge):**
```bash
cd backend-fees

# Migrationen ausführen (einmalig)
go run cmd/migrate/main.go up

# Server starten
go run cmd/server/main.go
```
Das Beiträge-Backend läuft auf http://localhost:8081

### 3. Frontend starten

```bash
cd frontend
bun install
bun run dev:plan  # Dienstplan auf :5173
bun run dev:zeit  # Zeiterfassung auf :5174
bun run dev:beitraege  # Beiträge auf :5175
```

## Projektstruktur

```
kita-apps/
├── backend-management/        # Go Backend (Dienstplan, Zeiterfassung)
│   ├── cmd/
│   │   ├── server/            # HTTP Server
│   │   └── migrate/           # Migration CLI
│   ├── internal/
│   │   ├── api/               # HTTP Handlers & Router
│   │   ├── auth/              # JWT Authentication
│   │   ├── config/            # Configuration
│   │   ├── domain/            # Domain Models
│   │   ├── repository/        # Database Layer
│   │   ├── service/           # Business Logic
│   │   └── testutil/          # Test Utilities
│   └── migrations/            # SQL Migrations
│
├── backend-fees/              # Go Backend (Beiträge)
│   ├── cmd/
│   │   ├── server/            # HTTP Server
│   │   └── migrate/           # Migration CLI
│   ├── internal/
│   │   ├── api/               # HTTP Handlers & Router
│   │   ├── auth/              # JWT Authentication
│   │   ├── config/            # Configuration
│   │   ├── domain/            # Domain Models
│   │   ├── repository/        # Database Layer
│   │   ├── service/           # Business Logic
│   │   └── csvparser/         # Bank CSV Import
│   └── migrations/            # SQL Migrations
│
├── frontend/
│   ├── apps/
│   │   ├── dienstplan/        # Vue App für plan.knirpsenstadt.de
│   │   ├── zeiterfassung/     # Vue App für zeit.knirpsenstadt.de
│   │   └── beitraege/         # Vue App für beitraege.knirpsenstadt.de
│   └── packages/
│       └── shared/            # Geteilte Komponenten, API-Client, Utils
│
├── docker/
│   ├── docker-compose.yml     # Development
│   ├── docker-compose.prod.yml
│   ├── Caddyfile
│   └── Dockerfile.*
│
└── scripts/
    └── generate-api.sh
```

## Entwicklung

### OpenAPI / Type Generation

Both backends use [swag](https://github.com/swaggo/swag) to generate OpenAPI specs from Go code annotations. The frontend then uses these specs to generate TypeScript types.

```
openapi/
├── management/
│   ├── swagger.yaml      # Swagger 2.0 spec
│   └── openapi3.yaml     # OpenAPI 3.0 spec (used by frontend)
└── fees/
    ├── swagger.yaml      # Swagger 2.0 spec
    └── openapi3.yaml     # OpenAPI 3.0 spec (used by frontend)
```

#### Regenerate OpenAPI specs and TypeScript types

```bash
# Management backend
cd backend-management
~/go/bin/swag init -g cmd/server/main.go -o ../openapi/management --outputTypes yaml
npx swagger2openapi ../openapi/management/swagger.yaml -o ../openapi/management/openapi3.yaml

# Fees backend
cd backend-fees
~/go/bin/swag init -g cmd/server/main.go -o ../openapi/fees --outputTypes yaml
npx swagger2openapi ../openapi/fees/swagger.yaml -o ../openapi/fees/openapi3.yaml

# Frontend types (shared package for Dienstplan/Zeiterfassung)
cd frontend/packages/shared
bun run generate:api

# Frontend types (Beiträge app)
cd frontend/apps/beitraege
bun run generate:api
```

See the individual backend READMEs for more details on the `@name` annotation syntax.

### Default Admin Login

**Dienstplan & Zeiterfassung:**
- **E-Mail**: admin@knirpsenstadt.de
- **Passwort**: admin123

**Beiträge:**
- **E-Mail**: admin@knirpsenstadt.de
- **Passwort**: admin123

> ⚠️ Die Passwörter nach dem ersten Login ändern!

### E2E Tests mit Playwright

Das Frontend verwendet Playwright für End-to-End Tests.

#### Voraussetzungen

```bash
cd frontend
bun install
bunx playwright install chromium
```

#### Tests ausführen

```bash
# Alle Tests (headless)
bun run test

# UI-Modus (interaktiv, empfohlen für Entwicklung)
bun run test:ui

# Mit sichtbarem Browser
bun run test:headed

# Debug-Modus (Schritt für Schritt)
bun run test:debug

# Nur Dienstplan-Tests
bun run test:plan

# Nur Zeiterfassung-Tests
bun run test:zeit

# Nur Beitraege-Tests
bun run test --project=beitraege

# Einzelnen Test ausführen
bunx playwright test -g "successfully logs in" --headed

# Test-Report anzeigen
bun run test:report
```

#### Test-Struktur

```
frontend/e2e/
├── fixtures/
│   └── index.ts              # Test-Utilities, Page Objects
├── tests/
│   ├── auth.setup.ts         # Authentifizierung (Dienstplan, Zeiterfassung)
│   ├── beitraege.setup.ts    # Authentifizierung (Beiträge)
│   ├── dienstplan/
│   │   ├── navigation.spec.ts    # Login, Navigation
│   │   ├── employees.spec.ts     # Mitarbeiter-CRUD
│   │   └── groups.spec.ts        # Gruppen, Besondere Tage
│   ├── zeiterfassung/
│   │   └── clock.spec.ts         # Ein-/Ausstempeln, Historie
│   └── beitraege/
│       └── children.spec.ts      # Kinder-CRUD, Login
└── .auth/                    # Gespeicherter Auth-State (gitignored)
```

#### Wichtig für CI/CD

Tests benötigen laufende Backends mit Testdaten:

```bash
# Terminal 1: Go Backend starten (Dienstplan, Zeiterfassung)
cd backend-management && go run cmd/server/main.go

# Terminal 2: Go Backend starten (Beiträge)
cd backend-fees && go run cmd/server/main.go

# Terminal 3: Tests ausführen
cd frontend && bun run test
```

Die Playwright-Konfiguration startet automatisch die Frontend-Dev-Server.

### Nützliche URLs (Development)

| URL | Beschreibung |
|-----|--------------|
| http://localhost:8080/health | Health Check (Management) |
| http://localhost:8081/health | Health Check (Fees) |
| http://localhost:8025 | MailHog (E-Mail Tester) |
| http://localhost:5173 | Dienstplan Frontend |
| http://localhost:5174 | Zeiterfassung Frontend |
| http://localhost:5175 | Beiträge Frontend |

## Deployment (Production)

### 1. Umgebungsvariablen konfigurieren

```bash
cd docker
cp .env.example .env
# .env bearbeiten und sichere Passwörter setzen
```

### 2. Container starten

```bash
docker compose -f docker-compose.prod.yml up -d
```

### 3. DNS konfigurieren

Die folgenden Subdomains müssen auf den Server zeigen:
- api.knirpsenstadt.de
- plan.knirpsenstadt.de
- zeit.knirpsenstadt.de
- beitraege.knirpsenstadt.de

Caddy kümmert sich automatisch um SSL-Zertifikate via Let's Encrypt.

## Dokumentation

- [PLAN.md](PLAN.md) - Detaillierter Projektplan
- [backend-management/README.md](backend-management/README.md) - Management Go-Backend Dokumentation
- [backend-fees/README.md](backend-fees/README.md) - Beiträge Go-Backend Dokumentation

## License

Privat - Kita Knirpsenstadt
