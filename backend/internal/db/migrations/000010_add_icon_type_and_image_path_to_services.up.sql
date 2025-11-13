-- Add icon_type and icon_image_path to services table
-- icon_type: 'emoji' (default), 'image_upload', or 'image_url'
-- icon_image_path: stores file path for uploads or URL for external images

-- Create enum type for icon_type (skip if already exists)
DO $$ BEGIN
    CREATE TYPE icon_type_enum AS ENUM ('emoji', 'image_upload', 'image_url');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Add new columns (skip if already exist)
DO $$ BEGIN
    ALTER TABLE services
    ADD COLUMN icon_type icon_type_enum NOT NULL DEFAULT 'emoji',
    ADD COLUMN icon_image_path TEXT NOT NULL DEFAULT '';
EXCEPTION
    WHEN duplicate_column THEN null;
END $$;

-- Ensure icon_image_path is NOT NULL and has default (in case it was added as nullable before)
ALTER TABLE services ALTER COLUMN icon_image_path SET DEFAULT '';
UPDATE services SET icon_image_path = '' WHERE icon_image_path IS NULL;
ALTER TABLE services ALTER COLUMN icon_image_path SET NOT NULL;

-- Add comment for clarity
COMMENT ON COLUMN services.icon_type IS 'Type of icon: emoji (text), image_upload (uploaded file), or image_url (external URL)';
COMMENT ON COLUMN services.icon_image_path IS 'File path for uploaded images or URL for external images';
