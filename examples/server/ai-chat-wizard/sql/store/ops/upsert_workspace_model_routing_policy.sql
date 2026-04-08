INSERT INTO workspace_model_routing_policies (
    workspace_id,
    policy_key,
    default_model_id,
    fallback_model_id,
    max_input_cost_per_million_usd,
    max_output_cost_per_million_usd,
    requires_approval,
    rules_json,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(workspace_id, policy_key) DO UPDATE SET
    default_model_id = excluded.default_model_id,
    fallback_model_id = excluded.fallback_model_id,
    max_input_cost_per_million_usd = excluded.max_input_cost_per_million_usd,
    max_output_cost_per_million_usd = excluded.max_output_cost_per_million_usd,
    requires_approval = excluded.requires_approval,
    rules_json = excluded.rules_json,
    updated_at = excluded.updated_at;
