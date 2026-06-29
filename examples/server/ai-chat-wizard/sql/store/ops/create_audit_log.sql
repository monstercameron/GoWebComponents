INSERT INTO audit_logs (
    actor_user_id,
    workspace_id,
    event_type,
    target_type,
    target_id,
    summary,
    payload_json,
    created_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?
WHERE (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM workspaces WHERE id = ?));
