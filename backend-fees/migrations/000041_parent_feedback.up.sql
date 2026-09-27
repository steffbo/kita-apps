ALTER TABLE fees.parent_reports ADD COLUMN response TEXT;

ALTER TABLE fees.parent_work_entries
    ADD COLUMN reviewed_by UUID REFERENCES fees.users(id) ON DELETE SET NULL,
    ADD COLUMN reviewed_at TIMESTAMPTZ;
-- Already reviewed parent submissions: the last editor was the reviewer (best available evidence).
UPDATE fees.parent_work_entries SET reviewed_by = updated_by, reviewed_at = updated_at
    WHERE source = 'PARENT' AND status IN ('APPROVED', 'REJECTED');
