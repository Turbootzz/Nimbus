-- Docker discovery: the containers a user was offered and dealt with. A row
-- means "do not offer this container again": service_id is the service it
-- became, NULL when the user ignored it or deleted that service.
-- discovery_key is "<integration id>/<container name>".
CREATE TABLE IF NOT EXISTS discovered_containers (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    discovery_key TEXT NOT NULL,
    service_id UUID REFERENCES services(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, discovery_key)
);

CREATE INDEX IF NOT EXISTS idx_discovered_containers_service_id ON discovered_containers(service_id);

-- Auto-add containers labelled nimbus.enable=true; off unless an admin turns it on
INSERT INTO system_settings (key, value) VALUES
    ('discovery_auto_add', 'false')
ON CONFLICT (key) DO NOTHING;
