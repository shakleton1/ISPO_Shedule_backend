-- +goose Up

-- 1) Deduplicate existing conflicts deterministically (keep best per key).
WITH ranked AS (
  SELECT
    id,
    ROW_NUMBER() OVER (
      PARTITION BY group_id, target_date, pair_number, COALESCE(subgroup, 0)
      ORDER BY
        (CASE action_type WHEN 'CANCEL' THEN 0 WHEN 'REPLACE' THEN 1 WHEN 'ADD' THEN 2 ELSE 3 END) ASC,
        updated_at DESC,
        id DESC
    ) AS rn
  FROM schedule_overrides
)
DELETE FROM schedule_overrides so
USING ranked r
WHERE so.id = r.id
  AND r.rn > 1;

-- 2) Enforce business constraints.
ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_overrides_cancel_has_no_new_fields;
ALTER TABLE schedule_overrides
  ADD CONSTRAINT chk_overrides_cancel_has_no_new_fields
  CHECK (
    action_type <> 'CANCEL'
    OR (new_subject_id IS NULL AND new_location_id IS NULL AND new_teacher_name IS NULL)
  );

ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_overrides_add_requires_subject_location;
ALTER TABLE schedule_overrides
  ADD CONSTRAINT chk_overrides_add_requires_subject_location
  CHECK (
    action_type <> 'ADD'
    OR (new_subject_id IS NOT NULL AND new_location_id IS NOT NULL)
  );

-- 3) Prevent multiple overrides on the same pair/subgroup.
CREATE UNIQUE INDEX IF NOT EXISTS uidx_overrides_one_per_pair
ON schedule_overrides (group_id, target_date, pair_number, COALESCE(subgroup, 0));

-- +goose Down

DROP INDEX IF EXISTS uidx_overrides_one_per_pair;

ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_overrides_add_requires_subject_location;
ALTER TABLE schedule_overrides
  DROP CONSTRAINT IF EXISTS chk_overrides_cancel_has_no_new_fields;
