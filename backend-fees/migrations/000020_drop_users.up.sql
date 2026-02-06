-- Drop users table but keep refresh_tokens for session persistence
-- First drop foreign key constraints that reference users
ALTER TABLE IF EXISTS fees.bank_transactions DROP CONSTRAINT IF EXISTS bank_transactions_hidden_by_fkey;
ALTER TABLE IF EXISTS fees.import_batches DROP CONSTRAINT IF EXISTS import_batches_imported_by_fkey;
ALTER TABLE IF EXISTS fees.payment_matches DROP CONSTRAINT IF EXISTS payment_matches_matched_by_fkey;
ALTER TABLE IF EXISTS fees.transaction_warnings DROP CONSTRAINT IF EXISTS transaction_warnings_resolved_by_fkey;
ALTER TABLE IF EXISTS fees.email_logs DROP CONSTRAINT IF EXISTS email_logs_sent_by_fkey;

-- Drop foreign key from refresh_tokens to users before dropping users
ALTER TABLE IF EXISTS fees.refresh_tokens DROP CONSTRAINT IF EXISTS refresh_tokens_user_id_fkey;

-- Drop users table only
DROP TABLE IF EXISTS fees.users CASCADE;

-- Drop type
DROP TYPE IF EXISTS fees.user_role CASCADE;
