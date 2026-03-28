INSERT INTO user_activation_milestones (
    user_id,
    milestone_key,
    status,
    achieved_at,
    metadata_json,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM users WHERE id = ?)
ON CONFLICT(user_id, milestone_key) DO UPDATE SET
    status = excluded.status,
    achieved_at = excluded.achieved_at,
    metadata_json = excluded.metadata_json,
    updated_at = excluded.updated_at;
