# Deployment to kita.remer.cc

This document describes the verified deployment workflow from the Ubuntu development workstation to `https://kita.remer.cc` on `vm-infra-dev`.

## Architecture

- **Backend**: `backend-fees` (Go) with the embedded Beiträge frontend
- **Database**: PostgreSQL (`kita` database, `fees` schema)
- **Routing**: edge Caddy in the homelab repo (`stacks/edge/caddy/Caddyfile`) sends `/beitraege/*`, `/api/fees/*` and `/health` to `backend-fees`; every other path redirects to `/beitraege/`
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

- `ghcr.io/steffbo/kita-backend-fees`
- `ghcr.io/steffbo/kita-banking-sync`

The Beiträge frontend is embedded in the `backend-fees` image; there are no separate frontend images in the current pipeline.

### 3. Deploy to Server

Deployment is handled via ansible (pulls latest image from GHCR and restarts the container):

```bash
cd /home/stefan/workspace/homelab/ansible
ansible-playbook playbooks/deploy-app.yml -e "app=kita" \
  -e "ansible_ssh_private_key_file=/home/stefan/.ssh/homelab_from_ubuntu"
```

The inventory still references a legacy key that is absent on this workstation. Use the one-run override above; do not create a replacement at the legacy path. The playbook deploys the database, `backend-fees`, and both banking-sync containers.

The playbook syncs `stacks/infra-dev/apps/kita/compose.yml` from the local homelab checkout, so compose changes take effect without pushing the homelab repo. Non-secret settings like `APP_BASE_URL=https://kita.remer.cc/beitraege` (base for invitation links) live there. Secrets stay in the encrypted `.env`.

If the command is interrupted or its result is otherwise unclear, treat the deployment state as unknown. Inspect the containers and deployed revisions before deciding whether a retry is needed.

### 4. Database migrations

`backend-fees` runs its migrations automatically on container start (`/app/migrate -direction up && /app/server`).

Do not run a manual migration as a routine release step. When investigating migration state, inspect the service logs and the applicable schema migration table first.

### 5. Verify Deployment

Verify the expected revision, container health, service endpoints, and the affected public page. `docker ps` alone is not release verification.

```bash
# From the kita-apps checkout
expected_sha=$(git rev-parse HEAD)

ssh vm-infra-dev 'sudo docker ps --filter name=kita --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"'

for container in kita-backend-fees kita-banking-sync-runner; do
  actual_sha=$(ssh vm-infra-dev \
    "sudo docker inspect $container --format '{{index .Config.Labels \"org.opencontainers.image.revision\"}}'")
  test "$actual_sha" = "$expected_sha" || {
    echo "$container runs $actual_sha, expected $expected_sha"
    exit 1
  }
done

curl -fsS https://kita.remer.cc/health   # expects {"status":"ok"}, not an empty body
curl -fsS -o /dev/null https://kita.remer.cc/beitraege/
```

Also inspect recent logs when migrations ran or the change is operationally sensitive:

```bash
ssh vm-infra-dev 'sudo docker logs kita-backend-fees --tail 50'
```

There is intentionally no quick deploy that bypasses the commit-pinned CI gate or the Ansible playbook.

## URLs

| Service | URL |
|---------|-----|
| Beitraege (Fees) | https://kita.remer.cc/beitraege |
| Fees health | https://kita.remer.cc/health |
| Fees API | https://kita.remer.cc/api/fees/v1/ |

## Backups

The homelab B2 job for `vm-infra-dev` (configured in the homelab repo: `ansible/playbooks/install-b2-restic-backup.yml`, described in `docs/backup.md`) runs daily at 06:30 and 18:30 Europe/Berlin. It writes a custom-format `pg_dump` of the whole `kita` database to `/srv/homelab-data/kita/restic-db/kita.dump` (validated with `pg_restore --list`, root-only), then uploads it Restic-encrypted to the B2 repository `homelab-restic-vm-infra-dev` (EU Central). Retention: 14 daily, 8 weekly, 12 monthly snapshots.

The physical Postgres data directory (`/srv/homelab-data/kita/postgres`) is **excluded**. Restoring the VM files alone does not bring the database back; recovery always goes through `kita.dump` and `pg_restore`. Data written after the last dump (up to 12 h) is lost and must be re-entered.

Access needs root on `vm-infra-dev` (`/srv/homelab/backup-b2-vm-infra-dev.env`) or the homelab AGE key (`ansible/secrets/backblaze-b2.env.enc`). Both are currently held only by the operator.

### Restore drill (verified 2026-10-11)

Restores the latest snapshot from B2 into an isolated throwaway container; the live database is only read. Snapshot `00a700f4` (06:30) restored in about a second; all 28 tables matched the live row counts except one bank import made after the snapshot, and `fees.schema_migrations` (version 44), routines and indexes were identical.

```bash
ssh vm-infra-dev
sudo -i
set -a; source /srv/homelab/backup-b2-vm-infra-dev.env; set +a
export RESTIC_REPOSITORY=s3:https://s3.eu-central-003.backblazeb2.com/homelab-restic-vm-infra-dev
restic --no-lock snapshots
mkdir -m 700 /var/tmp/restore-kita
restic --no-lock restore latest --target /var/tmp/restore-kita --include /srv/homelab-data/kita/restic-db
DUMP=/var/tmp/restore-kita/srv/homelab-data/kita/restic-db/kita.dump

docker run -d --name kita-restore-drill --network none \
  -e POSTGRES_USER=kita -e POSTGRES_DB=kita -e POSTGRES_HOST_AUTH_METHOD=trust postgres:16-alpine
docker exec kita-restore-drill pg_isready -U kita -d kita   # repeat until ready
docker exec -i kita-restore-drill pg_restore -U kita -d kita --exit-on-error < "$DUMP"
docker exec kita-restore-drill psql -U kita -d kita -c 'SELECT version, dirty FROM fees.schema_migrations;'
# compare row counts with kita-db (read-only), then clean up:
docker rm -f -v kita-restore-drill   # -v removes the anonymous data volume with family data
rm -rf /var/tmp/restore-kita
```

Use a snapshot ID from `restic snapshots` instead of `latest` for an older state. Without `-v`, `docker rm` leaves an anonymous volume holding the restored data; check `docker volume ls -f dangling=true`.

### Restoring production (not yet exercised)

Only with explicit authorization, after the drill above has succeeded with the chosen snapshot. The current database is renamed, not dropped, so the step is reversible.

```bash
docker stop kita-backend-fees kita-banking-sync-scheduler
docker exec kita-db psql -U kita -d postgres -c 'ALTER DATABASE kita RENAME TO kita_before_restore;'
docker exec kita-db createdb -U kita kita
docker exec -i kita-db pg_restore -U kita -d kita --exit-on-error < "$DUMP"
docker start kita-backend-fees kita-banking-sync-scheduler
```

Check that the running image's latest migration is not older than `fees.schema_migrations` in the dump, then verify `/health` and the Beiträge UI. Drop `kita_before_restore` only once the restored state is confirmed.

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
ssh vm-infra-dev \
  'sudo docker exec kita-db psql -U kita -d kita -c "SELECT * FROM fees.schema_migrations;"'
```

Do not blindly run `down`, force a migration version, or retry a partially applied migration. Inspect the migration, confirm backup/recovery options, and obtain explicit authorization before any corrective write.

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
