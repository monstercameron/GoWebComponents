INSERT INTO billing_plan_overages (
    plan_code,
    meter_key,
    included_units,
    soft_limit_units,
    hard_limit_units,
    overage_unit_size,
    overage_price_cents,
    billing_interval,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(plan_code, meter_key) DO UPDATE SET
    included_units = excluded.included_units,
    soft_limit_units = excluded.soft_limit_units,
    hard_limit_units = excluded.hard_limit_units,
    overage_unit_size = excluded.overage_unit_size,
    overage_price_cents = excluded.overage_price_cents,
    billing_interval = excluded.billing_interval,
    updated_at = excluded.updated_at;
