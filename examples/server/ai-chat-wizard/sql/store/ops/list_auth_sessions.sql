SELECT id, user_id, session_id, token_version, refresh_token_hash, user_agent, ip_address, last_seen_at, expires_at, revoked_at, created_at, updated_at
FROM auth_sessions
ORDER BY id DESC
LIMIT ?;
