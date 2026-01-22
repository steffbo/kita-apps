-- Remove exit_date column
ALTER TABLE fees.children DROP COLUMN IF EXISTS exit_date;
