-- Create service_status_logs table for historical uptime monitoring
CREATE TABLE IF NOT EXISTS service_status_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    response_time INTEGER,
    error_message TEXT,
    checked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Indexes for efficient queries
    CONSTRAINT chk_status CHECK (status IN ('online', 'offline', 'unknown'))
);

-- Index on service_id for fast lookups per service
CREATE INDEX IF NOT EXISTS idx_status_logs_service_id ON service_status_logs(service_id);

-- Composite index for time-range queries per service
CREATE INDEX IF NOT EXISTS idx_status_logs_service_time ON service_status_logs(service_id, checked_at DESC);

-- Index on checked_at for cleanup operations
CREATE INDEX IF NOT EXISTS idx_status_logs_checked_at ON service_status_logs(checked_at);

-- Add comments for clarity
COMMENT ON TABLE service_status_logs IS 'Historical log of service health check results';
COMMENT ON COLUMN service_status_logs.response_time IS 'Response time in milliseconds (NULL if check failed)';
COMMENT ON COLUMN service_status_logs.error_message IS 'Error details if check failed (NULL if successful)';
COMMENT ON COLUMN service_status_logs.checked_at IS 'Timestamp when the health check was performed';
