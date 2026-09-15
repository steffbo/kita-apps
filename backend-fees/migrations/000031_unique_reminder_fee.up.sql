-- At most one reminder fee per base fee: paid and unpaid base fees block a
-- second reminder alike. The DB-level unique index makes concurrent sends of
-- the same base fee fail on insert instead of creating duplicate 10/5 EUR fees.
--
-- Defensive dedupe first: if older data contains several reminder fees per
-- base fee, keep the earliest row (with its payment matches) and detach the
-- others so the index can always be created.
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY reminder_for_id
               ORDER BY created_at ASC, id ASC
           ) AS rn
    FROM fees.fee_expectations
    WHERE fee_type = 'REMINDER'
      AND reminder_for_id IS NOT NULL
)
UPDATE fees.fee_expectations fe
SET reminder_for_id = NULL
FROM ranked
WHERE fe.id = ranked.id
  AND ranked.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_fee_expectations_reminder_for_id
    ON fees.fee_expectations (reminder_for_id)
    WHERE fee_type = 'REMINDER' AND reminder_for_id IS NOT NULL;
