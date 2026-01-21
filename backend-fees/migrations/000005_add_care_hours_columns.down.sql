-- Remove care hours columns from children table
ALTER TABLE fees.children
    DROP COLUMN IF EXISTS legal_hours,
    DROP COLUMN IF EXISTS legal_hours_until,
    DROP COLUMN IF EXISTS care_hours;
