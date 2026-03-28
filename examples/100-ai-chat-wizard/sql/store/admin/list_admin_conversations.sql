SELECT
    c.id,
    c.public_id,
    c.user_id,
    u.email,
    COALESCE(NULLIF(TRIM(up.name), ''), 'User') AS display_name,
    c.started_at,
    COALESCE(
        c.title,
        (SELECT content
         FROM messages
         WHERE conversation_id = c.id
           AND role = 'user'
         ORDER BY id
         LIMIT 1),
        ''
    ) AS preview,
    (SELECT COUNT(*) FROM messages m WHERE m.conversation_id = c.id) AS message_count,
    (SELECT COUNT(*) FROM usage_events ue WHERE ue.conversation_id = c.id) AS usage_event_count,
    COALESCE((SELECT SUM(ue.total_cost_usd) FROM usage_events ue WHERE ue.conversation_id = c.id), 0) AS total_cost_usd,
    COALESCE((SELECT MAX(ue.created_at) FROM usage_events ue WHERE ue.conversation_id = c.id), '') AS last_activity_at
FROM conversations c
JOIN users u ON u.id = c.user_id
LEFT JOIN user_profile up ON up.user_id = c.user_id
ORDER BY c.id DESC
LIMIT ?;
