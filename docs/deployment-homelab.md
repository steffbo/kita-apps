# Deployment to kita.remer.cc

This document describes the verified deployment workflow from the Ubuntu development workstation to `https://kita.remer.cc` on `vm-infra-dev`.

## Architecture

- **Frontend**: Vue.js apps embedded in Go backends
- **Backend**: Go services with embedded frontends
- **Database**: PostgreSQL (shared `kita` database, `fees` schema for backend-fees)
- **Container Registry**: GitHub Container Registry (ghcr.io)
- **Hosting**: Docker Compose on infra-dev VM

## Deployment Steps

### 1. Verify, commit, and push

```bash
cd /home/stefan/workspace/kita-apps
# Run the tests/builds relevant to the change before committing.
git status --short
git add path/to/changed-file path/to/another-file
git commit -m "your commit message"
git push origin main
```

Note: the Docker image build workflow only runs on `main` (or via manual dispatch). If you work on a branch, merge to `main` before expecting images to be built.

### 2. Wait for the build of the exact commit

Do not sleep for a fixed duration and do not assume the newest workflow run belongs to this release. Resolve the run by the pushed commit SHA and require a successful conclusion:

```bash
expected_sha=$(git rev-parse HEAD)
run_id=$(gh run list -R steffbo/kita-apps --branch main --limit 20 \
  --json databaseId,headSha \
  --jq ".[] | select(.headSha == \"$expected_sha\") | .databaseId" | head -n1)
test -n "$run_id"
gh run watch "$run_id" -R steffbo/kita-apps --exit-status
```

The workflow builds and publishes these images:

- `ghcr.io/steffbo/kita-backend-management`
- `ghcr.io/steffbo/kita-backend-fees`
- `ghcr.io/steffbo/kita-banking-sync`

The Vue frontends are embedded in the Go backend images; there are no separate frontend images in the current pipeline.

### 3. Deploy to Server

Deployment is handled via ansible (pulls latest image from GHCR and restarts the container):

```bash
cd /home/stefan/workspace/homelab/ansible
ansible-playbook playbooks/deploy-app.yml -e "app=kita" \
  -e "ansible_ssh_private_key_file=/home/stefan/.ssh/homelab_from_ubuntu"
```

The inventory still references a legacy key that is absent on this workstation. Use the one-run override above; do not create a replacement at the legacy path. The playbook deploys the database, both backends, and both banking-sync containers.

If the command is interrupted or its result is otherwise unclear, treat the deployment state as unknown. Inspect the containers and deployed revisions before deciding whether a retry is needed.

### 4. Database migrations

Both backends run migrations automatically on container start:

- `backend-management`: `/app/migrate -direction up && /app/server`
- `backend-fees`: `/app/migrate -direction up && /app/server`

Do not run a manual migration as a routine release step. When investigating migration state, inspect the service logs and the applicable schema migration table first.

### 5. Verify Deployment

Verify the expected revision, container health, service endpoints, and the affected public page. `docker ps` alone is not release verification.

```bash
# From the kita-apps checkout
expected_sha=$(git rev-parse HEAD)

ssh vm-infra-dev 'sudo docker ps --filter name=kita --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"'

for container in kita-backend-management kita-backend-fees kita-banking-sync-runner; do
  actual_sha=$(ssh vm-infra-dev \
    "sudo docker inspect $container --format '{{index .Config.Labels \"org.opencontainers.image.revision\"}}'")
  test "$actual_sha" = "$expected_sha" || {
    echo "$container runs $actual_sha, expected $expected_sha"
    exit 1
  }
done

curl -fsS https://kita.remer.cc/health
curl -fsS https://kita.remer.cc/healthz
curl -fsS -o /dev/null https://kita.remer.cc/beitraege/
```

Also inspect recent logs when migrations ran or the change is operationally sensitive:

```bash
ssh vm-infra-dev 'sudo docker logs kita-backend-fees --tail 50'
ssh vm-infra-dev 'sudo docker logs kita-backend-management --tail 50'
```

There is intentionally no quick deploy that bypasses the commit-pinned CI gate or the Ansible playbook.

## URLs

| Service | URL |
|---------|-----|
| Beitraege (Fees) | https://kita.remer.cc/beitraege |
| Plan (Schedule) | https://kita.remer.cc/plan |
| Zeit (Time Tracking) | https://kita.remer.cc/zeit |
| Fees health | https://kita.remer.cc/health |
| Management health | https://kita.remer.cc/healthz |
| Fees API (public prefix) | https://kita.remer.cc/api-fees/v1/ |

## Backups

There is no app-level DB dump. The whole `infra-dev` VM (including the `kita-db` volume) is backed up twice daily to Backblaze by the homelab VM backup. Restoring means restoring the VM (or its disk) from that backup, not a `pg_restore`.

## Troubleshooting

### Build failed
```bash
# Check workflow logs
gh run view <run-id> --log-failed
```

### Container won't start
```bash
# Check logs
ssh vm-infra-dev 'sudo docker logs kita-backend-fees --tail 100'

# Check if port is in use
ssh vm-infra-dev 'sudo docker ps -a --filter publish=8081'
```

### Migration failed
```bash
# Inspect without changing state
ssh vm-infra-dev 'sudo docker logs kita-backend-fees --tail 100'
ssh vm-infra-dev 'sudo docker logs kita-backend-management --tail 100'
ssh vm-infra-dev \
  'sudo docker exec kita-db psql -U kita -d kita -c "SELECT * FROM fees.schema_migrations;"'
```

Do not blindly run `down`, force a migration version, or retry a partially applied migration. Determine which backend and schema are affected, inspect the migration, confirm backup/recovery options, and obtain explicit authorization before any corrective write.

### Database connection issues
```bash
# Check database container
ssh vm-infra-dev 'sudo docker logs kita-db --tail 100'

# Test connection
ssh vm-infra-dev 'sudo docker exec kita-db psql -U kita -d kita -c "SELECT 1;"'
```

## Live-data corrections after a release

Keep application deployment and business-data correction as separate, verifiable phases:

1. Inspect current state with read-only API and database queries.
2. Confirm the deployed revision and health before relying on new behavior.
3. Perform the authorized mutation through the application API, not direct SQL.
4. Verify the API response and independently verify database postconditions with `SELECT` queries.
5. If a request is interrupted, inspect the target before retrying; the outcome is unknown until verified.

Never print credentials or decrypted `.env` contents. Follow `.agents/skills/kita-live-data-access/` for authentication and write safeguards.
