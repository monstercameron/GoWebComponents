SELECT id, workspace_id, user_id, session_key, event_name, funnel_key, step_key, experiment_key, variant_key, event_props_json, created_at
FROM product_analytics_events
WHERE user_id = ?
ORDER BY id DESC
LIMIT ?;
