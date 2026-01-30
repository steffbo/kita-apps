-- Revert MULTIPLE_OPEN_FEES warning type

-- Drop and recreate the check constraint without MULTIPLE_OPEN_FEES
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
