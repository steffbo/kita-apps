-- The seeded fee regulation (000034) is the Elternbeitragsordnung in effect since
-- 2025-01-01, not since 2000. Move it to its real start date and add the
-- kindergarten (Ü3) table for reference; Ü3 care is free under the
-- Elternbeitragsentlastungsgesetz, so the calculation does not use it.
-- Only touches the untouched seed row, and only if no other version starts
-- before or on 2025-01-01.
UPDATE fees.fee_schedules
SET valid_from = '2025-01-01',
    name = 'Elternbeitragsordnung ab 01.01.2025',
    config = config || jsonb_build_object('kindergartenTable', '[
      {"minIncome": 20000.01, "rates": [49.28, 55.44, 61.60, 67.76, 73.92, 80.08]},
      {"minIncome": 22000.00, "rates": [68.99, 77.62, 86.24, 94.86, 103.49, 112.11]},
      {"minIncome": 25000.00, "rates": [95.20, 107.10, 119.00, 130.90, 142.80, 154.70]},
      {"minIncome": 28000.00, "rates": [125.44, 141.12, 156.80, 172.48, 188.16, 203.84]},
      {"minIncome": 31000.00, "rates": [138.88, 156.24, 173.60, 190.96, 208.32, 225.68]},
      {"minIncome": 34000.00, "rates": [152.32, 171.36, 190.40, 209.44, 228.48, 247.52]},
      {"minIncome": 37000.00, "rates": [165.76, 186.48, 207.20, 227.92, 248.64, 269.36]},
      {"minIncome": 40000.00, "rates": [179.20, 201.60, 224.00, 246.40, 268.80, 291.20]},
      {"minIncome": 43000.00, "rates": [192.64, 216.72, 240.80, 264.88, 288.96, 313.04]},
      {"minIncome": 46000.00, "rates": [206.08, 231.84, 257.60, 283.36, 309.12, 334.88]},
      {"minIncome": 49000.00, "rates": [219.52, 246.96, 274.40, 301.84, 329.28, 356.72]},
      {"minIncome": 52000.00, "rates": [232.96, 262.08, 291.20, 320.32, 349.44, 378.56]},
      {"minIncome": 55000.00, "rates": [246.40, 277.20, 308.00, 338.80, 369.60, 400.40]}
    ]'::jsonb),
    updated_at = NOW()
WHERE valid_from = '2000-01-01'
  AND name = 'Elternbeitragsordnung (Stand bei Einführung der Versionierung)'
  AND NOT EXISTS (
    SELECT 1 FROM fees.fee_schedules
    WHERE valid_from > '2000-01-01' AND valid_from <= '2025-01-01'
  );
