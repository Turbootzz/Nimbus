-- Rollback: Remove card_size from services table
ALTER TABLE services DROP COLUMN IF EXISTS card_size;
DROP TYPE IF EXISTS card_size_enum;
