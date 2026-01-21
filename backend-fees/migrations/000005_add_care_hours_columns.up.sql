-- Add care hours columns to children table
-- legal_hours: Rechtsanspruch - approved weekly childcare hours by Jugendamt
-- legal_hours_until: Optional end date for the Rechtsanspruch
-- care_hours: Betreuungszeit - agreed hours with the Kita

ALTER TABLE fees.children
    ADD COLUMN legal_hours INTEGER,
    ADD COLUMN legal_hours_until DATE,
    ADD COLUMN care_hours INTEGER;

COMMENT ON COLUMN fees.children.legal_hours IS 'Rechtsanspruch - approved weekly childcare hours by Jugendamt';
COMMENT ON COLUMN fees.children.legal_hours_until IS 'End date for Rechtsanspruch (NULL = unlimited)';
COMMENT ON COLUMN fees.children.care_hours IS 'Betreuungszeit - agreed weekly hours with Kita';
