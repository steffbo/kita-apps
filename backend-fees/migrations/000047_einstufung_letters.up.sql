-- Every Einstufung letter emailed from the app, with the exact PDF that was sent.
CREATE TABLE fees.einstufung_letters (
    id            UUID PRIMARY KEY,
    einstufung_id UUID REFERENCES fees.einstufungen(id) ON DELETE SET NULL,
    child_id      UUID NOT NULL REFERENCES fees.children(id) ON DELETE CASCADE,
    household_id  UUID REFERENCES fees.households(id) ON DELETE SET NULL,
    email_log_id  UUID REFERENCES fees.email_logs(id) ON DELETE SET NULL,
    sent_at       TIMESTAMPTZ NOT NULL,
    sent_by       UUID,
    recipients    TEXT NOT NULL,
    subject       TEXT NOT NULL,
    filename      TEXT NOT NULL,
    pdf           BYTEA NOT NULL
);

CREATE INDEX idx_einstufung_letters_child ON fees.einstufung_letters(child_id, sent_at DESC);
CREATE INDEX idx_einstufung_letters_einstufung ON fees.einstufung_letters(einstufung_id);
