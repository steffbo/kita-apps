-- Add reconciliation_year field for Kalendarjahresabrechnung
-- When set, this fee is a Nachzahlung for the specified year's annual reconciliation
ALTER TABLE fees.fee_expectations 
ADD COLUMN reconciliation_year INTEGER;

-- Create index for reconciliation queries
CREATE INDEX idx_fee_expectations_reconciliation_year 
ON fees.fee_expectations(reconciliation_year) 
WHERE reconciliation_year IS NOT NULL;

-- Transaction warnings table for flagging suspicious/unexpected transactions
CREATE TABLE fees.transaction_warnings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES fees.bank_transactions(id) ON DELETE CASCADE,
    warning_type VARCHAR(30) NOT NULL,
    message TEXT NOT NULL,
    expected_amount NUMERIC(10,2),
    actual_amount NUMERIC(10,2),
    child_id UUID REFERENCES fees.children(id) ON DELETE SET NULL,
    -- Resolution fields
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by UUID REFERENCES fees.users(id),
    resolution_type VARCHAR(20), -- 'dismissed', 'matched', 'auto_resolved'
    resolution_note TEXT,
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT transaction_warnings_type_check 
    CHECK (warning_type IN (
        'NO_MATCHING_FEE',      -- Trusted IBAN but no open fee found
        'UNEXPECTED_AMOUNT',    -- Amount doesn't match any expected fee
        'PARTIAL_PAYMENT',      -- Amount is less than expected
        'OVERPAYMENT',          -- Amount is more than expected  
        'POSSIBLE_BULK',        -- Amount could be multiple fees combined
        'DUPLICATE_PAYMENT'     -- Fee already paid, this might be duplicate
    ))
);

-- Index for listing unresolved warnings
CREATE INDEX idx_transaction_warnings_unresolved 
ON fees.transaction_warnings(created_at DESC) 
WHERE resolved_at IS NULL;

-- Index for looking up warnings by transaction
CREATE INDEX idx_transaction_warnings_transaction_id 
ON fees.transaction_warnings(transaction_id);

-- Index for looking up warnings by child
CREATE INDEX idx_transaction_warnings_child_id 
ON fees.transaction_warnings(child_id) 
WHERE child_id IS NOT NULL;
