UPDATE auth_oidc_states
SET consumed_at = ?, updated_at = ?
WHERE state_token_hash = ? AND TRIM(COALESCE(consumed_at, '')) = '';
