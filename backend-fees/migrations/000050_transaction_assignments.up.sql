-- A bank transaction that belongs to neither fees nor a Ratenzahlung but has a
-- named purpose (currently only donations). One row per transaction; a further
-- kind is a new CHECK value, not a new table.
CREATE TABLE fees.transaction_assignments (
 transaction_id UUID PRIMARY KEY REFERENCES fees.bank_transactions ON DELETE CASCADE,
 kind TEXT NOT NULL CHECK(kind IN ('DONATION')),
 note TEXT NOT NULL DEFAULT '',
 assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 assigned_by UUID
);
CREATE INDEX ON fees.transaction_assignments(kind);
