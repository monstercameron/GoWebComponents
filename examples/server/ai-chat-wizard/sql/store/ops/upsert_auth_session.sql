INSERT INTO auth_sessions (
    user_id,
    session_id,
    token_version,
    refresh_token_hash,
    user_agent,
    ip_address,
    last_seen_at,
    expires_at,
    revoked_at,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(session_id) DO UPDATE SET
    user_id = excluded.user_id,
    token_version = excluded.token_version,
    refresh_token_hash = excluded.refresh_token_hash,
    user_agent = excluded.user_agent,
    ip_address = excluded.ip_address,
    last_seen_at = excluded.last_seen_at,
    expires_at = excluded.expires_at,
    revoked_at = excluded.revoked_at,
    updated_at = excluded.updated_at;
