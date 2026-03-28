INSERT INTO billing_plans (
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
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(plan_code) DO UPDATE SET
    plan_name = excluded.plan_name,
    plan_rank = excluded.plan_rank,
    is_active = excluded.is_active,
    monthly_base_cents = excluded.monthly_base_cents,
    yearly_base_cents = excluded.yearly_base_cents,
    included_tokens_monthly = excluded.included_tokens_monthly,
    included_seats = excluded.included_seats,
    max_seats = excluded.max_seats,
    supports_priority = excluded.supports_priority,
    supports_team_workspace = excluded.supports_team_workspace,
    supports_sso = excluded.supports_sso,
    updated_at = excluded.updated_at;
