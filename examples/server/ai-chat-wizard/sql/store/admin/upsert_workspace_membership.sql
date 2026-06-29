INSERT INTO workspace_memberships (
    workspace_id,
    user_id,
    role_key,
    status,
    invited_by_user_id,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = ?)
  AND EXISTS (SELECT 1 FROM users WHERE id = ?)
ON CONFLICT(workspace_id, user_id) DO UPDATE SET
    role_key = excluded.role_key,
    status = excluded.status,
    invited_by_user_id = excluded.invited_by_user_id,
    updated_at = excluded.updated_at;
