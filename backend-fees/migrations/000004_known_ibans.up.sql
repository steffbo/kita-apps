-- Known IBANs table for tracking trusted and blacklisted payment sources
CREATE TABLE fees.known_ibans (
    iban VARCHAR(34) PRIMARY KEY,
    payer_name VARCHAR(255),
    status VARCHAR(20) NOT NULL CHECK (status IN ('trusted', 'blacklisted')),
    child_id UUID REFERENCES fees.children(id) ON DELETE SET NULL,
    reason TEXT,
    original_transaction_id UUID REFERENCES fees.bank_transactions(id) ON DELETE SET NULL,
    original_description TEXT,
    original_amount DECIMAL(12,2),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for status lookups
CREATE INDEX idx_known_ibans_status ON fees.known_ibans(status);

-- Index for child linkage
CREATE INDEX idx_known_ibans_child_id ON fees.known_ibans(child_id);

-- Apply updated_at trigger
CREATE TRIGGER update_known_ibans_updated_at
    BEFORE UPDATE ON fees.known_ibans
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();
