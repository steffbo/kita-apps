CREATE TABLE fees.app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE fees.email_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    to_email VARCHAR(255) NOT NULL,
    subject TEXT NOT NULL,
    body TEXT,
    email_type VARCHAR(50) NOT NULL,
    payload JSONB,
    sent_by UUID REFERENCES fees.users(id) ON DELETE SET NULL
);

CREATE INDEX idx_email_logs_sent_at ON fees.email_logs(sent_at);
CREATE INDEX idx_email_logs_type ON fees.email_logs(email_type);

INSERT INTO fees.app_settings (key, value)
VALUES ('reminder_auto_enabled', 'false')
ON CONFLICT (key) DO NOTHING;
