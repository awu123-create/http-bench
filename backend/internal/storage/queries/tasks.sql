-- name: CreateTask :one
INSERT INTO test_tasks (
    name,method,url,headers_json,body_type,body_raw,
    concurrency,qps,duration_seconds,timeout_millis,status,
    created_at,max_conns_per_host
)
VALUES (
    ?,?,?,?,?,?,?,?,?,?,?,?,?
)
RETURNING *;

-- name: GetTask :one
SELECT * FROM test_tasks WHERE id = ?;

-- name: UpdateTask :exec
UPDATE test_tasks
SET status = ?,started_at = ?,finished_at = ?
WHERE id = ?;

-- name: ListTasks :many
SELECT * FROM test_tasks;