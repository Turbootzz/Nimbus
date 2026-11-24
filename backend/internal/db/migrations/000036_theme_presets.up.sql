-- Theme preset, font, icon source and custom CSS per user. The defaults are
-- what everyone had before, so nothing changes until a user picks something.
ALTER TABLE user_preferences
  ADD COLUMN IF NOT EXISTS theme_preset VARCHAR(30) NOT NULL DEFAULT 'default',
  ADD COLUMN IF NOT EXISTS theme_font VARCHAR(20) NOT NULL DEFAULT 'default',
  ADD COLUMN IF NOT EXISTS kpi_mono BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS icon_source VARCHAR(30) NOT NULL DEFAULT 'dashboard-icons-svg',
  -- Checked by utils.ValidateCustomCSS before it is stored
  ADD COLUMN IF NOT EXISTS custom_css TEXT NOT NULL DEFAULT '';
