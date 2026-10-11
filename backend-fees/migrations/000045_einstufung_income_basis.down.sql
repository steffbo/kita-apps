-- The data correction of the up migration is not reverted: removed placeholder
-- incomes become 0.
ALTER TABLE fees.einstufungen
    DROP CONSTRAINT IF EXISTS chk_einstufungen_income_only_for_income_basis,
    DROP CONSTRAINT IF EXISTS chk_einstufungen_highest_rate_basis,
    DROP CONSTRAINT IF EXISTS chk_einstufungen_income_basis;

UPDATE fees.einstufungen SET annual_net_income = 0 WHERE annual_net_income IS NULL;

ALTER TABLE fees.einstufungen
    ALTER COLUMN annual_net_income SET DEFAULT 0,
    ALTER COLUMN annual_net_income SET NOT NULL,
    DROP COLUMN IF EXISTS provisional,
    DROP COLUMN IF EXISTS income_basis;
