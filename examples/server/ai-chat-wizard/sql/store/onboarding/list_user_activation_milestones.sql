SELECT id, user_id, milestone_key, status, achieved_at, metadata_json, updated_at
FROM user_activation_milestones
ORDER BY id DESC
LIMIT ?;
