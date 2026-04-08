INSERT INTO feature_flags (
    flag_key,
    description,
    is_enabled,
    rollout_percent,
    audience_json,
    payload_json,
    updated_by_user_id,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(flag_key) DO UPDATE SET
    description = excluded.description,
    is_enabled = excluded.is_enabled,
    rollout_percent = excluded.rollout_percent,
    audience_json = excluded.audience_json,
    payload_json = excluded.payload_json,
    updated_by_user_id = excluded.updated_by_user_id,
    updated_at = excluded.updated_at;
