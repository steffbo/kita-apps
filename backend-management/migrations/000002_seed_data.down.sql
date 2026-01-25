-- Remove seeded data

DELETE FROM special_days
WHERE date BETWEEN '2025-01-01' AND '2026-12-31'
  AND day_type = 'HOLIDAY';

DELETE FROM employees WHERE email = 'admin@knirpsenstadt.de';

DELETE FROM groups WHERE name IN ('Sonnenkinder', 'Mondkinder', 'Sternenkinder');
