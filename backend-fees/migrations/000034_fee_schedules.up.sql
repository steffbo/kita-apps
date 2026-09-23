-- Versioned fee regulation (Elternbeitragsordnung). Each row is valid from
-- valid_from until the next row starts. Replaces the tables that were hard-coded
-- in internal/domain/childcare_fee.go and the food/membership fee constants.
CREATE TABLE fees.fee_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    valid_from DATE NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    config JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed: the values that were in effect until now, valid for all history.
INSERT INTO fees.fee_schedules (valid_from, name, config) VALUES (
    '2000-01-01',
    'Elternbeitragsordnung (Stand bei Einführung der Versionierung)',
    '{
      "freeIncomeLimit": 35000.00,
      "entlastungIncomeLimit": 55000.00,
      "entlastungTable": [
        {"minIncome": 35000.01, "rates": [48.00, 54.00, 60.00, 66.00, 72.00, 78.00]},
        {"minIncome": 40000.01, "rates": [80.00, 90.00, 100.00, 110.00, 120.00, 130.00]},
        {"minIncome": 45000.01, "rates": [120.00, 135.00, 150.00, 165.00, 180.00, 195.00]},
        {"minIncome": 50000.01, "rates": [168.00, 189.00, 210.00, 231.00, 252.00, 273.00]}
      ],
      "satzungTable": [
        {"minIncome": 20000.01, "rates": [55.52, 62.46, 69.40, 76.34, 83.28, 90.22]},
        {"minIncome": 22000.00, "rates": [77.73, 87.44, 97.16, 106.88, 116.59, 126.31]},
        {"minIncome": 25000.00, "rates": [107.25, 120.66, 134.07, 147.48, 160.88, 174.29]},
        {"minIncome": 28000.00, "rates": [141.32, 158.99, 176.65, 194.32, 211.99, 229.65]},
        {"minIncome": 31000.00, "rates": [156.47, 176.02, 195.58, 215.14, 234.70, 254.26]},
        {"minIncome": 34000.00, "rates": [171.61, 193.06, 214.51, 235.96, 257.41, 278.86]},
        {"minIncome": 37000.00, "rates": [186.75, 210.09, 233.44, 256.78, 280.12, 303.47]},
        {"minIncome": 40000.00, "rates": [201.89, 227.13, 252.36, 277.60, 302.84, 328.07]},
        {"minIncome": 43000.00, "rates": [217.03, 244.16, 271.29, 298.42, 325.55, 352.68]},
        {"minIncome": 46000.00, "rates": [232.17, 261.20, 290.22, 319.24, 348.26, 377.28]},
        {"minIncome": 49000.00, "rates": [247.32, 278.23, 309.15, 340.06, 370.97, 401.89]},
        {"minIncome": 52000.00, "rates": [262.46, 295.27, 328.07, 360.88, 393.69, 426.49]},
        {"minIncome": 55000.01, "rates": [277.60, 312.30, 347.00, 381.70, 416.40, 451.10]}
      ],
      "siblingDiscountFactors": [1.00, 0.90, 0.80, 0.65, 0.45, 0.25],
      "siblingsFreeThreshold": 7,
      "monthlyFoodFee": 45.40,
      "annualMembershipFee": 30.00
    }'::jsonb
);
