-- Seed default admin user (password set via ADMIN_PASSWORD env var on first startup)
INSERT INTO fees.users (id, email, password_hash, first_name, last_name, role, is_active)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'admin@knirpsenstadt.de',
    '$2a$12$4ZRKskZF0GT/M/sd8skYbOesIDfYZTzjsdBGex39B0mevZJEoj/vC',
    'Admin',
    'Knirpsenstadt',
    'ADMIN',
    true
)
ON CONFLICT (id) DO UPDATE SET
    password_hash = EXCLUDED.password_hash;
