INSERT INTO workspace_invitations (
    workspace_id,
    email,
    role_key,
    invitation_token_hash,
    invited_by_user_id,
    status,
    expires_at,
    accepted_at,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = ?)
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?))
ON CONFLICT(invitation_token_hash) DO UPDATE SET
    workspace_id = excluded.workspace_id,
    email = excluded.email,
    role_key = excluded.role_key,
    invited_by_user_id = excluded.invited_by_user_id,
    status = excluded.status,
    expires_at = excluded.expires_at,
    accepted_at = excluded.accepted_at,
    updated_at = excluded.updated_at;
