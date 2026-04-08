SELECT id, workspace_id, email, role_key, invitation_token_hash, invited_by_user_id, status, expires_at, accepted_at, created_at, updated_at
FROM workspace_invitations
ORDER BY id DESC
LIMIT ?;
