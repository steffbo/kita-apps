# Beiträge Backend (Go)

Go-basiertes Backend für die Beitragsverwaltung der Kita Knirpsenstadt.

## Übersicht

| Eigenschaft | Wert |
|-------------|------|
| **Port** | 8081 |
| **Sprache** | Go 1.21+ |
| **Framework** | Chi Router |
| **Datenbank** | PostgreSQL 16 (Schema: `fees`) |
| **Auth** | JWT (Access + Refresh Tokens) |

## Schnellstart

### Voraussetzungen

- Go 1.21+
- PostgreSQL 16 (läuft via Docker)
- Datenbank `kita` muss existieren

### 1. Abhängigkeiten installieren

```bash
cd backend-fees
go mod download
```

### 2. Datenbank-Migrationen

```bash
# Migrationen anwenden
go run cmd/migrate/main.go up

# Migrationen zurückrollen (eine)
go run cmd/migrate/main.go down

# Alle Migrationen zurückrollen
go run cmd/migrate/main.go drop
```

### 3. Server starten

```bash
go run cmd/server/main.go
```

Der Server läuft auf http://localhost:8081

### 4. Health Check

```bash
curl http://localhost:8081/health
# {"status":"ok"}
```

## Konfiguration

Alle Einstellungen können über Umgebungsvariablen konfiguriert werden:

| Variable | Default | Beschreibung |
|----------|---------|--------------|
| `PORT` | `8081` | HTTP Server Port |
| `DATABASE_URL` | `postgres://kita:kita_dev_password@localhost:5432/kita?sslmode=disable&search_path=fees` | PostgreSQL Connection String |
| `CORS_ORIGINS` | `*` | Erlaubte CORS Origins (kommasepariert) |
| `JWT_SECRET` | `dev-secret-change-in-production` | Secret für JWT Signierung |
| `JWT_ACCESS_EXPIRY` | `15m` | Access Token Gültigkeit |
| `JWT_REFRESH_EXPIRY` | `168h` | Refresh Token Gültigkeit (7 Tage) |
| `READ_TIMEOUT` | `15s` | HTTP Read Timeout |
| `WRITE_TIMEOUT` | `15s` | HTTP Write Timeout |
| `DB_MAX_OPEN_CONNS` | `25` | Max offene DB-Verbindungen |
| `DB_MAX_IDLE_CONNS` | `5` | Max idle DB-Verbindungen |
| `DB_CONN_MAX_LIFETIME` | `5m` | Max Lebensdauer einer Verbindung |

## API Referenz

Base URL: `http://localhost:8081/api/fees/v1`

### Authentifizierung

| Methode | Endpoint | Beschreibung |
|---------|----------|--------------|
| `POST` | `/auth/login` | Login (Email + Passwort) |
| `POST` | `/auth/refresh` | Access Token erneuern |
| `POST` | `/auth/logout` | Logout (Token invalidieren) |
| `GET` | `/auth/me` | Aktueller Benutzer |

**Login Request:**
```json
{
  "email": "admin@knirpsenstadt.de",
  "password": "admin123"
}
```

**Login Response:**
```json
{
  "accessToken": "eyJhbG...",
  "refreshToken": "eyJhbG...",
  "expiresAt": "2026-01-21T19:00:00+01:00",
  "user": {
    "id": "uuid",
    "email": "admin@knirpsenstadt.de",
    "firstName": "Admin",
    "lastName": "Knirpsenstadt",
    "role": "ADMIN"
  }
}
```

### Kinder

| Methode | Endpoint | Beschreibung |
|---------|----------|--------------|
| `GET` | `/children` | Liste aller Kinder |
| `POST` | `/children` | Kind anlegen |
| `GET` | `/children/{id}` | Kind Details |
| `PUT` | `/children/{id}` | Kind aktualisieren |
| `DELETE` | `/children/{id}` | Kind löschen |
| `POST` | `/children/{id}/parents` | Elternteil verknüpfen |
| `DELETE` | `/children/{id}/parents/{parentId}` | Elternteil entfernen |

**Query Parameter (GET /children):**
- `activeOnly` (bool): Nur aktive Kinder
- `search` (string): Suche nach Name oder Mitgliedsnummer
- `limit` (int): Max. Ergebnisse
- `offset` (int): Pagination Offset

**Create Child Request:**
```json
{
  "memberNumber": "11072",
  "firstName": "Max",
  "lastName": "Mustermann",
  "birthDate": "2022-06-15",
  "entryDate": "2024-01-01"
}
```

### Eltern

| Methode | Endpoint | Beschreibung |
|---------|----------|--------------|
| `GET` | `/parents` | Liste aller Eltern |
| `POST` | `/parents` | Elternteil anlegen |
| `GET` | `/parents/{id}` | Elternteil Details |
| `PUT` | `/parents/{id}` | Elternteil aktualisieren |
| `DELETE` | `/parents/{id}` | Elternteil löschen |

### Beiträge

| Methode | Endpoint | Beschreibung |
|---------|----------|--------------|
| `GET` | `/fees` | Liste aller Beiträge |
| `GET` | `/fees/overview` | Übersicht (Statistiken) |
| `POST` | `/fees/generate` | Beiträge generieren |
| `GET` | `/fees/{id}` | Beitrag Details |
| `PUT` | `/fees/{id}` | Beitrag aktualisieren |
| `DELETE` | `/fees/{id}` | Beitrag löschen |
| `GET` | `/childcare-fee/calculate` | Platzgeld berechnen |

