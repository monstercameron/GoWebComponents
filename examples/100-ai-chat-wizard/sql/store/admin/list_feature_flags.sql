SELECT flag_key, description, is_enabled, rollout_percent, audience_json, payload_json, updated_by_user_id, updated_at
FROM feature_flags
ORDER BY flag_key ASC;
