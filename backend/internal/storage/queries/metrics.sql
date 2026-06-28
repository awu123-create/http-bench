-- name: CreateMetricBucket :one
INSERT INTO metrics_buckets (   
    test_id,second_offset,requests,success,failure,
    avg_latency_ms,min_latency_ms,max_latency_ms,
    errors_json,status_code_json,created_at
)
VALUES (
    ?,?,?,?,?,?,?,?,?,?,?
)
RETURNING *;

-- name: ListMetricsBucketsByTestID :many
SELECT * FROM metrics_buckets 
WHERE test_id = ?
ORDER BY second_offset ASC;

-- name: CreateTestResult :one
INSERT INTO test_results (
    test_id,total_requests,total_success,total_failure,
    avg_latency_ms,min_latency_ms,max_latency_ms,p99_latency_ms,
    p95_latency_ms,p90_latency_ms,p50_latency_ms,errors_json,status_code_json,
    created_at
)
VALUES (
    ?,?,?,?,?,?,?,?,?,?,?,?,?,?
)
RETURNING *;

-- name: GetTestResult :one
SELECT * FROM test_results WHERE test_id = ?;