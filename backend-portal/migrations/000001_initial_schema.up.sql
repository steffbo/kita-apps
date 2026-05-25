CREATE SCHEMA IF NOT EXISTS portal;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE portal.user_role AS ENUM ('ADMIN', 'BEITRAG', 'VORSTAND', 'LEITUNG', 'TEAM', 'PARENT');
CREATE TYPE portal.user_status AS ENUM ('INVITED', 'ACTIVE', 'DISABLED');
CREATE TYPE portal.parent_work_entry_status AS ENUM ('SUBMITTED', 'APPROVED', 'REJECTED', 'VOIDED');
CREATE TYPE portal.sync_run_status AS ENUM ('RUNNING', 'COMPLETED', 'FAILED');
CREATE TYPE portal.quarantine_status AS ENUM ('OPEN', 'RESOLVED', 'IGNORED');
CREATE TYPE portal.email_status AS ENUM ('PENDING', 'SENDING', 'SENT', 'FAILED');
CREATE TYPE portal.email_type AS ENUM ('INVITATION', 'PASSWORD_RESET', 'PARENT_WORK_REMINDER', 'PARENT_WORK_REVIEW');

CREATE TABLE portal.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    status portal.user_status NOT NULL DEFAULT 'INVITED',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_portal_users_email_lower ON portal.users (LOWER(email));
CREATE INDEX idx_portal_users_status ON portal.users(status);

CREATE TABLE portal.user_roles (
    user_id UUID NOT NULL REFERENCES portal.users(id) ON DELETE CASCADE,
    role portal.user_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role)
);

CREATE TABLE portal.invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    target_roles portal.user_role[] NOT NULL DEFAULT ARRAY['PARENT']::portal.user_role[],
    invited_by UUID REFERENCES portal.users(id),
    accepted_by UUID REFERENCES portal.users(id),
    accepted_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES portal.users(id),
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT invitation_not_accepted_and_revoked CHECK (accepted_at IS NULL OR revoked_at IS NULL)
);

CREATE INDEX idx_portal_invitations_email ON portal.invitations(LOWER(email));
CREATE INDEX idx_portal_invitations_open ON portal.invitations(created_at) WHERE accepted_at IS NULL AND revoked_at IS NULL;

CREATE TABLE portal.password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES portal.users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portal_password_reset_user ON portal.password_reset_tokens(user_id);
CREATE INDEX idx_portal_password_reset_expires ON portal.password_reset_tokens(expires_at);

CREATE TABLE portal.refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES portal.users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portal_refresh_tokens_user ON portal.refresh_tokens(user_id);
CREATE INDEX idx_portal_refresh_tokens_expires ON portal.refresh_tokens(expires_at);

CREATE TABLE portal.synced_households (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system VARCHAR(50) NOT NULL,
    source_id VARCHAR(100) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_checksum VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_system, source_id)
);

CREATE TABLE portal.synced_parents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system VARCHAR(50) NOT NULL,
    source_id VARCHAR(100) NOT NULL,
    household_id UUID REFERENCES portal.synced_households(id),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_checksum VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_system, source_id)
);

CREATE INDEX idx_portal_synced_parents_household ON portal.synced_parents(household_id);
CREATE INDEX idx_portal_synced_parents_email ON portal.synced_parents(LOWER(email));

CREATE TABLE portal.synced_children (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system VARCHAR(50) NOT NULL,
    source_id VARCHAR(100) NOT NULL,
    household_id UUID REFERENCES portal.synced_households(id),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    birth_date DATE NOT NULL,
    entry_date DATE NOT NULL,
    exit_date DATE,
    group_name VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_checksum VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_system, source_id)
);

CREATE INDEX idx_portal_synced_children_household ON portal.synced_children(household_id);
CREATE INDEX idx_portal_synced_children_active ON portal.synced_children(is_active);

CREATE TABLE portal.synced_child_parents (
    child_id UUID NOT NULL REFERENCES portal.synced_children(id) ON DELETE CASCADE,
    parent_id UUID NOT NULL REFERENCES portal.synced_parents(id) ON DELETE CASCADE,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (child_id, parent_id)
);

CREATE TABLE portal.user_households (
    user_id UUID NOT NULL REFERENCES portal.users(id) ON DELETE CASCADE,
    household_id UUID NOT NULL REFERENCES portal.synced_households(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, household_id)
);

CREATE TABLE portal.sync_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system VARCHAR(50) NOT NULL,
    status portal.sync_run_status NOT NULL DEFAULT 'RUNNING',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    imported_households INT NOT NULL DEFAULT 0,
    imported_parents INT NOT NULL DEFAULT 0,
    imported_children INT NOT NULL DEFAULT 0,
    quarantine_count INT NOT NULL DEFAULT 0,
    error_message TEXT
);

CREATE INDEX idx_portal_sync_runs_started ON portal.sync_runs(started_at DESC);

