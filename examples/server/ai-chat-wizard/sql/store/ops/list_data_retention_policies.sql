SELECT
    id,
    workspace_id,
    scope_key,
    retention_days,
    purge_mode,
    legal_hold_json,
    updated_at
FROM data_retention_policies
ORDER BY id DESC
LIMIT ?;
