SELECT
    rollup_day,
    provider_id,
    usage_event_count,
    total_cost_usd,
    prompt_tokens,
    completion_tokens,
    completed_event_count,
    failed_event_count,
    updated_at
FROM provider_usage_daily_rollups
WHERE rollup_day >= DATE(?)
ORDER BY rollup_day DESC, provider_id ASC
LIMIT ?;
