CREATE TABLE IF NOT EXISTS fees.einstufungen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id UUID NOT NULL REFERENCES fees.children(id) ON DELETE CASCADE,
    household_id UUID NOT NULL REFERENCES fees.households(id) ON DELETE CASCADE,
    year INTEGER NOT NULL,
    valid_from DATE NOT NULL,

    -- Income calculation stored as JSONB (parent1/parent2 line items)
    income_calculation JSONB NOT NULL DEFAULT '{}',

    -- Computed fee-relevant household income
    annual_net_income NUMERIC(12, 2) NOT NULL DEFAULT 0,

    -- Classification parameters
    highest_rate_voluntary BOOLEAN NOT NULL DEFAULT FALSE,
    care_hours_per_week INTEGER NOT NULL DEFAULT 45,
    care_type TEXT NOT NULL DEFAULT 'krippe',
    children_count INTEGER NOT NULL DEFAULT 1,

    -- Resulting fees
    monthly_childcare_fee NUMERIC(10, 2) NOT NULL DEFAULT 0,
    monthly_food_fee NUMERIC(10, 2) NOT NULL DEFAULT 45.40,
    annual_membership_fee NUMERIC(10, 2) NOT NULL DEFAULT 30.00,

    -- Fee calculation details
    fee_rule TEXT NOT NULL DEFAULT '',
    discount_percent INTEGER NOT NULL DEFAULT 0,
    discount_factor NUMERIC(5, 2) NOT NULL DEFAULT 1.00,
    base_fee NUMERIC(10, 2) NOT NULL DEFAULT 0,

    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One active classification per child per year
    CONSTRAINT uq_einstufung_child_year UNIQUE (child_id, year)
);

-- Index for listing by household
CREATE INDEX idx_einstufungen_household_id ON fees.einstufungen(household_id);

-- Index for listing by year
CREATE INDEX idx_einstufungen_year ON fees.einstufungen(year);
