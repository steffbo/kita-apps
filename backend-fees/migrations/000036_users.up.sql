-- Persistent user accounts (replaces the static admin from USER_NAME/USER_PASSWORD).
-- The server bootstraps the admin from those env vars on start when no user with
-- that email exists, reusing the former static admin ID so refresh tokens and
-- stored actor IDs (imported_by, matched_by, …) keep pointing at the same user.
CREATE TABLE fees.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    role TEXT NOT NULL DEFAULT 'USER' CHECK (role IN ('ADMIN', 'USER')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_users_email ON fees.users (LOWER(email));
