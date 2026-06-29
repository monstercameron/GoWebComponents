SELECT
    id,
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
FROM auth_oidc_states
WHERE state_token_hash = ?
LIMIT 1;
