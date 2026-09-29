ALTER TABLE fees.data_changes
    ADD COLUMN impersonated_by UUID REFERENCES fees.users(id) ON DELETE SET NULL;
