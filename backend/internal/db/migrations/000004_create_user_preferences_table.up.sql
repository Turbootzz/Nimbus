-- Create user_preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    theme_mode VARCHAR(10) NOT NULL DEFAULT 'light' CHECK (theme_mode IN ('light', 'dark')),
    theme_background TEXT,
    theme_accent_color VARCHAR(7), -- Hex color code like #3B82F6
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on user_id for faster lookups
CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);
