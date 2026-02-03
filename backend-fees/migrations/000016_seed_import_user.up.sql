-- Seed system import user for automated CSV uploads
INSERT INTO fees.users (
    id,
    email,
    password_hash,
    first_name,
    last_name,
    role,
    is_active,
    created_at,
    updated_at
)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'importer@system.local',
    '$2a$10$E46QlU6wkiXJtKH2AZPKy.pp7l83iUARHFQWCuAYe.zRwnbmQ1UK6',
    'Import',
    'Service',
    'USER',
    true,
    NOW(),
    NOW()
)
ON CONFLICT DO NOTHING;
