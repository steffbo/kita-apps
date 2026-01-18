-- V1__Initial_Schema.sql
-- Kita Knirpsenstadt Database Schema

-- =============================================
-- EMPLOYEES
-- =============================================
CREATE TABLE employees (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'EMPLOYEE',
    weekly_hours DECIMAL(4,2) NOT NULL CHECK (weekly_hours >= 0 AND weekly_hours <= 40),
    vacation_days_per_year INTEGER NOT NULL DEFAULT 30,
    remaining_vacation_days DECIMAL(5,2) NOT NULL DEFAULT 30,
    overtime_balance DECIMAL(6,2) NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_role CHECK (role IN ('ADMIN', 'EMPLOYEE'))
);

CREATE INDEX idx_employees_email ON employees(email);
CREATE INDEX idx_employees_active ON employees(active);

-- =============================================
-- PASSWORD RESET TOKENS
-- =============================================
CREATE TABLE password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_password_reset_token ON password_reset_tokens(token);

-- =============================================
-- REFRESH TOKENS
-- =============================================
CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    token VARCHAR(500) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_refresh_token ON refresh_tokens(token);
CREATE INDEX idx_refresh_token_employee ON refresh_tokens(employee_id);

-- =============================================
-- GROUPS
-- =============================================
CREATE TABLE groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    color VARCHAR(7) DEFAULT '#3B82F6',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- =============================================
-- GROUP ASSIGNMENTS
-- =============================================
CREATE TABLE group_assignments (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    assignment_type VARCHAR(20) NOT NULL DEFAULT 'PERMANENT',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_assignment_type CHECK (assignment_type IN ('PERMANENT', 'SPRINGER')),
    CONSTRAINT uq_employee_group UNIQUE (employee_id, group_id)
);

CREATE INDEX idx_group_assignments_employee ON group_assignments(employee_id);
CREATE INDEX idx_group_assignments_group ON group_assignments(group_id);

-- =============================================
-- SPECIAL DAYS (Holidays, Closures, Events)
-- =============================================
CREATE TABLE special_days (
    id BIGSERIAL PRIMARY KEY,
    date DATE NOT NULL,
    name VARCHAR(255) NOT NULL,
    day_type VARCHAR(20) NOT NULL,
    affects_all BOOLEAN NOT NULL DEFAULT TRUE,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_day_type CHECK (day_type IN ('HOLIDAY', 'CLOSURE', 'TEAM_DAY', 'EVENT')),
    CONSTRAINT uq_special_day_date_type UNIQUE (date, day_type, name)
);

CREATE INDEX idx_special_days_date ON special_days(date);
CREATE INDEX idx_special_days_year ON special_days(EXTRACT(YEAR FROM date));

-- =============================================
-- SCHEDULE ENTRIES
-- =============================================
CREATE TABLE schedule_entries (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    start_time TIME,
    end_time TIME,
    break_minutes INTEGER DEFAULT 0,
    group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    entry_type VARCHAR(20) NOT NULL DEFAULT 'WORK',
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_entry_type CHECK (entry_type IN ('WORK', 'VACATION', 'SICK', 'SPECIAL_LEAVE', 'TRAINING', 'EVENT')),
    CONSTRAINT chk_times CHECK (end_time IS NULL OR start_time IS NULL OR end_time > start_time)
);

CREATE INDEX idx_schedule_entries_employee ON schedule_entries(employee_id);
CREATE INDEX idx_schedule_entries_date ON schedule_entries(date);
CREATE INDEX idx_schedule_entries_group ON schedule_entries(group_id);
CREATE INDEX idx_schedule_entries_date_range ON schedule_entries(date, employee_id);

-- =============================================
-- TIME ENTRIES
-- =============================================
CREATE TABLE time_entries (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    clock_in TIMESTAMP WITH TIME ZONE NOT NULL,
    clock_out TIMESTAMP WITH TIME ZONE,
    break_minutes INTEGER DEFAULT 0,
    entry_type VARCHAR(20) NOT NULL DEFAULT 'WORK',
    worked_minutes INTEGER GENERATED ALWAYS AS (
        CASE 
            WHEN clock_out IS NOT NULL 
            THEN EXTRACT(EPOCH FROM (clock_out - clock_in)) / 60 - COALESCE(break_minutes, 0)
            ELSE NULL 
        END
    ) STORED,
    notes TEXT,
    edited_by BIGINT REFERENCES employees(id) ON DELETE SET NULL,
    edited_at TIMESTAMP WITH TIME ZONE,
    edit_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_time_entry_type CHECK (entry_type IN ('WORK', 'VACATION', 'SICK', 'SPECIAL_LEAVE', 'TRAINING', 'EVENT')),
    CONSTRAINT chk_clock_times CHECK (clock_out IS NULL OR clock_out > clock_in)
);

CREATE INDEX idx_time_entries_employee ON time_entries(employee_id);
CREATE INDEX idx_time_entries_date ON time_entries(date);
CREATE INDEX idx_time_entries_date_range ON time_entries(date, employee_id);

-- =============================================
-- AUDIT LOG (for tracking changes)
-- =============================================
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,
    entity_id BIGINT NOT NULL,
    action VARCHAR(20) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    performed_by BIGINT REFERENCES employees(id) ON DELETE SET NULL,
    performed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45),
    CONSTRAINT chk_action CHECK (action IN ('CREATE', 'UPDATE', 'DELETE'))
);

CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_performed_by ON audit_log(performed_by);
CREATE INDEX idx_audit_log_performed_at ON audit_log(performed_at);

-- =============================================
-- FUNCTIONS
-- =============================================

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply to tables with updated_at
CREATE TRIGGER update_employees_updated_at
    BEFORE UPDATE ON employees
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_groups_updated_at
    BEFORE UPDATE ON groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_schedule_entries_updated_at
    BEFORE UPDATE ON schedule_entries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
