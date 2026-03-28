SELECT id, workspace_id, provider_key, saml_entrypoint, saml_issuer, saml_certificate_pem, domains_json, is_enabled, updated_at
FROM workspace_sso_configs
WHERE workspace_id = ? AND provider_key = ?
LIMIT 1;
