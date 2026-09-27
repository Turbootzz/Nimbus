-- Latest data per polled widget or integration, so a restart doesn't blank
-- the dashboard. One row per source; payload keeps the last good result and
-- error the latest failure. Rows of deleted sources are pruned daily.
CREATE TABLE IF NOT EXISTS widget_snapshots (
    source_kind VARCHAR(20) NOT NULL CHECK (source_kind IN ('widget', 'integration')),
    source_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payload JSONB,
    error TEXT,
    fetched_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (source_kind, source_id)
);

CREATE INDEX IF NOT EXISTS idx_widget_snapshots_user_id ON widget_snapshots(user_id);
