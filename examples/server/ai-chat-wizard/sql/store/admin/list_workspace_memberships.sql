SELECT id, workspace_id, user_id, role_key, status, invited_by_user_id, created_at, updated_at
FROM workspace_memberships
ORDER BY id DESC
LIMIT ?;
