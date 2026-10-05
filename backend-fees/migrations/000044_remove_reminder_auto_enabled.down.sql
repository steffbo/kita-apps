INSERT INTO fees.app_settings (key, value)
VALUES ('reminder_auto_enabled', 'false')
ON CONFLICT (key) DO NOTHING;
