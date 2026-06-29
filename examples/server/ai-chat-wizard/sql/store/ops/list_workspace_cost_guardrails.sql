SELECT
    id,
    workspace_id,
    guardrail_key,
    daily_budget_cents,
    monthly_budget_cents,
    max_cost_per_request_cents,
    alert_threshold_percent,
    action_mode,
    updated_at
FROM workspace_cost_guardrails
ORDER BY id DESC
LIMIT ?;
