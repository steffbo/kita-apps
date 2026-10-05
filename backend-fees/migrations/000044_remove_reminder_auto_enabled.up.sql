-- Automatic reminder runs were removed; reminders are only sent per case.
DELETE FROM fees.app_settings WHERE key = 'reminder_auto_enabled';
