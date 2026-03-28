INSERT INTO provider_usage_daily_rollups (
    rollup_day,
    provider_id,
    usage_event_count,
    total_cost_usd,
    prompt_tokens,
    completion_tokens,
    completed_event_count,
    failed_event_count,
    updated_at
)
SELECT
    date(created_at)                                                    AS rollup_day,
    provider_id                                                         AS provider_id,
    COUNT(*)                                                            AS usage_event_count,
    COALESCE(SUM(total_cost_usd), 0)                                    AS total_cost_usd,
    COALESCE(SUM(prompt_tokens), 0)                                     AS prompt_tokens,
    COALESCE(SUM(completion_tokens), 0)                                 AS completion_tokens,
    COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) AS completed_event_count,
    COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0)     AS failed_event_count,
    ?                                                                    AS updated_at
FROM usage_events
WHERE created_at >= ?
GROUP BY date(created_at), provider_id
ON CONFLICT(rollup_day, provider_id) DO UPDATE SET
    usage_event_count = excluded.usage_event_count,
    total_cost_usd = excluded.total_cost_usd,
    prompt_tokens = excluded.prompt_tokens,
    completion_tokens = excluded.completion_tokens,
    completed_event_count = excluded.completed_event_count,
    failed_event_count = excluded.failed_event_count,
    updated_at = excluded.updated_at;
