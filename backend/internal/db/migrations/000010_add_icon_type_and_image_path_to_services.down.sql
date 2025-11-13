-- Rollback: Remove icon_type and icon_image_path from services table

ALTER TABLE services
DROP COLUMN IF EXISTS icon_image_path,
DROP COLUMN IF EXISTS icon_type;

DROP TYPE IF EXISTS icon_type_enum;
