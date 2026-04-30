CREATE TABLE schedule_entry_segments (
    id BIGSERIAL PRIMARY KEY,
    schedule_entry_id BIGINT NOT NULL REFERENCES schedule_entries(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    notes TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_schedule_entry_segment_times CHECK (end_time > start_time)
);

CREATE INDEX idx_schedule_entry_segments_entry ON schedule_entry_segments(schedule_entry_id);
CREATE INDEX idx_schedule_entry_segments_group ON schedule_entry_segments(group_id);

CREATE TRIGGER update_schedule_entry_segments_updated_at
    BEFORE UPDATE ON schedule_entry_segments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE schedule_requests (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    start_time TIME,
    end_time TIME,
    request_type VARCHAR(20) NOT NULL,
    text TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_schedule_request_type CHECK (request_type IN ('WISH', 'APPOINTMENT')),
    CONSTRAINT chk_schedule_request_status CHECK (status IN ('OPEN', 'DONE')),
    CONSTRAINT chk_schedule_request_times CHECK (end_time IS NULL OR start_time IS NULL OR end_time > start_time),
    CONSTRAINT chk_schedule_request_text_not_blank CHECK (length(trim(text)) > 0)
);

CREATE INDEX idx_schedule_requests_employee_date ON schedule_requests(employee_id, date);
CREATE INDEX idx_schedule_requests_date ON schedule_requests(date);
CREATE INDEX idx_schedule_requests_status ON schedule_requests(status);

CREATE TRIGGER update_schedule_requests_updated_at
    BEFORE UPDATE ON schedule_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
