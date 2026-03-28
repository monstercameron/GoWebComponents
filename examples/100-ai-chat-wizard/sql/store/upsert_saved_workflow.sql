INSERT INTO saved_workflows (
    workspace_id,
    user_id,
    workflow_key,
    name,
    description,
    workflow_json,
    is_public,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE (? = 0 OR EXISTS (SELECT 1 FROM workspaces WHERE id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?))
ON CONFLICT(workflow_key) DO UPDATE SET
    workspace_id = excluded.workspace_id,
    user_id = excluded.user_id,
    name = excluded.name,
    description = excluded.description,
    workflow_json = excluded.workflow_json,
    is_public = excluded.is_public,
    updated_at = excluded.updated_at;
