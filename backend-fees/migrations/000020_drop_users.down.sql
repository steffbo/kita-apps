-- Recreate users table (reverse of drop)
CREATE TYPE fees.user_role AS ENUM ('ADMIN', 'USER');

CREATE TABLE fees.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role fees.user_role DEFAULT 'USER',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Re-add foreign key constraint to refresh_tokens
ALTER TABLE fees.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES fees.users(id) ON DELETE CASCADE;
