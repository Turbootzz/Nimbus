-- Remove category_id from services
DROP INDEX IF EXISTS idx_services_category_id;
ALTER TABLE services DROP COLUMN IF EXISTS category_id;

-- Drop categories table and indexes
DROP INDEX IF EXISTS idx_categories_position;
DROP INDEX IF EXISTS idx_categories_user_id;
DROP TABLE IF EXISTS categories;
