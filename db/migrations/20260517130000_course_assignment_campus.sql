-- +goose Up
-- +goose StatementBegin

ALTER TABLE course_assignments
  ADD COLUMN IF NOT EXISTS campus_id INT NULL REFERENCES campuses(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_course_assignments_campus_id
ON course_assignments(campus_id)
WHERE campus_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_course_assignments_campus_id;

ALTER TABLE course_assignments
  DROP COLUMN IF EXISTS campus_id;

-- +goose StatementEnd
