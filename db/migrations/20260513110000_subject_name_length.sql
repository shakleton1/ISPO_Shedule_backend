-- +goose Up

ALTER TABLE subjects
  ALTER COLUMN name TYPE VARCHAR(200);

-- +goose Down

ALTER TABLE subjects
  ALTER COLUMN name TYPE VARCHAR(100) USING LEFT(name, 100);
