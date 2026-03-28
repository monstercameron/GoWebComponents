SELECT
    plan_code,
    plan_name,
    plan_rank,
    is_active,
    monthly_base_cents,
    yearly_base_cents,
    included_tokens_monthly,
    included_seats,
    max_seats,
    supports_priority,
    supports_team_workspace,
    supports_sso,
    created_at,
    updated_at
FROM billing_plans
ORDER BY plan_rank ASC, plan_code ASC
LIMIT ?;
