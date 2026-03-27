SELECT id, actor_user_id, workspace_id, event_type, target_type, target_id, summary, payload_json, created_at
FROM audit_logs
ORDER BY id DESC
LIMIT ?;
