CREATE TABLE fees.parent_work_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    valid_from DATE NOT NULL UNIQUE,
    hours_per_child_minutes INT NOT NULL CHECK (hours_per_child_minutes >= 0),
    missing_hour_rate_cents INT NOT NULL CHECK (missing_hour_rate_cents >= 0),
    max_carry_over_minutes INT NOT NULL CHECK (max_carry_over_minutes >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER update_parent_work_rules_updated_at BEFORE UPDATE ON fees.parent_work_rules
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();
INSERT INTO fees.parent_work_rules
    (valid_from, hours_per_child_minutes, missing_hour_rate_cents, max_carry_over_minutes)
VALUES ('2025-08-01', 540, 3000, 180);

CREATE TABLE fees.board_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES fees.members(id) ON DELETE CASCADE,
    office TEXT NOT NULL CHECK (btrim(office) <> ''),
    start_date DATE NOT NULL,
    end_date DATE CHECK (end_date IS NULL OR end_date >= start_date),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_board_terms_member_id ON fees.board_terms(member_id);
CREATE TRIGGER update_board_terms_updated_at BEFORE UPDATE ON fees.board_terms
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();

CREATE TABLE fees.parent_work_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id UUID NOT NULL REFERENCES fees.households(id),
    work_date DATE NOT NULL,
    duration_minutes INT NOT NULL CHECK (duration_minutes > 0 AND duration_minutes % 15 = 0),
    occasion TEXT NOT NULL CHECK (btrim(occasion) <> ''),
    member_name TEXT,
    child_name TEXT,
    status TEXT NOT NULL DEFAULT 'APPROVED'
        CHECK (status IN ('SUBMITTED', 'APPROVED', 'REJECTED', 'VOIDED')),
    void_reason TEXT,
    source TEXT NOT NULL DEFAULT 'MANUAL' CHECK (source IN ('MANUAL', 'IMPORT')),
    created_by UUID REFERENCES fees.users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES fees.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT parent_work_void_reason CHECK
        (status <> 'VOIDED' OR (void_reason IS NOT NULL AND btrim(void_reason) <> ''))
);
CREATE INDEX idx_parent_work_entries_household_date
    ON fees.parent_work_entries(household_id, work_date);
CREATE TRIGGER update_parent_work_entries_updated_at BEFORE UPDATE ON fees.parent_work_entries
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();

CREATE TABLE fees.parent_work_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id UUID NOT NULL REFERENCES fees.households(id),
    kita_year INT NOT NULL,
    required_minutes INT NOT NULL CHECK (required_minutes >= 0),
    reason TEXT NOT NULL CHECK (btrim(reason) <> ''),
    created_by UUID REFERENCES fees.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (household_id, kita_year)
);
CREATE TRIGGER update_parent_work_overrides_updated_at BEFORE UPDATE ON fees.parent_work_overrides
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();
