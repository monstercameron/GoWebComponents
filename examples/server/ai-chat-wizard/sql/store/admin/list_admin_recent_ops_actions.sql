SELECT
    id,
    actor_user_id,
    workspace_id,
    event_type,
    target_type,
    target_id,
    summary,
    payload_json,
    created_at
FROM audit_logs
WHERE created_at >= ?
  AND (
      LOWER(TRIM(event_type)) LIKE 'admin.ops.%'
      OR LOWER(TRIM(event_type)) = 'admin.incident.updated'
  )
ORDER BY created_at DESC, id DESC
LIMIT ?;
