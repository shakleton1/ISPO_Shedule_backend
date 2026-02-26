-- +goose Up

-- Extend role enum/check constraint with read-only viewer role.
-- Note: original constraint created by 20260226103000_auth_roles.sql is expected to be named students_role_check.

ALTER TABLE students
  DROP CONSTRAINT IF EXISTS students_role_check;

ALTER TABLE students
  ADD CONSTRAINT students_role_check CHECK (role IN ('student','dispatcher','admin','viewer'));

-- +goose Down

ALTER TABLE students
  DROP CONSTRAINT IF EXISTS students_role_check;

ALTER TABLE students
  ADD CONSTRAINT students_role_check CHECK (role IN ('student','dispatcher','admin'));
