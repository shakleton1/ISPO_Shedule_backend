-- +goose Up

-- Add draft/published lifecycle to schedule_templates and course_assignments.

-- schedule_templates: add status
ALTER TABLE schedule_templates
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'published';

ALTER TABLE schedule_templates
  DROP CONSTRAINT IF EXISTS chk_schedule_templates_status;
ALTER TABLE schedule_templates
  ADD CONSTRAINT chk_schedule_templates_status
  CHECK (status IN ('draft', 'published'));

-- Allow one draft and one published template per slot.
DROP INDEX IF EXISTS uidx_templates_one_per_pair;
CREATE UNIQUE INDEX IF NOT EXISTS uidx_templates_one_per_pair_status
ON schedule_templates (group_id, day_of_week, week_parity, pair_number, COALESCE(subgroup, 0), status);

CREATE INDEX IF NOT EXISTS idx_templates_group_status
ON schedule_templates (group_id, status);

-- course_assignments: add status
ALTER TABLE course_assignments
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'published';

ALTER TABLE course_assignments
  DROP CONSTRAINT IF EXISTS chk_course_assignments_status;
ALTER TABLE course_assignments
  ADD CONSTRAINT chk_course_assignments_status
  CHECK (status IN ('draft', 'published'));

-- Allow one draft and one published assignment per key.
DROP INDEX IF EXISTS uidx_course_assignments_key;
CREATE UNIQUE INDEX IF NOT EXISTS uidx_course_assignments_key_status
ON course_assignments (group_id, semester, subject_id, COALESCE(subgroup, 0), status);

CREATE INDEX IF NOT EXISTS idx_course_assignments_group_semester_status
ON course_assignments (group_id, semester, status);

-- +goose Down

DROP INDEX IF EXISTS idx_course_assignments_group_semester_status;
DROP INDEX IF EXISTS uidx_course_assignments_key_status;
DROP CONSTRAINT IF EXISTS chk_course_assignments_status;
ALTER TABLE course_assignments DROP COLUMN IF EXISTS status;

DROP INDEX IF EXISTS idx_templates_group_status;
DROP INDEX IF EXISTS uidx_templates_one_per_pair_status;
CREATE UNIQUE INDEX IF NOT EXISTS uidx_templates_one_per_pair
ON schedule_templates (group_id, day_of_week, week_parity, pair_number, COALESCE(subgroup, 0));

DROP CONSTRAINT IF EXISTS chk_schedule_templates_status;
ALTER TABLE schedule_templates DROP COLUMN IF EXISTS status;
