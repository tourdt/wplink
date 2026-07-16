CREATE TABLE IF NOT EXISTS map_object_bind_request (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  object_id bigint NOT NULL REFERENCES map_object(id),
  scene_code varchar(64) NOT NULL,
  applicant_user_id bigint REFERENCES users(id),
  evidence_images jsonb NOT NULL DEFAULT '[]'::jsonb,
  note text NOT NULL DEFAULT '',
  status varchar(20) NOT NULL DEFAULT 'pending',
  review_note text NOT NULL DEFAULT '',
  reviewed_by bigint REFERENCES admin_operators(id),
  reviewed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_map_bind_request_pending_object
  ON map_object_bind_request(merchant_id, object_id)
  WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_map_bind_request_status_created
  ON map_object_bind_request(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_map_bind_request_merchant_created
  ON map_object_bind_request(merchant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_map_bind_request_object
  ON map_object_bind_request(object_id);
