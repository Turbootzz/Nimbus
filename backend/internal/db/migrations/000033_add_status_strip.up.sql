-- Status strip: KPI chips at the top of the dashboard
ALTER TABLE user_preferences
  ADD COLUMN IF NOT EXISTS status_strip JSONB NOT NULL DEFAULT '{"enabled": false, "chips": []}';
