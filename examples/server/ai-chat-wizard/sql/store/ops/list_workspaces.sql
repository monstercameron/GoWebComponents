SELECT id, workspace_key, slug, name, plan_code, status, owner_user_id, settings_json, created_at, updated_at
FROM workspaces
ORDER BY id DESC
LIMIT ?;
