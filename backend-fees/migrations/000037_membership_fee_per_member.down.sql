-- Restoring the household-level rule fails while a household still has more
-- than one membership fee per year; resolve those rows manually first.
DROP INDEX IF EXISTS fees.idx_fee_expectations_membership_household_year_unique;
DROP INDEX IF EXISTS fees.idx_fee_expectations_membership_member_year_unique;
DROP INDEX IF EXISTS fees.idx_fee_expectations_member_id;

ALTER TABLE fees.fee_expectations DROP COLUMN IF EXISTS member_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_fee_expectations_membership_household_year_unique
ON fees.fee_expectations(household_id, year)
WHERE fee_type = 'MEMBERSHIP' AND month IS NULL AND year >= 2026;
