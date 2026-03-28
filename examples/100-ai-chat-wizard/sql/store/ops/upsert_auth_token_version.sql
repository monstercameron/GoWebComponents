INSERT INTO auth_token_versions (user_id, token_version, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
    token_version = excluded.token_version,
    updated_at = excluded.updated_at;
