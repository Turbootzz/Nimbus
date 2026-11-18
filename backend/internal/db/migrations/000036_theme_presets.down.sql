ALTER TABLE user_preferences
  DROP COLUMN IF EXISTS custom_css,
  DROP COLUMN IF EXISTS icon_source,
  DROP COLUMN IF EXISTS kpi_mono,
  DROP COLUMN IF EXISTS theme_font,
  DROP COLUMN IF EXISTS theme_preset;
