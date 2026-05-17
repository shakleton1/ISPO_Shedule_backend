-- +goose Up
-- +goose StatementBegin

ALTER TABLE academic_calendar_weeks
  ALTER COLUMN activity_code TYPE VARCHAR(50);

ALTER TABLE academic_calendar_weeks
  ALTER COLUMN activity_name TYPE VARCHAR(200);

ALTER TABLE study_activities
  DROP CONSTRAINT IF EXISTS chk_study_activities_kind;

ALTER TABLE study_activities
  ADD CONSTRAINT chk_study_activities_kind
  CHECK (activity_kind IN ('TEACHING', 'PRACTICE', 'EXAM', 'GIA', 'VACATION', 'EVENT', 'OTHER'));

INSERT INTO study_activities (code, name, activity_kind, allows_lessons)
VALUES
  ('TEACHING', 'Учебные занятия', 'TEACHING', TRUE),
  ('PRACTICE_EDU', 'Учебная практика', 'PRACTICE', FALSE),
  ('PRACTICE_PROD', 'Производственная практика', 'PRACTICE', FALSE),
  ('PRACTICE_PREGRAD', 'Преддипломная практика', 'PRACTICE', FALSE),
  ('EXAM', 'Экзаменационная сессия', 'EXAM', TRUE),
  ('GIA', 'Государственная итоговая аттестация', 'GIA', FALSE),
  ('VACATION', 'Каникулы', 'VACATION', FALSE),
  ('EVENT', 'Мероприятие', 'EVENT', TRUE),
  ('OTHER', 'Прочее', 'OTHER', TRUE)
ON CONFLICT (code)
DO UPDATE SET
  name = EXCLUDED.name,
  activity_kind = EXCLUDED.activity_kind,
  allows_lessons = EXCLUDED.allows_lessons;

UPDATE academic_calendar_weeks
SET activity_code = CASE
  WHEN activity_code = 'PRACTICE' AND lower(COALESCE(activity_name, '')) LIKE '%учебн%' THEN 'PRACTICE_EDU'
  WHEN activity_code = 'PRACTICE' AND lower(COALESCE(activity_name, '')) LIKE '%преддиплом%' THEN 'PRACTICE_PREGRAD'
  WHEN activity_code = 'PRACTICE' THEN 'PRACTICE_PROD'
  ELSE activity_code
END
WHERE activity_code = 'PRACTICE';

UPDATE academic_calendar_day_overrides
SET activity_code = CASE
  WHEN activity_code = 'PRACTICE' AND lower(COALESCE(activity_name, '')) LIKE '%учебн%' THEN 'PRACTICE_EDU'
  WHEN activity_code = 'PRACTICE' AND lower(COALESCE(activity_name, '')) LIKE '%преддиплом%' THEN 'PRACTICE_PREGRAD'
  WHEN activity_code = 'PRACTICE' THEN 'PRACTICE_PROD'
  ELSE activity_code
END
WHERE activity_code = 'PRACTICE';

WITH generic AS (
  SELECT id FROM study_activities WHERE code = 'PRACTICE'
),
targets AS (
  SELECT
    (SELECT id FROM study_activities WHERE code = 'PRACTICE_EDU') AS edu_id,
    (SELECT id FROM study_activities WHERE code = 'PRACTICE_PROD') AS prod_id,
    (SELECT id FROM study_activities WHERE code = 'PRACTICE_PREGRAD') AS pregrad_id
)
UPDATE study_calendar_weeks scw
SET activity_id = CASE
  WHEN lower(COALESCE(scw.comment, '')) LIKE '%учебн%' THEN targets.edu_id
  WHEN lower(COALESCE(scw.comment, '')) LIKE '%преддиплом%' THEN targets.pregrad_id
  ELSE targets.prod_id
END
FROM generic, targets
WHERE scw.activity_id = generic.id;

DELETE FROM study_activities
WHERE code = 'PRACTICE'
  AND NOT EXISTS (
    SELECT 1 FROM study_calendar_weeks WHERE activity_id = study_activities.id
  );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

UPDATE academic_calendar_weeks
SET activity_code = 'PRACTICE'
WHERE activity_code IN ('PRACTICE_EDU', 'PRACTICE_PROD', 'PRACTICE_PREGRAD');

UPDATE academic_calendar_day_overrides
SET activity_code = 'PRACTICE'
WHERE activity_code IN ('PRACTICE_EDU', 'PRACTICE_PROD', 'PRACTICE_PREGRAD');

INSERT INTO study_activities (code, name, activity_kind, allows_lessons)
VALUES ('PRACTICE', 'Практика', 'PRACTICE', FALSE)
ON CONFLICT (code)
DO UPDATE SET
  name = EXCLUDED.name,
  activity_kind = EXCLUDED.activity_kind,
  allows_lessons = EXCLUDED.allows_lessons;

UPDATE study_calendar_weeks
SET activity_id = (SELECT id FROM study_activities WHERE code = 'PRACTICE')
WHERE activity_id IN (
  SELECT id FROM study_activities WHERE code IN ('PRACTICE_EDU', 'PRACTICE_PROD', 'PRACTICE_PREGRAD')
);

DELETE FROM study_activities
WHERE code IN ('PRACTICE_EDU', 'PRACTICE_PROD', 'PRACTICE_PREGRAD', 'GIA', 'EVENT', 'OTHER');

ALTER TABLE study_activities
  DROP CONSTRAINT IF EXISTS chk_study_activities_kind;

ALTER TABLE study_activities
  ADD CONSTRAINT chk_study_activities_kind
  CHECK (activity_kind IN ('TEACHING', 'PRACTICE', 'EXAM', 'VACATION', 'EVENT', 'OTHER'));

ALTER TABLE academic_calendar_weeks
  ALTER COLUMN activity_code TYPE VARCHAR(10);

ALTER TABLE academic_calendar_weeks
  ALTER COLUMN activity_name TYPE VARCHAR(100);

-- +goose StatementEnd
