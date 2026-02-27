-- +goose Up

-- 1) Soft delete for key dictionaries.
ALTER TABLE teachers
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

ALTER TABLE subjects
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

ALTER TABLE curricula
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_teachers_deleted_at ON teachers(deleted_at);
CREATE INDEX IF NOT EXISTS idx_subjects_deleted_at ON subjects(deleted_at);
CREATE INDEX IF NOT EXISTS idx_curricula_deleted_at ON curricula(deleted_at);

-- 2) Indexes for teacher_subjects access patterns.
CREATE INDEX IF NOT EXISTS idx_teacher_subjects_teacher_id ON teacher_subjects(teacher_id);

-- 3) Hard guarantee: course_assignments.curriculum_item_id belongs to group.curriculum_id.
CREATE OR REPLACE FUNCTION ispo_validate_course_assignment_curriculum_item()
RETURNS trigger AS $$
DECLARE
  group_curriculum_id BIGINT;
  item_curriculum_id BIGINT;
BEGIN
  IF NEW.curriculum_item_id IS NULL THEN
    RETURN NEW;
  END IF;

  SELECT g.curriculum_id INTO group_curriculum_id
  FROM groups g
  WHERE g.id = NEW.group_id;

  IF group_curriculum_id IS NULL THEN
    RAISE EXCEPTION 'group % has no curriculum_id; curriculum_item_id must be NULL', NEW.group_id;
  END IF;

  SELECT ci.curriculum_id INTO item_curriculum_id
  FROM curriculum_items ci
  WHERE ci.id = NEW.curriculum_item_id;

  IF item_curriculum_id IS NULL THEN
    -- Should be prevented by FK, but keep error explicit.
    RAISE EXCEPTION 'curriculum_item_id % not found', NEW.curriculum_item_id;
  END IF;

  IF item_curriculum_id <> group_curriculum_id THEN
    RAISE EXCEPTION 'curriculum_item_id % belongs to curriculum %, expected % for group %',
      NEW.curriculum_item_id, item_curriculum_id, group_curriculum_id, NEW.group_id;
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_course_assignments_validate_curriculum_item ON course_assignments;
CREATE TRIGGER trg_course_assignments_validate_curriculum_item
BEFORE INSERT OR UPDATE ON course_assignments
FOR EACH ROW
EXECUTE FUNCTION ispo_validate_course_assignment_curriculum_item();

-- +goose Down

DROP TRIGGER IF EXISTS trg_course_assignments_validate_curriculum_item ON course_assignments;
DROP FUNCTION IF EXISTS ispo_validate_course_assignment_curriculum_item();

DROP INDEX IF EXISTS idx_teacher_subjects_teacher_id;

DROP INDEX IF EXISTS idx_curricula_deleted_at;
DROP INDEX IF EXISTS idx_subjects_deleted_at;
DROP INDEX IF EXISTS idx_teachers_deleted_at;

ALTER TABLE curricula DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE subjects DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE teachers DROP COLUMN IF EXISTS deleted_at;
