INSERT INTO api_keys (
    key_id,
    workspace_id,
    user_id,
    label,
    key_prefix,
    secret_hash,
    scopes_json,
    last_used_at,
    revoked_at,
    created_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, '', '', ?
WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = ?)
  AND EXISTS (SELECT 1 FROM users WHERE id = ?);
