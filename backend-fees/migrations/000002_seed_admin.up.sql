-- Seed default admin user
-- Password: admin123 (bcrypt hash with cost 12)
INSERT INTO fees.users (id, email, password_hash, first_name, last_name, role, is_active)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'admin@knirpsenstadt.de',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4.VTtYEMlpNQ3.Nm',
    'Admin',
    'Knirpsenstadt',
    'ADMIN',
    true
);
