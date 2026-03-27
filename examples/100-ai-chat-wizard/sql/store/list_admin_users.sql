SELECT
    u.id,
    u.email,
    COALESCE(NULLIF(TRIM(up.name), ''), 'User') AS display_name,
    u.created_at,
    (SELECT COUNT(*) FROM conversations c WHERE c.user_id = u.id) AS conversation_count,
    (SELECT COUNT(*)
     FROM messages m
     JOIN conversations c ON c.id = m.conversation_id
     WHERE c.user_id = u.id) AS message_count,
    (SELECT COUNT(*) FROM usage_events ue WHERE ue.user_id = u.id) AS usage_event_count,
    COALESCE((SELECT SUM(ue.total_cost_usd) FROM usage_events ue WHERE ue.user_id = u.id), 0) AS total_cost_usd,
    COALESCE((SELECT MAX(ue.created_at) FROM usage_events ue WHERE ue.user_id = u.id), '') AS last_seen_at
FROM users u
LEFT JOIN user_profile up ON up.user_id = u.id
ORDER BY u.id DESC
LIMIT ?;
