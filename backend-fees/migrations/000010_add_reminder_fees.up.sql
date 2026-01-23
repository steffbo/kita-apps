-- Add reminder_for_id column to link reminder fees to their original fee
ALTER TABLE fees.fee_expectations 
ADD COLUMN reminder_for_id UUID REFERENCES fees.fee_expectations(id) ON DELETE SET NULL;

-- Create index for looking up reminders by original fee
CREATE INDEX idx_fee_expectations_reminder_for_id ON fees.fee_expectations(reminder_for_id);

-- Update fee_type check constraint to include REMINDER
ALTER TABLE fees.fee_expectations 
DROP CONSTRAINT fee_expectations_fee_type_check;

ALTER TABLE fees.fee_expectations 
ADD CONSTRAINT fee_expectations_fee_type_check 
CHECK (fee_type IN ('MEMBERSHIP', 'FOOD', 'CHILDCARE', 'REMINDER'));
