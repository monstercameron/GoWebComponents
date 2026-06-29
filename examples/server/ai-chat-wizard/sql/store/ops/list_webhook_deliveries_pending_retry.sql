SELECT id, endpoint_id, event_type, delivery_key, request_headers_json, request_body_json, response_status, response_body, attempt_count, delivered_at, failed_at, next_retry_at, created_at, updated_at
FROM webhook_deliveries
WHERE TRIM(COALESCE(next_retry_at, '')) <> ''
  AND next_retry_at <= ?
ORDER BY next_retry_at ASC, id ASC
LIMIT ?;
