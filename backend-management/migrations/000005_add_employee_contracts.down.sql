ALTER TABLE schedule_entries
    DROP CONSTRAINT IF EXISTS chk_shift_kind,
    DROP COLUMN IF EXISTS shift_kind;

DROP TRIGGER IF EXISTS update_employee_contracts_updated_at ON employee_contracts;
DROP TABLE IF EXISTS employee_contract_workdays;
DROP TABLE IF EXISTS employee_contracts;
