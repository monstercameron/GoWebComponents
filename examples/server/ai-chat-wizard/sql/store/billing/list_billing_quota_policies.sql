SELECT
    id,
    plan_code,
    quota_key,
    soft_limit_value,
    hard_limit_value,
    reset_interval,
    enforcement_mode,
    updated_at
FROM billing_quota_policies
ORDER BY id DESC
LIMIT ?;
