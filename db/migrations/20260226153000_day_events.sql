-- +goose Up

CREATE TABLE IF NOT EXISTS schedule_day_events (
  id BIGSERIAL PRIMARY KEY,
  target_date DATE NOT NULL,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL CHECK (event_type IN ('INFO','EXAM','PRACTICE','ATTESTATION','GIA','OTHER')),
  title VARCHAR(200) NOT NULL,
  details TEXT NULL,
  location_id INT NULL REFERENCES locations(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_schedule_day_events_lookup ON schedule_day_events(group_id, target_date);

DROP TRIGGER IF EXISTS trg_schedule_day_events_set_updated_at ON schedule_day_events;
CREATE TRIGGER trg_schedule_day_events_set_updated_at
BEFORE UPDATE ON schedule_day_events
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_schedule_day_events_set_updated_at ON schedule_day_events;
DROP TABLE IF EXISTS schedule_day_events;
