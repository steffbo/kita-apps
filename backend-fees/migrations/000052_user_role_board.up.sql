ALTER TABLE fees.users
    DROP CONSTRAINT users_role_check,
    ADD CONSTRAINT users_role_check CHECK (role IN ('ADMIN', 'USER', 'BOARD', 'PARENT_WORK', 'LEITUNG', 'PARENT'));
