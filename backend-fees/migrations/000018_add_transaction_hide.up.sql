-- Add hidden flag for bank transactions
ALTER TABLE fees.bank_transactions
    ADD COLUMN is_hidden BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN hidden_at TIMESTAMPTZ,
    ADD COLUMN hidden_by UUID REFERENCES fees.users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_bank_transactions_hidden ON fees.bank_transactions(is_hidden);
