-- Add children_count_for_fees column to households table
-- This allows overriding the automatic sibling count for fee calculation (e.g., for foster families)
ALTER TABLE fees.households ADD COLUMN children_count_for_fees INTEGER;
