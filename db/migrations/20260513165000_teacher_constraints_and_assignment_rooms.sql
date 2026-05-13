-- +goose Up
-- +goose StatementBegin

ALTER TABLE teacher_day_constraints
  ADD COLUMN IF NOT EXISTS constraint_level TEXT NOT NULL DEFAULT 'warning',
  ADD COLUMN IF NOT EXISTS requires_confirmation BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE teacher_day_constraints
SET
  constraint_level = 'warning',
  requires_confirmation = NOT allows_lessons;

ALTER TABLE teacher_day_constraints
  DROP CONSTRAINT IF EXISTS chk_teacher_day_constraints_level;
ALTER TABLE teacher_day_constraints
  ADD CONSTRAINT chk_teacher_day_constraints_level
  CHECK (constraint_level IN ('warning', 'hard_block'));

ALTER TABLE teacher_day_constraints
  DROP COLUMN IF EXISTS allows_lessons;

ALTER TABLE course_assignments
  DROP COLUMN IF EXISTS location_id;

ALTER TABLE room_assignments
  DROP CONSTRAINT IF EXISTS chk_room_assignments_source;
UPDATE room_assignments
SET source = 'manual'
WHERE source = 'course_assignment';
ALTER TABLE room_assignments
  ADD CONSTRAINT chk_room_assignments_source
  CHECK (source IN ('manual', 'auto', 'imported', 'teacher_preference', 'request'));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE room_assignments
  DROP CONSTRAINT IF EXISTS chk_room_assignments_source;
ALTER TABLE room_assignments
  ADD CONSTRAINT chk_room_assignments_source
  CHECK (source IN ('manual', 'auto', 'imported', 'teacher_preference', 'course_assignment', 'request'));

ALTER TABLE course_assignments
  ADD COLUMN IF NOT EXISTS location_id INT NULL REFERENCES locations(id) ON DELETE SET NULL;

ALTER TABLE teacher_day_constraints
  ADD COLUMN IF NOT EXISTS allows_lessons BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE teacher_day_constraints
SET allows_lessons = CASE
  WHEN constraint_level = 'hard_block' THEN FALSE
  WHEN requires_confirmation THEN FALSE
  ELSE TRUE
END;

ALTER TABLE teacher_day_constraints
  DROP CONSTRAINT IF EXISTS chk_teacher_day_constraints_level;
ALTER TABLE teacher_day_constraints
  DROP COLUMN IF EXISTS constraint_level,
  DROP COLUMN IF EXISTS requires_confirmation;

-- +goose StatementEnd
