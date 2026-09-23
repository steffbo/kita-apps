-- Enforce import deduplication in the database, so concurrent imports (banking-sync
-- and a manual upload) cannot insert the same booking twice. Mirrors the
-- application-level check in TransactionRepository.Exists; md5 keeps the index
-- small for long descriptions. Verified on 2026-09-23 that live data has no
-- duplicates for this key; if a dev database has some, this migration fails and
-- they must be cleaned up manually.
CREATE UNIQUE INDEX uq_bank_transactions_booking
    ON fees.bank_transactions (
        booking_date,
        COALESCE(payer_iban, ''),
        amount,
        md5(COALESCE(description, ''))
    );
