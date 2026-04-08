INSERT INTO data_retention_policies (
    workspace_id,
    scope_key,
    retention_days,
    purge_mode,
    legal_hold_json,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = ?)
ON CONFLICT(workspace_id, scope_key) DO UPDATE SET
    retention_days = excluded.retention_days,
    purge_mode = excluded.purge_mode,
    legal_hold_json = excluded.legal_hold_json,
    updated_at = excluded.updated_at;
