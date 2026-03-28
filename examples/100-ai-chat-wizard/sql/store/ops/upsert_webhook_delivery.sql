INSERT INTO webhook_deliveries (
    endpoint_id,
    event_type,
    delivery_key,
    request_headers_json,
    request_body_json,
    response_status,
    response_body,
    attempt_count,
    delivered_at,
    failed_at,
    next_retry_at,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM webhook_endpoints WHERE id = ?)
ON CONFLICT(delivery_key) DO UPDATE SET
    endpoint_id = excluded.endpoint_id,
    event_type = excluded.event_type,
    request_headers_json = excluded.request_headers_json,
    request_body_json = excluded.request_body_json,
    response_status = excluded.response_status,
    response_body = excluded.response_body,
    attempt_count = excluded.attempt_count,
    delivered_at = excluded.delivered_at,
    failed_at = excluded.failed_at,
    next_retry_at = excluded.next_retry_at,
    updated_at = excluded.updated_at;
