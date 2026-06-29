INSERT INTO billing_access_overrides (
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
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE EXISTS (SELECT 1 FROM billing_customers WHERE id = ?)
ON CONFLICT(customer_id, override_key) DO UPDATE SET
    override_value = excluded.override_value,
    reason = excluded.reason,
    is_enabled = excluded.is_enabled,
    starts_at = excluded.starts_at,
    ends_at = excluded.ends_at,
    actor_user_id = excluded.actor_user_id,
    updated_at = excluded.updated_at;
