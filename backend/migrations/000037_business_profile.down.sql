ALTER TABLE merchants
  DROP COLUMN map_visible,
  DROP COLUMN business_profile;

ALTER TABLE merchants
  ALTER COLUMN merchant_type DROP DEFAULT,
  ALTER COLUMN profile_status SET DEFAULT 'completed';
