-- Create import_batches table to store metadata about CSV imports
CREATE TABLE fees.import_batches (
    id UUID PRIMARY KEY,
    file_name VARCHAR(255) NOT NULL,
    imported_by UUID NOT NULL REFERENCES fees.users(id),
    imported_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add index for faster lookups
CREATE INDEX idx_import_batches_imported_at ON fees.import_batches(imported_at DESC);

-- Note: We don't add FK constraint to bank_transactions.import_batch_id 
-- because existing data has batch IDs that won't exist in the new table.
-- New imports will create records in import_batches first.
