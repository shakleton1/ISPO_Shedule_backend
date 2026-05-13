-- +goose Up

CREATE TABLE IF NOT EXISTS campuses (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  address TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_campuses_name UNIQUE (name)
);

DROP TRIGGER IF EXISTS trg_campuses_set_updated_at ON campuses;
CREATE TRIGGER trg_campuses_set_updated_at
BEFORE UPDATE ON campuses
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

CREATE TABLE IF NOT EXISTS location_types (
  id SERIAL PRIMARY KEY,
  code VARCHAR(50) NOT NULL,
  name VARCHAR(100) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_location_types_code UNIQUE (code)
);

DROP TRIGGER IF EXISTS trg_location_types_set_updated_at ON location_types;
CREATE TRIGGER trg_location_types_set_updated_at
BEFORE UPDATE ON location_types
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

INSERT INTO location_types (code, name)
VALUES
  ('classroom', 'Обычная аудитория'),
  ('computer_class', 'ВЦ / компьютерный класс'),
  ('lab', 'Лаборатория'),
  ('gym', 'Спортивный зал'),
  ('pool', 'Бассейн'),
  ('lecture_hall', 'Потоковая аудитория'),
  ('special', 'Спецкабинет'),
  ('online', 'Онлайн / дистант')
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name;

ALTER TABLE locations
  ADD COLUMN IF NOT EXISTS campus_id INT NULL REFERENCES campuses(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'physical',
  ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE locations
  DROP CONSTRAINT IF EXISTS chk_locations_physical_kind;
ALTER TABLE locations
  ADD CONSTRAINT chk_locations_physical_kind CHECK (kind IN ('physical', 'virtual'));

CREATE INDEX IF NOT EXISTS idx_locations_campus_id ON locations(campus_id);
CREATE INDEX IF NOT EXISTS idx_locations_kind_new ON locations(kind);
CREATE INDEX IF NOT EXISTS idx_locations_is_active ON locations(is_active);

ALTER TABLE locations
  DROP CONSTRAINT IF EXISTS chk_locations_kind;

DROP INDEX IF EXISTS idx_locations_kind;
DROP INDEX IF EXISTS idx_locations_campus;

ALTER TABLE locations
  DROP COLUMN IF EXISTS is_virtual,
  DROP COLUMN IF EXISTS campus,
  DROP COLUMN IF EXISTS location_kind;

CREATE TABLE IF NOT EXISTS location_type_links (
  location_id INT NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  type_id INT NOT NULL REFERENCES location_types(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (location_id, type_id)
);

CREATE INDEX IF NOT EXISTS idx_location_type_links_type_id ON location_type_links(type_id);

ALTER TABLE schedule_templates
  ADD COLUMN IF NOT EXISTS lesson_format TEXT NOT NULL DEFAULT 'offline';

ALTER TABLE schedule_overrides
  ADD COLUMN IF NOT EXISTS new_lesson_format TEXT NULL;

ALTER TABLE schedule_templates
  DROP CONSTRAINT IF EXISTS chk_schedule_templates_lesson_format;
ALTER TABLE schedule_templates
  ADD CONSTRAINT chk_schedule_templates_lesson_format
  CHECK (lesson_format IN ('offline', 'online', 'hybrid'));

ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_schedule_overrides_lesson_format;
ALTER TABLE schedule_overrides
  ADD CONSTRAINT chk_schedule_overrides_lesson_format
  CHECK (new_lesson_format IS NULL OR new_lesson_format IN ('offline', 'online', 'hybrid'));

ALTER TABLE schedule_templates
  ALTER COLUMN location_id DROP NOT NULL;

CREATE TABLE IF NOT EXISTS teacher_location_preferences (
  id BIGSERIAL PRIMARY KEY,
  teacher_id INT NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
  location_id INT NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  priority INT NOT NULL DEFAULT 100 CHECK (priority > 0),
  valid_from DATE NULL,
  valid_to DATE NULL,
  comment TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chk_teacher_location_preferences_dates CHECK (valid_to IS NULL OR valid_from IS NULL OR valid_to >= valid_from),
  CONSTRAINT uidx_teacher_location_preferences UNIQUE (teacher_id, location_id, priority)
);

CREATE INDEX IF NOT EXISTS idx_teacher_location_preferences_teacher ON teacher_location_preferences(teacher_id, priority, id);
CREATE INDEX IF NOT EXISTS idx_teacher_location_preferences_location ON teacher_location_preferences(location_id);

DROP TRIGGER IF EXISTS trg_teacher_location_preferences_set_updated_at ON teacher_location_preferences;
CREATE TRIGGER trg_teacher_location_preferences_set_updated_at
BEFORE UPDATE ON teacher_location_preferences
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

CREATE TABLE IF NOT EXISTS room_requests (
  id BIGSERIAL PRIMARY KEY,
  teacher_id INT NULL REFERENCES teachers(id) ON DELETE SET NULL,
  subject_id INT NULL REFERENCES subjects(id) ON DELETE SET NULL,
  group_id INT NULL REFERENCES groups(id) ON DELETE CASCADE,
  semester SMALLINT NULL CHECK (semester IS NULL OR semester BETWEEN 1 AND 12),
  required_type_id INT NULL REFERENCES location_types(id) ON DELETE SET NULL,
  preferred_location_id INT NULL REFERENCES locations(id) ON DELETE SET NULL,
  priority INT NOT NULL DEFAULT 100 CHECK (priority > 0),
  comment TEXT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chk_room_requests_status CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_room_requests_lookup ON room_requests(group_id, semester, subject_id, teacher_id, priority);
CREATE INDEX IF NOT EXISTS idx_room_requests_type ON room_requests(required_type_id);
CREATE INDEX IF NOT EXISTS idx_room_requests_location ON room_requests(preferred_location_id);

DROP TRIGGER IF EXISTS trg_room_requests_set_updated_at ON room_requests;
CREATE TRIGGER trg_room_requests_set_updated_at
BEFORE UPDATE ON room_requests
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

CREATE TABLE IF NOT EXISTS room_assignments (
  id BIGSERIAL PRIMARY KEY,
  schedule_template_id BIGINT NULL REFERENCES schedule_templates(id) ON DELETE CASCADE,
  schedule_override_id BIGINT NULL REFERENCES schedule_overrides(id) ON DELETE CASCADE,
  location_id INT NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
  source TEXT NOT NULL DEFAULT 'manual',
  status TEXT NOT NULL DEFAULT 'published',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT chk_room_assignments_owner CHECK (
    (schedule_template_id IS NOT NULL AND schedule_override_id IS NULL)
    OR (schedule_template_id IS NULL AND schedule_override_id IS NOT NULL)
  ),
  CONSTRAINT chk_room_assignments_source CHECK (source IN ('manual', 'auto', 'imported', 'teacher_preference', 'course_assignment', 'request')),
  CONSTRAINT chk_room_assignments_status CHECK (status IN ('draft', 'published'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_room_assignments_template
ON room_assignments(schedule_template_id)
WHERE schedule_template_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_room_assignments_override
ON room_assignments(schedule_override_id)
WHERE schedule_override_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_room_assignments_location ON room_assignments(location_id);

DROP TRIGGER IF EXISTS trg_room_assignments_set_updated_at ON room_assignments;
CREATE TRIGGER trg_room_assignments_set_updated_at
BEFORE UPDATE ON room_assignments
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_room_assignments_set_updated_at ON room_assignments;
DROP TABLE IF EXISTS room_assignments;

DROP TRIGGER IF EXISTS trg_room_requests_set_updated_at ON room_requests;
DROP TABLE IF EXISTS room_requests;

DROP TRIGGER IF EXISTS trg_teacher_location_preferences_set_updated_at ON teacher_location_preferences;
DROP TABLE IF EXISTS teacher_location_preferences;

ALTER TABLE schedule_templates
  ALTER COLUMN location_id SET NOT NULL;

ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_schedule_overrides_lesson_format;
ALTER TABLE schedule_templates
  DROP CONSTRAINT IF EXISTS chk_schedule_templates_lesson_format;

ALTER TABLE schedule_overrides
  DROP COLUMN IF EXISTS new_lesson_format;
ALTER TABLE schedule_templates
  DROP COLUMN IF EXISTS lesson_format;

DROP TABLE IF EXISTS location_type_links;

ALTER TABLE locations
  ADD COLUMN IF NOT EXISTS is_virtual BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS campus VARCHAR(50) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS location_kind TEXT NOT NULL DEFAULT 'classroom';

ALTER TABLE locations
  DROP CONSTRAINT IF EXISTS chk_locations_physical_kind;

DROP INDEX IF EXISTS idx_locations_is_active;
DROP INDEX IF EXISTS idx_locations_kind_new;
DROP INDEX IF EXISTS idx_locations_campus_id;

ALTER TABLE locations
  DROP COLUMN IF EXISTS is_active,
  DROP COLUMN IF EXISTS kind,
  DROP COLUMN IF EXISTS campus_id;

ALTER TABLE locations
  ADD CONSTRAINT chk_locations_kind
  CHECK (location_kind IN ('classroom', 'computer_lab', 'gym', 'pool', 'virtual', 'other'));

CREATE INDEX IF NOT EXISTS idx_locations_campus ON locations(campus);
CREATE INDEX IF NOT EXISTS idx_locations_kind ON locations(location_kind);

DROP TRIGGER IF EXISTS trg_location_types_set_updated_at ON location_types;
DROP TABLE IF EXISTS location_types;

DROP TRIGGER IF EXISTS trg_campuses_set_updated_at ON campuses;
DROP TABLE IF EXISTS campuses;
