-- +goose Up

ALTER TABLE audit_logs
  ADD COLUMN IF NOT EXISTS request_id TEXT NULL,
  ADD COLUMN IF NOT EXISTS ip TEXT NULL,
  ADD COLUMN IF NOT EXISTS user_agent TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs(request_id);

-- +goose Down

DROP INDEX IF EXISTS idx_audit_logs_request_id;

ALTER TABLE audit_logs
  DROP COLUMN IF EXISTS request_id,
  DROP COLUMN IF EXISTS ip,
  DROP COLUMN IF EXISTS user_agent;
