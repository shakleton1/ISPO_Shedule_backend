-- +goose Up

CREATE TABLE IF NOT EXISTS schedule_lessons (
  id BIGSERIAL PRIMARY KEY,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  lesson_date DATE NOT NULL,
  pair_number SMALLINT NOT NULL CHECK (pair_number BETWEEN 1 AND 8),
  subgroup SMALLINT NULL CHECK (subgroup IN (1, 2)),
  subject_id INT NULL REFERENCES subjects(id) ON DELETE SET NULL,
  teacher_id INT NULL REFERENCES teachers(id) ON DELETE SET NULL,
  lesson_format TEXT NOT NULL DEFAULT 'offline',
  status TEXT NOT NULL DEFAULT 'published',
  source TEXT NOT NULL DEFAULT 'manual',
  flow_key VARCHAR(80) NULL,
  comment TEXT NULL,
  version INT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chk_schedule_lessons_lesson_format
    CHECK (lesson_format IN ('offline', 'online', 'hybrid')),
  CONSTRAINT chk_schedule_lessons_status
    CHECK (status IN ('draft', 'published', 'cancelled')),
  CONSTRAINT chk_schedule_lessons_source
    CHECK (source IN ('manual', 'imported', 'auto', 'generated', 'replacement'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_schedule_lessons_slot
ON schedule_lessons (
  group_id,
  lesson_date,
  pair_number,
  COALESCE(subgroup, 0)
)
WHERE status <> 'cancelled';

CREATE INDEX IF NOT EXISTS idx_schedule_lessons_group_date
ON schedule_lessons(group_id, lesson_date);

CREATE INDEX IF NOT EXISTS idx_schedule_lessons_teacher_date
ON schedule_lessons(teacher_id, lesson_date)
WHERE teacher_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_schedule_lessons_subject
ON schedule_lessons(subject_id)
WHERE subject_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_schedule_lessons_flow_key
ON schedule_lessons(flow_key)
WHERE flow_key IS NOT NULL AND flow_key <> '';

DROP TRIGGER IF EXISTS trg_schedule_lessons_set_updated_at ON schedule_lessons;
CREATE TRIGGER trg_schedule_lessons_set_updated_at
BEFORE UPDATE ON schedule_lessons
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

ALTER TABLE schedule_replacements
  DROP CONSTRAINT IF EXISTS schedule_replacements_schedule_override_id_fkey;

DROP TRIGGER IF EXISTS trg_room_assignments_set_updated_at ON room_assignments;
DROP INDEX IF EXISTS uidx_room_assignments_template;
DROP INDEX IF EXISTS uidx_room_assignments_override;
DROP INDEX IF EXISTS idx_room_assignments_location;

ALTER TABLE room_assignments
  DROP CONSTRAINT IF EXISTS chk_room_assignments_owner,
  DROP CONSTRAINT IF EXISTS chk_room_assignments_source,
  DROP CONSTRAINT IF EXISTS chk_room_assignments_status;

ALTER TABLE room_assignments
  ADD COLUMN IF NOT EXISTS schedule_lesson_id BIGINT NULL REFERENCES schedule_lessons(id) ON DELETE CASCADE;

DELETE FROM room_assignments WHERE schedule_lesson_id IS NULL;

ALTER TABLE room_assignments
  DROP COLUMN IF EXISTS schedule_template_id,
  DROP COLUMN IF EXISTS schedule_override_id;

ALTER TABLE room_assignments
  ALTER COLUMN schedule_lesson_id SET NOT NULL;

ALTER TABLE room_assignments
  ADD CONSTRAINT chk_room_assignments_source
  CHECK (source IN ('manual', 'auto', 'imported', 'teacher_preference', 'request', 'replacement')),
  ADD CONSTRAINT chk_room_assignments_status
  CHECK (status IN ('draft', 'published'));

CREATE UNIQUE INDEX IF NOT EXISTS uidx_room_assignments_lesson
ON room_assignments(schedule_lesson_id);

CREATE INDEX IF NOT EXISTS idx_room_assignments_location
ON room_assignments(location_id);

DROP TRIGGER IF EXISTS trg_room_assignments_set_updated_at ON room_assignments;
CREATE TRIGGER trg_room_assignments_set_updated_at
BEFORE UPDATE ON room_assignments
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

DROP TRIGGER IF EXISTS trg_schedule_overrides_set_updated_at ON schedule_overrides;
DROP TABLE IF EXISTS schedule_overrides;

CREATE TABLE schedule_overrides (
  id BIGSERIAL PRIMARY KEY,
  schedule_lesson_id BIGINT NULL REFERENCES schedule_lessons(id) ON DELETE SET NULL,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  lesson_date DATE NOT NULL,
  pair_number SMALLINT NOT NULL CHECK (pair_number BETWEEN 1 AND 8),
  subgroup SMALLINT NULL CHECK (subgroup IN (1, 2)),
  action_type TEXT NOT NULL
    CHECK (action_type IN ('add', 'replace', 'cancel', 'restore')),
  source_subject_id INT NULL REFERENCES subjects(id) ON DELETE SET NULL,
  source_teacher_id INT NULL REFERENCES teachers(id) ON DELETE SET NULL,
  source_location_id INT NULL REFERENCES locations(id) ON DELETE SET NULL,
  source_lesson_format TEXT NULL
    CHECK (source_lesson_format IS NULL OR source_lesson_format IN ('offline', 'online', 'hybrid')),
  replacement_subject_id INT NULL REFERENCES subjects(id) ON DELETE SET NULL,
  replacement_teacher_id INT NULL REFERENCES teachers(id) ON DELETE SET NULL,
  replacement_location_id INT NULL REFERENCES locations(id) ON DELETE SET NULL,
  replacement_lesson_format TEXT NULL
    CHECK (replacement_lesson_format IS NULL OR replacement_lesson_format IN ('offline', 'online', 'hybrid')),
  reason TEXT NULL,
  status TEXT NOT NULL DEFAULT 'applied'
    CHECK (status IN ('draft', 'applied', 'cancelled')),
  expected_lesson_version INT NULL,
  applied_lesson_version INT NULL,
  created_by INT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  applied_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_schedule_overrides_lesson
ON schedule_overrides(schedule_lesson_id);

CREATE INDEX IF NOT EXISTS idx_schedule_overrides_group_date
ON schedule_overrides(group_id, lesson_date);

CREATE INDEX IF NOT EXISTS idx_schedule_overrides_created_at
ON schedule_overrides(created_at);

ALTER TABLE schedule_replacements
  ADD CONSTRAINT schedule_replacements_schedule_override_id_fkey
  FOREIGN KEY (schedule_override_id) REFERENCES schedule_overrides(id) ON DELETE SET NULL;

DROP TABLE IF EXISTS calendar_exceptions;
DROP TABLE IF EXISTS schedule_templates;

-- +goose Down

DROP TABLE IF EXISTS schedule_templates;

CREATE TABLE IF NOT EXISTS schedule_templates (
  id BIGSERIAL PRIMARY KEY,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  day_of_week SMALLINT NOT NULL,
  week_parity TEXT NOT NULL,
  pair_number SMALLINT NOT NULL,
  subject_id INT NOT NULL REFERENCES subjects(id) ON DELETE RESTRICT,
  location_id INT NULL REFERENCES locations(id) ON DELETE SET NULL,
  lesson_format TEXT NOT NULL DEFAULT 'offline',
  status TEXT NOT NULL DEFAULT 'published',
  teacher_manual BOOLEAN NOT NULL DEFAULT FALSE,
  location_manual BOOLEAN NOT NULL DEFAULT FALSE,
  teacher_id INT NULL REFERENCES teachers(id) ON DELETE SET NULL,
  subgroup SMALLINT NULL,
  flow_key VARCHAR(80) NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS calendar_exceptions (
  id BIGSERIAL PRIMARY KEY,
  target_date DATE NOT NULL UNIQUE,
  works_as_day SMALLINT NOT NULL,
  comment TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE schedule_replacements
  DROP CONSTRAINT IF EXISTS schedule_replacements_schedule_override_id_fkey;

DROP TABLE IF EXISTS schedule_overrides;

CREATE TABLE schedule_overrides (
  id BIGSERIAL PRIMARY KEY,
  target_date DATE NOT NULL,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  pair_number SMALLINT NOT NULL,
  action_type TEXT NOT NULL,
  new_subject_id INT NULL REFERENCES subjects(id) ON DELETE SET NULL,
  new_location_id INT NULL REFERENCES locations(id) ON DELETE SET NULL,
  new_lesson_format TEXT NULL,
  new_teacher_id INT NULL REFERENCES teachers(id) ON DELETE SET NULL,
  new_teacher_manual BOOLEAN NOT NULL DEFAULT FALSE,
  comment TEXT NULL,
  subgroup SMALLINT NULL,
  flow_key VARCHAR(80) NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE schedule_replacements
  ADD CONSTRAINT schedule_replacements_schedule_override_id_fkey
  FOREIGN KEY (schedule_override_id) REFERENCES schedule_overrides(id) ON DELETE SET NULL;

DROP TRIGGER IF EXISTS trg_room_assignments_set_updated_at ON room_assignments;
DROP INDEX IF EXISTS uidx_room_assignments_lesson;
DROP INDEX IF EXISTS idx_room_assignments_location;

ALTER TABLE room_assignments
  DROP CONSTRAINT IF EXISTS chk_room_assignments_source,
  DROP CONSTRAINT IF EXISTS chk_room_assignments_status,
  DROP COLUMN IF EXISTS schedule_lesson_id,
  ADD COLUMN IF NOT EXISTS schedule_template_id BIGINT NULL REFERENCES schedule_templates(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS schedule_override_id BIGINT NULL REFERENCES schedule_overrides(id) ON DELETE CASCADE;

ALTER TABLE room_assignments
  ADD CONSTRAINT chk_room_assignments_owner CHECK (
    (schedule_template_id IS NOT NULL AND schedule_override_id IS NULL)
    OR (schedule_template_id IS NULL AND schedule_override_id IS NOT NULL)
  ),
  ADD CONSTRAINT chk_room_assignments_source
  CHECK (source IN ('manual', 'auto', 'imported', 'teacher_preference', 'request')),
  ADD CONSTRAINT chk_room_assignments_status
  CHECK (status IN ('draft', 'published'));

CREATE UNIQUE INDEX IF NOT EXISTS uidx_room_assignments_template
ON room_assignments(schedule_template_id)
WHERE schedule_template_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_room_assignments_override
ON room_assignments(schedule_override_id)
WHERE schedule_override_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_room_assignments_location
ON room_assignments(location_id);

CREATE TRIGGER trg_room_assignments_set_updated_at
BEFORE UPDATE ON room_assignments
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

DROP TRIGGER IF EXISTS trg_schedule_lessons_set_updated_at ON schedule_lessons;
DROP TABLE IF EXISTS schedule_lessons;
