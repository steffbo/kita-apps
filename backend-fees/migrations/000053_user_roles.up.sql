-- Accounts can hold several roles and always act in exactly one: `roles` lists the
-- assigned roles, `role` stays the active one and thereby the last one used.
ALTER TABLE fees.users ADD COLUMN roles TEXT[];
UPDATE fees.users SET roles = ARRAY[role::TEXT];
ALTER TABLE fees.users
    ALTER COLUMN roles SET NOT NULL,
    ADD CONSTRAINT users_roles_check CHECK (
        cardinality(roles) >= 1
        AND roles <@ ARRAY['ADMIN', 'BOARD', 'PARENT_WORK', 'LEITUNG', 'PARENT']::TEXT[]
        AND role::TEXT = ANY(roles)
    );

COMMENT ON COLUMN fees.users.role IS 'Active role, one of roles; survives logout as the last one used';
COMMENT ON COLUMN fees.users.roles IS 'All roles assigned to the account';
