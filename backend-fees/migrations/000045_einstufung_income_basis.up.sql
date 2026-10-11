-- What the childcare fee of an Einstufung is based on, and whether income
-- proofs are still missing. annual_net_income is only set for INCOME; the
-- former 0/-3000 placeholders for Höchstsatz and Kindergarten children
-- (old form default: WK-Pauschale without wages) become NULL.
ALTER TABLE fees.einstufungen
    ADD COLUMN income_basis VARCHAR(20) NOT NULL DEFAULT 'INCOME',
    ADD COLUMN provisional BOOLEAN NOT NULL DEFAULT false,
    ALTER COLUMN annual_net_income DROP NOT NULL,
    ALTER COLUMN annual_net_income DROP DEFAULT;

UPDATE fees.einstufungen
SET income_basis = 'HIGHEST_RATE', annual_net_income = NULL
WHERE highest_rate_voluntary;

-- Kindergarten children whose parents were never asked for income: nothing
-- but the WK-Pauschale was entered. Their care is free.
UPDATE fees.einstufungen e
SET income_basis = 'NOT_REQUIRED',
    annual_net_income = NULL,
    notes = CASE
        WHEN e.notes = '' THEN 'Einkommen unbekannt – nicht abgefragt (Kindergartenkind, beitragsfrei)'
        ELSE 'Einkommen unbekannt – nicht abgefragt (Kindergartenkind, beitragsfrei)' || E'\n' || e.notes
    END
WHERE e.income_basis = 'INCOME'
  AND e.care_type = 'kindergarten'
  AND NOT EXISTS (
      SELECT 1
      FROM jsonb_each(e.income_calculation) AS p(parent, details),
           jsonb_each(CASE WHEN jsonb_typeof(p.details) = 'object' THEN p.details ELSE '{}'::jsonb END) AS f(key, value)
      WHERE f.key <> 'advertisingCosts'
        AND jsonb_typeof(f.value) = 'number'
        AND (f.value)::numeric <> 0
  );

ALTER TABLE fees.einstufungen
    ADD CONSTRAINT chk_einstufungen_income_basis
        CHECK (income_basis IN ('INCOME', 'HIGHEST_RATE', 'NOT_REQUIRED')),
    ADD CONSTRAINT chk_einstufungen_highest_rate_basis
        CHECK (highest_rate_voluntary = (income_basis = 'HIGHEST_RATE')),
    ADD CONSTRAINT chk_einstufungen_income_only_for_income_basis
        CHECK ((annual_net_income IS NOT NULL) = (income_basis = 'INCOME'));

-- Households: the highest rate needs no income; households whose open
-- Einstufungen were all corrected to NOT_REQUIRED carried the same placeholder.
UPDATE fees.households
SET annual_household_income = NULL, updated_at = NOW()
WHERE income_status = 'MAX_ACCEPTED'
  AND annual_household_income IS NOT NULL;

UPDATE fees.households h
SET income_status = 'NOT_REQUIRED', annual_household_income = NULL, updated_at = NOW()
WHERE h.income_status = 'PROVIDED'
  AND h.annual_household_income <= 0
  AND EXISTS (
      SELECT 1 FROM fees.einstufungen e
      WHERE e.household_id = h.id AND e.valid_until IS NULL
  )
  AND NOT EXISTS (
      SELECT 1 FROM fees.einstufungen e
      WHERE e.household_id = h.id AND e.valid_until IS NULL AND e.income_basis <> 'NOT_REQUIRED'
  );
