-- +goose Up

-- We treat existing `students` as unified `users` table (student/dispatcher/admin).

ALTER TABLE students
  ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'student' CHECK (role IN ('student','dispatcher','admin'));

-- For non-students (admin/dispatcher) group_id can stay NULL.
-- password_hash already exists.

CREATE INDEX IF NOT EXISTS idx_students_role ON students(role);

-- +goose Down

DROP INDEX IF EXISTS idx_students_role;

ALTER TABLE students
  DROP COLUMN IF EXISTS role;
