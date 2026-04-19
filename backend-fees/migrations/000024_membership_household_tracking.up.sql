ALTER TABLE fees.households
ADD COLUMN IF NOT EXISTS membership_parent_id UUID REFERENCES fees.parents(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS membership_assignment_status VARCHAR(20) NOT NULL DEFAULT 'ASSUMED';

ALTER TABLE fees.households
DROP CONSTRAINT IF EXISTS households_membership_assignment_status_check;

ALTER TABLE fees.households
ADD CONSTRAINT households_membership_assignment_status_check
CHECK (membership_assignment_status IN ('ASSUMED', 'CONFIRMED'));

CREATE INDEX IF NOT EXISTS idx_households_membership_parent_id ON fees.households(membership_parent_id);

ALTER TABLE fees.fee_expectations
ADD COLUMN IF NOT EXISTS household_id UUID REFERENCES fees.households(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_fee_expectations_household_id ON fees.fee_expectations(household_id);

-- Backfill household_id from child relation.
UPDATE fees.fee_expectations fe
SET household_id = c.household_id
FROM fees.children c
WHERE fe.child_id = c.id
  AND fe.household_id IS NULL;

-- Keep reminder household links in sync with their base expectation where possible.
UPDATE fees.fee_expectations rem
SET household_id = base.household_id
FROM fees.fee_expectations base
WHERE rem.fee_type = 'REMINDER'
  AND rem.reminder_for_id = base.id
  AND rem.household_id IS DISTINCT FROM base.household_id;

-- Auto-assign a household membership parent where none exists yet.
WITH ranked_parents AS (
    SELECT
        p.household_id,
        p.id AS parent_id,
        (p.member_id IS NOT NULL) AS has_member,
        ROW_NUMBER() OVER (
            PARTITION BY p.household_id
            ORDER BY
                CASE WHEN p.member_id IS NOT NULL THEN 0 ELSE 1 END,
                p.created_at ASC,
                p.id ASC
        ) AS rn
    FROM fees.parents p
    WHERE p.household_id IS NOT NULL
), chosen AS (
    SELECT household_id, parent_id, has_member
    FROM ranked_parents
    WHERE rn = 1
)
UPDATE fees.households h
SET
    membership_parent_id = c.parent_id,
    membership_assignment_status = CASE WHEN c.has_member THEN 'CONFIRMED' ELSE 'ASSUMED' END
FROM chosen c
WHERE h.id = c.household_id
  AND h.membership_parent_id IS NULL;

-- Consolidate duplicate membership expectations for 2026: keep one canonical fee per household/year.
WITH membership_ranked AS (
    SELECT
        fe.id,
        fe.household_id,
        fe.year,
        fe.created_at,
        COALESCE(SUM(pm.amount), 0) AS matched_amount,
        FIRST_VALUE(fe.id) OVER (
            PARTITION BY fe.household_id, fe.year
            ORDER BY
                CASE WHEN COALESCE(SUM(pm.amount), 0) > 0 THEN 0 ELSE 1 END,
                COALESCE(SUM(pm.amount), 0) DESC,
                fe.created_at ASC,
                fe.id ASC
        ) AS canonical_id,
        ROW_NUMBER() OVER (
            PARTITION BY fe.household_id, fe.year
            ORDER BY
                CASE WHEN COALESCE(SUM(pm.amount), 0) > 0 THEN 0 ELSE 1 END,
                COALESCE(SUM(pm.amount), 0) DESC,
                fe.created_at ASC,
                fe.id ASC
        ) AS rn
    FROM fees.fee_expectations fe
    LEFT JOIN fees.payment_matches pm ON pm.expectation_id = fe.id
    WHERE fe.fee_type = 'MEMBERSHIP'
      AND fe.month IS NULL
      AND fe.year = 2026
      AND fe.household_id IS NOT NULL
    GROUP BY fe.id, fe.household_id, fe.year, fe.created_at
), duplicates AS (
    SELECT id AS duplicate_id, canonical_id
    FROM membership_ranked
    WHERE rn > 1
)
UPDATE fees.payment_matches pm
SET expectation_id = d.canonical_id
FROM duplicates d
WHERE pm.expectation_id = d.duplicate_id
  AND NOT EXISTS (
      SELECT 1
      FROM fees.payment_matches pm2
      WHERE pm2.transaction_id = pm.transaction_id
        AND pm2.expectation_id = d.canonical_id
  );

WITH membership_ranked AS (
    SELECT
        fe.id,
        fe.household_id,
        fe.year,
        fe.created_at,
        COALESCE(SUM(pm.amount), 0) AS matched_amount,
        FIRST_VALUE(fe.id) OVER (
            PARTITION BY fe.household_id, fe.year
            ORDER BY
                CASE WHEN COALESCE(SUM(pm.amount), 0) > 0 THEN 0 ELSE 1 END,
                COALESCE(SUM(pm.amount), 0) DESC,
                fe.created_at ASC,
                fe.id ASC
        ) AS canonical_id,
        ROW_NUMBER() OVER (
            PARTITION BY fe.household_id, fe.year
            ORDER BY
                CASE WHEN COALESCE(SUM(pm.amount), 0) > 0 THEN 0 ELSE 1 END,
                COALESCE(SUM(pm.amount), 0) DESC,
                fe.created_at ASC,
                fe.id ASC
        ) AS rn
    FROM fees.fee_expectations fe
    LEFT JOIN fees.payment_matches pm ON pm.expectation_id = fe.id
    WHERE fe.fee_type = 'MEMBERSHIP'
      AND fe.month IS NULL
      AND fe.year = 2026
      AND fe.household_id IS NOT NULL
    GROUP BY fe.id, fe.household_id, fe.year, fe.created_at
), duplicates AS (
    SELECT id AS duplicate_id, canonical_id
    FROM membership_ranked
    WHERE rn > 1
)
UPDATE fees.fee_expectations rem
SET reminder_for_id = d.canonical_id
FROM duplicates d
WHERE rem.reminder_for_id = d.duplicate_id;

WITH membership_ranked AS (
    SELECT
        fe.id,
        fe.household_id,
        fe.year,
        fe.created_at,
        COALESCE(SUM(pm.amount), 0) AS matched_amount,
        ROW_NUMBER() OVER (
            PARTITION BY fe.household_id, fe.year
            ORDER BY
                CASE WHEN COALESCE(SUM(pm.amount), 0) > 0 THEN 0 ELSE 1 END,
                COALESCE(SUM(pm.amount), 0) DESC,
                fe.created_at ASC,
                fe.id ASC
        ) AS rn
    FROM fees.fee_expectations fe
    LEFT JOIN fees.payment_matches pm ON pm.expectation_id = fe.id
    WHERE fe.fee_type = 'MEMBERSHIP'
      AND fe.month IS NULL
      AND fe.year = 2026
      AND fe.household_id IS NOT NULL
    GROUP BY fe.id, fe.household_id, fe.year, fe.created_at
), duplicates AS (
    SELECT id AS duplicate_id
    FROM membership_ranked
    WHERE rn > 1
)
DELETE FROM fees.fee_expectations fe
USING duplicates d
WHERE fe.id = d.duplicate_id
  AND NOT EXISTS (
      SELECT 1
      FROM fees.payment_matches pm
      WHERE pm.expectation_id = fe.id
  );

-- Prevent new duplicates from 2026 onward.
CREATE UNIQUE INDEX IF NOT EXISTS idx_fee_expectations_membership_household_year_unique
ON fees.fee_expectations(household_id, year)
WHERE fee_type = 'MEMBERSHIP' AND month IS NULL AND year >= 2026;
