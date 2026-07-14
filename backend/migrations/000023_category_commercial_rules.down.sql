DROP TABLE IF EXISTS resource_contact_unlocks;
DROP TABLE IF EXISTS resource_contact_unlock_orders;

ALTER TABLE resource_type_configs
  DROP COLUMN IF EXISTS commercial_rules;
