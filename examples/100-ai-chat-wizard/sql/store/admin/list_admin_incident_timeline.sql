SELECT
    id,
    incident_id,
    status,
    message,
    is_public,
    published_at,
    created_by_user_id,
    created_at
FROM incident_updates
WHERE created_at >= ?
ORDER BY created_at DESC, id DESC
LIMIT ?;
