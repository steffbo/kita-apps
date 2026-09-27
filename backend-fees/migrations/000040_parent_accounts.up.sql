ALTER TABLE fees.users DROP CONSTRAINT users_role_check,
    ADD CONSTRAINT users_role_check CHECK (role IN ('ADMIN', 'USER', 'PARENT_WORK', 'PARENT'));
ALTER TABLE fees.users ADD COLUMN parent_id UUID UNIQUE REFERENCES fees.parents(id) ON DELETE SET NULL;

CREATE TABLE fees.data_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR NOT NULL CHECK (entity_type IN ('PARENT', 'CHILD')),
    entity_id UUID NOT NULL,
    parent_id UUID REFERENCES fees.parents(id) ON DELETE SET NULL,
    user_id UUID REFERENCES fees.users(id) ON DELETE SET NULL,
    field VARCHAR NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_data_changes_changed_at ON fees.data_changes(changed_at DESC);
CREATE INDEX idx_data_changes_entity ON fees.data_changes(entity_type, entity_id, changed_at DESC);

CREATE TABLE fees.parent_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID NOT NULL REFERENCES fees.parents(id) ON DELETE CASCADE,
    user_id UUID REFERENCES fees.users(id) ON DELETE SET NULL,
    household_id UUID NOT NULL REFERENCES fees.households(id) ON DELETE CASCADE,
    topic VARCHAR NOT NULL CHECK (topic IN ('GENERAL', 'CHILD', 'FEE', 'PARENT_WORK', 'CONTACT')),
    reference_id UUID,
    message TEXT NOT NULL,
    status VARCHAR NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'DONE')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES fees.users(id) ON DELETE SET NULL
);
CREATE INDEX idx_parent_reports_status_created ON fees.parent_reports(status, created_at DESC);
CREATE INDEX idx_parent_reports_household_created ON fees.parent_reports(household_id, created_at DESC);

ALTER TABLE fees.parent_work_entries ADD COLUMN reject_reason TEXT;
ALTER TABLE fees.parent_work_entries DROP CONSTRAINT parent_work_entries_source_check,
    ADD CONSTRAINT parent_work_entries_source_check CHECK (source IN ('MANUAL', 'IMPORT', 'PARENT'));
