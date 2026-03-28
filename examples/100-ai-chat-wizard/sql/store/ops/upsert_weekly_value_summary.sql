INSERT INTO weekly_value_summaries (
    workspace_id,
    user_id,
    summary_week,
    summary_text,
    metrics_json,
    sent_at,
    created_at
)
SELECT ?, ?, ?, ?, ?, ?, ?
WHERE (? = 0 OR EXISTS (SELECT 1 FROM workspaces WHERE id = ?))
  AND (? = 0 OR EXISTS (SELECT 1 FROM users WHERE id = ?))
ON CONFLICT(workspace_id, user_id, summary_week) DO UPDATE SET
    summary_text = excluded.summary_text,
    metrics_json = excluded.metrics_json,
    sent_at = excluded.sent_at;
