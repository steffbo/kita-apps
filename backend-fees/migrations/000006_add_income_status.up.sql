-- Add income_status column to track why we do or don't have income information
-- Possible values:
-- '' (empty)      - Unknown/legacy data
-- 'PROVIDED'      - Income was provided, use for fee calculation
-- 'MAX_ACCEPTED'  - Family accepted Höchstsatz, no income needed
-- 'PENDING'       - Waiting for documents to calculate income
-- 'NOT_REQUIRED'  - Child was >3y when joining, income not required
-- 'HISTORIC'      - Child is now >3y, income kept for historic reference

ALTER TABLE fees.parents 
ADD COLUMN income_status VARCHAR(20) NOT NULL DEFAULT '';

-- Update existing records: if they have income, mark as PROVIDED
UPDATE fees.parents 
SET income_status = 'PROVIDED' 
WHERE annual_household_income IS NOT NULL;
