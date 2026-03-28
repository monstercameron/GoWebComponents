INSERT INTO incident_updates (
    id,
    incident_id,
    status,
    message,
    is_public,
    published_at,
    created_by_user_id,
    created_at
)
SELECT NULLIF(?, 0), ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM incidents WHERE id = ?)
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?))
ON CONFLICT(id) DO UPDATE SET
    incident_id = excluded.incident_id,
    status = excluded.status,
    message = excluded.message,
    is_public = excluded.is_public,
    published_at = excluded.published_at,
    created_by_user_id = excluded.created_by_user_id,
    created_at = excluded.created_at;
