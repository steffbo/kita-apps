-- Drop triggers
DROP TRIGGER IF EXISTS update_children_updated_at ON fees.children;
DROP TRIGGER IF EXISTS update_parents_updated_at ON fees.parents;
DROP TRIGGER IF EXISTS update_users_updated_at ON fees.users;

-- Drop trigger function
DROP FUNCTION IF EXISTS fees.update_updated_at_column();

-- Drop tables in reverse order
DROP TABLE IF EXISTS fees.payment_matches;
DROP TABLE IF EXISTS fees.bank_transactions;
DROP TABLE IF EXISTS fees.fee_expectations;
DROP TABLE IF EXISTS fees.refresh_tokens;
DROP TABLE IF EXISTS fees.users;
DROP TABLE IF EXISTS fees.child_parents;
DROP TABLE IF EXISTS fees.parents;
DROP TABLE IF EXISTS fees.children;

-- Drop schema
DROP SCHEMA IF EXISTS fees;
