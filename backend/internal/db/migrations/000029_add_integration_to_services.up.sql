-- Link a service to an integration, so its tile shows the app's KPIs
-- (e.g. Sonarr's queue). Deleting the integration unlinks the service.
ALTER TABLE services ADD COLUMN IF NOT EXISTS integration_id UUID REFERENCES integrations(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_services_integration_id ON services(integration_id);
