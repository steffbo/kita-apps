DROP INDEX IF EXISTS fees.idx_fee_expectations_membership_household_year_unique;
DROP INDEX IF EXISTS fees.idx_fee_expectations_household_id;

ALTER TABLE fees.fee_expectations
DROP COLUMN IF EXISTS household_id;

DROP INDEX IF EXISTS fees.idx_households_membership_parent_id;

ALTER TABLE fees.households
DROP CONSTRAINT IF EXISTS households_membership_assignment_status_check;

ALTER TABLE fees.households
DROP COLUMN IF EXISTS membership_assignment_status,
DROP COLUMN IF EXISTS membership_parent_id;
