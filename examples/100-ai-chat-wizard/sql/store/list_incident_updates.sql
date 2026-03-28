SELECT id, incident_id, status, message, is_public, published_at, created_by_user_id, created_at
FROM incident_updates
ORDER BY id DESC
LIMIT ?;
