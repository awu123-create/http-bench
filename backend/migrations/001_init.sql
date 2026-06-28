-- +goose Up
-- 压测任务表
CREATE TABLE IF NOT EXISTS test_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    method TEXT NOT NULL,
    url TEXT NOT NULL,
    headers_json TEXT NOT NULL DEFAULT '{}',
    body_type TEXT NOT NULL DEFAULT 'none',
    body_raw TEXT NOT NULL DEFAULT '',
    concurrency INTEGER NOT NULL,
    qps INTEGER NOT NULL,
    duration_seconds INTEGER NOT NULL,
    timeout_seconds INTEGER NOT NULL,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT
);

-- 每秒聚合指标表
CREATE TABLE IF NOT EXISTS metrics_buckets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    test_id INTEGER NOT NULL,
    second_offset INTEGER NOT NULL,
    requests INTEGER NOT NULL,
    success INTEGER NOT NULL,
    failure INTEGER NOT NULL,
    avg_latency_ms REAL NOT NULL,
    min_latency_ms INTEGER NOT NULL,
    max_latency_ms INTEGER NOT NULL,
    errors_json TEXT NOT NULL DEFAULT '{}',
    status_code_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    FOREIGN KEY (test_id) REFERENCES test_tasks(id)
);

-- 压测任务结果表
CREATE TABLE IF NOT EXISTS test_results (
    test_id INTEGER NOT NULL,
    total_requests INTEGER NOT NULL,
    total_success INTEGER NOT NULL,
    total_failure INTEGER NOT NULL,
    avg_latency_ms REAL NOT NULL,
    min_latency_ms INTEGER NOT NULL,
    max_latency_ms INTEGER NOT NULL,
    p99_latency_ms INTEGER NOT NULL,
    p95_latency_ms INTEGER NOT NULL,
    p90_latency_ms INTEGER NOT NULL,
    p50_latency_ms INTEGER NOT NULL,
    errors_json TEXT NOT NULL DEFAULT '{}',
    status_code_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    FOREIGN KEY (test_id) REFERENCES test_tasks(id)
);

-- +goose Down
DROP TABLE IF EXISTS test_results;
DROP TABLE IF EXISTS metrics_buckets;
DROP TABLE IF EXISTS test_tasks;