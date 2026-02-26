-- +goose Up

-- Teachers dictionary (normalized)
CREATE TABLE IF NOT EXISTS teachers (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  name_key TEXT GENERATED ALWAYS AS (lower(btrim(name))) STORED,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uidx_teachers_name_key UNIQUE (name_key)
);

DROP TRIGGER IF EXISTS trg_teachers_set_updated_at ON teachers;
CREATE TRIGGER trg_teachers_set_updated_at
BEFORE UPDATE ON teachers
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- Add FK columns (nullable: teacher may be empty)
ALTER TABLE schedule_templates
  ADD COLUMN IF NOT EXISTS teacher_id INT NULL REFERENCES teachers(id);

ALTER TABLE schedule_overrides
  ADD COLUMN IF NOT EXISTS new_teacher_id INT NULL REFERENCES teachers(id);

-- Backfill teachers from existing string columns
INSERT INTO teachers (name)
SELECT DISTINCT btrim(teacher_name)
FROM schedule_templates
WHERE btrim(teacher_name) <> ''
ON CONFLICT (name_key) DO NOTHING;

INSERT INTO teachers (name)
SELECT DISTINCT btrim(new_teacher_name)
FROM schedule_overrides
WHERE new_teacher_name IS NOT NULL AND btrim(new_teacher_name) <> ''
ON CONFLICT (name_key) DO NOTHING;

-- Backfill FK references
UPDATE schedule_templates st
SET teacher_id = t.id
FROM teachers t
WHERE btrim(st.teacher_name) <> ''
  AND t.name_key = lower(btrim(st.teacher_name));

UPDATE schedule_overrides so
SET new_teacher_id = t.id
FROM teachers t
WHERE so.new_teacher_name IS NOT NULL
  AND btrim(so.new_teacher_name) <> ''
  AND t.name_key = lower(btrim(so.new_teacher_name));

-- Drop denormalized columns
ALTER TABLE schedule_templates
  DROP COLUMN IF EXISTS teacher_name;

-- Override constraints migration created a CHECK referencing new_teacher_name;
-- we must drop it before dropping the column.
ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_overrides_cancel_has_no_new_fields;

ALTER TABLE schedule_overrides
  DROP COLUMN IF EXISTS new_teacher_name;

-- Re-add CANCEL constraint using new_teacher_id.
ALTER TABLE schedule_overrides
  ADD CONSTRAINT chk_overrides_cancel_has_no_new_fields
  CHECK (
    action_type <> 'CANCEL'
    OR (new_subject_id IS NULL AND new_location_id IS NULL AND new_teacher_id IS NULL)
  );

-- +goose Down

-- Recreate old columns
ALTER TABLE schedule_templates
  ADD COLUMN IF NOT EXISTS teacher_name VARCHAR(100) NOT NULL DEFAULT '';

ALTER TABLE schedule_overrides
  ADD COLUMN IF NOT EXISTS new_teacher_name VARCHAR(100) NULL;

-- Switch CANCEL constraint back to new_teacher_name.
ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_overrides_cancel_has_no_new_fields;

ALTER TABLE schedule_overrides
  ADD CONSTRAINT chk_overrides_cancel_has_no_new_fields
  CHECK (
    action_type <> 'CANCEL'
    OR (new_subject_id IS NULL AND new_location_id IS NULL AND new_teacher_name IS NULL)
  );

-- Restore values from teachers
UPDATE schedule_templates st
SET teacher_name = COALESCE(t.name, '')
FROM teachers t
WHERE st.teacher_id = t.id;

UPDATE schedule_overrides so
SET new_teacher_name = t.name
FROM teachers t
WHERE so.new_teacher_id = t.id;

-- Drop FK columns
ALTER TABLE schedule_overrides
  DROP COLUMN IF EXISTS new_teacher_id;

ALTER TABLE schedule_templates
  DROP COLUMN IF EXISTS teacher_id;

-- Drop trigger + table
DROP TRIGGER IF EXISTS trg_teachers_set_updated_at ON teachers;
DROP TABLE IF EXISTS teachers;
