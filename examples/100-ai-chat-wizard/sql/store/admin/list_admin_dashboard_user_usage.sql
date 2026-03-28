SELECT
    u.id,
    u.email,
    COALESCE(NULLIF(TRIM(up.name), ''), 'User') AS display_name,
    (SELECT COUNT(*) FROM conversations c WHERE c.user_id = u.id) AS conversation_count,
    (SELECT COUNT(*)
     FROM messages m
     JOIN conversations c ON c.id = m.conversation_id
     WHERE c.user_id = u.id) AS message_count,
    COUNT(*) AS usage_event_count,
    COALESCE(SUM(ue.total_cost_usd), 0) AS total_cost_usd,
    COALESCE(SUM(ue.prompt_tokens), 0) AS prompt_tokens,
    COALESCE(SUM(ue.completion_tokens), 0) AS completion_tokens,
    COALESCE(MAX(ue.created_at), '') AS last_seen_at
FROM usage_events ue
JOIN users u ON u.id = ue.user_id
LEFT JOIN user_profile up ON up.user_id = u.id
WHERE ue.created_at >= ?
GROUP BY u.id, u.email, up.name
ORDER BY total_cost_usd DESC, usage_event_count DESC, u.id DESC
LIMIT ?;
