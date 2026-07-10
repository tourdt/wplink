CREATE SEQUENCE IF NOT EXISTS merchant_no_seq AS bigint
  START WITH 100000
  MINVALUE 100000
  MAXVALUE 999999
  NO CYCLE;

CREATE OR REPLACE FUNCTION merchant_no_luhn_check_digit(payload text)
RETURNS integer
LANGUAGE plpgsql
IMMUTABLE
STRICT
AS $$
DECLARE
  sum_value integer := 0;
  digit integer;
  doubled integer;
  pos integer;
BEGIN
  IF payload !~ '^[0-9]+$' THEN
    RAISE EXCEPTION 'merchant no payload must be digits';
  END IF;

  FOR pos IN 1..length(payload) LOOP
    digit := substring(payload from length(payload) - pos + 1 for 1)::integer;
    IF pos % 2 = 1 THEN
      doubled := digit * 2;
      IF doubled > 9 THEN
        doubled := doubled - 9;
      END IF;
      sum_value := sum_value + doubled;
    ELSE
      sum_value := sum_value + digit;
    END IF;
  END LOOP;

  RETURN (10 - (sum_value % 10)) % 10;
END;
$$;

CREATE OR REPLACE FUNCTION next_merchant_no()
RETURNS varchar(9)
LANGUAGE plpgsql
VOLATILE
AS $$
DECLARE
  payload text;
  seq_text text;
BEGIN
  seq_text := lpad(nextval('merchant_no_seq')::text, 6, '0');
  IF length(seq_text) > 6 THEN
    RAISE EXCEPTION 'merchant no sequence exceeded 6 digits';
  END IF;

  payload := right(to_char(clock_timestamp(), 'YYYY'), 1) || seq_text;
  RETURN 'M' || payload || merchant_no_luhn_check_digit(payload)::text;
END;
$$;

ALTER TABLE merchants
  ADD COLUMN IF NOT EXISTS merchant_no varchar(9);

UPDATE merchants
SET merchant_no = next_merchant_no()
WHERE merchant_no IS NULL OR merchant_no = '';

ALTER TABLE merchants
  ALTER COLUMN merchant_no SET DEFAULT next_merchant_no(),
  ALTER COLUMN merchant_no SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'chk_merchants_merchant_no_format'
  ) THEN
    ALTER TABLE merchants
      ADD CONSTRAINT chk_merchants_merchant_no_format
      CHECK (merchant_no ~ '^M[0-9]{8}$');
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_merchants_merchant_no
  ON merchants(merchant_no);
