INSERT INTO workspace_sso_configs (
    workspace_id,
    provider_key,
    saml_entrypoint,
    saml_issuer,
    saml_certificate_pem,
    domains_json,
    is_enabled,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = ?)
ON CONFLICT(workspace_id, provider_key) DO UPDATE SET
    saml_entrypoint = excluded.saml_entrypoint,
    saml_issuer = excluded.saml_issuer,
    saml_certificate_pem = excluded.saml_certificate_pem,
    domains_json = excluded.domains_json,
    is_enabled = excluded.is_enabled,
    updated_at = excluded.updated_at;
