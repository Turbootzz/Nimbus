-- Wallpaper blur and dim, and glass cards (card opacity and blur)
ALTER TABLE user_preferences
  ADD COLUMN IF NOT EXISTS wallpaper_blur INTEGER NOT NULL DEFAULT 0 CHECK (wallpaper_blur BETWEEN 0 AND 20),
  ADD COLUMN IF NOT EXISTS wallpaper_dim INTEGER NOT NULL DEFAULT 0 CHECK (wallpaper_dim BETWEEN 0 AND 80),
  ADD COLUMN IF NOT EXISTS card_opacity INTEGER NOT NULL DEFAULT 100 CHECK (card_opacity BETWEEN 0 AND 100),
  ADD COLUMN IF NOT EXISTS card_blur INTEGER NOT NULL DEFAULT 0 CHECK (card_blur BETWEEN 0 AND 40);
