CREATE TABLE IF NOT EXISTS verification_payment_orders (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  verification_id bigint NOT NULL REFERENCES verifications(id),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  user_id bigint NOT NULL REFERENCES users(id),
  out_trade_no varchar(64) UNIQUE NOT NULL,
  transaction_id varchar(128),
  amount_total integer NOT NULL,
  currency varchar(8) NOT NULL DEFAULT 'CNY',
  status varchar(32) NOT NULL DEFAULT 'pending',
  prepay_id varchar(128),
  notify_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  paid_at timestamptz,
  closed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_verification_payment_order_verification
  ON verification_payment_orders(verification_id);
CREATE INDEX IF NOT EXISTS idx_verification_payment_orders_merchant_status
  ON verification_payment_orders(merchant_id, status);
CREATE INDEX IF NOT EXISTS idx_verification_payment_orders_out_trade_no
  ON verification_payment_orders(out_trade_no);
