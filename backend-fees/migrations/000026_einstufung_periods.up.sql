CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE fees.einstufungen
    ADD COLUMN IF NOT EXISTS valid_until DATE,
    ADD COLUMN IF NOT EXISTS source_einstufung_id UUID REFERENCES fees.einstufungen(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS change_date DATE,
    ADD COLUMN IF NOT EXISTS effective_from_month DATE;

UPDATE fees.einstufungen
SET
    change_date = COALESCE(change_date, valid_from),
    effective_from_month = COALESCE(effective_from_month, date_trunc('month', valid_from)::date);

ALTER TABLE fees.einstufungen
    ALTER COLUMN change_date SET NOT NULL,
    ALTER COLUMN effective_from_month SET NOT NULL;

ALTER TABLE fees.einstufungen
    DROP CONSTRAINT IF EXISTS uq_einstufung_child_year;

ALTER TABLE fees.einstufungen
    ADD CONSTRAINT chk_einstufungen_effective_month_start
        CHECK (date_part('day', effective_from_month) = 1),
    ADD CONSTRAINT chk_einstufungen_valid_period
        CHECK (valid_until IS NULL OR valid_until >= effective_from_month);

ALTER TABLE fees.einstufungen
    ADD CONSTRAINT ex_einstufungen_child_period
    EXCLUDE USING gist (
        child_id WITH =,
        daterange(effective_from_month, COALESCE(valid_until + 1, 'infinity'::date), '[)') WITH &&
    );

CREATE INDEX IF NOT EXISTS idx_einstufungen_source_id ON fees.einstufungen(source_einstufung_id);
CREATE INDEX IF NOT EXISTS idx_einstufungen_effective_from_month ON fees.einstufungen(effective_from_month);
