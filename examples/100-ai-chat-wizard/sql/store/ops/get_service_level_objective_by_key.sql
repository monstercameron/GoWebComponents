SELECT id, slo_key, service_name, objective_percent, window_days, error_budget_minutes, status_page_url, updated_at
FROM service_level_objectives
WHERE slo_key = ?
LIMIT 1;
