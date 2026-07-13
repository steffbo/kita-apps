# Write and Verify

## Before sending

- Confirm that the user requested the specific live mutation.
- Check the current resource to avoid overwriting newer work.
- Resolve the exact route and payload from current code and OpenAPI.
- Validate business calculations with the relevant domain skill.
- Save no credential or token to disk.
- Prepare a read-only verification query before writing.

## Sending

- Use the application API, not SQL.
- Capture the HTTP status, returned resource ID, and documented side-effect summary.
- Treat timeouts and interrupted responses as unknown outcomes. Check current state before retrying.
- Avoid broad environment inspection and never print complete container environments.

## Verifying

Use both API readback and minimal read-only SQL where the risk warrants it. Check:

- target record values and timestamps
- predecessor/successor or other relationship changes
- derived household or aggregate values
- downstream expectations, matches, credits, or audit records
- absence of unintended duplicate records

Report the direct app link, final identifiers, effective period, derived amounts, and any unresolved discrepancy. Do not include secrets or unrelated personal data.
