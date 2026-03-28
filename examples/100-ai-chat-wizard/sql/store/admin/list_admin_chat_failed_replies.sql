SELECT
    ue.event_id,
    ue.user_id,
    u.email,
    COALESCE(NULLIF(TRIM(up.name), ''), 'User') AS display_name,
    ue.conversation_id,
    COALESCE(c.public_id, '') AS conversation_public_id,
    COALESCE(c.title, '') AS conversation_title,
    ue.provider_id,
    ue.model_id,
    ue.prompt_tokens,
    ue.completion_tokens,
    ue.usage_source,
    ue.provider_request_id,
    ue.input_cost_per_million_usd,
    ue.output_cost_per_million_usd,
    ue.pricing_currency,
    ue.input_cost_usd,
    ue.output_cost_usd,
    ue.total_cost_usd,
    ue.client_id,
    ue.trace_id,
    ue.span_id,
    ue.trace_state,
    ue.status,
    ue.error_message,
    ue.created_at
FROM usage_events ue
JOIN users u ON u.id = ue.user_id
LEFT JOIN user_profile up ON up.user_id = u.id
LEFT JOIN conversations c ON c.id = ue.conversation_id
WHERE ue.created_at >= ?
  AND LOWER(TRIM(ue.status)) = 'failed'
ORDER BY ue.created_at DESC, ue.id DESC
LIMIT ?;
