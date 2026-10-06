ALTER TABLE user_preferences
  DROP COLUMN IF EXISTS wallpaper_blur,
  DROP COLUMN IF EXISTS wallpaper_dim,
  DROP COLUMN IF EXISTS card_opacity,
  DROP COLUMN IF EXISTS card_blur;
