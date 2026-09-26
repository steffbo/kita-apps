-- Adding the old constraint fails if any PARENT_WORK account still exists.
ALTER TABLE fees.users
    DROP CONSTRAINT users_role_check,
    ADD CONSTRAINT users_role_check CHECK (role IN ('ADMIN', 'USER'));
