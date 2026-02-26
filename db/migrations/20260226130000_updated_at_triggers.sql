-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION ispo_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_groups_set_updated_at ON groups;
CREATE TRIGGER trg_groups_set_updated_at
BEFORE UPDATE ON groups
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

DROP TRIGGER IF EXISTS trg_subjects_set_updated_at ON subjects;
CREATE TRIGGER trg_subjects_set_updated_at
BEFORE UPDATE ON subjects
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

DROP TRIGGER IF EXISTS trg_locations_set_updated_at ON locations;
CREATE TRIGGER trg_locations_set_updated_at
BEFORE UPDATE ON locations
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

DROP TRIGGER IF EXISTS trg_students_set_updated_at ON students;
CREATE TRIGGER trg_students_set_updated_at
BEFORE UPDATE ON students
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

DROP TRIGGER IF EXISTS trg_schedule_templates_set_updated_at ON schedule_templates;
CREATE TRIGGER trg_schedule_templates_set_updated_at
BEFORE UPDATE ON schedule_templates
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

DROP TRIGGER IF EXISTS trg_schedule_overrides_set_updated_at ON schedule_overrides;
CREATE TRIGGER trg_schedule_overrides_set_updated_at
BEFORE UPDATE ON schedule_overrides
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

DROP TRIGGER IF EXISTS trg_schedule_day_overlays_set_updated_at ON schedule_day_overlays;
CREATE TRIGGER trg_schedule_day_overlays_set_updated_at
BEFORE UPDATE ON schedule_day_overlays
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

DROP TRIGGER IF EXISTS trg_calendar_exceptions_set_updated_at ON calendar_exceptions;
CREATE TRIGGER trg_calendar_exceptions_set_updated_at
BEFORE UPDATE ON calendar_exceptions
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_calendar_exceptions_set_updated_at ON calendar_exceptions;
DROP TRIGGER IF EXISTS trg_schedule_day_overlays_set_updated_at ON schedule_day_overlays;
DROP TRIGGER IF EXISTS trg_schedule_overrides_set_updated_at ON schedule_overrides;
DROP TRIGGER IF EXISTS trg_schedule_templates_set_updated_at ON schedule_templates;
DROP TRIGGER IF EXISTS trg_students_set_updated_at ON students;
DROP TRIGGER IF EXISTS trg_locations_set_updated_at ON locations;
DROP TRIGGER IF EXISTS trg_subjects_set_updated_at ON subjects;
DROP TRIGGER IF EXISTS trg_groups_set_updated_at ON groups;

DROP FUNCTION IF EXISTS ispo_set_updated_at();
