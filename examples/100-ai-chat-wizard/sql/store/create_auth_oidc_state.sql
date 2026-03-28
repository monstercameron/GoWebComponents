INSERT INTO auth_oidc_states (
    provider_key,
    workspace_id,
    session_key,
    state_token_hash,
    nonce_token_hash,
    return_to_url,
    expected_subject,
    expires_at,
    consumed_at,
    created_by_user_id,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE (? = 0 OR EXISTS (SELECT 1 FROM workspaces WHERE id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?));
