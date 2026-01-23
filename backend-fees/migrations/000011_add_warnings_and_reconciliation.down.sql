-- Drop transaction warnings table
DROP TABLE IF EXISTS fees.transaction_warnings;

-- Remove reconciliation_year field
DROP INDEX IF EXISTS fees.idx_fee_expectations_reconciliation_year;
ALTER TABLE fees.fee_expectations DROP COLUMN IF EXISTS reconciliation_year;
