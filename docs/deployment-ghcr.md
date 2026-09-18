# GHCR images and release identity

This document describes the container images produced by the current GitHub Actions pipeline. The supported deployment of `kita.remer.cc` is the Homelab workflow in `docs/deployment-homelab.md`.

## Current images

| Image | Contents | Runtime port |
| --- | --- | --- |
| `ghcr.io/steffbo/kita-backend-management` | Management API plus embedded Dienstplan and Zeiterfassung frontends | 8080 |
| `ghcr.io/steffbo/kita-backend-fees` | Fees API plus embedded Beiträge frontend | 8081 |
| `ghcr.io/steffbo/kita-banking-sync` | Banking-sync runner used by the scheduled Homelab service | 3333 |

There are no separately published frontend images. The portal backend and frontend are not part of the image pipeline or production deployment.

## Build pipeline

`.github/workflows/build-images.yml` runs on every push to `main` and through manual dispatch. Its matrix builds all three images and publishes:

- `latest` on the default branch;
- a Git-SHA tag such as `sha-abc1234`;
- OCI metadata labels, including the full commit in `org.opencontainers.image.revision`.

A release must be tied to the workflow run whose `headSha` equals the intended commit. Do not rely on an arbitrary delay or assume that the newest run belongs to the change being released.

```bash
expected_sha=$(git rev-parse HEAD)
run_id=$(gh run list -R steffbo/kita-apps --branch main --limit 20 \
  --json databaseId,headSha \
  --jq ".[] | select(.headSha == \"$expected_sha\") | .databaseId" | head -n1)
test -n "$run_id"
gh run watch "$run_id" -R steffbo/kita-apps --exit-status
```

## Consuming images

Prefer an immutable SHA tag when deploying outside the managed Homelab workflow:

```bash
docker pull ghcr.io/steffbo/kita-backend-fees:sha-abc1234
```

If a deployment intentionally uses `latest`, wait for the commit-pinned workflow to finish successfully before pulling it. After startup, compare the container's revision label with the expected full SHA:

```bash
expected_sha=$(git rev-parse HEAD)
actual_sha=$(docker inspect <container> \
  --format '{{index .Config.Labels "org.opencontainers.image.revision"}}')
test "$actual_sha" = "$expected_sha"
```

A successful pull, a running container, or a passing healthcheck does not by itself prove that the intended release is active.

## Legacy standalone Compose file

`docker/docker-compose.ghcr.yml` is a legacy standalone configuration. It still models the former split-frontend architecture and references frontend images that the current workflow no longer builds. It is retained for historical reference but is not wired to the current image set and must not be used for a new deployment without migration and testing.

The former domain, backup, and Caddy examples belonged to that legacy stack and are not the source of truth for the Homelab. Current routing, encrypted environment handling, persistent storage, migration startup, and backup behavior live in the Homelab repository.

## Security and recovery

- Never print registry credentials, decrypted environment files, or image pull tokens.
- Diagnose a failed migration from container logs and schema state before making any corrective write.
- Do not blindly run down migrations or force schema versions; confirm backup and recovery options first.
- Treat an interrupted deployment as having unknown state until containers, revisions, and health endpoints have been inspected.
