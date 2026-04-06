SELECT
    id,
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
FROM workspace_sso_configs
WHERE workspace_id = ? AND provider_key = ?
LIMIT 1;
