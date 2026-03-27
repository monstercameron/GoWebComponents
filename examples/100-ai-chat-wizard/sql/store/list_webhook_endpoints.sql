SELECT id, workspace_id, label, target_url, secret_hash, events_json, is_enabled, last_delivery_at, failure_count, created_at, updated_at
FROM webhook_endpoints
ORDER BY id DESC
LIMIT ?;
