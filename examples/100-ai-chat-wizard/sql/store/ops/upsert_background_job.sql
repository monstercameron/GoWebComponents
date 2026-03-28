INSERT INTO background_jobs (
    job_key,
    job_type,
    queue_key,
    status,
    attempt_count,
    max_attempts,
    payload_json,
    run_after,
    started_at,
    finished_at,
    error_message,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(job_key) DO UPDATE SET
    job_type = excluded.job_type,
    queue_key = excluded.queue_key,
    status = excluded.status,
    attempt_count = excluded.attempt_count,
    max_attempts = excluded.max_attempts,
    payload_json = excluded.payload_json,
    run_after = excluded.run_after,
    started_at = excluded.started_at,
    finished_at = excluded.finished_at,
    error_message = excluded.error_message,
    updated_at = excluded.updated_at;
