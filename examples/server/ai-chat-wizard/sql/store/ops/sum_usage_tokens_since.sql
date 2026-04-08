SELECT COALESCE(SUM(prompt_tokens + completion_tokens), 0)
FROM usage_events
WHERE user_id = ?
  AND created_at >= ?;
