WITH customer_scope AS (
    SELECT id
    FROM billing_customers
    WHERE user_id = ?
    LIMIT 1
),
active_plan AS (
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
),
resolved_plan AS (
    SELECT ap.plan_code
    FROM active_plan ap
    UNION ALL
    SELECT 'pro'
    FROM customer_scope cs
    WHERE NOT EXISTS (SELECT 1 FROM active_plan)
    LIMIT 1
)
SELECT
    bpma.model_id AS model_id,
    bpma.is_default AS is_default,
    rp.plan_code AS plan_code
FROM resolved_plan rp
JOIN billing_plan_model_access bpma ON bpma.plan_code = rp.plan_code
WHERE bpma.is_enabled = 1
ORDER BY bpma.is_default DESC, bpma.model_id ASC;
