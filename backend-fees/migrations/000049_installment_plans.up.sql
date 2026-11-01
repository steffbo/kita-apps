CREATE TABLE fees.installment_plans (
 id UUID PRIMARY KEY, household_id UUID REFERENCES fees.households ON DELETE SET NULL,
 debtor_name TEXT NOT NULL, debtor_address TEXT NOT NULL DEFAULT '', emails TEXT[] NOT NULL DEFAULT '{}',
 total_amount NUMERIC(12,2) NOT NULL CHECK(total_amount > 0), installment_amount NUMERIC(12,2) NOT NULL CHECK(installment_amount > 0),
 interval_months INT NOT NULL DEFAULT 1 CHECK(interval_months IN (1,2,3,6,12)), first_due_date DATE NOT NULL, final_due_date DATE,
 reference TEXT NOT NULL DEFAULT '', payer_iban TEXT, note TEXT NOT NULL DEFAULT '',
 status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK(status IN ('ACTIVE','CLOSED')), close_reason TEXT, closed_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), created_by UUID
);
CREATE TABLE fees.installment_payments (
 id UUID PRIMARY KEY, plan_id UUID NOT NULL REFERENCES fees.installment_plans ON DELETE CASCADE,
 transaction_id UUID UNIQUE REFERENCES fees.bank_transactions ON DELETE CASCADE,
 amount NUMERIC(12,2) NOT NULL CHECK(amount > 0), paid_on DATE NOT NULL,
 source TEXT NOT NULL CHECK(source IN ('BANK','MANUAL')), note TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT now(), created_by UUID
);
ALTER TABLE fees.email_logs ADD COLUMN installment_plan_id UUID REFERENCES fees.installment_plans ON DELETE SET NULL;
CREATE TABLE fees.installment_charges (
 id UUID PRIMARY KEY, plan_id UUID NOT NULL REFERENCES fees.installment_plans ON DELETE CASCADE,
 amount NUMERIC(12,2) NOT NULL CHECK(amount > 0), label TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), email_log_id UUID REFERENCES fees.email_logs ON DELETE SET NULL
);
CREATE TABLE fees.installment_contracts (
 plan_id UUID PRIMARY KEY REFERENCES fees.installment_plans ON DELETE CASCADE, filename TEXT NOT NULL,
 size_bytes INT NOT NULL CHECK(size_bytes > 0 AND size_bytes <= 10485760), content BYTEA NOT NULL,
 uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now(), uploaded_by UUID
);
CREATE INDEX ON fees.installment_payments(plan_id);
CREATE INDEX ON fees.installment_charges(plan_id);
CREATE INDEX ON fees.installment_plans(payer_iban) WHERE status = 'ACTIVE';
CREATE INDEX ON fees.installment_plans(household_id);
CREATE INDEX ON fees.email_logs(installment_plan_id, sent_at DESC);
