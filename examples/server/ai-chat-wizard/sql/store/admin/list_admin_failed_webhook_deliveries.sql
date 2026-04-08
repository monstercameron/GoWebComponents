SELECT
    id,
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
FROM webhook_deliveries
WHERE updated_at >= ?
  AND TRIM(COALESCE(failed_at, '')) <> ''
  AND TRIM(COALESCE(delivered_at, '')) = ''
ORDER BY updated_at DESC, id DESC
LIMIT ?;
