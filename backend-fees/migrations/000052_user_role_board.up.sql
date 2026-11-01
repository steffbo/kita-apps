-- USER only identifies the import token now. A leftover account has no purpose
-- any more: it is deactivated with the narrowest staff role until an admin
-- assigns a real one.
UPDATE fees.users SET role = 'PARENT_WORK', is_active = false WHERE role = 'USER';

ALTER TABLE fees.users
    DROP CONSTRAINT users_role_check,
    ADD CONSTRAINT users_role_check CHECK (role IN ('ADMIN', 'BOARD', 'PARENT_WORK', 'LEITUNG', 'PARENT'));
