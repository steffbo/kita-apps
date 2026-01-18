# Kita-Apps Knirpsenstadt

Zeiterfassung und Dienstplanung für die Kita Knirpsenstadt.

## Projekt-Übersicht

| App | URL | Beschreibung |
|-----|-----|--------------|
| Dienstplan | plan.knirpsenstadt.de | Wochenplanung, Gruppenübersicht |
| Zeiterfassung | zeit.knirpsenstadt.de | Ein-/Ausstempeln, Zeitübersicht |

## Tech-Stack

- **Backend**: Spring Boot 3.x, PostgreSQL, JWT Auth
- **Frontend**: Vue 3, TypeScript, Tailwind CSS, shadcn-vue
- **Build**: Bun, Vite
- **Deployment**: Docker, Caddy

## Schnellstart (Entwicklung)

### Voraussetzungen

- Java 21+
- Bun 1.x
- Docker & Docker Compose
- Maven 3.9+

### 1. Datenbank starten

```bash
cd docker
docker compose up db -d
```

### 2. Backend starten

```bash
cd backend
mvn spring-boot:run
```

Das Backend läuft auf http://localhost:8080

### 3. Frontend starten

```bash
cd frontend
bun install
bun run dev:plan  # Dienstplan auf :5173
bun run dev:zeit  # Zeiterfassung auf :5174
```

### 4. API-Typen generieren

```bash
./scripts/generate-api.sh
```

## Projektstruktur

```
kita-apps/
├── openapi/
│   └── kita-api.yaml          # API-Spezifikation (Single Source of Truth)
│
├── backend/
│   ├── pom.xml
│   └── src/main/java/de/knirpsenstadt/
│       ├── KitaApplication.java
│       ├── config/            # Security, CORS
│       ├── controller/        # REST Controller (implementieren generierte Interfaces)
│       ├── service/           # Business Logic
│       ├── repository/        # JPA Repositories
│       └── model/             # Entities
│
├── frontend/
│   ├── apps/
│   │   ├── dienstplan/        # Vue App für plan.knirpsenstadt.de
│   │   └── zeiterfassung/     # Vue App für zeit.knirpsenstadt.de
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

### API-First Workflow

1. Änderungen in `openapi/kita-api.yaml` vornehmen
2. `./scripts/generate-api.sh` ausführen
3. Backend: Maven generiert Interfaces automatisch bei `mvn compile`
4. Frontend: TypeScript-Typen werden in `packages/shared/src/api/schema.d.ts` generiert

### Default Admin Login

- **E-Mail**: admin@knirpsenstadt.de
- **Passwort**: admin123

> ⚠️ Das Passwort nach dem ersten Login ändern!

### Nützliche URLs (Development)

| URL | Beschreibung |
|-----|--------------|
| http://localhost:8080/api/swagger-ui.html | API Dokumentation |
| http://localhost:8080/api/actuator/health | Health Check |
| http://localhost:8025 | MailHog (E-Mail Tester) |
| http://localhost:5173 | Dienstplan Frontend |
| http://localhost:5174 | Zeiterfassung Frontend |

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

Caddy kümmert sich automatisch um SSL-Zertifikate via Let's Encrypt.

## Dokumentation

- [PLAN.md](PLAN.md) - Detaillierter Projektplan
- [openapi/kita-api.yaml](openapi/kita-api.yaml) - API-Spezifikation

## License

Privat - Kita Knirpsenstadt
