# Deployment Guide

This guide covers deploying Kita-Apps to a production server using pre-built Docker images from GitHub Container Registry (GHCR).

## Prerequisites

- A server with Docker and Docker Compose installed
- A domain name with DNS configured (e.g., `knirpsenstadt.de`)
- Ports 80 and 443 open for HTTP/HTTPS traffic

## Architecture Overview

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                      Caddy                              │
                    │              (Reverse Proxy + Auto HTTPS)               │
                    └─────────────────────────────────────────────────────────┘
                                              │
        ┌─────────────────┬─────────────────┬─┴───────────────┬───────────────┐
        │                 │                 │                 │               │
        ▼                 ▼                 ▼                 ▼               ▼
┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────┐
│  frontend-    │ │  frontend-    │ │  frontend-    │ │   backend-    │ │  backend- │
│    plan       │ │    zeit       │ │  beitraege    │ │  management   │ │   fees    │
│   :80         │ │   :80         │ │   :80         │ │   :8080       │ │  :8081    │
└───────────────┘ └───────────────┘ └───────────────┘ └───────────────┘ └───────────┘
                                                              │               │
                                                              └───────┬───────┘
                                                                      ▼
                                                              ┌───────────────┐
                                                              │   PostgreSQL  │
                                                              │    :5432      │
                                                              └───────────────┘
```

## Domain Configuration

Configure DNS A records pointing to your server:

| Subdomain | Purpose |
|-----------|---------|
| `plan.knirpsenstadt.de` | Dienstplan Frontend |
| `zeit.knirpsenstadt.de` | Zeiterfassung Frontend |
| `beitraege.knirpsenstadt.de` | Beitraege Frontend |
| `api.knirpsenstadt.de` | Backend Management API |
| `api-fees.knirpsenstadt.de` | Backend Fees API |

## Setup on Server

```bash
# Clone the repository (or copy the docker/ directory)
git clone https://github.com/steffbo/kita-apps.git
cd kita-apps/docker

# Copy and configure environment variables
cp .env.example .env
nano .env  # or use your preferred editor
```

### Required Environment Variables

Edit `.env` and set these values:

```bash
# Database (use strong passwords!)
DB_NAME=kita
DB_USER=kita
DB_PASSWORD=your_secure_database_password

# JWT Secret (generate with: openssl rand -base64 64)
JWT_SECRET=your_generated_jwt_secret

# Mail settings
MAIL_HOST=smtp.your-provider.com
MAIL_PORT=587
MAIL_USER=noreply@knirpsenstadt.de
MAIL_PASSWORD=your_mail_password

# Domains (customize if needed)
DOMAIN_BASE=knirpsenstadt.de
DOMAIN_PLAN=plan.knirpsenstadt.de
DOMAIN_ZEIT=zeit.knirpsenstadt.de
DOMAIN_BEITRAEGE=beitraege.knirpsenstadt.de
DOMAIN_API=api.knirpsenstadt.de
DOMAIN_API_FEES=api-fees.knirpsenstadt.de

# GitHub user (for image paths)
GITHUB_USER=steffbo
```

## Start the Stack

```bash
# Pull all images (no authentication needed - public repo!)
docker compose -f docker-compose.ghcr.yml pull

# Start all services
docker compose -f docker-compose.ghcr.yml up -d

# Check status
docker compose -f docker-compose.ghcr.yml ps

# View logs
docker compose -f docker-compose.ghcr.yml logs -f
```

## Updating to New Versions

When new images are pushed to GHCR:

```bash
cd /path/to/kita-apps/docker

# Pull latest images
docker compose -f docker-compose.ghcr.yml pull

# Restart with new images
docker compose -f docker-compose.ghcr.yml up -d

# Clean up old images (optional)
docker image prune -f
```

## Database Migrations

Migrations run automatically when the backend containers start. To run migrations manually:

```bash
# backend-management migrations
docker compose -f docker-compose.ghcr.yml exec backend-management ./server migrate up

# backend-fees migrations
docker compose -f docker-compose.ghcr.yml exec backend-fees ./server migrate up
```

## Backup & Restore

### Automatic Backups

A backup service runs daily at 3 AM, storing backups in `./backup/`:

```bash
# List backups
ls -la backup/

# Manual backup
docker compose -f docker-compose.ghcr.yml exec backup /pg-backup.sh
```

### Manual Backup

```bash
# Create backup
docker compose -f docker-compose.ghcr.yml exec db pg_dump -U $DB_USER $DB_NAME > backup_$(date +%Y%m%d).sql

# Restore backup
docker compose -f docker-compose.ghcr.yml exec -T db psql -U $DB_USER $DB_NAME < backup_20240101.sql
```

## Monitoring

### Health Checks

```bash
# Check backend-management health
curl https://api.knirpsenstadt.de/healthz

# Check backend-fees health  
curl https://api-fees.knirpsenstadt.de/health
```

### Container Status

```bash
# View all containers
docker compose -f docker-compose.ghcr.yml ps

# View resource usage
docker stats

# View logs for specific service
docker compose -f docker-compose.ghcr.yml logs -f backend-management
```

## Troubleshooting

### Caddy Certificate Issues

If HTTPS isn't working:

```bash
# Check Caddy logs
docker compose -f docker-compose.ghcr.yml logs caddy

# Verify DNS is configured correctly
dig +short plan.knirpsenstadt.de
```

### Database Connection Issues

```bash
# Check if database is healthy
docker compose -f docker-compose.ghcr.yml exec db pg_isready -U $DB_USER

# View database logs
docker compose -f docker-compose.ghcr.yml logs db
```

## Security Recommendations

1. **Firewall**: Only allow ports 80, 443, and SSH
2. **SSH**: Use key-based authentication, disable password login
3. **Updates**: Keep the host OS and Docker updated
4. **Secrets**: Never commit `.env` files to version control
5. **Backups**: Regularly test backup restoration

## CI/CD Pipeline

Images are automatically built and pushed to GHCR on every push to `main`:

- `ghcr.io/steffbo/kita-backend-management:latest`
- `ghcr.io/steffbo/kita-backend-fees:latest`
- `ghcr.io/steffbo/kita-frontend-plan:latest`
- `ghcr.io/steffbo/kita-frontend-zeit:latest`
- `ghcr.io/steffbo/kita-frontend-beitraege:latest`

Each image is also tagged with the Git SHA (e.g., `sha-abc1234`) for rollback capability.

Since the repository is public, **no authentication is required** to pull images.
