# Beitragsverwaltungs-App ("Beiträge")

## Overview

A fee management application for tracking parent contributions at Kita Knirpsenstadt. The app manages three types of fees, provides child and parent data management, and matches bank transactions via CSV import.

## Fee Types

| Fee Type | German Name | Amount | Frequency | Applies To |
|----------|-------------|--------|-----------|------------|
| `MEMBERSHIP` | Vereinsbeitrag | 30.00 EUR | yearly | All parents |
| `FOOD` | Essensgeld | 45.40 EUR | monthly | All children |
| `CHILDCARE` | Platzgeld | variable (default: 100 EUR) | monthly | Children under 3 years |

The childcare fee amount is determined by the household's annual net income. For now, the calculation API returns a fixed 100 EUR regardless of input. The actual fee table (Brandenburg regulations) will be implemented later.

## Architecture

### Technology Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.22+ |
| Database | PostgreSQL 16 (shared instance, separate schema `fees`) |
| API Framework | Chi Router |
| Migrations | golang-migrate |
| Auth | JWT (access + refresh tokens) |
| Frontend | Vue 3 + TypeScript + Vite |
| UI Components | Tailwind CSS + shadcn-vue |
| State Management | TanStack Query + Pinia |
| E2E Testing | Playwright |
| Deployment | Docker + Caddy |

### Project Structure

```
kita-apps/
├── backend-fees/                         # Go Backend
│   ├── cmd/
│   │   └── server/
│   │       └── main.go                   # Application entry point
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handler/                  # HTTP request handlers
│   │   │   │   ├── auth_handler.go
│   │   │   │   ├── child_handler.go
│   │   │   │   ├── parent_handler.go
│   │   │   │   ├── fee_handler.go
│   │   │   │   └── import_handler.go
│   │   │   ├── middleware/               # HTTP middleware
│   │   │   │   ├── auth.go
│   │   │   │   ├── cors.go
│   │   │   │   └── logging.go
│   │   │   ├── request/                  # Request DTOs
│   │   │   ├── response/                 # Response DTOs
│   │   │   └── router.go                 # Route definitions
│   │   ├── auth/
│   │   │   └── jwt.go                    # JWT token handling
│   │   ├── config/
│   │   │   └── config.go                 # Environment configuration
│   │   ├── domain/                       # Domain entities
│   │   │   ├── child.go
│   │   │   ├── parent.go
│   │   │   ├── fee_expectation.go
│   │   │   ├── bank_transaction.go
│   │   │   ├── payment_match.go
│   │   │   └── user.go
│   │   ├── repository/                   # Database access layer
│   │   │   ├── child_repository.go
│   │   │   ├── parent_repository.go
│   │   │   ├── fee_repository.go
│   │   │   ├── transaction_repository.go
│   │   │   └── user_repository.go
│   │   ├── service/                      # Business logic layer
│   │   │   ├── auth_service.go
│   │   │   ├── child_service.go
│   │   │   ├── parent_service.go
│   │   │   ├── fee_service.go
│   │   │   ├── import_service.go
│   │   │   └── childcare_fee_calculator.go
│   │   └── csvparser/
│   │       └── bank_csv_parser.go        # CSV parsing for bank exports
│   ├── migrations/
│   │   ├── 000001_create_schema.up.sql
│   │   ├── 000001_create_schema.down.sql
│   │   └── ...
│   ├── tests/
│   │   ├── integration/
│   │   │   ├── api_test.go
│   │   │   └── testcontainers_setup.go
│   │   └── unit/
│   │       ├── matching_test.go
│   │       └── csv_parser_test.go
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── apps/
│   │   └── beitraege/                    # Fee Management Vue App
│   │       ├── src/
│   │       │   ├── App.vue
│   │       │   ├── main.ts
│   │       │   ├── pages/
│   │       │   │   ├── LoginPage.vue
│   │       │   │   ├── DashboardPage.vue
│   │       │   │   ├── ChildrenPage.vue
│   │       │   │   ├── ParentsPage.vue
│   │       │   │   ├── FeesPage.vue
│   │       │   │   └── ImportPage.vue
│   │       │   ├── components/
│   │       │   │   ├── ChildFormDialog.vue
│   │       │   │   ├── ParentFormDialog.vue
│   │       │   │   ├── FeeTable.vue
│   │       │   │   ├── ImportPreview.vue
│   │       │   │   ├── MainLayout.vue
│   │       │   │   └── ui/               # shadcn-vue components
│   │       │   ├── composables/
│   │       │   │   ├── useAuth.ts
│   │       │   │   ├── useChildren.ts
│   │       │   │   ├── useParents.ts
│   │       │   │   ├── useFees.ts
│   │       │   │   └── useImport.ts
│   │       │   ├── api/
│   │       │   │   ├── client.ts
│   │       │   │   └── schema.d.ts
│   │       │   ├── stores/
│   │       │   │   └── auth.ts
│   │       │   └── router/
│   │       │       └── index.ts
│   │       ├── index.html
│   │       ├── vite.config.ts
│   │       ├── tailwind.config.js
│   │       ├── tsconfig.json
│   │       └── package.json
│   │
│   ├── e2e/
│   │   └── tests/
│   │       └── beitraege/
│   │           ├── auth.spec.ts
│   │           ├── children.spec.ts
│   │           ├── fees.spec.ts
│   │           └── import.spec.ts
│   │
│   └── packages/shared/                  # Shared code (existing)
│
├── openapi/
│   └── fees-api.yaml                     # OpenAPI specification
│
└── docker/
    ├── docker-compose.yml                # Development setup
    ├── docker-compose.prod.yml           # Production setup
    ├── Dockerfile.backend-fees
    ├── Dockerfile.frontend-beitraege
    └── Caddyfile                         # Reverse proxy config
```

