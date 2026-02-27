-- +goose Up

-- Manual override flags to control auto-resolve from course_assignments.

-- Templates: allow "manual empty" (keep blank even if assignment exists).
ALTER TABLE schedule_templates
  ADD COLUMN IF NOT EXISTS teacher_manual BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE schedule_templates
  ADD COLUMN IF NOT EXISTS location_manual BOOLEAN NOT NULL DEFAULT FALSE;

-- Overrides: persist whether new_teacher_name was explicitly set (including empty string).
ALTER TABLE schedule_overrides
  ADD COLUMN IF NOT EXISTS new_teacher_manual BOOLEAN NOT NULL DEFAULT FALSE;

-- Backfill: existing overrides with new_teacher_id are considered manual.
UPDATE schedule_overrides
SET new_teacher_manual = TRUE
WHERE new_teacher_id IS NOT NULL;

-- +goose Down

ALTER TABLE schedule_overrides
  DROP COLUMN IF EXISTS new_teacher_manual;

ALTER TABLE schedule_templates
  DROP COLUMN IF EXISTS location_manual;

ALTER TABLE schedule_templates
  DROP COLUMN IF EXISTS teacher_manual;
