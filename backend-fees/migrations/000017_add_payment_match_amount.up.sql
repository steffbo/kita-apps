ALTER TABLE fees.payment_matches
ADD COLUMN amount NUMERIC(10,2) NOT NULL DEFAULT 0;

-- Backfill existing matches with the fee's amount
UPDATE fees.payment_matches pm
SET amount = fe.amount
FROM fees.fee_expectations fe
WHERE fe.id = pm.expectation_id;