## Database Schema

The application uses the shared PostgreSQL instance with a dedicated schema `fees`.

### Tables

```sql
-- Schema
CREATE SCHEMA IF NOT EXISTS fees;

-- Children
CREATE TABLE fees.children (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_number VARCHAR(10) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    birth_date DATE NOT NULL,
    entry_date DATE NOT NULL,
    street VARCHAR(200),
    house_number VARCHAR(20),
    postal_code VARCHAR(10),
    city VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Parents
CREATE TABLE fees.parents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    birth_date DATE,
    email VARCHAR(255),
    phone VARCHAR(50),
    street VARCHAR(200),
    house_number VARCHAR(20),
    postal_code VARCHAR(10),
    city VARCHAR(100),
    annual_household_income DECIMAL(12,2),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Child-Parent relationship (n:m)
CREATE TABLE fees.child_parents (
    child_id UUID REFERENCES fees.children(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES fees.parents(id) ON DELETE CASCADE,
    is_primary BOOLEAN DEFAULT false,
    PRIMARY KEY (child_id, parent_id)
);

-- Fee expectations (what should be paid)
CREATE TABLE fees.fee_expectations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id UUID REFERENCES fees.children(id) ON DELETE CASCADE,
    fee_type VARCHAR(20) NOT NULL CHECK (fee_type IN ('MEMBERSHIP', 'FOOD', 'CHILDCARE')),
    year INT NOT NULL,
    month INT CHECK (month IS NULL OR (month >= 1 AND month <= 12)),
    amount DECIMAL(10,2) NOT NULL,
    due_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (child_id, fee_type, year, month)
);

-- Bank transactions (imported from CSV)
CREATE TABLE fees.bank_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_date DATE NOT NULL,
    value_date DATE NOT NULL,
    payer_name VARCHAR(255),
    payer_iban VARCHAR(34),
    description TEXT,
    amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'EUR',
    import_batch_id UUID,
    imported_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (booking_date, payer_iban, amount, description)
);

-- Payment matches (linking transactions to expectations)
CREATE TABLE fees.payment_matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID REFERENCES fees.bank_transactions(id) ON DELETE CASCADE,
    expectation_id UUID REFERENCES fees.fee_expectations(id) ON DELETE CASCADE,
    match_type VARCHAR(20) NOT NULL CHECK (match_type IN ('AUTO', 'MANUAL')),
    confidence DECIMAL(3,2),
    matched_at TIMESTAMPTZ DEFAULT NOW(),
    matched_by UUID REFERENCES fees.users(id),
    UNIQUE (transaction_id, expectation_id)
);

-- Users (authentication)
CREATE TABLE fees.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(20) DEFAULT 'USER' CHECK (role IN ('ADMIN', 'USER')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Refresh tokens
CREATE TABLE fees.refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES fees.users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_children_member_number ON fees.children(member_number);
CREATE INDEX idx_children_last_name ON fees.children(last_name);
CREATE INDEX idx_fee_expectations_child_id ON fees.fee_expectations(child_id);
CREATE INDEX idx_fee_expectations_year_month ON fees.fee_expectations(year, month);
CREATE INDEX idx_bank_transactions_booking_date ON fees.bank_transactions(booking_date);
CREATE INDEX idx_bank_transactions_import_batch ON fees.bank_transactions(import_batch_id);
CREATE INDEX idx_payment_matches_transaction ON fees.payment_matches(transaction_id);
CREATE INDEX idx_payment_matches_expectation ON fees.payment_matches(expectation_id);
```

