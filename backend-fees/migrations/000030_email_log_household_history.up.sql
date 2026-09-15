ALTER TABLE fees.email_logs
    ADD COLUMN household_id UUID REFERENCES fees.households(id) ON DELETE SET NULL;

CREATE INDEX idx_email_logs_household_id ON fees.email_logs(household_id);

-- Backfills email_logs.household_id from payload.feeIds. Only logs whose fee
-- IDs resolve to exactly one household are mapped. Re-runnable: skips logs
-- that already carry a household_id.
CREATE OR REPLACE FUNCTION fees.backfill_email_log_households() RETURNS INTEGER AS $$
DECLARE
    updated INTEGER;
BEGIN
    UPDATE fees.email_logs el
    SET household_id = resolved.household_id
    FROM (
        SELECT el2.id,
               (ARRAY_AGG(fe.household_id ORDER BY fe.household_id NULLS LAST))[1] AS household_id
        FROM fees.email_logs el2
        CROSS JOIN LATERAL jsonb_array_elements_text(
            COALESCE(el2.payload -> 'feeIds', '[]'::jsonb)
        ) AS fid(value)
        JOIN fees.fee_expectations fe ON fe.id::text = fid.value
        WHERE el2.payload IS NOT NULL
          AND el2.household_id IS NULL
        GROUP BY el2.id
        HAVING COUNT(DISTINCT fe.household_id) = 1
    ) resolved
    WHERE el.id = resolved.id;
    GET DIAGNOSTICS updated = ROW_COUNT;
    RETURN updated;
END;
$$ LANGUAGE plpgsql;

SELECT fees.backfill_email_log_households();

-- Reliability cutoff for fee contact history: fees created on or before this
-- date without any resolvable log are treated as history_unknown; fees created
-- after it without a log count as reliably never contacted.
INSERT INTO fees.app_settings (key, value)
VALUES ('reminder_history_reliable_from', to_char(NOW() AT TIME ZONE 'UTC', 'YYYY-MM-DD'))
ON CONFLICT (key) DO NOTHING;
