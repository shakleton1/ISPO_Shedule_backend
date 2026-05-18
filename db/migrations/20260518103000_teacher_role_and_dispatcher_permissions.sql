-- +goose Up
-- +goose StatementBegin

ALTER TABLE students
  ADD COLUMN IF NOT EXISTS teacher_id INT NULL REFERENCES teachers(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_students_teacher_id
ON students(teacher_id)
WHERE teacher_id IS NOT NULL;

ALTER TABLE students
  DROP CONSTRAINT IF EXISTS students_role_check;

ALTER TABLE students
  ADD CONSTRAINT students_role_check
  CHECK (role IN ('student','dispatcher','admin','viewer','teacher'));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

UPDATE students
SET role = 'viewer'
WHERE role = 'teacher';

ALTER TABLE students
  DROP CONSTRAINT IF EXISTS students_role_check;

ALTER TABLE students
  ADD CONSTRAINT students_role_check
  CHECK (role IN ('student','dispatcher','admin','viewer'));

DROP INDEX IF EXISTS idx_students_teacher_id;

ALTER TABLE students
  DROP COLUMN IF EXISTS teacher_id;

-- +goose StatementEnd
