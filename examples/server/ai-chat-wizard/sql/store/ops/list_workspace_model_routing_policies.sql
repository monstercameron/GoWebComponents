SELECT
    id,
    workspace_id,
    policy_key,
    default_model_id,
    fallback_model_id,
    max_input_cost_per_million_usd,
    max_output_cost_per_million_usd,
    requires_approval,
    rules_json,
    updated_at
FROM workspace_model_routing_policies
ORDER BY id DESC
LIMIT ?;
