ALTER TABLE schedule_entries
    DROP CONSTRAINT IF EXISTS chk_entry_type,
    ADD CONSTRAINT chk_entry_type CHECK (entry_type IN ('WORK', 'VACATION', 'SICK', 'CHILD_SICK', 'RECOVERY_DAY', 'SPECIAL_LEAVE', 'TRAINING', 'EVENT'));

ALTER TABLE time_entries
    DROP CONSTRAINT IF EXISTS chk_time_entry_type,
    ADD CONSTRAINT chk_time_entry_type CHECK (entry_type IN ('WORK', 'VACATION', 'SICK', 'CHILD_SICK', 'RECOVERY_DAY', 'SPECIAL_LEAVE', 'TRAINING', 'EVENT'));
