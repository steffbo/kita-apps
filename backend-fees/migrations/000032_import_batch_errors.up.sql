-- Per-row import failures, so automated (banking-sync) imports surface errors
-- in the import history instead of silently counting rows as skipped.
ALTER TABLE fees.import_batches
    ADD COLUMN error_count INT NOT NULL DEFAULT 0,
    ADD COLUMN errors JSONB NOT NULL DEFAULT '[]'::jsonb;
