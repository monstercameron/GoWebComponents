SELECT id, workspace_id, scope_key, retention_days, purge_mode, legal_hold_json, updated_at
FROM data_retention_policies
WHERE workspace_id = ? AND scope_key = ?
LIMIT 1;
