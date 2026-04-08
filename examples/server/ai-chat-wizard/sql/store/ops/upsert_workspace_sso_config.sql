INSERT INTO workspace_sso_configs (
    workspace_id,
    provider_key,
    provider_type,
    oidc_issuer_url,
    oidc_client_id,
    oidc_client_secret_ref,
    oidc_scopes_json,
    oidc_claims_json,
    saml_entrypoint,
    saml_issuer,
    saml_certificate_pem,
    domains_json,
    is_enabled,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = ?)
ON CONFLICT(workspace_id, provider_key) DO UPDATE SET
    provider_type = excluded.provider_type,
    oidc_issuer_url = excluded.oidc_issuer_url,
    oidc_client_id = excluded.oidc_client_id,
    oidc_client_secret_ref = excluded.oidc_client_secret_ref,
    oidc_scopes_json = excluded.oidc_scopes_json,
    oidc_claims_json = excluded.oidc_claims_json,
    saml_entrypoint = excluded.saml_entrypoint,
    saml_issuer = excluded.saml_issuer,
    saml_certificate_pem = excluded.saml_certificate_pem,
    domains_json = excluded.domains_json,
    is_enabled = excluded.is_enabled,
    updated_at = excluded.updated_at;
