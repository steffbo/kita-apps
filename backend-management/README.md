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

The API follows the existing OpenAPI spec in `openapi/management/`.
Base URL: `http://localhost:8080/api`

## OpenAPI Spec Generation

The backend uses [swag](https://github.com/swaggo/swag) to generate OpenAPI specs from Go annotations. The generated specs are used by the frontend to generate TypeScript types.

### Prerequisites

```bash
# Install swag CLI (once)
go install github.com/swaggo/swag/cmd/swag@latest

# Install swagger2openapi for OpenAPI 3 conversion
npm install -g swagger2openapi
```

### Generate OpenAPI Spec

```bash
cd backend-management

# Generate Swagger 2.0 spec
~/go/bin/swag init -g cmd/server/main.go -o ../openapi/management --outputTypes yaml

# Convert to OpenAPI 3.0
npx swagger2openapi ../openapi/management/swagger.yaml -o ../openapi/management/openapi3.yaml
```

Generated files:
- `openapi/management/swagger.yaml` - Swagger 2.0 spec
- `openapi/management/openapi3.yaml` - OpenAPI 3.0 spec (used by frontend)

### Adding `@name` Annotations

To get clean schema names in the generated spec (without `handler.` prefixes), add `@name` annotations to type definitions:

```go
// MyResponse represents the response
// @Description API response description
type MyResponse struct {
    Field string `json:"field"`
} //@name MyResponse
```

The `@name` annotation must be on the same line as the closing brace.

### Frontend Type Generation

After regenerating the OpenAPI spec, update the frontend types:

```bash
cd frontend/packages/shared
bun run generate:api
```

This generates `src/api/schema.d.ts` with TypeScript types from the OpenAPI spec.

## Database Schema

Migrations create tables for:

- Employees (employees)
- Groups and assignments (groups, group_assignments)
- Schedule entries (schedule_entries)
- Time tracking entries (time_entries)
- Special days (special_days)
