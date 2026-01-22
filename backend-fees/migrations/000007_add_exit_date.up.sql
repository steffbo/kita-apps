-- Add exit_date column to track when a child leaves the Kita
ALTER TABLE fees.children ADD COLUMN exit_date DATE;

-- Add comment for documentation
COMMENT ON COLUMN fees.children.exit_date IS 'Date when the child is expected to leave the Kita (e.g., moving or starting school)';
