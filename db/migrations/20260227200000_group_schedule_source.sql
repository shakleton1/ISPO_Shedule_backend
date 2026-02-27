-- +goose Up

ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS schedule_source_group_id INT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_groups_schedule_source_group'
  ) THEN
    ALTER TABLE groups
      ADD CONSTRAINT fk_groups_schedule_source_group
      FOREIGN KEY (schedule_source_group_id)
      REFERENCES groups(id)
      ON DELETE SET NULL;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_groups_schedule_source_group_id
  ON groups(schedule_source_group_id);

-- +goose Down

DROP INDEX IF EXISTS idx_groups_schedule_source_group_id;

ALTER TABLE groups
  DROP CONSTRAINT IF EXISTS fk_groups_schedule_source_group;

ALTER TABLE groups
  DROP COLUMN IF EXISTS schedule_source_group_id;
