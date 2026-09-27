-- Widgets: dashboard tiles that are not links (clock, notes, app data, ...).
-- They share the per-user position space with services, so one grid can
-- hold both. config is validated per type in internal/widgets.
CREATE TABLE IF NOT EXISTS widgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(100) NOT NULL DEFAULT '',
    group_id UUID REFERENCES groups(id) ON DELETE SET NULL,
    integration_id UUID REFERENCES integrations(id) ON DELETE SET NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    card_size card_size_enum NOT NULL DEFAULT '2x1',
    position INTEGER NOT NULL DEFAULT 0,
    refresh_seconds INT NOT NULL DEFAULT 300 CHECK (refresh_seconds >= 10),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_widgets_user_position ON widgets(user_id, position);
CREATE INDEX IF NOT EXISTS idx_widgets_group_id ON widgets(group_id);
CREATE INDEX IF NOT EXISTS idx_widgets_integration_id ON widgets(integration_id);
