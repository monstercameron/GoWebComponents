SELECT user_id, status, reason, disabled_by_user_id, disabled_at, updated_at
FROM user_access_states
WHERE user_id = ?;
