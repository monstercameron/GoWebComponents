INSERT INTO service_level_objectives (
    slo_key,
    service_name,
    objective_percent,
    window_days,
    error_budget_minutes,
    status_page_url,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(slo_key) DO UPDATE SET
    service_name = excluded.service_name,
    objective_percent = excluded.objective_percent,
    window_days = excluded.window_days,
    error_budget_minutes = excluded.error_budget_minutes,
    status_page_url = excluded.status_page_url,
    updated_at = excluded.updated_at;
