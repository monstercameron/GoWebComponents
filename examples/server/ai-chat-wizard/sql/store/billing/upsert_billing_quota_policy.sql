INSERT INTO billing_quota_policies (
    plan_code,
    quota_key,
    soft_limit_value,
    hard_limit_value,
    reset_interval,
    enforcement_mode,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(plan_code, quota_key) DO UPDATE SET
    soft_limit_value = excluded.soft_limit_value,
    hard_limit_value = excluded.hard_limit_value,
    reset_interval = excluded.reset_interval,
    enforcement_mode = excluded.enforcement_mode,
    updated_at = excluded.updated_at;
