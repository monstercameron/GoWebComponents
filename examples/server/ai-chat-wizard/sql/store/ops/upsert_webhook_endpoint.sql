INSERT INTO webhook_endpoints (
    workspace_id,
    label,
    target_url,
    secret_hash,
    events_json,
    is_enabled,
    last_delivery_at,
    failure_count,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = ?)
ON CONFLICT(workspace_id, target_url) DO UPDATE SET
    label = excluded.label,
    secret_hash = excluded.secret_hash,
    events_json = excluded.events_json,
    is_enabled = excluded.is_enabled,
    last_delivery_at = excluded.last_delivery_at,
    failure_count = excluded.failure_count,
    updated_at = excluded.updated_at;
