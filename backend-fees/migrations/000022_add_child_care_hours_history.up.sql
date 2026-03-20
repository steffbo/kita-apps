CREATE TABLE fees.child_care_hours_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id UUID NOT NULL REFERENCES fees.children(id) ON DELETE CASCADE,
    care_hours INTEGER,
    effective_from DATE NOT NULL,
    effective_until DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT child_care_hours_history_valid_range
        CHECK (effective_until IS NULL OR effective_until >= effective_from)
);

CREATE INDEX idx_child_care_hours_history_child_id
    ON fees.child_care_hours_history(child_id, effective_from);

CREATE TRIGGER update_child_care_hours_history_updated_at
    BEFORE UPDATE ON fees.child_care_hours_history
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();

INSERT INTO fees.child_care_hours_history (child_id, care_hours, effective_from, effective_until)
SELECT id, care_hours, entry_date, NULL
FROM fees.children
WHERE care_hours IS NOT NULL;
