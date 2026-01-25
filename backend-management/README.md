# Management Backend (Go)

Go-based backend for schedule planning and time tracking for Kita Knirpsenstadt.

## Overview

| Property | Value |
|----------|-------|
| **Port** | 8080 |
| **Language** | Go 1.21+ |
| **Framework** | Chi Router |
| **Database** | PostgreSQL 16 (schema: `public`) |
| **Auth** | JWT (Access + Refresh Tokens) |

## Quickstart

### Prerequisites

- Go 1.21+
- PostgreSQL 16 (via Docker)
- Database `kita` must exist

### 1. Install dependencies

```bash
cd backend-management
go mod download
```

### 2. Database migrations

```bash
# Apply migrations
go run cmd/migrate/main.go -direction up

# Roll back one migration
go run cmd/migrate/main.go -direction down -steps 1

# Roll back all migrations
go run cmd/migrate/main.go -direction down
```

### 3. Start the server

```bash
go run cmd/server/main.go
```

The server runs at http://localhost:8080

### 4. Health Check

```bash
curl http://localhost:8080/api/actuator/health
# {"status":"UP"}
```

## Configuration

All settings can be configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | `postgres://kita:kita_dev_password@localhost:5432/kita?sslmode=disable` | PostgreSQL connection string |
| `CORS_ORIGINS` | `http://localhost:5173,http://localhost:5174,https://plan.knirpsenstadt.de,https://zeit.knirpsenstadt.de` | Allowed CORS origins |
| `JWT_SECRET` | `YrDhyo+bnIfg3WxnBoyZHGbZTMqtLjKWtlIEY25UDAI=` | JWT signing secret (Base64) |
| `JWT_ACCESS_EXPIRY` | `15m` | Access token lifetime |
| `JWT_REFRESH_EXPIRY` | `168h` | Refresh token lifetime (7 days) |
| `JWT_ISSUER` | `kita-management` | JWT issuer |
| `READ_TIMEOUT` | `15s` | HTTP read timeout |
| `WRITE_TIMEOUT` | `15s` | HTTP write timeout |
| `DB_MAX_OPEN_CONNS` | `25` | Max open DB connections |
| `DB_MAX_IDLE_CONNS` | `5` | Max idle DB connections |
| `DB_CONN_MAX_LIFETIME` | `5m` | Max connection lifetime |

## API Reference

The API follows the existing OpenAPI spec in `openapi/kita-api.yaml`.
Base URL: `http://localhost:8080/api`

## Database Schema

Migrations create tables for:

- Employees (employees)
- Groups and assignments (groups, group_assignments)
- Schedule entries (schedule_entries)
- Time tracking entries (time_entries)
- Special days (special_days)
