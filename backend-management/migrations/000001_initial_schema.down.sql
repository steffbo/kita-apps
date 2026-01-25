-- Drop initial schema

DROP TRIGGER IF EXISTS update_schedule_entries_updated_at ON schedule_entries;
DROP TRIGGER IF EXISTS update_groups_updated_at ON groups;
DROP TRIGGER IF EXISTS update_employees_updated_at ON employees;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS time_entries;
DROP TABLE IF EXISTS schedule_entries;
DROP TABLE IF EXISTS special_days;
DROP TABLE IF EXISTS group_assignments;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS employees;