**Query Parameter (GET /fees):**
- `year` (int): Jahr filtern
- `month` (int): Monat filtern
- `feeType` (string): MEMBERSHIP, FOOD, CHILDCARE
- `status` (string): OPEN, PAID, OVERDUE
- `childId` (uuid): Nach Kind filtern

**Generate Fees Request:**
```json
{
  "year": 2026,
  "fromMonth": 1,
  "toMonth": 12,
  "feeTypes": ["MEMBERSHIP", "FOOD"]
}
```

### Import

| Methode | Endpoint | Beschreibung |
|---------|----------|--------------|
| `POST` | `/import/upload` | CSV-Datei hochladen |
| `POST` | `/import/confirm` | Import bestätigen |
| `GET` | `/import/history` | Import-Historie |
| `GET` | `/import/transactions` | Ungematchte Transaktionen |
| `POST` | `/import/match` | Manuelles Matching |

**CSV Upload:**
- Content-Type: `multipart/form-data`
- Field: `file` (CSV-Datei im Sparkasse-Format)

## Datenbank Schema

Das Backend verwendet ein eigenes Schema `fees` in der PostgreSQL-Datenbank:

```
fees.children         - Kinder
fees.parents          - Eltern
fees.child_parents    - Kind-Eltern-Verknüpfung (M:N)
fees.users            - Benutzer (separate Auth)
fees.refresh_tokens   - JWT Refresh Tokens
fees.fee_expectations - Erwartete Beiträge
fees.bank_transactions - Importierte Kontobewegungen
fees.payment_matches  - Zuordnungen Zahlung <-> Beitrag
```

### Beitragsarten

| Typ | Bezeichnung | Betrag | Fälligkeit |
|-----|-------------|--------|------------|
| `MEMBERSHIP` | Vereinsbeitrag | 30,00 € | Jährlich (Januar) |
| `FOOD` | Essensgeld | 45,40 € | Monatlich |
| `CHILDCARE` | Platzgeld (U3) | 100,00 € | Monatlich |

## Projektstruktur

```
backend-fees/
├── cmd/
│   ├── server/
│   │   └── main.go          # HTTP Server Einstiegspunkt
│   └── migrate/
│       └── main.go          # Migration CLI
│
├── internal/
│   ├── api/
│   │   ├── router.go        # Chi Router Setup
│   │   ├── handler/         # HTTP Handler
│   │   │   ├── auth_handler.go
│   │   │   ├── child_handler.go
│   │   │   ├── parent_handler.go
│   │   │   ├── fee_handler.go
│   │   │   └── import_handler.go
│   │   ├── middleware/      # Auth, Logging
│   │   ├── request/         # Request Parsing
│   │   └── response/        # JSON Response Helpers
│   │
│   ├── auth/
│   │   └── jwt.go           # JWT Token Handling
│   │
│   ├── config/
│   │   └── config.go        # Umgebungsvariablen
│   │
│   ├── domain/              # Domain Models
│   │   ├── child.go
│   │   ├── parent.go
│   │   ├── fee.go
│   │   ├── transaction.go
│   │   └── user.go
│   │
│   ├── repository/          # Database Layer
│   │   ├── interfaces.go
│   │   ├── child_repository.go
│   │   ├── parent_repository.go
│   │   ├── fee_repository.go
│   │   ├── transaction_repository.go
│   │   ├── match_repository.go
│   │   └── user_repository.go
│   │
│   ├── service/             # Business Logic
│   │   ├── auth_service.go
│   │   ├── child_service.go
│   │   ├── parent_service.go
│   │   ├── fee_service.go
│   │   ├── import_service.go
│   │   └── errors.go
│   │
│   └── csvparser/
│       └── bank_csv_parser.go  # Sparkasse CSV Parser
│
├── migrations/              # SQL Migrationen
│   ├── 000001_initial_schema.up.sql
│   ├── 000001_initial_schema.down.sql
│   ├── 000002_seed_admin.up.sql
│   └── 000002_seed_admin.down.sql
│
├── tests/                   # Integration Tests
├── Dockerfile
├── go.mod
└── go.sum
```

## Default Admin

Nach den Migrationen existiert ein Admin-Benutzer:

- **Email**: admin@knirpsenstadt.de
- **Passwort**: admin123

> ⚠️ Das Passwort in Production ändern!

## Development

### Build

```bash
go build -o bin/server cmd/server/main.go
go build -o bin/migrate cmd/migrate/main.go
```

### Tests

```bash
go test ./...
```

### Docker

```bash
docker build -t kita-fees-backend .
docker run -p 8081:8081 \
  -e DATABASE_URL="postgres://..." \
  -e JWT_SECRET="production-secret" \
  kita-fees-backend
```

## Unterschied zum Management Backend

| Aspekt | Management Backend | Fees Backend |
|--------|-------------------|--------------|
| Apps | Dienstplan, Zeiterfassung | Beiträge |
| Port | 8080 | 8081 |
| DB Schema | public | fees |

Die Backends sind vollständig unabhängig und teilen nur die PostgreSQL-Datenbank (verschiedene Schemas).
