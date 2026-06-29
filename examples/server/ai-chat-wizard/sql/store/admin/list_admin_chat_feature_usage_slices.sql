SELECT
    pae.workspace_id,
    COALESCE(NULLIF(TRIM(pae.event_name), ''), 'unknown_feature') AS feature_key,
    COUNT(*) AS event_count,
    COUNT(DISTINCT pae.user_id) AS user_count,
    COALESCE(MAX(pae.created_at), '') AS last_seen_at
FROM product_analytics_events pae
WHERE pae.created_at >= ?
GROUP BY pae.workspace_id, feature_key
ORDER BY event_count DESC, last_seen_at DESC
LIMIT ?;
