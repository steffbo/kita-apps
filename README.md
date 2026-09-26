# Kita-Apps Knirpsenstadt

Beitragsverwaltung für die Kita Knirpsenstadt, läuft unter https://kita.remer.cc/beitraege. Enthält außerdem die Elternstunden (Erfassung und Übersicht je Familie); ein Eltern-Zugang ist geplant.

## Bestandteile

| Teil | Beschreibung |
|------|--------------|
| `backend-fees/` | Go-API (Port 8081): Kinder, Eltern, Haushalte, Beiträge, Bankabgleich, Erinnerungen, Benutzer |
| `frontend/apps/beitraege/` | Vue-App (Dev-Port 5175), in Produktion ins Backend eingebettet |
| `banking-sync/` | Bun-Dienst: lädt Bank-CSVs herunter und importiert sie ins Backend |

## Tech-Stack

- **Backend**: Go, Chi Router, PostgreSQL, JWT
- **Frontend**: Vue 3, TypeScript, Tailwind CSS
- **Build**: Bun, Vite
- **Deployment**: GHCR-Images, Docker Compose im Homelab

## Schnellstart (Entwicklung)

Voraussetzungen: Go, Bun, Docker.

```bash
# Datenbank
cd docker && docker compose up db -d

# Backend (http://localhost:8081)
cd backend-fees
go run cmd/migrate/main.go
USER_NAME=admin@example.org USER_PASSWORD='change-me' go run cmd/server/main.go

# Frontend (http://localhost:5175)
cd frontend
bun install
bun run dev:beitraege
```

`USER_NAME` / `USER_PASSWORD` legen beim Start einen Admin an, falls es noch keinen Account mit dieser E-Mail gibt.

## OpenAPI / Typen

`backend-fees` erzeugt die Spec mit [swag](https://github.com/swaggo/swag) aus Go-Annotationen. Eingecheckt ist nur `openapi/fees/openapi3.yaml`; die Beiträge-App generiert daraus `src/api/schema.d.ts`.

```bash
scripts/generate-api.sh
```

Details zur Annotation-Syntax: `backend-fees/README.md`.

## E2E-Tests

Playwright-Suite für die Beiträge-App, startet ihren eigenen Stack (Wegwerf-PostgreSQL, Backend mit eingebettetem Frontend) und läuft in CI:

```bash
cd frontend
bunx playwright install chromium
bun run test:e2e
```

Details: [`docs/e2e-beitraege.md`](docs/e2e-beitraege.md).

## Dokumentation

- [AGENTS.md](AGENTS.md) – kompakter Einstieg: Layout, Ports, Kommandos, Dokumentenkarte
- [docs/status-fees.md](docs/status-fees.md) – Entscheidungen und Änderungshistorie
- [docs/deployment-homelab.md](docs/deployment-homelab.md), [docs/deployment-ghcr.md](docs/deployment-ghcr.md) – Deployment
- [backend-fees/README.md](backend-fees/README.md) – Backend-Dokumentation

Der frühere Dienstplan/Zeiterfassung-Teil (`backend-management`) und das Portal-Grundgerüst (`backend-portal`) wurden im September 2026 entfernt. Der letzte Stand liegt im Git-Tag `archive/pre-cleanup-2026-09`.

## License

Privat - Kita Knirpsenstadt
