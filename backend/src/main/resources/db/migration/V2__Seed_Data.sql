-- V2__Seed_Data.sql
-- Initial data for Kita Knirpsenstadt

-- =============================================
-- DEFAULT GROUPS (3 Kita-Gruppen)
-- =============================================
INSERT INTO groups (name, description, color) VALUES
    ('Sonnenkinder', 'Gruppe 1 - Die Sonnenkinder', '#F59E0B'),
    ('Mondkinder', 'Gruppe 2 - Die Mondkinder', '#6366F1'),
    ('Sternenkinder', 'Gruppe 3 - Die Sternenkinder', '#10B981');

-- =============================================
-- DEFAULT ADMIN USER
-- Password: 'admin123' (BCrypt encoded)
-- IMPORTANT: Change this password immediately after first login!
-- =============================================
INSERT INTO employees (
    email, 
    password_hash, 
    first_name, 
    last_name, 
    role, 
    weekly_hours, 
    vacation_days_per_year,
    remaining_vacation_days
) VALUES (
    'admin@knirpsenstadt.de',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.qTHvL5pXVMBxKS1z1e',
    'Admin',
    'Leitung',
    'ADMIN',
    38.0,
    30,
    30
);

-- =============================================
-- BRANDENBURG HOLIDAYS 2025
-- =============================================
INSERT INTO special_days (date, name, day_type, affects_all) VALUES
    ('2025-01-01', 'Neujahr', 'HOLIDAY', TRUE),
    ('2025-04-18', 'Karfreitag', 'HOLIDAY', TRUE),
    ('2025-04-21', 'Ostermontag', 'HOLIDAY', TRUE),
    ('2025-05-01', 'Tag der Arbeit', 'HOLIDAY', TRUE),
    ('2025-05-29', 'Christi Himmelfahrt', 'HOLIDAY', TRUE),
    ('2025-06-09', 'Pfingstmontag', 'HOLIDAY', TRUE),
    ('2025-10-03', 'Tag der Deutschen Einheit', 'HOLIDAY', TRUE),
    ('2025-10-31', 'Reformationstag', 'HOLIDAY', TRUE),
    ('2025-12-25', '1. Weihnachtsfeiertag', 'HOLIDAY', TRUE),
    ('2025-12-26', '2. Weihnachtsfeiertag', 'HOLIDAY', TRUE);

-- =============================================
-- BRANDENBURG HOLIDAYS 2026
-- =============================================
INSERT INTO special_days (date, name, day_type, affects_all) VALUES
    ('2026-01-01', 'Neujahr', 'HOLIDAY', TRUE),
    ('2026-04-03', 'Karfreitag', 'HOLIDAY', TRUE),
    ('2026-04-06', 'Ostermontag', 'HOLIDAY', TRUE),
    ('2026-05-01', 'Tag der Arbeit', 'HOLIDAY', TRUE),
    ('2026-05-14', 'Christi Himmelfahrt', 'HOLIDAY', TRUE),
    ('2026-05-25', 'Pfingstmontag', 'HOLIDAY', TRUE),
    ('2026-10-03', 'Tag der Deutschen Einheit', 'HOLIDAY', TRUE),
    ('2026-10-31', 'Reformationstag', 'HOLIDAY', TRUE),
    ('2026-12-25', '1. Weihnachtsfeiertag', 'HOLIDAY', TRUE),
    ('2026-12-26', '2. Weihnachtsfeiertag', 'HOLIDAY', TRUE);

-- =============================================
-- EXAMPLE CLOSURE DAYS (can be adjusted)
-- =============================================
-- Summer closure (3 weeks in July/August - example)
-- INSERT INTO special_days (date, name, day_type, affects_all, notes) VALUES
--     ('2025-07-21', 'Sommerschließzeit', 'CLOSURE', TRUE, 'Sommerschließzeit 21.07. - 08.08.2025'),
--     ... (would be multiple days)

-- Christmas closure (example)
-- INSERT INTO special_days (date, name, day_type, affects_all, notes) VALUES
--     ('2025-12-22', 'Weihnachtsschließzeit', 'CLOSURE', TRUE, 'Weihnachtsschließzeit'),
--     ... (would continue through New Year)
