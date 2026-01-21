-- Drop trigger first
DROP TRIGGER IF EXISTS update_known_ibans_updated_at ON fees.known_ibans;

-- Drop table
DROP TABLE IF EXISTS fees.known_ibans;
