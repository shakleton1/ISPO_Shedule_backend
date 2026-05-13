-- +goose Up

ALTER TABLE subjects
  ALTER COLUMN name TYPE TEXT;

-- +goose Down

ALTER TABLE subjects
  ALTER COLUMN name TYPE VARCHAR(200) USING LEFT(name, 200);
