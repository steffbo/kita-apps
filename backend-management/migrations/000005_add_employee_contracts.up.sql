-- Historical employee contracts and shift kind support

CREATE TABLE employee_contracts (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    valid_from DATE NOT NULL,
    weekly_hours DECIMAL(4,2) NOT NULL CHECK (weekly_hours >= 0 AND weekly_hours <= 40),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_employee_contract_month UNIQUE (employee_id, valid_from),
    CONSTRAINT chk_employee_contract_month CHECK (valid_from = date_trunc('month', valid_from)::date)
);

CREATE TABLE employee_contract_workdays (
    id BIGSERIAL PRIMARY KEY,
    contract_id BIGINT NOT NULL REFERENCES employee_contracts(id) ON DELETE CASCADE,
    weekday INTEGER NOT NULL CHECK (weekday >= 1 AND weekday <= 5),
    planned_minutes INTEGER NOT NULL CHECK (planned_minutes >= 0 AND planned_minutes <= 600),
    CONSTRAINT uq_contract_weekday UNIQUE (contract_id, weekday)
);

CREATE INDEX idx_employee_contracts_employee_valid_from ON employee_contracts(employee_id, valid_from DESC);
CREATE INDEX idx_employee_contract_workdays_contract ON employee_contract_workdays(contract_id);

CREATE TRIGGER update_employee_contracts_updated_at
    BEFORE UPDATE ON employee_contracts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

ALTER TABLE schedule_entries
    ADD COLUMN shift_kind VARCHAR(20) NOT NULL DEFAULT 'MANUAL',
    ADD CONSTRAINT chk_shift_kind CHECK (shift_kind IN ('EARLY', 'LATE', 'MANUAL'));

INSERT INTO employee_contracts (employee_id, valid_from, weekly_hours)
SELECT id, date_trunc('month', CURRENT_DATE)::date, weekly_hours
FROM employees
ON CONFLICT (employee_id, valid_from) DO NOTHING;

INSERT INTO employee_contract_workdays (contract_id, weekday, planned_minutes)
SELECT c.id, d.weekday,
       CASE
           WHEN c.weekly_hours = 33 AND d.weekday <= 3 THEN 420
           WHEN c.weekly_hours = 33 THEN 360
           ELSE ROUND(c.weekly_hours * 60 / 5)::integer
       END
FROM employee_contracts c
CROSS JOIN generate_series(1, 5) AS d(weekday)
ON CONFLICT (contract_id, weekday) DO NOTHING;
