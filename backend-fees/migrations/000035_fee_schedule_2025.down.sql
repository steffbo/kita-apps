UPDATE fees.fee_schedules
SET valid_from = '2000-01-01',
    name = 'Elternbeitragsordnung (Stand bei Einführung der Versionierung)',
    config = config - 'kindergartenTable',
    updated_at = NOW()
WHERE valid_from = '2025-01-01'
  AND name = 'Elternbeitragsordnung ab 01.01.2025';
