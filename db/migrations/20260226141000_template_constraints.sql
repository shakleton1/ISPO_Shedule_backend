-- +goose Up

-- Deduplicate templates deterministically: keep newest per key.
WITH ranked AS (
  SELECT
    id,
    ROW_NUMBER() OVER (
      PARTITION BY group_id, day_of_week, week_parity, pair_number, COALESCE(subgroup, 0)
      ORDER BY updated_at DESC, id DESC
    ) AS rn
  FROM schedule_templates
)
DELETE FROM schedule_templates st
USING ranked r
WHERE st.id = r.id
  AND r.rn > 1;

-- Prevent multiple templates on same slot/subgroup.
CREATE UNIQUE INDEX IF NOT EXISTS uidx_templates_one_per_pair
ON schedule_templates (group_id, day_of_week, week_parity, pair_number, COALESCE(subgroup, 0));

-- +goose Down

DROP INDEX IF EXISTS uidx_templates_one_per_pair;
