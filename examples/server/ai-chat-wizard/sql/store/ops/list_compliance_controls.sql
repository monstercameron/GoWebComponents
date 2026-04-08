SELECT
    id,
    workspace_id,
    control_key,
    framework_key,
    status,
    owner_user_id,
    evidence_url,
    reviewed_at,
    updated_at
FROM compliance_controls
ORDER BY id DESC
LIMIT ?;
