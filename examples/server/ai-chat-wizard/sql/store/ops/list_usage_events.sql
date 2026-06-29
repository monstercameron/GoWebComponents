SELECT
    event_id,
    conversation_id,
    provider_id,
    model_id,
    prompt_tokens,
    completion_tokens,
    usage_source,
    provider_request_id,
    input_cost_per_million_usd,
    output_cost_per_million_usd,
    pricing_currency,
    input_cost_usd,
    output_cost_usd,
    total_cost_usd,
    client_id,
    trace_id,
    span_id,
    trace_state,
    status,
    error_message,
    created_at
FROM usage_events
WHERE user_id = ?
ORDER BY id DESC
LIMIT ?;
