-- +goose Up

-- 1) Curriculum items: hierarchy + index codes
ALTER TABLE curriculum_items
  ADD COLUMN IF NOT EXISTS parent_id BIGINT NULL REFERENCES curriculum_items(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS index_code VARCHAR(30) NULL;

CREATE INDEX IF NOT EXISTS idx_curriculum_items_parent ON curriculum_items(parent_id);
CREATE INDEX IF NOT EXISTS idx_curriculum_items_index_code ON curriculum_items(index_code);

-- 2) Allocations: detailed hour breakdown + assessment type
ALTER TABLE curriculum_item_allocations
  ADD COLUMN IF NOT EXISTS hours_total INT NULL CHECK (hours_total IS NULL OR hours_total BETWEEN 0 AND 10000),
  ADD COLUMN IF NOT EXISTS hours_lectures INT NULL CHECK (hours_lectures IS NULL OR hours_lectures BETWEEN 0 AND 10000),
  ADD COLUMN IF NOT EXISTS hours_practice INT NULL CHECK (hours_practice IS NULL OR hours_practice BETWEEN 0 AND 10000),
  ADD COLUMN IF NOT EXISTS hours_lab INT NULL CHECK (hours_lab IS NULL OR hours_lab BETWEEN 0 AND 10000),
  ADD COLUMN IF NOT EXISTS hours_independent INT NULL CHECK (hours_independent IS NULL OR hours_independent BETWEEN 0 AND 10000),
  ADD COLUMN IF NOT EXISTS hours_exam INT NULL CHECK (hours_exam IS NULL OR hours_exam BETWEEN 0 AND 10000),
  ADD COLUMN IF NOT EXISTS assessment_type TEXT NULL CHECK (assessment_type IN ('EXAM', 'CREDIT', 'GRADED_CREDIT', 'MODULE_EXAM'));

-- Migrate existing flat hours into hours_total (if present)
UPDATE curriculum_item_allocations
SET hours_total = hours
WHERE hours_total IS NULL AND hours IS NOT NULL;

-- Drop legacy flat hours column
ALTER TABLE curriculum_item_allocations
  DROP COLUMN IF EXISTS hours;

-- +goose Down

-- Recreate legacy hours column (best-effort)
ALTER TABLE curriculum_item_allocations
  ADD COLUMN IF NOT EXISTS hours INT NULL CHECK (hours IS NULL OR hours BETWEEN 0 AND 10000);

UPDATE curriculum_item_allocations
SET hours = hours_total
WHERE hours IS NULL AND hours_total IS NOT NULL;

ALTER TABLE curriculum_item_allocations
  DROP COLUMN IF EXISTS assessment_type,
  DROP COLUMN IF EXISTS hours_exam,
  DROP COLUMN IF EXISTS hours_independent,
  DROP COLUMN IF EXISTS hours_lab,
  DROP COLUMN IF EXISTS hours_practice,
  DROP COLUMN IF EXISTS hours_lectures,
  DROP COLUMN IF EXISTS hours_total;

DROP INDEX IF EXISTS idx_curriculum_items_index_code;
DROP INDEX IF EXISTS idx_curriculum_items_parent;

ALTER TABLE curriculum_items
  DROP COLUMN IF EXISTS index_code,
  DROP COLUMN IF EXISTS parent_id;
