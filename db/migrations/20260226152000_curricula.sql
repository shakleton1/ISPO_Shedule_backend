-- +goose Up

-- Specialties (направление/специальность)
CREATE TABLE IF NOT EXISTS specialties (
  id SERIAL PRIMARY KEY,
  code VARCHAR(20) NOT NULL,
  name VARCHAR(200) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_specialties_code UNIQUE (code)
);

DROP TRIGGER IF EXISTS trg_specialties_set_updated_at ON specialties;
CREATE TRIGGER trg_specialties_set_updated_at
BEFORE UPDATE ON specialties
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- Curricula (учебный план) — один на год набора + вариант (опционально)
CREATE TABLE IF NOT EXISTS curricula (
  id BIGSERIAL PRIMARY KEY,
  specialty_id INT NOT NULL REFERENCES specialties(id) ON DELETE RESTRICT,
  admission_year SMALLINT NOT NULL CHECK (admission_year BETWEEN 2000 AND 2100),
  variant VARCHAR(50) NOT NULL DEFAULT '',
  title VARCHAR(200) NOT NULL DEFAULT '',
  notes TEXT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_curricula_unique UNIQUE (specialty_id, admission_year, variant)
);

CREATE INDEX IF NOT EXISTS idx_curricula_specialty_year ON curricula(specialty_id, admission_year);

DROP TRIGGER IF EXISTS trg_curricula_set_updated_at ON curricula;
CREATE TRIGGER trg_curricula_set_updated_at
BEFORE UPDATE ON curricula
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- Link groups to curriculum (optional)
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS curriculum_id BIGINT NULL REFERENCES curricula(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS admission_year SMALLINT NULL CHECK (admission_year BETWEEN 2000 AND 2100),
  ADD COLUMN IF NOT EXISTS specialty_id INT NULL REFERENCES specialties(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_groups_curriculum_id ON groups(curriculum_id);
CREATE INDEX IF NOT EXISTS idx_groups_specialty_id ON groups(specialty_id);

-- Academic calendar per curriculum (календарный учебный график)
CREATE TABLE IF NOT EXISTS academic_calendars (
  id BIGSERIAL PRIMARY KEY,
  curriculum_id BIGINT NOT NULL REFERENCES curricula(id) ON DELETE CASCADE,
  academic_year_start DATE NOT NULL,
  weeks_total SMALLINT NOT NULL DEFAULT 52 CHECK (weeks_total BETWEEN 1 AND 60),
  notes TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_academic_calendars UNIQUE (curriculum_id, academic_year_start)
);

DROP TRIGGER IF EXISTS trg_academic_calendars_set_updated_at ON academic_calendars;
CREATE TRIGGER trg_academic_calendars_set_updated_at
BEFORE UPDATE ON academic_calendars
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

CREATE TABLE IF NOT EXISTS academic_calendar_weeks (
  id BIGSERIAL PRIMARY KEY,
  calendar_id BIGINT NOT NULL REFERENCES academic_calendars(id) ON DELETE CASCADE,
  week_number SMALLINT NOT NULL CHECK (week_number BETWEEN 1 AND 60),
  week_start_date DATE NOT NULL,
  activity_code VARCHAR(10) NOT NULL,
  activity_name VARCHAR(100) NULL,
  is_teaching BOOLEAN NOT NULL DEFAULT TRUE,
  comment TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_academic_calendar_weeks UNIQUE (calendar_id, week_start_date),
  CONSTRAINT uidx_academic_calendar_weeknum UNIQUE (calendar_id, week_number)
);

CREATE INDEX IF NOT EXISTS idx_academic_calendar_weeks_lookup ON academic_calendar_weeks(calendar_id, week_start_date);

DROP TRIGGER IF EXISTS trg_academic_calendar_weeks_set_updated_at ON academic_calendar_weeks;
CREATE TRIGGER trg_academic_calendar_weeks_set_updated_at
BEFORE UPDATE ON academic_calendar_weeks
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- Curriculum items (строки учебного плана) + распределение по семестрам
CREATE TABLE IF NOT EXISTS curriculum_items (
  id BIGSERIAL PRIMARY KEY,
  curriculum_id BIGINT NOT NULL REFERENCES curricula(id) ON DELETE CASCADE,
  item_type TEXT NOT NULL CHECK (item_type IN ('DISCIPLINE','PRACTICE','ATTESTATION','GIA','VACATION','OTHER')),
  name VARCHAR(200) NOT NULL,
  subject_id INT NULL REFERENCES subjects(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_curriculum_items_curriculum ON curriculum_items(curriculum_id);

DROP TRIGGER IF EXISTS trg_curriculum_items_set_updated_at ON curriculum_items;
CREATE TRIGGER trg_curriculum_items_set_updated_at
BEFORE UPDATE ON curriculum_items
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

CREATE TABLE IF NOT EXISTS curriculum_item_allocations (
  id BIGSERIAL PRIMARY KEY,
  item_id BIGINT NOT NULL REFERENCES curriculum_items(id) ON DELETE CASCADE,
  semester SMALLINT NOT NULL CHECK (semester BETWEEN 1 AND 12),
  weeks SMALLINT NULL CHECK (weeks IS NULL OR weeks BETWEEN 0 AND 60),
  hours INT NULL CHECK (hours IS NULL OR hours BETWEEN 0 AND 10000),
  comment TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_curriculum_item_allocations UNIQUE (item_id, semester)
);

CREATE INDEX IF NOT EXISTS idx_curriculum_item_allocations_item ON curriculum_item_allocations(item_id);

DROP TRIGGER IF EXISTS trg_curriculum_item_allocations_set_updated_at ON curriculum_item_allocations;
CREATE TRIGGER trg_curriculum_item_allocations_set_updated_at
BEFORE UPDATE ON curriculum_item_allocations
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_curriculum_item_allocations_set_updated_at ON curriculum_item_allocations;
DROP TABLE IF EXISTS curriculum_item_allocations;

DROP TRIGGER IF EXISTS trg_curriculum_items_set_updated_at ON curriculum_items;
DROP TABLE IF EXISTS curriculum_items;

DROP TRIGGER IF EXISTS trg_academic_calendar_weeks_set_updated_at ON academic_calendar_weeks;
DROP TABLE IF EXISTS academic_calendar_weeks;

DROP TRIGGER IF EXISTS trg_academic_calendars_set_updated_at ON academic_calendars;
DROP TABLE IF EXISTS academic_calendars;

ALTER TABLE groups
  DROP COLUMN IF EXISTS curriculum_id,
  DROP COLUMN IF EXISTS admission_year,
  DROP COLUMN IF EXISTS specialty_id;

DROP TRIGGER IF EXISTS trg_curricula_set_updated_at ON curricula;
DROP TABLE IF EXISTS curricula;

DROP TRIGGER IF EXISTS trg_specialties_set_updated_at ON specialties;
DROP TABLE IF EXISTS specialties;
