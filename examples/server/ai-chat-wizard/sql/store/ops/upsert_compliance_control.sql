INSERT INTO compliance_controls (
    workspace_id,
    control_key,
    framework_key,
    status,
    owner_user_id,
    evidence_url,
    reviewed_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM workspaces WHERE id = ?)
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?))
ON CONFLICT(workspace_id, control_key, framework_key) DO UPDATE SET
    status = excluded.status,
    owner_user_id = excluded.owner_user_id,
    evidence_url = excluded.evidence_url,
    reviewed_at = excluded.reviewed_at,
    updated_at = excluded.updated_at;
