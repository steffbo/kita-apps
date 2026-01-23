-- Restore original fee_type check constraint (without REMINDER)
ALTER TABLE fees.fee_expectations 
DROP CONSTRAINT IF EXISTS fee_expectations_fee_type_check;

ALTER TABLE fees.fee_expectations 
ADD CONSTRAINT fee_expectations_fee_type_check 
CHECK (fee_type IN ('MEMBERSHIP', 'FOOD', 'CHILDCARE'));

-- Remove reminder_for_id column
DROP INDEX IF EXISTS fees.idx_fee_expectations_reminder_for_id;
ALTER TABLE fees.fee_expectations DROP COLUMN IF EXISTS reminder_for_id;
