-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS calendar_day_constraints (
  id BIGSERIAL PRIMARY KEY,
  target_date DATE NOT NULL UNIQUE,
  title VARCHAR(200) NOT NULL,
  reason TEXT NULL,
  constraint_type TEXT NOT NULL DEFAULT 'blocked',
  affects_lessons BOOLEAN NOT NULL DEFAULT TRUE,
  requires_confirmation BOOLEAN NOT NULL DEFAULT FALSE,
  style_preset VARCHAR(30) NOT NULL DEFAULT 'warning',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chk_calendar_day_constraints_type
    CHECK (constraint_type IN ('blocked', 'warning', 'info')),
  CONSTRAINT chk_calendar_day_constraints_style
    CHECK (style_preset IN ('standard', 'warning', 'danger', 'info'))
);

CREATE INDEX IF NOT EXISTS idx_calendar_day_constraints_date
ON calendar_day_constraints(target_date);

CREATE INDEX IF NOT EXISTS idx_calendar_day_constraints_type
ON calendar_day_constraints(constraint_type);

CREATE OR REPLACE FUNCTION set_calendar_day_constraints_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_calendar_day_constraints_updated_at
ON calendar_day_constraints;

CREATE TRIGGER trg_calendar_day_constraints_updated_at
BEFORE UPDATE ON calendar_day_constraints
FOR EACH ROW
EXECUTE FUNCTION set_calendar_day_constraints_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS trg_calendar_day_constraints_updated_at
ON calendar_day_constraints;

DROP FUNCTION IF EXISTS set_calendar_day_constraints_updated_at();

DROP TABLE IF EXISTS calendar_day_constraints;

-- +goose StatementEnd
