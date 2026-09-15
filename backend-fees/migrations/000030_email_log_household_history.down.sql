DELETE FROM fees.app_settings WHERE key = 'reminder_history_reliable_from';

DROP FUNCTION IF EXISTS fees.backfill_email_log_households();

DROP INDEX IF EXISTS fees.idx_email_logs_household_id;

ALTER TABLE fees.email_logs
    DROP COLUMN IF EXISTS household_id;
