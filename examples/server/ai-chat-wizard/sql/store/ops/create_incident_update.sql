INSERT INTO incident_updates (
    incident_id,
    status,
    message,
    is_public,
    published_at,
    created_by_user_id,
    created_at
)
SELECT ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM incidents WHERE id = ?)
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?));
