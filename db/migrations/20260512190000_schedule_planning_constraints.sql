-- +goose Up

-- Locations stay in one DB, but carry enough metadata to split/filter by campus.
ALTER TABLE locations
  ADD COLUMN IF NOT EXISTS campus VARCHAR(50) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS location_kind TEXT NOT NULL DEFAULT 'classroom',
  ADD COLUMN IF NOT EXISTS capacity SMALLINT NULL CHECK (capacity IS NULL OR capacity > 0);

ALTER TABLE locations
  DROP CONSTRAINT IF EXISTS chk_locations_kind;
ALTER TABLE locations
  ADD CONSTRAINT chk_locations_kind
  CHECK (location_kind IN ('classroom', 'computer_lab', 'gym', 'pool', 'virtual', 'other'));

CREATE INDEX IF NOT EXISTS idx_locations_campus ON locations(campus);
CREATE INDEX IF NOT EXISTS idx_locations_kind ON locations(location_kind);

-- Dictionary for study calendar activity cells.
CREATE TABLE IF NOT EXISTS study_activities (
  id SERIAL PRIMARY KEY,
  code VARCHAR(50) NOT NULL,
  name VARCHAR(200) NOT NULL,
  activity_kind TEXT NOT NULL DEFAULT 'OTHER',
  allows_lessons BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_study_activities_code UNIQUE (code),
  CONSTRAINT chk_study_activities_kind CHECK (activity_kind IN ('TEACHING', 'PRACTICE', 'EXAM', 'VACATION', 'EVENT', 'OTHER'))
);

DROP TRIGGER IF EXISTS trg_study_activities_set_updated_at ON study_activities;
CREATE TRIGGER trg_study_activities_set_updated_at
BEFORE UPDATE ON study_activities
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

INSERT INTO study_activities (code, name, activity_kind, allows_lessons)
VALUES
  ('TEACHING', 'Учебные занятия', 'TEACHING', TRUE),
  ('PRACTICE', 'Практика', 'PRACTICE', FALSE),
  ('EXAM', 'Экзамен', 'EXAM', TRUE),
  ('VACATION', 'Каникулы', 'VACATION', FALSE)
ON CONFLICT (code) DO NOTHING;

-- Group-specific week calendar. It overrides/extends curriculum academic calendar.
CREATE TABLE IF NOT EXISTS study_calendar_weeks (
  id BIGSERIAL PRIMARY KEY,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  week_number SMALLINT NOT NULL CHECK (week_number BETWEEN 1 AND 60),
  week_start_date DATE NULL,
  activity_id INT NULL REFERENCES study_activities(id) ON DELETE SET NULL,
  allows_lessons BOOLEAN NOT NULL DEFAULT TRUE,
  comment TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_study_calendar_weeks_group_week UNIQUE (group_id, week_number)
);

CREATE INDEX IF NOT EXISTS idx_study_calendar_weeks_group_start ON study_calendar_weeks(group_id, week_start_date);
CREATE INDEX IF NOT EXISTS idx_study_calendar_weeks_activity ON study_calendar_weeks(activity_id);

DROP TRIGGER IF EXISTS trg_study_calendar_weeks_set_updated_at ON study_calendar_weeks;
CREATE TRIGGER trg_study_calendar_weeks_set_updated_at
BEFORE UPDATE ON study_calendar_weeks
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- Per-teacher day restrictions, e.g. illness/training/administrative ban.
CREATE TABLE IF NOT EXISTS teacher_day_constraints (
  id BIGSERIAL PRIMARY KEY,
  teacher_id INT NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
  target_date DATE NOT NULL,
  reason VARCHAR(255) NOT NULL DEFAULT '',
  allows_lessons BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_teacher_day_constraints_teacher_date UNIQUE (teacher_id, target_date)
);

CREATE INDEX IF NOT EXISTS idx_teacher_day_constraints_date ON teacher_day_constraints(target_date);

DROP TRIGGER IF EXISTS trg_teacher_day_constraints_set_updated_at ON teacher_day_constraints;
CREATE TRIGGER trg_teacher_day_constraints_set_updated_at
BEFORE UPDATE ON teacher_day_constraints
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- Replacement journal: explicit "what was replaced by what" records.
CREATE TABLE IF NOT EXISTS schedule_replacements (
  id BIGSERIAL PRIMARY KEY,
  target_date DATE NOT NULL,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  pair_number SMALLINT NOT NULL CHECK (pair_number BETWEEN 1 AND 8),
  subgroup SMALLINT NULL CHECK (subgroup IN (1, 2)),
  source_subject_id INT NULL REFERENCES subjects(id) ON DELETE SET NULL,
  source_location_id INT NULL REFERENCES locations(id) ON DELETE SET NULL,
  source_teacher_id INT NULL REFERENCES teachers(id) ON DELETE SET NULL,
  replacement_subject_id INT NULL REFERENCES subjects(id) ON DELETE SET NULL,
  replacement_location_id INT NULL REFERENCES locations(id) ON DELETE SET NULL,
  replacement_teacher_id INT NULL REFERENCES teachers(id) ON DELETE SET NULL,
  reason TEXT NULL,
  schedule_override_id BIGINT NULL REFERENCES schedule_overrides(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_schedule_replacements_slot
ON schedule_replacements (group_id, target_date, pair_number, COALESCE(subgroup, 0));

CREATE INDEX IF NOT EXISTS idx_schedule_replacements_group_date ON schedule_replacements(group_id, target_date);
CREATE INDEX IF NOT EXISTS idx_schedule_replacements_override ON schedule_replacements(schedule_override_id);

DROP TRIGGER IF EXISTS trg_schedule_replacements_set_updated_at ON schedule_replacements;
CREATE TRIGGER trg_schedule_replacements_set_updated_at
BEFORE UPDATE ON schedule_replacements
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_schedule_replacements_set_updated_at ON schedule_replacements;
DROP TABLE IF EXISTS schedule_replacements;

DROP TRIGGER IF EXISTS trg_teacher_day_constraints_set_updated_at ON teacher_day_constraints;
DROP TABLE IF EXISTS teacher_day_constraints;

DROP TRIGGER IF EXISTS trg_study_calendar_weeks_set_updated_at ON study_calendar_weeks;
DROP TABLE IF EXISTS study_calendar_weeks;

DROP TRIGGER IF EXISTS trg_study_activities_set_updated_at ON study_activities;
DROP TABLE IF EXISTS study_activities;

DROP INDEX IF EXISTS idx_locations_kind;
DROP INDEX IF EXISTS idx_locations_campus;

ALTER TABLE locations
  DROP CONSTRAINT IF EXISTS chk_locations_kind;

ALTER TABLE locations
  DROP COLUMN IF EXISTS capacity,
  DROP COLUMN IF EXISTS location_kind,
  DROP COLUMN IF EXISTS campus;
