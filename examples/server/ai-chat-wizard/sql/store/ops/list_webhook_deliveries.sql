SELECT id, endpoint_id, event_type, delivery_key, request_headers_json, request_body_json, response_status, response_body, attempt_count, delivered_at, failed_at, next_retry_at, created_at, updated_at
FROM webhook_deliveries
ORDER BY id DESC
LIMIT ?;
