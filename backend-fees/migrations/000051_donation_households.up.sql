-- Optional household a donation is credited to (parents, grandparents of a family).
ALTER TABLE fees.transaction_assignments
 ADD COLUMN household_id UUID REFERENCES fees.households ON DELETE SET NULL;
CREATE INDEX ON fees.transaction_assignments(household_id);
