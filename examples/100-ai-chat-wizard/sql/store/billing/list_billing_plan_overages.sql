SELECT
    id,
    plan_code,
    meter_key,
    included_units,
    soft_limit_units,
    hard_limit_units,
    overage_unit_size,
    overage_price_cents,
    billing_interval,
    updated_at
FROM billing_plan_overages
ORDER BY id DESC
LIMIT ?;
