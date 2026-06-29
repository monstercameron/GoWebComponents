DELETE FROM auth_oidc_states
WHERE expires_at <= ?;
