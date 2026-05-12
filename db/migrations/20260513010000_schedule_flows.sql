-- +goose Up

ALTER TABLE schedule_templates
  ADD COLUMN IF NOT EXISTS flow_key VARCHAR(80) NULL;

ALTER TABLE schedule_overrides
  ADD COLUMN IF NOT EXISTS flow_key VARCHAR(80) NULL;

CREATE INDEX IF NOT EXISTS idx_schedule_templates_flow_key
  ON schedule_templates(flow_key)
  WHERE flow_key IS NOT NULL AND flow_key <> '';

CREATE INDEX IF NOT EXISTS idx_schedule_overrides_flow_key
  ON schedule_overrides(flow_key)
  WHERE flow_key IS NOT NULL AND flow_key <> '';

-- +goose Down

DROP INDEX IF EXISTS idx_schedule_overrides_flow_key;
DROP INDEX IF EXISTS idx_schedule_templates_flow_key;

ALTER TABLE schedule_overrides
  DROP COLUMN IF EXISTS flow_key;

ALTER TABLE schedule_templates
  DROP COLUMN IF EXISTS flow_key;
