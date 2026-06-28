-- +goose Up
ALTER TABLE test_tasks ADD COLUMN max_conns_per_host INTEGER
NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE test_tasks DROP COLUMN max_conns_per_host;