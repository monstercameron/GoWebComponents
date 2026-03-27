SELECT id, key_id, workspace_id, user_id, label, key_prefix, secret_hash, scopes_json, last_used_at, revoked_at, created_at
FROM api_keys
ORDER BY id DESC
LIMIT ?;
