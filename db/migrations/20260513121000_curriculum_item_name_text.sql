-- +goose Up

ALTER TABLE curriculum_items
  ALTER COLUMN name TYPE TEXT;

-- +goose Down

ALTER TABLE curriculum_items
  ALTER COLUMN name TYPE VARCHAR(200) USING LEFT(name, 200);
