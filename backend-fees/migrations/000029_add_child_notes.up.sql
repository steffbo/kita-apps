CREATE TABLE fees.child_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id UUID NOT NULL REFERENCES fees.children(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_child_notes_child_created
    ON fees.child_notes(child_id, created_at DESC, id DESC);

CREATE INDEX idx_child_notes_created
    ON fees.child_notes(created_at DESC, id DESC);

CREATE TRIGGER update_child_notes_updated_at
    BEFORE UPDATE ON fees.child_notes
    FOR EACH ROW EXECUTE FUNCTION fees.update_updated_at_column();