## API Endpoints

Base URL: `/api/fees/v1`

### Authentication

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/login` | Login with email/password, returns JWT tokens |
| POST | `/auth/refresh` | Refresh access token |
| POST | `/auth/logout` | Invalidate refresh token |
| GET | `/auth/me` | Get current user info |

### Children

| Method | Path | Description |
|--------|------|-------------|
| GET | `/children` | List all children (supports `?active=true`) |
| POST | `/children` | Create a new child |
| GET | `/children/:id` | Get child details with parents |
| PUT | `/children/:id` | Update child data |
| DELETE | `/children/:id` | Deactivate child (soft delete) |
| POST | `/children/:id/parents` | Link parent to child |
| DELETE | `/children/:id/parents/:parentId` | Unlink parent from child |

### Parents

| Method | Path | Description |
|--------|------|-------------|
| GET | `/parents` | List all parents |
| POST | `/parents` | Create a new parent |
| GET | `/parents/:id` | Get parent details with children |
| PUT | `/parents/:id` | Update parent data |
| DELETE | `/parents/:id` | Delete parent |

### Fees

| Method | Path | Description |
|--------|------|-------------|
| GET | `/fees` | List fee expectations (supports filters) |
| GET | `/fees/overview` | Dashboard summary (open/paid per month) |
| POST | `/fees/generate` | Generate fee expectations for a period |
| GET | `/fees/:id` | Get single fee expectation with payment status |
| PUT | `/fees/:id` | Update fee amount |
| DELETE | `/fees/:id` | Delete fee expectation |

### Import

| Method | Path | Description |
|--------|------|-------------|
| POST | `/import/upload` | Upload CSV file, returns preview with matches |
| POST | `/import/confirm` | Confirm selected matches |
| GET | `/import/history` | List import batches |
| GET | `/import/transactions` | List unmatched transactions |
| POST | `/import/match` | Manually match transaction to fee |

### Childcare Fee Calculator

| Method | Path | Description |
|--------|------|-------------|
| GET | `/childcare-fee/calculate` | Calculate fee based on income |

Query parameters: `?income=55000` (annual household net income)

Returns: `{ "amount": 100.00, "bracket": "default" }`

## CSV Format (Bank Export)

The bank export CSV uses the following format:

- **Encoding**: ISO-8859-1 (Latin-1)
- **Delimiter**: Semicolon (`;`)
- **Date format**: `DD.MM.YYYY`
- **Decimal separator**: Comma (`,`)
- **Thousands separator**: None

### Columns

| Column | Description | Example |
|--------|-------------|---------|
| Bezeichnung Auftragskonto | Account name | BFS Komfort |
| IBAN Auftragskonto | Own IBAN | DE33370205000003321400 |
| BIC Auftragskonto | Own BIC | BFSWDE33XXX |
| Bankname Auftragskonto | Bank name | SozialBank AG |
| Buchungstag | Booking date | 20.01.2026 |
| Valutadatum | Value date | 20.01.2026 |
| Name Zahlungsbeteiligter | Payer name | Stefan Remer |
| IBAN Zahlungsbeteiligter | Payer IBAN | DE50500105175432192422 |
| BIC (SWIFT-Code) Zahlungsbeteiligter | Payer BIC | INGDDEFFXXX |
| Buchungstext | Transaction type | Dauerauftragsgutschr |
| Verwendungszweck | Description | Viktoria Remer 11072 Essensgeld |
| Betrag | Amount | 45,40 |
| Waehrung | Currency | EUR |
| Saldo nach Buchung | Balance | 55942,50 |
| Bemerkung | Notes | (usually empty) |
| Gekennzeichneter Umsatz | Flagged | (usually empty) |
| Glaeubiger ID | Creditor ID | (for direct debits) |
| Mandatsreferenz | Mandate reference | (for direct debits) |

## Matching Algorithm

### Step 1: Filter Transactions

Only consider incoming payments (positive amounts). Skip:
- Outgoing payments (negative amounts)
- Bank fees ("Abschluss")
- Known non-parent transactions (Krankenkassen, Lieferanten, Gehälter)

### Step 2: Extract Identifiers

From the `Verwendungszweck` field, extract:

1. **Member number**: Pattern `/\b(\d{5})\b/` or `/Mitgliedsnummer\s*:?\s*(\d{5})/i`
2. **Child name**: Fuzzy match against known children's first and last names

### Step 3: Determine Fee Type

Based on amount:
- `45.40 EUR` → FOOD (Essensgeld)
- `30.00 EUR` → MEMBERSHIP (Vereinsbeitrag)
- Other amounts → CHILDCARE (Platzgeld) or manual review

### Step 4: Calculate Confidence Score

| Scenario | Confidence |
|----------|------------|
| Member number found + exact amount match | 0.95 |
| Child name found + exact amount match | 0.85 |
| Member number found + different amount | 0.70 |
| Child name found + different amount | 0.60 |
| Only amount matches known fee type | 0.40 |

### Step 5: Generate Match Suggestions

For each transaction, suggest the most likely fee expectation(s) based on:
- Matched child (via member number or name)
- Fee type (via amount)
- Time period (booking date → year/month)

Matches with confidence >= 0.80 can be auto-confirmed.
Matches with confidence < 0.80 require manual review.

## Frontend Pages

### Login (Anmeldung)

- Email and password input
- JWT token stored in localStorage
- Redirect to Dashboard on success

### Dashboard (Übersicht)

- Summary cards: Total open fees, Paid this month, Pending matches
- Bar chart: Fees by month (open vs. paid)
- Recent activity list
- Quick actions: Import CSV, Generate fees

### Children (Kinder)

- Table with columns: Mitgliedsnummer, Name, Geburtsdatum, Alter, Status
- Filter: Active/All
- Actions: Add, Edit, View details, Deactivate
- Child detail view: Personal data, linked parents, fee history

### Parents (Eltern)

- Table with columns: Name, E-Mail, Telefon, Kinder
- Actions: Add, Edit, Delete
- Parent detail view: Personal data, income info, linked children

### Fees (Beiträge)

- Table with columns: Kind, Beitragsart, Jahr/Monat, Betrag, Status, Bezahlt am
- Filters: Year, Month, Fee type, Status (open/paid)
- Batch actions: Generate fees for month
- Status indicators: Green (paid), Yellow (pending), Red (overdue)

### Import (CSV Import)

1. **Upload step**: Drag & drop CSV file
2. **Preview step**: Table showing matched transactions
   - Columns: Datum, Einzahler, Verwendungszweck, Betrag, Erkanntes Kind, Beitragsart, Konfidenz
   - Checkboxes for selection
   - Manual correction option
3. **Confirm step**: Summary and confirm button
4. **Result step**: Success message with stats

## Testing Strategy

### Backend Unit Tests

- CSV parser edge cases (encoding, malformed data)
- Matching algorithm accuracy
- Fee calculation logic
- JWT token generation/validation

### Backend Integration Tests

Using Testcontainers with PostgreSQL:
- Full API endpoint tests
- Database migrations
- Transaction handling
- Concurrent import handling

### Frontend E2E Tests (Playwright)

- **Auth flow**: Login, logout, session expiry
- **Children CRUD**: Create, edit, view, deactivate
- **Parents CRUD**: Create, edit, link to children
- **Fee generation**: Generate for month, view results
- **CSV Import**: Upload, review matches, confirm

## Docker Configuration

### Development (docker-compose.yml)

```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: kita
      POSTGRES_USER: kita
      POSTGRES_PASSWORD: kita
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  backend-fees:
    build:
      context: ./backend-fees
      dockerfile: Dockerfile
    environment:
      DATABASE_URL: postgres://kita:kita@db:5432/kita?sslmode=disable&search_path=fees
      JWT_SECRET: ${JWT_SECRET:-dev-secret-change-in-production}
      PORT: 8081
    ports:
      - "8081:8081"
    depends_on:
      - db

  frontend-beitraege:
    build:
      context: ./frontend
      dockerfile: ../docker/Dockerfile.frontend-beitraege
    ports:
      - "5175:80"
    depends_on:
      - backend-fees
