SELECT
    id,
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
FROM background_jobs
WHERE updated_at >= ?
  AND LOWER(TRIM(status)) IN ('failed', 'error')
ORDER BY updated_at DESC, id DESC
LIMIT ?;
