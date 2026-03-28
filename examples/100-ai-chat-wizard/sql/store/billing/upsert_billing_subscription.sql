INSERT INTO billing_subscriptions (
    customer_id,
    provider_id,
    provider_subscription_id,
    plan_code,
    price_code,
    status,
    billing_interval,
    quantity,
    current_period_start,
    current_period_end,
    cancel_at_period_end,
    canceled_at,
    trial_ends_at,
    access_expires_at,
    created_at,
    updated_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM billing_customers WHERE id = ?)
  AND EXISTS (SELECT 1 FROM billing_plans WHERE plan_code = ?)
ON CONFLICT(provider_subscription_id) DO UPDATE SET
    customer_id = excluded.customer_id,
    provider_id = excluded.provider_id,
    plan_code = excluded.plan_code,
    price_code = excluded.price_code,
    status = excluded.status,
    billing_interval = excluded.billing_interval,
    quantity = excluded.quantity,
    current_period_start = excluded.current_period_start,
    current_period_end = excluded.current_period_end,
    cancel_at_period_end = excluded.cancel_at_period_end,
    canceled_at = excluded.canceled_at,
    trial_ends_at = excluded.trial_ends_at,
    access_expires_at = excluded.access_expires_at,
    updated_at = excluded.updated_at;
