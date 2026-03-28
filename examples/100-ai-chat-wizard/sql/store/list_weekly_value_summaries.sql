SELECT id, workspace_id, user_id, summary_week, summary_text, metrics_json, sent_at, created_at
FROM weekly_value_summaries
ORDER BY id DESC
LIMIT ?;
