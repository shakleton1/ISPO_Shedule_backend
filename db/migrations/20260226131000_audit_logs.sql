-- +goose Up

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  actor_type TEXT NOT NULL,
  actor_id BIGINT NULL,
  actor_login VARCHAR(50) NULL,
  actor_role TEXT NULL,

  method VARCHAR(10) NOT NULL,
  path VARCHAR(200) NOT NULL,

  action TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,

  payload JSONB NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id ON audit_logs(actor_id);

-- +goose Down

DROP TABLE IF EXISTS audit_logs;
