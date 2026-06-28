-- +goose Up
ALTER TABLE test_tasks ADD COLUMN timeout_millis INTEGER NOT NULL DEFAULT 30000;
UPDATE test_tasks SET timeout_millis = timeout_seconds * 1000;
ALTER TABLE test_tasks DROP COLUMN timeout_seconds;

-- +goose Down
ALTER TABLE test_tasks DROP COLUMN timeout_millis;