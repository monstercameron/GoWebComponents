SELECT
    SUBSTR(created_at, 1, 10) AS usage_day,
    COUNT(*) AS usage_event_count,
    COALESCE(SUM(total_cost_usd), 0) AS total_cost_usd,
    COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
    COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
    COUNT(DISTINCT user_id) AS active_users,
    COUNT(DISTINCT conversation_id) AS active_conversations,
    COUNT(DISTINCT CASE
        WHEN TRIM(COALESCE(client_id, '')) <> '' THEN client_id
        ELSE NULL
    END) AS active_clients
FROM usage_events
WHERE created_at >= ?
GROUP BY SUBSTR(created_at, 1, 10)
ORDER BY usage_day ASC;
