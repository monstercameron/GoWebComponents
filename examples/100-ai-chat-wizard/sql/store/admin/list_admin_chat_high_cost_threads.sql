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
         FROM messages m
         WHERE m.conversation_id = c.id
           AND LOWER(TRIM(m.role)) = 'user'
         ORDER BY m.id ASC
         LIMIT 1),
        ''
    ) AS preview,
    (SELECT COUNT(*) FROM messages m WHERE m.conversation_id = c.id) AS message_count,
    (SELECT COUNT(*) FROM usage_events ue2 WHERE ue2.conversation_id = c.id) AS usage_event_count,
    COALESCE(SUM(ue.total_cost_usd), 0) AS total_cost_usd,
    COALESCE(MAX(ue.created_at), c.started_at) AS last_activity_at
FROM conversations c
JOIN users u ON u.id = c.user_id
LEFT JOIN user_profile up ON up.user_id = c.user_id
LEFT JOIN usage_events ue ON ue.conversation_id = c.id
GROUP BY
    c.id,
    c.public_id,
    c.user_id,
    u.email,
    up.name,
    c.started_at,
    c.title
HAVING COALESCE(MAX(ue.created_at), c.started_at) >= ?
   AND COALESCE(SUM(ue.total_cost_usd), 0) >= ?
ORDER BY total_cost_usd DESC, last_activity_at DESC, c.id DESC
LIMIT ?;
