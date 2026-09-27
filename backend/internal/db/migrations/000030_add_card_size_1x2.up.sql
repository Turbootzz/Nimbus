-- 1x2: one column, two rows. Widgets only (lists, notes, embedded pages);
-- the API keeps services on 1x1, 2x1 and 2x2.
ALTER TYPE card_size_enum ADD VALUE IF NOT EXISTS '1x2';

-- Embeds now fill their card and only come in tall sizes
UPDATE widgets SET card_size = '2x2' WHERE type = 'iframe' AND card_size IN ('1x1', '2x1');
