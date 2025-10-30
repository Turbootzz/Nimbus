-- Remove open_in_new_tab preference from user_preferences table
ALTER TABLE user_preferences DROP COLUMN IF EXISTS open_in_new_tab;
