-- +goose Up

CREATE TABLE IF NOT EXISTS groups (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE,
  course INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS subjects (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  short_name VARCHAR(30) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS locations (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  is_virtual BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS students (
  id BIGSERIAL PRIMARY KEY,
  login VARCHAR(50) NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  group_id INT REFERENCES groups(id) ON DELETE SET NULL,
  subgroup SMALLINT NULL CHECK (subgroup IN (1,2)),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS schedule_templates (
  id BIGSERIAL PRIMARY KEY,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 5),
  week_parity TEXT NOT NULL CHECK (week_parity IN ('numerator','denominator','both')),
  pair_number SMALLINT NOT NULL CHECK (pair_number BETWEEN 1 AND 8),
  subject_id INT NOT NULL REFERENCES subjects(id),
  location_id INT NOT NULL REFERENCES locations(id),
  teacher_name VARCHAR(100) NOT NULL DEFAULT '',
  subgroup SMALLINT NULL CHECK (subgroup IN (1,2)),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tpl_query ON schedule_templates(group_id, day_of_week, week_parity, pair_number);

CREATE TABLE IF NOT EXISTS schedule_overrides (
  id BIGSERIAL PRIMARY KEY,
  target_date DATE NOT NULL,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  pair_number SMALLINT NOT NULL CHECK (pair_number BETWEEN 1 AND 8),
  action_type TEXT NOT NULL CHECK (action_type IN ('CANCEL','REPLACE','ADD')),
  new_subject_id INT NULL REFERENCES subjects(id),
  new_location_id INT NULL REFERENCES locations(id),
  new_teacher_name VARCHAR(100) NULL,
  comment TEXT NULL,
  subgroup SMALLINT NULL CHECK (subgroup IN (1,2)),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ovr_query ON schedule_overrides(group_id, target_date);

CREATE TABLE IF NOT EXISTS schedule_day_overlays (
  id BIGSERIAL PRIMARY KEY,
  target_date DATE NOT NULL,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  text VARCHAR(255) NOT NULL,
  style_preset VARCHAR(30) NOT NULL DEFAULT 'standard',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(target_date, group_id)
);

CREATE TABLE IF NOT EXISTS calendar_exceptions (
  id BIGSERIAL PRIMARY KEY,
  target_date DATE NOT NULL UNIQUE,
  works_as_day SMALLINT NOT NULL CHECK (works_as_day BETWEEN 0 AND 5),
  comment TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS system_state (
  id SMALLINT PRIMARY KEY,
  schedule_version TIMESTAMPTZ NOT NULL
);

INSERT INTO system_state (id, schedule_version)
VALUES (1, now())
ON CONFLICT (id) DO NOTHING;

-- +goose Down

DROP TABLE IF EXISTS system_state;
DROP TABLE IF EXISTS calendar_exceptions;
DROP TABLE IF EXISTS schedule_day_overlays;
DROP TABLE IF EXISTS schedule_overrides;
DROP TABLE IF EXISTS schedule_templates;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS groups;
