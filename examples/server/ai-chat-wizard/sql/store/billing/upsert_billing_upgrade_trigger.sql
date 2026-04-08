INSERT INTO billing_upgrade_triggers (
    plan_code,
    trigger_key,
    threshold_percent,
    upgrade_plan_code,
    message,
    cta_label,
    cta_url,
    is_enabled,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(plan_code, trigger_key) DO UPDATE SET
    threshold_percent = excluded.threshold_percent,
    upgrade_plan_code = excluded.upgrade_plan_code,
    message = excluded.message,
    cta_label = excluded.cta_label,
    cta_url = excluded.cta_url,
    is_enabled = excluded.is_enabled,
    updated_at = excluded.updated_at;
