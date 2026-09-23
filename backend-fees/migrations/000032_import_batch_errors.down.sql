ALTER TABLE fees.import_batches
    DROP COLUMN IF EXISTS errors,
    DROP COLUMN IF EXISTS error_count;
