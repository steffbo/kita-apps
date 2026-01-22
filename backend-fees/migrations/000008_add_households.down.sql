-- Remove member references from parents
ALTER TABLE fees.parents DROP COLUMN IF EXISTS member_id;
ALTER TABLE fees.parents DROP COLUMN IF EXISTS household_id;

-- Remove household references from children
ALTER TABLE fees.children DROP COLUMN IF EXISTS household_id;

-- Drop members table
DROP TRIGGER IF EXISTS update_members_updated_at ON fees.members;
DROP TABLE IF EXISTS fees.members;

-- Drop households table
DROP TRIGGER IF EXISTS update_households_updated_at ON fees.households;
DROP TABLE IF EXISTS fees.households;
