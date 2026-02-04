DROP INDEX IF EXISTS idx_bank_transactions_hidden;

ALTER TABLE fees.bank_transactions
    DROP COLUMN IF EXISTS hidden_by,
    DROP COLUMN IF EXISTS hidden_at,
    DROP COLUMN IF EXISTS is_hidden;
