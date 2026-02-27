-- +goose Up

ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_overrides_add_requires_subject_location;
ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_overrides_add_requires_subject;

ALTER TABLE schedule_overrides
  ADD CONSTRAINT chk_overrides_add_requires_subject
  CHECK (
    action_type <> 'ADD'
    OR (new_subject_id IS NOT NULL)
  );

-- +goose Down

ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_overrides_add_requires_subject;

ALTER TABLE schedule_overrides
  ADD CONSTRAINT chk_overrides_add_requires_subject_location
  CHECK (
    action_type <> 'ADD'
    OR (new_subject_id IS NOT NULL AND new_location_id IS NOT NULL)
  );
