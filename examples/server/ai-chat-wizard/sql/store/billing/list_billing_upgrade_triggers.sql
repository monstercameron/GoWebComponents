SELECT
    id,
    plan_code,
    trigger_key,
    threshold_percent,
    upgrade_plan_code,
    message,
    cta_label,
    cta_url,
    is_enabled,
    updated_at
FROM billing_upgrade_triggers
ORDER BY id DESC
LIMIT ?;
