SELECT
    id,
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
FROM billing_subscriptions
WHERE customer_id = ?
ORDER BY id DESC
LIMIT ?;
