-- +goose Up

CREATE TABLE IF NOT EXISTS location_week_availability (
  id BIGSERIAL PRIMARY KEY,
  week_start_date DATE NOT NULL,
  location_id INT NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  is_available BOOLEAN NOT NULL DEFAULT TRUE,
  comment TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_location_week_availability
  ON location_week_availability(week_start_date, location_id);

CREATE INDEX IF NOT EXISTS idx_location_week_availability_location
  ON location_week_availability(location_id, week_start_date);

DROP TRIGGER IF EXISTS trg_location_week_availability_set_updated_at ON location_week_availability;
CREATE TRIGGER trg_location_week_availability_set_updated_at
BEFORE UPDATE ON location_week_availability
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_location_week_availability_set_updated_at ON location_week_availability;
DROP INDEX IF EXISTS idx_location_week_availability_location;
DROP INDEX IF EXISTS uidx_location_week_availability;
DROP TABLE IF EXISTS location_week_availability;
