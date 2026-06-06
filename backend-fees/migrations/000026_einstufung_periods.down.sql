DROP INDEX IF EXISTS fees.idx_einstufungen_effective_from_month;
DROP INDEX IF EXISTS fees.idx_einstufungen_source_id;

ALTER TABLE fees.einstufungen
    DROP CONSTRAINT IF EXISTS ex_einstufungen_child_period,
    DROP CONSTRAINT IF EXISTS chk_einstufungen_valid_period,
    DROP CONSTRAINT IF EXISTS chk_einstufungen_effective_month_start;

ALTER TABLE fees.einstufungen
    ADD CONSTRAINT uq_einstufung_child_year UNIQUE (child_id, year);

ALTER TABLE fees.einstufungen
    DROP COLUMN IF EXISTS effective_from_month,
    DROP COLUMN IF EXISTS change_date,
    DROP COLUMN IF EXISTS source_einstufung_id,
    DROP COLUMN IF EXISTS valid_until;
