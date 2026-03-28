UPDATE auth_sessions
SET
    revoked_at = ?,
    updated_at = ?
WHERE session_id = ?;
