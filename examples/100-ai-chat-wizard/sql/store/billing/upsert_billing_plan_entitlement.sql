INSERT INTO billing_plan_entitlements (
    plan_code,
    entitlement_key,
    entitlement_value,
    updated_at
)
VALUES (?, ?, ?, ?)
ON CONFLICT(plan_code, entitlement_key) DO UPDATE SET
    entitlement_value = excluded.entitlement_value,
    updated_at = excluded.updated_at;
