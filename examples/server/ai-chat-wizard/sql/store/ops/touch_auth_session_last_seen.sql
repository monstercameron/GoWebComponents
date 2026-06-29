UPDATE auth_sessions
SET
    user_agent = ?,
    ip_address = ?,
    last_seen_at = ?,
    expires_at = ?,
    updated_at = ?
WHERE session_id = ?;
