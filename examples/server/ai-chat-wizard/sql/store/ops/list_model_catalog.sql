SELECT
    id,
    provider_id,
    provider_label,
    label,
    note,
    description,
    supports_thinking,
    supports_speech,
    input_cost_per_million_usd,
    output_cost_per_million_usd,
    pricing_currency,
    max_output_tokens,
    throughput_tokens_per_second,
    onboarding_ready,
    is_default,
    use_for_title_generation,
    use_for_memory_extraction,
    sort_order
FROM model_catalog
ORDER BY sort_order ASC, provider_id ASC, label ASC
