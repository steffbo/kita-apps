CREATE TABLE fees.banking_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_name VARCHAR(255) NOT NULL DEFAULT 'SozialBank',
    bank_blz VARCHAR(8) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    account_number VARCHAR(255),
    encrypted_pin TEXT NOT NULL,
    fints_url TEXT NOT NULL,
    tan_method VARCHAR(20) DEFAULT '911',
    product_id VARCHAR(255) DEFAULT 'KITABEITRAEGE',
    last_sync_at TIMESTAMPTZ,
    sync_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for faster lookups
CREATE INDEX idx_banking_configs_sync ON fees.banking_configs(sync_enabled);
