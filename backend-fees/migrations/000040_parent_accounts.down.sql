DROP TABLE fees.parent_reports;
DROP TABLE fees.data_changes;
ALTER TABLE fees.parent_work_entries DROP COLUMN reject_reason;
ALTER TABLE fees.parent_work_entries DROP CONSTRAINT parent_work_entries_source_check,
    ADD CONSTRAINT parent_work_entries_source_check CHECK (source IN ('MANUAL', 'IMPORT'));
ALTER TABLE fees.users DROP COLUMN parent_id;
ALTER TABLE fees.users DROP CONSTRAINT users_role_check,
    ADD CONSTRAINT users_role_check CHECK (role IN ('ADMIN', 'USER', 'PARENT_WORK'));
