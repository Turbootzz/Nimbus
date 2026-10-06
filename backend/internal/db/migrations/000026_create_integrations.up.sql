-- Integrations: user-scoped connections to external apps (Sonarr, Proxmox, ...)
-- credentials_enc holds AES-256-GCM encrypted JSON (see utils/crypto.go)
CREATE TABLE IF NOT EXISTS integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    base_url TEXT NOT NULL,
    auth_type VARCHAR(20) NOT NULL DEFAULT 'none'
        CHECK (auth_type IN ('none', 'api_key', 'basic', 'token')),
    credentials_enc BYTEA,
    verify_tls BOOLEAN NOT NULL DEFAULT TRUE,
    options JSONB NOT NULL DEFAULT '{}'::jsonb,
    refresh_seconds INT NOT NULL DEFAULT 60 CHECK (refresh_seconds >= 10),
    last_test_at TIMESTAMP WITH TIME ZONE,
    last_test_ok BOOLEAN,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_integrations_user_id ON integrations(user_id);
