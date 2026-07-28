# Deployment to kita.remer.cc

This document describes the deployment workflow for the Kita apps to the production environment at `https://kita.remer.cc`.

## Architecture

- **Frontend**: Vue.js apps embedded in Go backends
- **Backend**: Go services with embedded frontends
- **Database**: PostgreSQL (shared `kita` database, `fees` schema for backend-fees)
- **Container Registry**: GitHub Container Registry (ghcr.io)
- **Hosting**: Docker Compose on infra-dev VM

## Deployment Steps

### 1. Commit and Push Changes

```bash
cd /Users/stefan.remer/workspace/kita-apps
git add -A
git commit -m "your commit message"
git push
```

Note: the Docker image build workflow only runs on `main` (or via manual dispatch). If you work on a branch, merge to `main` before expecting images to be built.

### 2. Monitor GitHub Actions Build

Watch the build progress using the GitHub CLI:

```bash
# List recent workflow runs
gh run list -R steffbo/kita-apps --branch main --limit 5

# Watch a specific run (get run ID from list above)
gh run watch <run-id> -R steffbo/kita-apps

# Or watch the latest run
gh run watch $(gh run list -R steffbo/kita-apps --branch main --limit 1 --json databaseId --jq '.[0].databaseId')
```

The workflow builds Docker images and pushes them to `ghcr.io/steffbo/kita-backend-fees:latest` and `ghcr.io/steffbo/kita-backend-management:latest`.

### 3. Deploy to Server

Deployment is handled via ansible (pulls latest image from GHCR and restarts the container):

```bash
cd ~/workspace/homelab/ansible
ansible-playbook playbooks/deploy-app.yml -e "app=kita"
```

### 4. Run Database Migrations (if needed)

`backend-fees` runs migrations automatically on container start (compose command is `/app/migrate -direction up && /app/server`).

If you added new migrations for `backend-management`, run them manually:

```bash
# SSH to server (if not already connected)
ssh -i ~/.ssh/PVE_id_ed25519 stefan@192.168.188.207

# Run migrations for backend-management (manual)
sudo docker exec kita-backend-management ./migrate -direction up

# Check current migration version
sudo docker exec kita-db psql -U kita -d kita -c "SELECT * FROM fees.schema_migrations;"
```

### 5. Verify Deployment

```bash
# Check container status
sudo docker ps | grep kita

# Check logs
sudo docker logs kita-backend-fees --tail 50
sudo docker logs kita-backend-management --tail 50

# Health check
curl -s https://kita.remer.cc/api-fees/health
curl -s https://kita.remer.cc/api/health
```

## Quick One-Liner Deploy

For a quick deploy after pushing (wait ~90 seconds for build):

```bash
# From local machine - deploy backend-fees
ssh -i ~/.ssh/PVE_id_ed25519 stefan@192.168.188.207 \
  "cd /srv/homelab/stacks/infra-dev && sudo docker compose pull backend-fees && sudo docker compose up -d --force-recreate backend-fees"
```

## URLs

| Service | URL |
|---------|-----|
| Beitraege (Fees) | https://kita.remer.cc/beitraege |
| Plan (Schedule) | https://kita.remer.cc/plan |
| Zeit (Time Tracking) | https://kita.remer.cc/zeit |
| Fees API | https://kita.remer.cc/api-fees/v1/ |
| Management API | https://kita.remer.cc/api/v1/ |

## Troubleshooting

### Build failed
```bash
# Check workflow logs
gh run view <run-id> --log-failed
```

### Container won't start
```bash
# Check logs
sudo docker logs kita-backend-fees

# Check if port is in use
sudo docker ps -a | grep 8081
```

### Migration failed
```bash
# Check migration status (look for dirty=true)
sudo docker exec kita-db psql -U kita -d kita -c "SELECT * FROM fees.schema_migrations;"

# If dirty, fix the issue and force version
sudo docker exec kita-backend-fees ./migrate -direction down -steps 1
sudo docker exec kita-backend-fees ./migrate -direction up
```

### Database connection issues
```bash
# Check database container
sudo docker logs kita-db

# Test connection
sudo docker exec kita-db psql -U kita -d kita -c "SELECT 1;"
```
