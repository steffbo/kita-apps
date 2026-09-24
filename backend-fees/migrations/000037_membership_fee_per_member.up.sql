-- Membership fees are owed per club member, not per household: a household
-- with two members owes two fees per year. Link each MEMBERSHIP expectation to
-- the member it belongs to.
ALTER TABLE fees.fee_expectations
ADD COLUMN IF NOT EXISTS member_id UUID REFERENCES fees.members(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_fee_expectations_member_id ON fees.fee_expectations(member_id);

-- Backfill only the unambiguous case: households with exactly one member.
-- Fees of households with several members stay unassigned; the yearly
-- generation adopts them for the first member without a fee.
WITH single_member AS (
    SELECT household_id, MIN(id::text)::uuid AS member_id
    FROM fees.members
    WHERE household_id IS NOT NULL
    GROUP BY household_id
    HAVING COUNT(*) = 1
)
UPDATE fees.fee_expectations fe
SET member_id = sm.member_id
FROM single_member sm
WHERE fe.fee_type = 'MEMBERSHIP'
  AND fe.month IS NULL
  AND fe.household_id = sm.household_id
  AND fe.member_id IS NULL;

-- One fee per member and year; the household-level rule remains for fees
-- without a member (households without known members).
DROP INDEX IF EXISTS fees.idx_fee_expectations_membership_household_year_unique;

CREATE UNIQUE INDEX IF NOT EXISTS idx_fee_expectations_membership_member_year_unique
ON fees.fee_expectations(member_id, year)
WHERE fee_type = 'MEMBERSHIP' AND month IS NULL AND member_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_fee_expectations_membership_household_year_unique
ON fees.fee_expectations(household_id, year)
WHERE fee_type = 'MEMBERSHIP' AND month IS NULL AND member_id IS NULL AND year >= 2026;
