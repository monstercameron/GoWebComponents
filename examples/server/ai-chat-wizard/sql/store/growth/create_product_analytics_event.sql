INSERT INTO product_analytics_events (
    workspace_id,
    user_id,
    session_key,
    event_name,
    funnel_key,
    step_key,
    experiment_key,
    variant_key,
    event_props_json,
    created_at
)
SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
WHERE (? = 0 OR EXISTS (SELECT 1 FROM workspaces WHERE id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?))
  AND (TRIM(COALESCE(?, '')) = '' OR EXISTS (SELECT 1 FROM experiments WHERE experiment_key = ?));
