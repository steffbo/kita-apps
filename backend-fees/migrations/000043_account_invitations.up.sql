ALTER TABLE fees.users ADD COLUMN invitation_pending BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE fees.account_invitations (
    user_id UUID PRIMARY KEY REFERENCES fees.users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL
);
