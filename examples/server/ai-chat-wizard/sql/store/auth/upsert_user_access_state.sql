INSERT INTO user_access_states (
    user_id,
    status,
    reason,
    disabled_by_user_id,
    disabled_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
    status = excluded.status,
    reason = excluded.reason,
    disabled_by_user_id = excluded.disabled_by_user_id,
    disabled_at = excluded.disabled_at,
    updated_at = excluded.updated_at;
