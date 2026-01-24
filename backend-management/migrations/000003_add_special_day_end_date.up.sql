-- Add end_date column to special_days for multi-day closures
ALTER TABLE special_days ADD COLUMN end_date DATE;
