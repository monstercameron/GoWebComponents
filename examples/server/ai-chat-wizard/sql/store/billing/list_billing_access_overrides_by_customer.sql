SELECT
    id,
    customer_id,
    override_key,
    override_value,
    reason,
    is_enabled,
    starts_at,
    ends_at,
    actor_user_id,
    created_at,
    updated_at
FROM billing_access_overrides
WHERE customer_id = ?
ORDER BY updated_at DESC, id DESC;
