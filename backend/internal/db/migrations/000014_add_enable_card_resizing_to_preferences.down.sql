-- Remove enable_card_resizing from user_preferences table
ALTER TABLE user_preferences DROP COLUMN IF EXISTS enable_card_resizing;
