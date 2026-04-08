SELECT
    workspace_id,
    is_password_allowed,
    is_external_login_allowed,
    is_sso_required,
    required_provider_key,
    is_jit_provisioning_allowed,
    is_local_password_qa_allowed,
    updated_by_user_id,
    updated_at
FROM workspace_auth_policies
WHERE workspace_id = ?
LIMIT 1;
