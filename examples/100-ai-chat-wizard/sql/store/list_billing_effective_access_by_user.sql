WITH customer_scope AS (
    SELECT id
    FROM billing_customers
    WHERE user_id = ?
    LIMIT 1
),
current_plan AS (
    SELECT bs.plan_code
    FROM billing_subscriptions bs
    JOIN customer_scope cs ON cs.id = bs.customer_id
    WHERE bs.status IN ('active', 'trialing', 'past_due')
    ORDER BY
        CASE bs.status
            WHEN 'active' THEN 0
            WHEN 'trialing' THEN 1
            ELSE 2
        END ASC,
        COALESCE(NULLIF(bs.current_period_end, ''), '9999-12-31T23:59:59Z') DESC,
        bs.id DESC
    LIMIT 1
)
SELECT
    bpe.entitlement_key AS access_key,
    bpe.entitlement_value AS access_value,
    'plan' AS source_type,
    bpe.updated_at AS source_updated_at,
    '' AS reason
FROM current_plan cp
JOIN billing_plan_entitlements bpe ON bpe.plan_code = cp.plan_code
UNION ALL
SELECT
    bao.override_key AS access_key,
    bao.override_value AS access_value,
    'override' AS source_type,
    bao.updated_at AS source_updated_at,
    bao.reason AS reason
FROM customer_scope cs
JOIN billing_access_overrides bao ON bao.customer_id = cs.id
WHERE bao.is_enabled = 1
  AND (TRIM(bao.starts_at) = '' OR bao.starts_at <= ?)
  AND (TRIM(bao.ends_at) = '' OR bao.ends_at >= ?)
ORDER BY access_key ASC, source_type DESC, source_updated_at DESC;
