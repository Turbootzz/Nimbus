-- Add card_size to services table
-- card_size: '1x1' (compact), '2x1' (standard/default), '1x2' (tall), '2x2' (large)

-- Create enum type for card_size
DO $$ BEGIN
    CREATE TYPE card_size_enum AS ENUM ('1x1', '2x1', '1x2', '2x2');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Add new column with default '2x1' for backwards compatibility
ALTER TABLE services ADD COLUMN IF NOT EXISTS card_size card_size_enum NOT NULL DEFAULT '2x1';

-- Comment for clarity
COMMENT ON COLUMN services.card_size IS 'Card display size: 1x1 (compact), 2x1 (standard/default), 1x2 (tall), 2x2 (large)';
