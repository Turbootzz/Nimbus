-- Canvas layout (full-bleed dashboard with a top bar) is opt-in
ALTER TABLE user_preferences
  ADD COLUMN IF NOT EXISTS layout_mode VARCHAR(10) NOT NULL DEFAULT 'classic' CHECK (layout_mode IN ('classic', 'canvas'));
