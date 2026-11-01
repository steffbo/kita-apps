-- Every account keeps only its active role.
ALTER TABLE fees.users DROP COLUMN roles;
COMMENT ON COLUMN fees.users.role IS NULL;
