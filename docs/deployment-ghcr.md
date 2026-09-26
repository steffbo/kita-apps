# GHCR images and release identity

This document describes the container images produced by the current GitHub Actions pipeline. The supported deployment of `kita.remer.cc` is the Homelab workflow in `docs/deployment-homelab.md`.

## Current images

| Image | Contents | Runtime port |
| --- | --- | --- |
| `ghcr.io/steffbo/kita-backend-fees` | Fees API plus embedded Beiträge frontend | 8081 |
| `ghcr.io/steffbo/kita-banking-sync` | Banking-sync runner used by the scheduled Homelab service | 3333 |

There are no separately published frontend images. `kita-backend-management` is no longer built (service removed 2026-09); its old tags remain in GHCR but are not deployed.

## Build pipeline

`.github/workflows/build-images.yml` runs on every push to `main` and through manual dispatch. Its matrix builds both images and publishes:

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

## Security and recovery

- Never print registry credentials, decrypted environment files, or image pull tokens.
- Diagnose a failed migration from container logs and schema state before making any corrective write.
- Do not blindly run down migrations or force schema versions; confirm backup and recovery options first.
- Treat an interrupted deployment as having unknown state until containers, revisions, and health endpoints have been inspected.
