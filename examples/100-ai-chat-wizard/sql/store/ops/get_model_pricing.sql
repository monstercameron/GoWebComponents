SELECT provider_id, input_cost_per_million_usd, output_cost_per_million_usd, pricing_currency
FROM model_catalog
WHERE id = ?
LIMIT 1;
