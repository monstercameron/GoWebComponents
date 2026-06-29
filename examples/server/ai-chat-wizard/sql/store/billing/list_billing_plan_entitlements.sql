SELECT
    entitlement_key,
    entitlement_value,
    updated_at
FROM billing_plan_entitlements
WHERE plan_code = ?
ORDER BY entitlement_key ASC;
