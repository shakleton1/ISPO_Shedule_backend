-- +goose Up
-- +goose StatementBegin

ALTER TABLE course_assignments
  ADD COLUMN IF NOT EXISTS is_flow BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_course_assignments_is_flow
ON course_assignments(is_flow)
WHERE is_flow = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_course_assignments_is_flow;

ALTER TABLE course_assignments
  DROP COLUMN IF EXISTS is_flow;

-- +goose StatementEnd
