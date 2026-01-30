-- Revert LATE_PAYMENT warning type addition

-- Drop and recreate the check constraint without LATE_PAYMENT
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
    'DUPLICATE_PAYMENT'     -- Fee already paid, this might be duplicate
));

-- Drop index for matched_fee_id
DROP INDEX IF EXISTS fees.idx_transaction_warnings_matched_fee_id;

-- Remove matched_fee_id column
ALTER TABLE fees.transaction_warnings 
DROP COLUMN IF EXISTS matched_fee_id;
