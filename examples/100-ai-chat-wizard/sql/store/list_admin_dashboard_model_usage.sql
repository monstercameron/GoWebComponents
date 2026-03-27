SELECT
    provider_id,
    model_id,
    COUNT(*) AS usage_event_count,
    COALESCE(SUM(total_cost_usd), 0) AS total_cost_usd,
    COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
    COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
    COUNT(DISTINCT user_id) AS active_users,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS completed_event_count,
    SUM(CASE WHEN status <> 'completed' THEN 1 ELSE 0 END) AS failed_event_count
FROM usage_events
WHERE created_at >= ?
GROUP BY provider_id, model_id
ORDER BY total_cost_usd DESC, usage_event_count DESC, provider_id ASC, model_id ASC
LIMIT ?;
