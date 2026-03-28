SELECT
    id,
    workspace_id,
    provider_key,
    saml_entrypoint,
    saml_issuer,
    saml_certificate_pem,
    domains_json,
    is_enabled,
    updated_at
FROM workspace_sso_configs
ORDER BY id DESC
LIMIT ?;
