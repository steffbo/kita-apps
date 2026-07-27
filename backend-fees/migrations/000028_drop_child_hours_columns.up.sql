-- Care hours (Betreuungszeit) and legal hours (Rechtsanspruch) were stored twice:
-- as denormalized "currently effective" columns on fees.children and as periods in
-- fees.child_care_hours_history / fees.child_legal_hours_history.
--
-- The columns were only refreshed while a child row was written, so they went stale as
-- soon as a period started or ended without an accompanying write. The history tables are
-- the single source of truth from now on; the current values are derived in queries.

-- Safety net: keep column values that are not represented in the history at all.
INSERT INTO fees.child_care_hours_history (child_id, care_hours, effective_from, effective_until)
SELECT c.id, c.care_hours, c.entry_date, NULL
FROM fees.children c
WHERE c.care_hours IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM fees.child_care_hours_history h
      WHERE h.child_id = c.id AND h.care_hours IS NOT NULL
  );

INSERT INTO fees.child_legal_hours_history (child_id, legal_hours, effective_from, effective_until)
SELECT c.id, c.legal_hours, c.entry_date, c.legal_hours_until
FROM fees.children c
WHERE c.legal_hours IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM fees.child_legal_hours_history h
      WHERE h.child_id = c.id AND h.legal_hours IS NOT NULL
  );

ALTER TABLE fees.children
    DROP COLUMN IF EXISTS care_hours,
    DROP COLUMN IF EXISTS legal_hours,
    DROP COLUMN IF EXISTS legal_hours_until;
