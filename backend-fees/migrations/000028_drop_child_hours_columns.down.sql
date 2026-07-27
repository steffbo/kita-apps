-- Restores the denormalized hour columns and backfills them from the history periods
-- that are effective today.
ALTER TABLE fees.children
    ADD COLUMN IF NOT EXISTS legal_hours INTEGER,
    ADD COLUMN IF NOT EXISTS legal_hours_until DATE,
    ADD COLUMN IF NOT EXISTS care_hours INTEGER;

COMMENT ON COLUMN fees.children.legal_hours IS 'Rechtsanspruch - approved weekly childcare hours by Jugendamt';
COMMENT ON COLUMN fees.children.legal_hours_until IS 'End date for Rechtsanspruch (NULL = unlimited)';
COMMENT ON COLUMN fees.children.care_hours IS 'Betreuungszeit - agreed weekly hours with Kita';

UPDATE fees.children c
SET care_hours = (
    SELECT h.care_hours
    FROM fees.child_care_hours_history h
    WHERE h.child_id = c.id
      AND h.effective_from <= CURRENT_DATE
      AND (h.effective_until IS NULL OR h.effective_until >= CURRENT_DATE)
    ORDER BY h.effective_from DESC, h.created_at DESC
    LIMIT 1
);

UPDATE fees.children c
SET legal_hours = (
        SELECT h.legal_hours
        FROM fees.child_legal_hours_history h
        WHERE h.child_id = c.id
          AND h.effective_from <= CURRENT_DATE
          AND (h.effective_until IS NULL OR h.effective_until >= CURRENT_DATE)
        ORDER BY h.effective_from DESC, h.created_at DESC
        LIMIT 1
    ),
    legal_hours_until = (
        SELECT h.effective_until
        FROM fees.child_legal_hours_history h
        WHERE h.child_id = c.id
          AND h.effective_from <= CURRENT_DATE
          AND (h.effective_until IS NULL OR h.effective_until >= CURRENT_DATE)
        ORDER BY h.effective_from DESC, h.created_at DESC
        LIMIT 1
    );
