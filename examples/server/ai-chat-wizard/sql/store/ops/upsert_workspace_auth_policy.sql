INSERT INTO workspace_auth_policies (
    workspace_id,
    is_password_allowed,
    is_external_login_allowed,
    is_sso_required,
    required_provider_key,
    is_jit_provisioning_allowed,
    is_local_password_qa_allowed,
    updated_by_user_id,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = ?)
ON CONFLICT(workspace_id) DO UPDATE SET
    is_password_allowed = excluded.is_password_allowed,
    is_external_login_allowed = excluded.is_external_login_allowed,
    is_sso_required = excluded.is_sso_required,
    required_provider_key = excluded.required_provider_key,
    is_jit_provisioning_allowed = excluded.is_jit_provisioning_allowed,
    is_local_password_qa_allowed = excluded.is_local_password_qa_allowed,
    updated_by_user_id = excluded.updated_by_user_id,
    updated_at = excluded.updated_at;
