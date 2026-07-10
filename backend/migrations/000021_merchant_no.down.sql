DROP INDEX IF EXISTS uniq_merchants_merchant_no;

ALTER TABLE merchants
  DROP CONSTRAINT IF EXISTS chk_merchants_merchant_no_format,
  ALTER COLUMN merchant_no DROP DEFAULT,
  DROP COLUMN IF EXISTS merchant_no;

DROP FUNCTION IF EXISTS next_merchant_no();
DROP FUNCTION IF EXISTS merchant_no_luhn_check_digit(text);
DROP SEQUENCE IF EXISTS merchant_no_seq;
