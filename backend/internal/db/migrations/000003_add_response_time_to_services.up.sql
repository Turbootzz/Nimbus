-- Add response_time column to services table for health check monitoring
ALTER TABLE services ADD COLUMN response_time INTEGER;

-- Create index on response_time for performance queries
CREATE INDEX idx_services_response_time ON services(response_time) WHERE response_time IS NOT NULL;

-- Add comment for clarity
COMMENT ON COLUMN services.response_time IS 'Response time in milliseconds from health checks (NULL if never checked)';
