-- When the Einstufung letter was last emailed from the app; the letter date
-- follows it while the Einstufung stays unchanged.
ALTER TABLE fees.einstufungen ADD COLUMN letter_sent_at TIMESTAMPTZ;
