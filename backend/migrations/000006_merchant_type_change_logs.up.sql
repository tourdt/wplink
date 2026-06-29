CREATE TABLE IF NOT EXISTS merchant_type_change_logs (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  old_merchant_type varchar(64) NOT NULL,
  new_merchant_type varchar(64) NOT NULL,
  old_verification_status varchar(32) NOT NULL,
  new_verification_status varchar(32) NOT NULL,
  changed_by_user_id bigint REFERENCES users(id),
  changed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_merchant_type_change_logs_merchant
  ON merchant_type_change_logs(merchant_id, changed_at DESC);