```

### Production (docker-compose.prod.yml)

Additional services and Caddy configuration for:
- `beitraege.knirpsenstadt.de` → frontend-beitraege
- `api.knirpsenstadt.de/fees/*` → backend-fees

## Environment Variables

### Backend

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | required |
| `JWT_SECRET` | Secret for JWT signing | required |
| `JWT_ACCESS_EXPIRY` | Access token expiry | 15m |
| `JWT_REFRESH_EXPIRY` | Refresh token expiry | 7d |
| `PORT` | HTTP server port | 8081 |
| `LOG_LEVEL` | Logging level | info |
| `CORS_ORIGINS` | Allowed CORS origins | * |

### Frontend

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_API_URL` | Backend API URL | http://localhost:8081/api/fees/v1 |

## Implementation Phases

### Phase 1: Backend Foundation
- [x] Create PLAN-BEITRAEGE.md
- [ ] Initialize Go module
- [ ] Setup project structure
- [ ] Configure database connection
- [ ] Create migrations
- [ ] Implement JWT authentication

### Phase 2: Core Entities
- [ ] User repository and service
- [ ] Child CRUD (handler, service, repository)
- [ ] Parent CRUD (handler, service, repository)
- [ ] Child-Parent linking

### Phase 3: Fee Management
- [ ] Fee expectation entity
- [ ] Fee generation service
- [ ] Fee overview endpoints

### Phase 4: CSV Import
- [ ] CSV parser
- [ ] Matching algorithm
- [ ] Import preview endpoint
- [ ] Match confirmation endpoint

### Phase 5: Backend Testing
- [ ] Unit tests for matching
- [ ] Unit tests for CSV parser
- [ ] Integration tests with Testcontainers

### Phase 6: Frontend Setup
- [ ] Initialize Vue app
- [ ] Configure Tailwind + shadcn-vue
- [ ] Setup router and auth store
- [ ] Create API client

### Phase 7: Frontend Pages
- [ ] Login page
- [ ] Dashboard page
- [ ] Children page
- [ ] Parents page
- [ ] Fees page
- [ ] Import page

### Phase 8: Frontend Testing
- [ ] Playwright configuration
- [ ] Auth tests
- [ ] CRUD tests
- [ ] Import flow tests

### Phase 9: Deployment
- [ ] Backend Dockerfile
- [ ] Frontend Dockerfile
- [ ] Update docker-compose files
- [ ] Update Caddyfile

## Notes

### Language Guidelines
- **Code**: All code, comments, variable names, and API responses in English
- **Frontend UI**: All user-facing text in German
- **Documentation**: English

### Naming Conventions
- Use `fee` instead of `contribution` in all code
- API paths: kebab-case (`/fee-expectations`)
- Go packages: lowercase (`feeservice`)
- TypeScript: camelCase for variables, PascalCase for types

### Security Considerations
- Passwords hashed with bcrypt (cost 12)
- JWT tokens with short expiry (15 min access, 7 days refresh)
- CORS restricted to known origins in production
- SQL injection prevention via parameterized queries
- Input validation on all endpoints
