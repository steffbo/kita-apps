---
name: kita-live-data-access
description: Inspect, verify, or mutate live kita-apps data through the repository's approved API and database access paths. Use when Codex must query the infra-dev PostgreSQL database, inspect fees, management, or portal records, authenticate to a live Kita API, perform an explicitly authorized application write, or verify the database and downstream effects after a live operation.
---

# Kita Live Data Access

Use the application API as the write boundary and PostgreSQL as a read-only verification surface.

## Establish authorization

1. Treat inspection, diagnosis, and verification as read-only unless the user explicitly requests a mutation.
2. Confirm that the requested live change is within the user's stated scope before sending it.
3. Do not broaden an authorized record change into unrelated cleanup, migration, deployment, or account management.

## Load only the needed references

- Read [references/current-access.md](references/current-access.md) before accessing `infra-dev`, authenticating, or choosing a schema.
- Read [references/write-and-verify.md](references/write-and-verify.md) before any live mutation.
- Use the domain-specific repository skill, such as `$kita-fees-einstufung`, to construct and validate business payloads.
- Use the homelab operations skill when deployment, container lifecycle, service logs, or runtime recovery is part of the request.

## Prefer access paths in this order

1. Use an already authenticated browser or an explicitly supplied short-lived token when available.
2. Use a dedicated agent service account when the application supports one and its credentials are injected securely.
3. Use the current environment-configured application account only when authorized and necessary.
4. Use SSH plus containerized `psql` for read-only inspection and verification.

Never print, return, commit, log, or persist passwords, access tokens, refresh tokens, JWT secrets, or complete environment dumps. Keep credentials inside the smallest possible process scope.

## Read live data

1. Re-read the current access instructions in `AGENTS.md` and the schema/repository code relevant to the query.
2. Qualify database objects with `fees`, `portal`, or `public` as appropriate.
3. Use `SELECT` by default. Do not use SQL writes as an implementation shortcut.
4. Select only the columns and rows required for the task, especially when personal data is involved.
5. Avoid returning unnecessary personal or secret data in tool output or the final response.

## Write through the application

1. Resolve the current API route and request schema from router, handler, service, and OpenAPI sources.
2. Authenticate through the application's supported login flow.
3. Calculate and review the exact payload before sending it.
4. Perform only the explicitly authorized request.
5. Record the returned resource ID and structured side-effect summary without exposing tokens.
6. Do not retry a non-idempotent write until checking whether the first request succeeded.

## Verify after a write

1. Read the created or changed resource through the API when possible.
2. Run narrowly scoped read-only SQL to verify persistence and relationships.
3. Check domain side effects such as predecessor periods, household aggregates, fee expectations, matches, credits, or audit records as applicable.
4. Compare actual state with the API response and report any discrepancy.
5. Provide a direct application link when a stable route exists.

## Treat the agent account as a target state

The current fees authentication supports one environment-configured static admin identity, not independent database-backed users. Do not claim that a dedicated agent account already exists.

Prefer a future dedicated, revocable agent service account over sharing the human admin identity. Design it with least privilege, API-only use, credential rotation, an unambiguous audit identity, and explicit production-write permissions. Until that capability is implemented, follow the current authentication path in `current-access.md` and keep the limitation visible.