CREATE TABLE portal.sync_quarantine_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sync_run_id UUID REFERENCES portal.sync_runs(id) ON DELETE SET NULL,
    source_system VARCHAR(50) NOT NULL,
    source_entity VARCHAR(50) NOT NULL,
    source_id VARCHAR(100),
    reason TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::JSONB,
    status portal.quarantine_status NOT NULL DEFAULT 'OPEN',
    resolved_by UUID REFERENCES portal.users(id),
    resolved_at TIMESTAMPTZ,
    resolution_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portal_sync_quarantine_open ON portal.sync_quarantine_items(created_at DESC) WHERE status = 'OPEN';

CREATE TABLE portal.parent_work_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id UUID NOT NULL REFERENCES portal.synced_children(id) ON DELETE CASCADE,
    kita_year_start_year INT NOT NULL,
    required_hours NUMERIC(5,2) NOT NULL CHECK (required_hours >= 0),
    override_hours NUMERIC(5,2) CHECK (override_hours >= 0),
    effective_required_hours NUMERIC(5,2) GENERATED ALWAYS AS (COALESCE(override_hours, required_hours)) STORED,
    override_reason TEXT,
    override_by UUID REFERENCES portal.users(id),
    overridden_at TIMESTAMPTZ,
    exemption_reason TEXT,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (child_id, kita_year_start_year)
);

CREATE INDEX idx_portal_parent_work_requirements_year ON portal.parent_work_requirements(kita_year_start_year);

CREATE TABLE portal.parent_work_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    child_id UUID NOT NULL REFERENCES portal.synced_children(id) ON DELETE CASCADE,
    submitted_by UUID NOT NULL REFERENCES portal.users(id),
    work_date DATE NOT NULL,
    duration_hours NUMERIC(5,2) NOT NULL CHECK (duration_hours > 0),
    approved_duration_hours NUMERIC(5,2) CHECK (approved_duration_hours > 0),
    category VARCHAR(80) NOT NULL,
    description TEXT NOT NULL,
    status portal.parent_work_entry_status NOT NULL DEFAULT 'SUBMITTED',
    review_note TEXT,
    reviewed_by UUID REFERENCES portal.users(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT rejected_entries_need_review_note CHECK (status <> 'REJECTED' OR LENGTH(TRIM(COALESCE(review_note, ''))) > 0),
    CONSTRAINT reviewed_entries_need_reviewer CHECK (status NOT IN ('APPROVED', 'REJECTED', 'VOIDED') OR reviewed_by IS NOT NULL)
);

CREATE INDEX idx_portal_parent_work_entries_child ON portal.parent_work_entries(child_id);
CREATE INDEX idx_portal_parent_work_entries_submitter ON portal.parent_work_entries(submitted_by);
CREATE INDEX idx_portal_parent_work_entries_status ON portal.parent_work_entries(status);
CREATE INDEX idx_portal_parent_work_entries_work_date ON portal.parent_work_entries(work_date DESC);

CREATE TABLE portal.email_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email_type portal.email_type NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::JSONB,
    status portal.email_status NOT NULL DEFAULT 'PENDING',
    attempts INT NOT NULL DEFAULT 0,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portal_email_outbox_pending ON portal.email_outbox(scheduled_at) WHERE status = 'PENDING';
CREATE INDEX idx_portal_email_outbox_recipient ON portal.email_outbox(LOWER(recipient_email));

CREATE TABLE portal.audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES portal.users(id),
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(80) NOT NULL,
    entity_id UUID,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portal_audit_events_actor ON portal.audit_events(actor_user_id);
CREATE INDEX idx_portal_audit_events_entity ON portal.audit_events(entity_type, entity_id);
CREATE INDEX idx_portal_audit_events_created ON portal.audit_events(created_at DESC);

CREATE OR REPLACE FUNCTION portal.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_portal_users_updated_at
    BEFORE UPDATE ON portal.users
    FOR EACH ROW EXECUTE FUNCTION portal.update_updated_at_column();

CREATE TRIGGER update_portal_synced_households_updated_at
    BEFORE UPDATE ON portal.synced_households
    FOR EACH ROW EXECUTE FUNCTION portal.update_updated_at_column();

CREATE TRIGGER update_portal_synced_parents_updated_at
    BEFORE UPDATE ON portal.synced_parents
    FOR EACH ROW EXECUTE FUNCTION portal.update_updated_at_column();

CREATE TRIGGER update_portal_synced_children_updated_at
    BEFORE UPDATE ON portal.synced_children
    FOR EACH ROW EXECUTE FUNCTION portal.update_updated_at_column();

CREATE TRIGGER update_portal_parent_work_requirements_updated_at
    BEFORE UPDATE ON portal.parent_work_requirements
    FOR EACH ROW EXECUTE FUNCTION portal.update_updated_at_column();

CREATE TRIGGER update_portal_parent_work_entries_updated_at
    BEFORE UPDATE ON portal.parent_work_entries
    FOR EACH ROW EXECUTE FUNCTION portal.update_updated_at_column();

CREATE TRIGGER update_portal_email_outbox_updated_at
    BEFORE UPDATE ON portal.email_outbox
    FOR EACH ROW EXECUTE FUNCTION portal.update_updated_at_column();
