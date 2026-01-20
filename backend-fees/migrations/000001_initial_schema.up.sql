-- Create fees schema
CREATE SCHEMA IF NOT EXISTS fees;

-- Children table
CREATE TABLE fees.children (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_number VARCHAR(10) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    birth_date DATE NOT NULL,
    entry_date DATE NOT NULL,
    street VARCHAR(200),
    house_number VARCHAR(20),
    postal_code VARCHAR(10),
    city VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Parents table
CREATE TABLE fees.parents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    birth_date DATE,
    email VARCHAR(255),
    phone VARCHAR(50),
    street VARCHAR(200),
    house_number VARCHAR(20),
    postal_code VARCHAR(10),
    city VARCHAR(100),
    annual_household_income DECIMAL(12,2),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Child-Parent relationship
CREATE TABLE fees.child_parents (
    child_id UUID REFERENCES fees.children(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES fees.parents(id) ON DELETE CASCADE,
    is_primary BOOLEAN DEFAULT false,
    PRIMARY KEY (child_id, parent_id)
);

-- Users table
CREATE TABLE fees.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(20) DEFAULT 'USER' CHECK (role IN ('ADMIN', 'USER')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Refresh tokens table
CREATE TABLE fees.refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES fees.users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Fee expectations table
CREATE TABLE fees.fee_expectations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id UUID REFERENCES fees.children(id) ON DELETE CASCADE,
    fee_type VARCHAR(20) NOT NULL CHECK (fee_type IN ('MEMBERSHIP', 'FOOD', 'CHILDCARE')),
    year INT NOT NULL,
    month INT CHECK (month IS NULL OR (month >= 1 AND month <= 12)),
    amount DECIMAL(10,2) NOT NULL,
    due_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (child_id, fee_type, year, month)
);

-- Bank transactions table
CREATE TABLE fees.bank_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_date DATE NOT NULL,
    value_date DATE NOT NULL,
    payer_name VARCHAR(255),
    payer_iban VARCHAR(34),
    description TEXT,
    amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'EUR',
    import_batch_id UUID,
    imported_at TIMESTAMPTZ DEFAULT NOW()
);

-- Payment matches table
CREATE TABLE fees.payment_matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID REFERENCES fees.bank_transactions(id) ON DELETE CASCADE,
    expectation_id UUID REFERENCES fees.fee_expectations(id) ON DELETE CASCADE,
    match_type VARCHAR(20) NOT NULL CHECK (match_type IN ('AUTO', 'MANUAL')),
    confidence DECIMAL(3,2),
    matched_at TIMESTAMPTZ DEFAULT NOW(),
    matched_by UUID REFERENCES fees.users(id),
    UNIQUE (transaction_id, expectation_id)
);

-- Indexes
CREATE INDEX idx_children_member_number ON fees.children(member_number);
CREATE INDEX idx_children_last_name ON fees.children(last_name);
CREATE INDEX idx_children_is_active ON fees.children(is_active);
CREATE INDEX idx_parents_last_name ON fees.parents(last_name);
CREATE INDEX idx_parents_email ON fees.parents(email);
CREATE INDEX idx_users_email ON fees.users(email);
CREATE INDEX idx_refresh_tokens_user_id ON fees.refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON fees.refresh_tokens(expires_at);
CREATE INDEX idx_fee_expectations_child_id ON fees.fee_expectations(child_id);
CREATE INDEX idx_fee_expectations_year_month ON fees.fee_expectations(year, month);
CREATE INDEX idx_fee_expectations_fee_type ON fees.fee_expectations(fee_type);
CREATE INDEX idx_bank_transactions_booking_date ON fees.bank_transactions(booking_date);
CREATE INDEX idx_bank_transactions_import_batch ON fees.bank_transactions(import_batch_id);
CREATE INDEX idx_bank_transactions_amount ON fees.bank_transactions(amount);
CREATE INDEX idx_payment_matches_transaction ON fees.payment_matches(transaction_id);
CREATE INDEX idx_payment_matches_expectation ON fees.payment_matches(expectation_id);

-- Trigger function for updated_at
CREATE OR REPLACE FUNCTION fees.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_children_updated_at
    BEFORE UPDATE ON fees.children
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();

CREATE TRIGGER update_parents_updated_at
    BEFORE UPDATE ON fees.parents
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON fees.users
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();
