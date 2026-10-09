ALTER TABLE merchants
  ADD COLUMN business_profile JSONB NOT NULL DEFAULT '{"schemaVersion":1,"entityKind":"","primaryRole":"","roles":[],"serviceArea":"","processingMethods":[],"propertyTypes":[],"serviceItems":[]}'::jsonb,
  ADD COLUMN map_visible BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE merchants
  ALTER COLUMN merchant_type SET DEFAULT '',
  ALTER COLUMN profile_status SET DEFAULT 'incomplete';
