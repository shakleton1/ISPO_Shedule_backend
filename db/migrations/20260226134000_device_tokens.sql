-- +goose Up

CREATE TABLE IF NOT EXISTS device_tokens (
  id BIGSERIAL PRIMARY KEY,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  token TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_group_id ON device_tokens(group_id);

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'ispo_set_updated_at') THEN
    EXECUTE 'DROP TRIGGER IF EXISTS trg_device_tokens_set_updated_at ON device_tokens';
    EXECUTE 'CREATE TRIGGER trg_device_tokens_set_updated_at BEFORE UPDATE ON device_tokens FOR EACH ROW EXECUTE FUNCTION ispo_set_updated_at()';
  END IF;
END $$;

-- +goose Down

DROP TABLE IF EXISTS device_tokens;
