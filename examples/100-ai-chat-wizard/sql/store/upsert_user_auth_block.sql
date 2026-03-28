INSERT INTO user_auth_blocks (
    user_id,
    block_key,
    block_source,
    block_reason,
    blocked_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, block_key) DO UPDATE SET
    block_source = excluded.block_source,
    block_reason = excluded.block_reason,
    blocked_at = excluded.blocked_at,
    updated_at = excluded.updated_at;
