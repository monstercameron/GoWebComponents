INSERT INTO workspace_cost_guardrails (
    workspace_id,
    guardrail_key,
    daily_budget_cents,
    monthly_budget_cents,
    max_cost_per_request_cents,
    alert_threshold_percent,
    action_mode,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(workspace_id, guardrail_key) DO UPDATE SET
    daily_budget_cents = excluded.daily_budget_cents,
    monthly_budget_cents = excluded.monthly_budget_cents,
    max_cost_per_request_cents = excluded.max_cost_per_request_cents,
    alert_threshold_percent = excluded.alert_threshold_percent,
    action_mode = excluded.action_mode,
    updated_at = excluded.updated_at;
