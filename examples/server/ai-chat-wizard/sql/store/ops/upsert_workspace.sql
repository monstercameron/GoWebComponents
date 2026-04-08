INSERT INTO workspaces (
    workspace_key,
    slug,
    name,
    plan_code,
    status,
    owner_user_id,
    settings_json,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM users WHERE id = ?)
ON CONFLICT(workspace_key) DO UPDATE SET
    slug = excluded.slug,
    name = excluded.name,
    plan_code = excluded.plan_code,
    status = excluded.status,
    owner_user_id = excluded.owner_user_id,
    settings_json = excluded.settings_json,
    updated_at = excluded.updated_at;
