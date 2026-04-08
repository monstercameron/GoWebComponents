SELECT
    plan_code,
    plan_name,
    plan_rank,
    is_active,
    monthly_base_cents,
    monthly_platform_fee_cents,
    yearly_base_cents,
    usage_premium_basis_points,
    included_tokens_monthly,
    included_seats,
    min_seats,
    workspace_mode,
    max_seats,
    supports_priority,
    supports_collaboration,
    supports_workspace_admin,
    supports_team_workspace,
    supports_sso,
    created_at,
    updated_at
FROM billing_plans
ORDER BY plan_rank ASC, plan_code ASC
LIMIT ?;
