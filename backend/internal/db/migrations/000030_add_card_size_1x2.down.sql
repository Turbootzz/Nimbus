-- Postgres can't drop an enum value, so rebuild the type without 1x2
UPDATE widgets SET card_size = '2x2' WHERE card_size = '1x2';
UPDATE services SET card_size = '2x2' WHERE card_size = '1x2';

ALTER TYPE card_size_enum RENAME TO card_size_enum_old;
CREATE TYPE card_size_enum AS ENUM ('1x1', '2x1', '2x2');

ALTER TABLE services ALTER COLUMN card_size DROP DEFAULT;
ALTER TABLE services ALTER COLUMN card_size TYPE card_size_enum USING card_size::text::card_size_enum;
ALTER TABLE services ALTER COLUMN card_size SET DEFAULT '2x1';

ALTER TABLE widgets ALTER COLUMN card_size DROP DEFAULT;
ALTER TABLE widgets ALTER COLUMN card_size TYPE card_size_enum USING card_size::text::card_size_enum;
ALTER TABLE widgets ALTER COLUMN card_size SET DEFAULT '2x1';

DROP TYPE card_size_enum_old;
