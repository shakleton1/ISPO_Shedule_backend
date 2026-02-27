-- +goose Up

-- 1) teacher_subjects: which teacher can teach which subject.
-- Minimal schema by request: (teacher_id, subject_id).
CREATE TABLE IF NOT EXISTS teacher_subjects (
  teacher_id INT NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
  subject_id INT NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  PRIMARY KEY (teacher_id, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_teacher_subjects_subject_id ON teacher_subjects(subject_id);

-- 2) course_assignments: who teaches a subject for a group in a semester.
CREATE TABLE IF NOT EXISTS course_assignments (
  id BIGSERIAL PRIMARY KEY,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  semester SMALLINT NOT NULL CHECK (semester BETWEEN 1 AND 12),
  subject_id INT NOT NULL REFERENCES subjects(id) ON DELETE RESTRICT,
  teacher_id INT NULL REFERENCES teachers(id) ON DELETE SET NULL,
  location_id INT NULL REFERENCES locations(id) ON DELETE SET NULL,
  curriculum_item_id BIGINT NULL REFERENCES curriculum_items(id) ON DELETE SET NULL,
  subgroup SMALLINT NULL CHECK (subgroup IN (1, 2)),
  notes TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TRIGGER IF EXISTS trg_course_assignments_set_updated_at ON course_assignments;
CREATE TRIGGER trg_course_assignments_set_updated_at
BEFORE UPDATE ON course_assignments
FOR EACH ROW
EXECUTE FUNCTION ispo_set_updated_at();

-- Enforce one assignment per key, NULL-safe for subgroup.
CREATE UNIQUE INDEX IF NOT EXISTS uidx_course_assignments_key
ON course_assignments (group_id, semester, subject_id, COALESCE(subgroup, 0));

CREATE INDEX IF NOT EXISTS idx_course_assignments_group_semester ON course_assignments (group_id, semester);
CREATE INDEX IF NOT EXISTS idx_course_assignments_teacher_id ON course_assignments (teacher_id);

-- +goose Down

DROP INDEX IF EXISTS idx_course_assignments_teacher_id;
DROP INDEX IF EXISTS idx_course_assignments_group_semester;
DROP INDEX IF EXISTS uidx_course_assignments_key;

DROP TRIGGER IF EXISTS trg_course_assignments_set_updated_at ON course_assignments;
DROP TABLE IF EXISTS course_assignments;

DROP INDEX IF EXISTS idx_teacher_subjects_subject_id;
DROP TABLE IF EXISTS teacher_subjects;
