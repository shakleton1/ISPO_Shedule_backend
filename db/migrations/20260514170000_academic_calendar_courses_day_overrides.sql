-- +goose Up
-- +goose StatementBegin

ALTER TABLE academic_calendar_weeks
  ADD COLUMN IF NOT EXISTS course_number SMALLINT NOT NULL DEFAULT 1;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'chk_academic_calendar_weeks_course_number'
      AND conrelid = 'academic_calendar_weeks'::regclass
  ) THEN
    ALTER TABLE academic_calendar_weeks
      ADD CONSTRAINT chk_academic_calendar_weeks_course_number
      CHECK (course_number BETWEEN 1 AND 6);
  END IF;
END $$;

ALTER TABLE academic_calendar_weeks
  DROP CONSTRAINT IF EXISTS uidx_academic_calendar_weeknum,
  DROP CONSTRAINT IF EXISTS uidx_academic_calendar_weeks;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_academic_calendar_weeks_calendar_course_week
ON academic_calendar_weeks(calendar_id, course_number, week_number);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_academic_calendar_weeks_calendar_course_start
ON academic_calendar_weeks(calendar_id, course_number, week_start_date);

CREATE INDEX IF NOT EXISTS idx_academic_calendar_weeks_calendar_course_start
ON academic_calendar_weeks(calendar_id, course_number, week_start_date);

CREATE TABLE IF NOT EXISTS academic_calendar_day_overrides (
  id BIGSERIAL PRIMARY KEY,

  calendar_id BIGINT NOT NULL REFERENCES academic_calendars(id) ON DELETE CASCADE,
  course_number SMALLINT NOT NULL CHECK (course_number BETWEEN 1 AND 6),
  week_number SMALLINT NOT NULL CHECK (week_number BETWEEN 1 AND 60),
  day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),

  activity_code VARCHAR(50) NOT NULL,
  activity_name VARCHAR(200) NULL,
  is_teaching BOOLEAN NOT NULL DEFAULT TRUE,

  comment TEXT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT uidx_academic_calendar_day_overrides
    UNIQUE(calendar_id, course_number, week_number, day_of_week)
);

CREATE INDEX IF NOT EXISTS idx_academic_calendar_day_overrides_lookup
ON academic_calendar_day_overrides(calendar_id, course_number, week_number);

DROP TRIGGER IF EXISTS trg_academic_calendar_day_overrides_set_updated_at
ON academic_calendar_day_overrides;

CREATE TRIGGER trg_academic_calendar_day_overrides_set_updated_at
BEFORE UPDATE ON academic_calendar_day_overrides
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS trg_academic_calendar_day_overrides_set_updated_at
ON academic_calendar_day_overrides;

DROP TABLE IF EXISTS academic_calendar_day_overrides;

DROP INDEX IF EXISTS idx_academic_calendar_day_overrides_lookup;
DROP INDEX IF EXISTS idx_academic_calendar_weeks_calendar_course_start;
DROP INDEX IF EXISTS uidx_academic_calendar_weeks_calendar_course_start;
DROP INDEX IF EXISTS uidx_academic_calendar_weeks_calendar_course_week;

DELETE FROM academic_calendar_weeks WHERE course_number <> 1;

ALTER TABLE academic_calendar_weeks
  DROP CONSTRAINT IF EXISTS chk_academic_calendar_weeks_course_number,
  DROP COLUMN IF EXISTS course_number;

ALTER TABLE academic_calendar_weeks
  ADD CONSTRAINT uidx_academic_calendar_weeks UNIQUE (calendar_id, week_start_date),
  ADD CONSTRAINT uidx_academic_calendar_weeknum UNIQUE (calendar_id, week_number);

-- +goose StatementEnd
