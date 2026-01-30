-- Add LATE_PAYMENT warning type for transactions that paid after the 15th of the fee month
-- Late payments should go to review queue, and if confirmed, generate a 10 EUR REMINDER fee

-- Add matched_fee_id column to track which fee the warning is associated with
ALTER TABLE fees.transaction_warnings 
ADD COLUMN matched_fee_id UUID REFERENCES fees.fee_expectations(id) ON DELETE SET NULL;

-- Create index for looking up warnings by matched fee
CREATE INDEX idx_transaction_warnings_matched_fee_id 
ON fees.transaction_warnings(matched_fee_id) 
WHERE matched_fee_id IS NOT NULL;

-- Drop and recreate the check constraint to add LATE_PAYMENT
ALTER TABLE fees.transaction_warnings 
DROP CONSTRAINT transaction_warnings_type_check;

ALTER TABLE fees.transaction_warnings 
ADD CONSTRAINT transaction_warnings_type_check 
CHECK (warning_type IN (
    'NO_MATCHING_FEE',      -- Trusted IBAN but no open fee found
    'UNEXPECTED_AMOUNT',    -- Amount doesn't match any expected fee
    'PARTIAL_PAYMENT',      -- Amount is less than expected
    'OVERPAYMENT',          -- Amount is more than expected  
    'POSSIBLE_BULK',        -- Amount could be multiple fees combined
    'DUPLICATE_PAYMENT',    -- Fee already paid, this might be duplicate
    'LATE_PAYMENT'          -- Payment received after the 15th of fee month
));
